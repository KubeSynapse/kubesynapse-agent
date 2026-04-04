// Package models defines all shared data types for the k8synapse system.
package models

import "time"

// PodIncident represents a detected pod crash/failure event with full diagnostic context.
type PodIncident struct {
	// Pod identification
	Microservice  string `json:"microservice"`
	PodName       string `json:"pod"`
	Namespace     string `json:"namespace"`
	Cluster       string `json:"cluster"`
	NodeName      string `json:"node_name,omitempty"`
	RepositoryURL string `json:"repository_url,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`

	// Failure details
	Reason       string `json:"reason"`
	RestartCount int32  `json:"restart_count"`
	ExitCode     int32  `json:"exit_code,omitempty"`
	Signal       int32  `json:"signal,omitempty"`

	// Timestamps
	Timestamp     time.Time `json:"timestamp"`
	LastRestartAt time.Time `json:"last_restart_at,omitempty"`

	// Diagnostic data (redacted)
	Diagnostics Diagnostics `json:"diagnostics"`

	// Computed fields
	Severity string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
}

// InfrastructureAudit contains proactive configuration safety data.
type InfrastructureAudit struct {
	HPAStatus              string   `json:"hpa_status"`
	PDBStatus              string   `json:"pdb_status"`
	ProbesConfig           string   `json:"probes_config"`
	BestPracticeViolations []string `json:"best_practice_violations"`
}

type Diagnostics struct {
	Events              string              `json:"events"`
	CrashLogs           string              `json:"crash_logs"`
	ResourceUsage       string              `json:"resource_usage"`
	RealTimeMetrics     RealTimeMetrics     `json:"real_time_metrics"`
	PodDescribe         string              `json:"pod_describe"`
	Conditions          string              `json:"conditions,omitempty"`
	InfrastructureAudit InfrastructureAudit `json:"infrastructure_audit"`
}

// RealTimeMetrics holds the actual CPU/Memory usage at the time of the incident.
type RealTimeMetrics struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Window string `json:"window"`
}

// WebhookPayload is the payload sent to n8n/Jenkins webhooks.
type WebhookPayload struct {
	Microservice  string      `json:"microservice"`
	Namespace     string      `json:"namespace"`
	RestartCount  int32       `json:"restart_count"`
	Reason        string      `json:"reason"`
	Cluster       string      `json:"cluster"`
	Pod           string      `json:"pod"`
	Timestamp     string      `json:"timestamp"`
	Severity      string      `json:"severity"`
	RepositoryURL string      `json:"repository_url,omitempty"`
	CommitID      string      `json:"commit_id,omitempty"`
	Diagnostics   Diagnostics `json:"diagnostics"`
}
