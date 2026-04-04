// Package collector gathers diagnostic data from Kubernetes for a crashing pod.
// It fetches: pod events, previous container logs, resource usage, and pod describe info.
package collector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/KubeSynapse/kubesynapse/internal/config"
	"github.com/KubeSynapse/kubesynapse/internal/models"
	"github.com/KubeSynapse/kubesynapse/internal/redactor"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Collector gathers diagnostic data about crashing pods.
type Collector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv1beta1.Clientset
	cfg           *config.Config
	redactor      *redactor.Redactor
}

// New creates a new Collector.
func New(clientset *kubernetes.Clientset, metricsClient *metricsv1beta1.Clientset, cfg *config.Config) *Collector {
	return &Collector{
		clientset:     clientset,
		metricsClient: metricsClient,
		cfg:           cfg,
		redactor:      redactor.New(),
	}
}

// Collect gathers all diagnostic data for a pod and returns redacted results.
func (c *Collector) Collect(ctx context.Context, pod *corev1.Pod) models.Diagnostics {
	namespace := pod.Namespace
	podName := pod.Name

	diag := models.Diagnostics{}

	// Metrics and Logs (Actionable data for n8n)
	diag.RealTimeMetrics = c.collectRealTimeMetrics(ctx, namespace, podName)
	diag.CrashLogs = c.collectPreviousLogs(ctx, namespace, podName, pod)
	
	// Environment and State
	diag.Events = c.collectEvents(ctx, namespace, podName)
	diag.ResourceUsage = c.collectResourceUsage(pod)
	diag.PodDescribe = c.collectPodDescribe(pod)
	diag.Conditions = c.collectConditions(pod)
	diag.InfrastructureAudit = c.collectAudit(ctx, pod)

	// Redact all collected data
	diag.Events = c.redactor.Redact(diag.Events)
	diag.CrashLogs = c.redactor.Redact(diag.CrashLogs)
	diag.ResourceUsage = c.redactor.Redact(diag.ResourceUsage)
	diag.PodDescribe = c.redactor.Redact(diag.PodDescribe)
	diag.Conditions = c.redactor.Redact(diag.Conditions)

	return diag
}

// collectRealTimeMetrics fetches current resource usage from metrics server.
func (c *Collector) collectRealTimeMetrics(ctx context.Context, namespace, podName string) models.RealTimeMetrics {
	if c.metricsClient == nil {
		return models.RealTimeMetrics{}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	metrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return models.RealTimeMetrics{}
	}

	var cpu, mem int64
	for _, container := range metrics.Containers {
		cpu += container.Usage.Cpu().MilliValue()
		mem += container.Usage.Memory().Value()
	}

	return models.RealTimeMetrics{
		CPU:    fmt.Sprintf("%dm", cpu),
		Memory: fmt.Sprintf("%dMi", mem/(1024*1024)),
		Window: metrics.Window.String(),
	}
}

// collectEvents fetches recent events related to the pod.
func (c *Collector) collectEvents(ctx context.Context, namespace, podName string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		return fmt.Sprintf("Error fetching events: %v", err)
	}

	if len(events.Items) == 0 {
		return "No events found"
	}

	var sb strings.Builder
	// Show last 20 events
	start := 0
	if len(events.Items) > 20 {
		start = len(events.Items) - 20
	}
	for _, event := range events.Items[start:] {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			event.LastTimestamp.Format(time.RFC3339),
			event.Type,
			event.Reason,
			event.Source.Component,
			event.Message,
		))
	}

	return sb.String()
}

// collectPreviousLogs fetches logs from the previous container instance (crash logs).
func (c *Collector) collectPreviousLogs(ctx context.Context, namespace, podName string, pod *corev1.Pod) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Find the main container
	containerName := ""
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}
	if containerName == "" {
		return "No containers found in pod"
	}

	tailLines := int64(100)
	if c.cfg != nil && c.cfg.LogTailLines > 0 {
		tailLines = int64(c.cfg.LogTailLines)
	}
	logOpts := &corev1.PodLogOptions{
		Container: containerName,
		Previous:  true,
		TailLines: &tailLines,
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		// Try current logs if previous not available
		logOpts.Previous = false
		req = c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
		stream, err = req.Stream(ctx)
		if err != nil {
			return fmt.Sprintf("No logs available: %v", err)
		}
	}
	defer stream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stream); err != nil {
		return fmt.Sprintf("Error reading logs: %v", err)
	}

	logs := buf.String()
	if logs == "" {
		return "No log output"
	}

	return logs
}

// collectResourceUsage extracts resource requests/limits from the pod spec.
func (c *Collector) collectResourceUsage(pod *corev1.Pod) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-30s %-15s %-15s %-15s %-15s\n",
		"CONTAINER", "CPU-REQ", "CPU-LIM", "MEM-REQ", "MEM-LIM"))

	for _, container := range pod.Spec.Containers {
		cpuReq := "none"
		cpuLim := "none"
		memReq := "none"
		memLim := "none"

		if req, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
			cpuReq = req.String()
		}
		if lim, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
			cpuLim = lim.String()
		}
		if req, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			memReq = req.String()
		}
		if lim, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
			memLim = lim.String()
		}

		sb.WriteString(fmt.Sprintf("%-30s %-15s %-15s %-15s %-15s\n",
			container.Name, cpuReq, cpuLim, memReq, memLim))
	}

	return sb.String()
}

