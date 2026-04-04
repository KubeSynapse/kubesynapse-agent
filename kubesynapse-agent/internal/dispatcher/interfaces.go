package dispatcher

import (
	"context"

	"github.com/KubeSynapse/kubesynapse/internal/models"
)

// Source represents a producer of pod incidents (e.g., Kubernetes Watcher).
type Source interface {
	// Run starts the source and produces incidents on the provided channel.
	Run(ctx context.Context, incidents chan<- models.PodIncident) error
	// Name returns a human-readable name for the source.
	Name() string
}

// Sink represents a consumer of pod incidents (e.g., Webhook, SMTP, Slack).
type Sink interface {
	// Handle processes a single pod incident.
	Handle(ctx context.Context, incident models.PodIncident) error
	// Name returns a human-readable name for the sink.
	Name() string
}
