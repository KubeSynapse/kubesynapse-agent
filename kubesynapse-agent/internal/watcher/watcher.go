// Package watcher monitors Kubernetes pods for crash events using the
// SharedInformer pattern. It detects CrashLoopBackOff, OOMKilled, Failed
// states, and high restart counts, then dispatches alerts through the
// configured pipeline (webhook → LLM → Jira/email).
package watcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/KubeSynapse/kubesynapse/internal/collector"
	"github.com/KubeSynapse/kubesynapse/internal/config"
	"github.com/KubeSynapse/kubesynapse/internal/dedup"
	"github.com/KubeSynapse/kubesynapse/internal/discovery"
	"github.com/KubeSynapse/kubesynapse/internal/health"
	"github.com/KubeSynapse/kubesynapse/internal/models"
	"github.com/KubeSynapse/kubesynapse/pkg/logger"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Watcher monitors Kubernetes pods and triggers the incident response pipeline.
type Watcher struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv1beta1.Clientset
	cfg           *config.Config
	collector     *collector.Collector
	dedupCache    dedup.Deduplicator
	health        *health.Server
}

// New creates a new Watcher with all dependencies initialized.
func New(clientset *kubernetes.Clientset, metricsClient *metricsv1beta1.Clientset, cfg *config.Config, healthServer *health.Server) *Watcher {
	var deduplicator dedup.Deduplicator

	// Discovery: Identify the cluster if not explicitly named
	if cfg.ClusterName == "" || cfg.ClusterName == "kubernetes-production" || cfg.ClusterName == "k8s-cluster" {
		cfg.ClusterName = discovery.DiscoverClusterName(clientset)
	}

	switch cfg.DeduplicationMode {
	case "redis":
		if cfg.RedisURL != "" {
			logger.Info("Connecting to Redis", "url", cfg.RedisURL, "tls", cfg.RedisUseTLS)
			rCache, err := dedup.NewRedis(cfg.RedisURL, cfg.RedisPassword, cfg.RedisUseTLS, int(cfg.CooldownSeconds))
			if err != nil {
				logger.Warn("Redis connection failed, falling back to memory", "error", err)
				deduplicator = dedup.NewInMemory(int(cfg.CooldownSeconds))
			} else {
				logger.Info("Connected to Redis for distributed deduplication")
				deduplicator = rCache
			}
		} else {
			logger.Warn("Redis URL not set, falling back to memory")
			deduplicator = dedup.NewInMemory(int(cfg.CooldownSeconds))
		}
	case "file":
		logger.Info("Using file-based persistence", "path", cfg.PersistencePath)
		deduplicator = dedup.NewFile(cfg.PersistencePath, int(cfg.CooldownSeconds))
	default:
		logger.Info("Using in-memory deduplication (local state only)")
		deduplicator = dedup.NewInMemory(int(cfg.CooldownSeconds))
	}

	return &Watcher{
		clientset:     clientset,
		metricsClient: metricsClient,
		cfg:           cfg,
		collector:     collector.New(clientset, metricsClient, cfg),
		dedupCache:    deduplicator,
		health:        healthServer,
	}
}

// Name returns the name of the source.
func (w *Watcher) Name() string {
	return "k8s-pod-watcher"
}

// Run starts watching pods in the configured namespaces.
// It sends detected incidents to the provided channel.
func (w *Watcher) Run(ctx context.Context, incidents chan<- models.PodIncident) error {
	namespaces := w.cfg.WatchNamespaces
	if len(namespaces) == 0 {
		namespaces = []string{""} // Empty string = all namespaces
	}

	logger.Info("Starting pod watcher",
		"namespaces", namespacesDisplay(namespaces),
		"restart_threshold", w.cfg.RestartThreshold,
		"cooldown", w.cfg.CooldownSeconds)

	// Create informer factory
	factory := informers.NewSharedInformerFactory(w.clientset, 30*time.Second)
	podInformer := factory.Core().V1().Pods().Informer()

	// Register event handlers
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod, ok := newObj.(*corev1.Pod)
			if !ok {
				return
			}

			// Filter by namespace if configured
			if !w.shouldWatchNamespace(pod.Namespace) {
				return
			}

			w.health.GetMetrics().PodsWatched.Add(1)
			w.checkPod(ctx, pod, incidents)
		},
	})

	// Mark as ready once informer syncs
	w.health.SetReady(true)

	// Start informer
	stopCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stopCh)
	}()

	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	logger.Info("Pod informer synced and watching")

	// Block until context is cancelled
	<-ctx.Done()
	logger.Info("Watcher shutting down...")
	return nil
}