// collectPodDescribe creates a human-readable summary of the pod state.
func (c *Collector) collectPodDescribe(pod *corev1.Pod) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Name:      %s\n", pod.Name))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n", pod.Namespace))
	sb.WriteString(fmt.Sprintf("Node:      %s\n", pod.Spec.NodeName))
	sb.WriteString(fmt.Sprintf("Phase:     %s\n", pod.Status.Phase))
	sb.WriteString(fmt.Sprintf("Start:     %s\n", pod.Status.StartTime))

	// Container statuses
	sb.WriteString("\nContainer Statuses:\n")
	for _, cs := range pod.Status.ContainerStatuses {
		sb.WriteString(fmt.Sprintf("  %s:\n", cs.Name))
		sb.WriteString(fmt.Sprintf("    Ready:         %v\n", cs.Ready))
		sb.WriteString(fmt.Sprintf("    Restart Count: %d\n", cs.RestartCount))

		if cs.State.Waiting != nil {
			sb.WriteString(fmt.Sprintf("    State:         Waiting (%s: %s)\n",
				cs.State.Waiting.Reason, cs.State.Waiting.Message))
		}
		if cs.State.Running != nil {
			sb.WriteString(fmt.Sprintf("    State:         Running (since %s)\n",
				cs.State.Running.StartedAt.Format(time.RFC3339)))
		}
		if cs.State.Terminated != nil {
			sb.WriteString(fmt.Sprintf("    State:         Terminated (reason: %s, exit: %d)\n",
				cs.State.Terminated.Reason, cs.State.Terminated.ExitCode))
		}

		// Last termination state (crucial for crash analysis)
		if cs.LastTerminationState.Terminated != nil {
			lt := cs.LastTerminationState.Terminated
			sb.WriteString("    Last Terminated:\n")
			sb.WriteString(fmt.Sprintf("      Reason:    %s\n", lt.Reason))
			sb.WriteString(fmt.Sprintf("      Exit Code: %d\n", lt.ExitCode))
			sb.WriteString(fmt.Sprintf("      Signal:    %d\n", lt.Signal))
			sb.WriteString(fmt.Sprintf("      Started:   %s\n", lt.StartedAt.Format(time.RFC3339)))
			sb.WriteString(fmt.Sprintf("      Finished:  %s\n", lt.FinishedAt.Format(time.RFC3339)))
			if lt.Message != "" {
				sb.WriteString(fmt.Sprintf("      Message:   %s\n", lt.Message))
			}
		}
	}

	return sb.String()
}

// collectConditions extracts pod conditions (Ready, Scheduled, etc.).
func (c *Collector) collectConditions(pod *corev1.Pod) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-25s %-10s %-30s %s\n", "TYPE", "STATUS", "REASON", "MESSAGE"))

	for _, cond := range pod.Status.Conditions {
		sb.WriteString(fmt.Sprintf("%-25s %-10s %-30s %s\n",
			cond.Type, cond.Status, cond.Reason, cond.Message))
	}

	return sb.String()
}
// collectAudit performs a proactive configuration audit of the pod's infrastructure.
func (c *Collector) collectAudit(ctx context.Context, pod *corev1.Pod) models.InfrastructureAudit {
	audit := models.InfrastructureAudit{
		BestPracticeViolations: []string{},
	}

	audit.HPAStatus = c.collectHPAStatus(ctx, pod)
	audit.PDBStatus = c.collectPDBStatus(ctx, pod)
	audit.ProbesConfig, audit.BestPracticeViolations = c.auditProbes(pod)

	return audit
}

// collectHPAStatus checks if an HPA is targeting the pod's owner.
func (c *Collector) collectHPAStatus(ctx context.Context, pod *corev1.Pod) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	hpas, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("Error checking HPA: %v", err)
	}

	// Try to find if any HPA targets this pod's owners
	for _, hpa := range hpas.Items {
		for _, owner := range pod.OwnerReferences {
			if hpa.Spec.ScaleTargetRef.Name == owner.Name && hpa.Spec.ScaleTargetRef.Kind == owner.Kind {
				return fmt.Sprintf("Active (Min: %d, Max: %d, Current: %d)",
					*hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas, hpa.Status.CurrentReplicas)
			}
		}
	}

	return "Missing (No HPA found targeting this resource)"
}

// collectPDBStatus checks if a PDB is covering the pod's labels.
func (c *Collector) collectPDBStatus(ctx context.Context, pod *corev1.Pod) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pdbs, err := c.clientset.PolicyV1().PodDisruptionBudgets(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("Error checking PDB: %v", err)
	}

	for _, pdb := range pdbs.Items {
		selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			return fmt.Sprintf("Active (Min Available: %s, Current Healthy: %d)",
				pdb.Spec.MinAvailable.String(), pdb.Status.CurrentHealthy)
		}
	}

	return "Missing (Pod not covered by any PodDisruptionBudget)"
}

// auditProbes validates liveness/readiness configuration.
func (c *Collector) auditProbes(pod *corev1.Pod) (string, []string) {
	var sb strings.Builder
	violations := []string{}

	for _, container := range pod.Spec.Containers {
		sb.WriteString(fmt.Sprintf("Container %s:\n", container.Name))

		if container.LivenessProbe != nil {
			sb.WriteString("  Liveness:  Configured\n")
		} else {
			sb.WriteString("  Liveness:  MISSING\n")
			violations = append(violations, fmt.Sprintf("Container %s missing LivenessProbe", container.Name))
		}

		if container.ReadinessProbe != nil {
			sb.WriteString("  Readiness: Configured\n")
		} else {
			sb.WriteString("  Readiness: MISSING\n")
			violations = append(violations, fmt.Sprintf("Container %s missing ReadinessProbe", container.Name))
		}
	}

	return sb.String(), violations
}
