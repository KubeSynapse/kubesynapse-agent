package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/KubeSynapse/kubesynapse/internal/models"
	"github.com/KubeSynapse/kubesynapse/pkg/logger"
)

// Sink dispatches webhook payloads to the configured endpoint.
type Sink struct {
	url        string
	headers    map[string]string
	httpClient *http.Client
	maxRetries int
}

// New creates a new webhook Sink.
func New(webhookURL string, headers map[string]string) *Sink {
	return &Sink{
		url:     webhookURL,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxRetries: 3,
	}
}

// Name returns the name of the sink.
func (s *Sink) Name() string {
	return "n8n-webhook-sink"
}

// Handle processes a single pod incident by sending a webhook.
func (s *Sink) Handle(ctx context.Context, incident models.PodIncident) error {
	payload := models.WebhookPayload{
		Microservice:  incident.Microservice,
		Namespace:     incident.Namespace,
		RestartCount:  incident.RestartCount,
		Reason:        incident.Reason,
		Cluster:       incident.Cluster,
		Pod:           incident.PodName,
		Timestamp:     incident.Timestamp.Format(time.RFC3339),
		Severity:      incident.Severity,
		RepositoryURL: incident.RepositoryURL,
		CommitID:      incident.CommitID,
		Diagnostics:   incident.Diagnostics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			logger.Info("Retrying webhook", "attempt", attempt, "max", s.maxRetries, "backoff", backoff)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "k8synapse-agent/2.0")

		for k, v := range s.headers {
			req.Header.Set(k, v)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			logger.Error("Webhook attempt failed", "attempt", attempt+1, "error", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Info("Webhook sent successfully", "url", s.url, "status", resp.StatusCode)
			return nil
		}

		logger.Error("Webhook attempt returned non-2xx status", "attempt", attempt+1, "status", resp.StatusCode)
	}

	return fmt.Errorf("webhook failed after %d retries", s.maxRetries)
}
