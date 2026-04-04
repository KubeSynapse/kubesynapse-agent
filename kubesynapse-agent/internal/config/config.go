// Package config loads and validates configuration from environment variables.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/KubeSynapse/kubesynapse/pkg/logger"
)

// Config holds all configuration for the k8synapse agent.
type Config struct {
	// Kubernetes
	WatchNamespaces  []string `json:"watch_namespaces" yaml:"watch_namespaces"`
	ClusterName      string   `json:"cluster_name" yaml:"cluster_name"`
	RedisURL         string   `json:"redis_url" yaml:"redis_url"`
	RedisPassword    string   `json:"redis_password" yaml:"redis_password"`
	RedisUseTLS      bool     `json:"redis_use_tls" yaml:"redis_use_tls"`
	KubeconfigPath   string   `json:"kubeconfig_path" yaml:"kubeconfig_path"`
	RestartThreshold int32    `json:"restart_threshold" yaml:"restart_threshold"`

	// Webhook
	WebhookURL      string            `json:"webhook_url" yaml:"webhook_url"`
	WebhookType     string            `json:"webhook_type" yaml:"webhook_type"`
	WebhookHeaders  map[string]string `json:"webhook_headers" yaml:"webhook_headers"`
	LogTailLines    int               `json:"log_tail_lines" yaml:"log_tail_lines"`

	// Deduplication
	DeduplicationMode string `json:"deduplication_mode" yaml:"deduplication_mode"`
	PersistencePath   string `json:"persistence_path" yaml:"persistence_path"`
	CooldownSeconds   int    `json:"cooldown_seconds" yaml:"cooldown_seconds"`

	// Observability
	HealthPort  int    `json:"health_port" yaml:"health_port"`
	MetricsPort int    `json:"metrics_port" yaml:"metrics_port"`
	LogLevel    string `json:"log_level" yaml:"log_level"`
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		ClusterName:      getEnv("CLUSTER_NAME", "kubernetes-cluster"),
		RedisURL:         getEnv("REDIS_URL", ""),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisUseTLS:      getEnvBool("REDIS_USE_TLS", false),
		KubeconfigPath:   getEnv("KUBECONFIG", ""),
		RestartThreshold: int32(getEnvInt("RESTART_THRESHOLD", 3)),

		WebhookURL:      getEnv("WEBHOOK_URL", "http://localhost:5678/webhook/k8s-incident"),
		WebhookType:     getEnv("WEBHOOK_TYPE", "n8n"),
		WebhookHeaders:  make(map[string]string),
		LogTailLines:    getEnvInt("LOG_TAIL_LINES", 100),

		DeduplicationMode: getEnv("DEDUPLICATION_MODE", "memory"),
		PersistencePath:   getEnv("PERSISTENCE_PATH", "/data/dedup.json"),
		CooldownSeconds:   getEnvInt("COOLDOWN_SECONDS", 300),

		HealthPort:  getEnvInt("HEALTH_PORT", 8080),
		MetricsPort: getEnvInt("METRICS_PORT", 9090),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	// Parse Custom Webhook Headers
	headersJSON := getEnv("WEBHOOK_HEADERS", "")
	if headersJSON != "" {
		if err := json.Unmarshal([]byte(headersJSON), &cfg.WebhookHeaders); err != nil {
			return nil, fmt.Errorf("failed to parse WEBHOOK_HEADERS: %v", err)
		}
	}

	// Parse namespaces
	ns := getEnv("WATCH_NAMESPACES", "")
	if ns != "" {
		cfg.WatchNamespaces = strings.Split(ns, ",")
		for i := range cfg.WatchNamespaces {
			cfg.WatchNamespaces[i] = strings.TrimSpace(cfg.WatchNamespaces[i])
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	logger.Info("Configuration loaded successfully", 
		"cluster", cfg.ClusterName, 
		"namespaces", cfg.WatchNamespaces,
		"webhook", cfg.WebhookURL)

	return cfg, nil
}

// Validate checks required configuration values.
func (c *Config) Validate() error {
	if c.WebhookURL == "" {
		return fmt.Errorf("WEBHOOK_URL is required")
	}

	return nil
}

// helper functions

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