// checkPod evaluates a pod for crash conditions.
func (w *Watcher) checkPod(ctx context.Context, pod *corev1.Pod, incidentCh chan<- models.PodIncident) {
	for _, cs := range pod.Status.ContainerStatuses {
		incident := w.detectIncident(pod, cs)
		if incident == nil {
			continue
		}

		// Deduplication check
		if !w.dedupCache.ShouldAlert(ctx, pod.Namespace, pod.Name) {
			logger.Debug("Skipping duplicate alert (cooldown active)",
				"namespace", pod.Namespace,
				"pod", pod.Name)
			return
		}

		logger.Warn("INCIDENT DETECTED",
			"namespace", pod.Namespace,
			"pod", pod.Name,
			"reason", incident.Reason,
			"restarts", incident.RestartCount)

		// Collect diagnostics (with redaction)
		logger.Info("Collecting diagnostics", "namespace", pod.Namespace, "pod", pod.Name)
		incident.Diagnostics = w.collector.Collect(ctx, pod)

		// Send to dispatcher
		incidentCh <- *incident

		w.health.GetMetrics().AlertsFired.Add(1)
		logger.Info("Incident telemetry collected & dispatched",
			"namespace", pod.Namespace,
			"pod", pod.Name)
		return // Only process first failing container per pod
	}
}

// detectIncident checks if a container status represents a crash/failure event.
func (w *Watcher) detectIncident(pod *corev1.Pod, cs corev1.ContainerStatus) *models.PodIncident {
	var reason string
	var exitCode int32
	var signal int32

	// Check 1: CrashLoopBackOff
	if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
		reason = "CrashLoopBackOff"
	}

	// Check 2: OOMKilled (from last termination state)
	if cs.LastTerminationState.Terminated != nil {
		lt := cs.LastTerminationState.Terminated
		exitCode = lt.ExitCode
		signal = lt.Signal

		if lt.Reason == "OOMKilled" {
			reason = "OOMKilled"
		} else if lt.Reason == "Error" {
			reason = "Error"
		}
	}

	// Check 3: Current terminated state
	if cs.State.Terminated != nil {
		t := cs.State.Terminated
		exitCode = t.ExitCode

		if t.Reason == "OOMKilled" {
			reason = "OOMKilled"
		} else if t.Reason == "Error" || t.ExitCode != 0 {
			if reason == "" {
				reason = fmt.Sprintf("Terminated(%s, exit=%d)", t.Reason, t.ExitCode)
			}
		}
	}

	// Check 4: High restart count
	if reason == "" && cs.RestartCount >= w.cfg.RestartThreshold {
		reason = fmt.Sprintf("HighRestartCount(%d)", cs.RestartCount)
	}

	// Check 5: Pod phase Failed
	if reason == "" && pod.Status.Phase == corev1.PodFailed {
		reason = "PodFailed"
	}

	// No incident detected
	if reason == "" {
		return nil
	}

	// Only trigger if restart count meets threshold
	if cs.RestartCount < w.cfg.RestartThreshold && reason != "OOMKilled" && reason != "PodFailed" {
		return nil
	}

	// Compute severity
	severity := computeSeverity(reason, cs.RestartCount)

	// Extract microservice name (label app or pod name prefix)
	microservice := extractMicroservice(pod)

	// Extract SCM metadata from annotations
	repoURL, commitID := extractSCMMetadata(pod)

	return &models.PodIncident{
		Microservice:  microservice,
		PodName:       pod.Name,
		Namespace:     pod.Namespace,
		Cluster:       w.cfg.ClusterName,
		NodeName:      pod.Spec.NodeName,
		RepositoryURL: repoURL,
		CommitID:      commitID,
		Reason:        reason,
		RestartCount:  cs.RestartCount,
		ExitCode:      exitCode,
		Signal:        signal,
		Timestamp:     time.Now(),
		Severity:      severity,
	}
}

// extractSCMMetadata looks for repo/commit info in pod annotations.
func extractSCMMetadata(pod *corev1.Pod) (string, string) {
	repo := pod.Annotations["app.kubernetes.io/repo"]
	if repo == "" {
		repo = pod.Annotations["git-repo"]
	}

	commit := pod.Annotations["app.kubernetes.io/commit"]
	if commit == "" {
		commit = pod.Annotations["git-commit"]
	}

	return repo, commit
}

// shouldWatchNamespace checks if the pod's namespace is in our watch list.
func (w *Watcher) shouldWatchNamespace(namespace string) bool {
	if len(w.cfg.WatchNamespaces) == 0 {
		return true // Watch all
	}
	for _, ns := range w.cfg.WatchNamespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

// computeSeverity determines incident severity based on reason and restart count.
func computeSeverity(reason string, restartCount int32) string {
	if reason == "OOMKilled" || restartCount > 10 {
		return "CRITICAL"
	}
	if reason == "CrashLoopBackOff" || restartCount > 5 {
		return "HIGH"
	}
	if restartCount > 3 {
		return "MEDIUM"
	}
	return "LOW"
}

// extractMicroservice gets the microservice name from pod labels or name.
func extractMicroservice(pod *corev1.Pod) string {
	// Try common labels
	for _, label := range []string{"app", "app.kubernetes.io/name", "service"} {
		if val, ok := pod.Labels[label]; ok {
			return val
		}
	}

	// Fall back to pod name prefix (strip the random suffix)
	name := pod.Name
	parts := strings.Split(name, "-")
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], "-")
	}
	return name
}

func namespacesDisplay(ns []string) string {
	if len(ns) == 0 || (len(ns) == 1 && ns[0] == "") {
		return "[all namespaces]"
	}
	return fmt.Sprintf("%v", ns)
}
