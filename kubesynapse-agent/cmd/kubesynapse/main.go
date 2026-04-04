package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/KubeSynapse/kubesynapse/internal/config"
	"github.com/KubeSynapse/kubesynapse/internal/dispatcher"
	"github.com/KubeSynapse/kubesynapse/internal/health"
	"github.com/KubeSynapse/kubesynapse/internal/sink/webhook"
	"github.com/KubeSynapse/kubesynapse/internal/watcher"
	"github.com/KubeSynapse/kubesynapse/pkg/logger"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	// Initialize context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Configuration error: %v\n", err)
		os.Exit(1)
	}

	logger.Info("kubesynapse-agent booting up...", 
		"version", "2.0-enterprise",
		"cluster", cfg.ClusterName)

	// 2. Initialize Kubernetes client
	config, err := buildKubeConfig(cfg.KubeconfigPath)
	if err != nil {
		logger.Error("Failed to build kubeconfig", "error", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("Failed to create Kubernetes client", "error", err)
		os.Exit(1)
	}
	logger.Info("Kubernetes client initialized")

	// Initialize Metrics client
	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		logger.Warn("Failed to initialize metrics client (real-time metrics will be disabled)", "error", err)
	}

	// 3. Initialize health and metrics server
	healthServer := health.New(cfg.HealthPort)
	healthServer.Start()
	logger.Info("Health server started", "port", cfg.HealthPort)

	// 4. Initialize Dispatcher and modular components
	d := dispatcher.New()

	// Add Watcher Source
	w := watcher.New(clientset, metricsClient, cfg, healthServer)
	d.AddSource(w)

	// Add Webhook Sink
	whSink := webhook.New(cfg.WebhookURL, cfg.WebhookHeaders)
	d.AddSink(whSink)

	// 5. Run Dispatcher
	logger.Info("Starting event dispatcher...")
	if err := d.Run(ctx); err != nil {
		logger.Error("Dispatcher failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Shutdown complete")
}

func buildKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var k8sConfig *rest.Config
	var err error

	if kubeconfigPath != "" {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	} else {
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			home, _ := os.UserHomeDir()
			defaultPath := home + "/.kube/config"
			k8sConfig, err = clientcmd.BuildConfigFromFlags("", defaultPath)
			if err != nil {
				return nil, err
			}
		}
	}

	return kubernetes.NewForConfig(k8sConfig)
}

func buildKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		home, _ := os.UserHomeDir()
		defaultPath := home + "/.kube/config"
		return clientcmd.BuildConfigFromFlags("", defaultPath)
	}
	return config, nil
}
