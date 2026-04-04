package dispatcher

import (
	"context"
	"sync"

	"github.com/KubeSynapse/kubesynapse/internal/models"
	"github.com/KubeSynapse/kubesynapse/pkg/logger"
)

// Dispatcher coordinates the flow of incidents from sources to sinks.
type Dispatcher struct {
	sources []Source
	sinks   []Sink
	mu      sync.RWMutex
}

// New creates a new Dispatcher.
func New() *Dispatcher {
	return &Dispatcher{
		sources: make([]Source, 0),
		sinks:   make([]Sink, 0),
	}
}

// AddSource adds a source to the dispatcher.
func (d *Dispatcher) AddSource(s Source) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sources = append(d.sources, s)
}

// AddSink adds a sink to the dispatcher.
func (d *Dispatcher) AddSink(s Sink) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sinks = append(d.sinks, s)
}

// Run starts all sources and dispatches received incidents to all sinks.
func (d *Dispatcher) Run(ctx context.Context) error {
	incidentCh := make(chan models.PodIncident, 100)
	var wg sync.WaitGroup

	// Start all sources
	for _, src := range d.sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			logger.Info("Starting source", "name", s.Name())
			if err := s.Run(ctx, incidentCh); err != nil {
				logger.Error("Source failed", "name", s.Name(), "error", err)
			}
		}(src)
	}

	// Main dispatch loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case incident, ok := <-incidentCh:
				if !ok {
					return
				}
				d.dispatch(ctx, incident)
			}
		}
	}()

	// Wait for all sources to finish (usually when context is cancelled)
	wg.Wait()
	close(incidentCh)
	logger.Info("Dispatcher stopped")
	return nil
}

// dispatch sends an incident to all registered sinks asynchronously.
func (d *Dispatcher) dispatch(ctx context.Context, incident models.PodIncident) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, sink := range d.sinks {
		go func(s Sink) {
			logger.Debug("Dispatching incident to sink", 
				"pod", incident.PodName, 
				"sink", s.Name())
			
			if err := s.Handle(ctx, incident); err != nil {
				logger.Error("Sink failed to handle incident", 
					"sink", s.Name(), 
					"pod", incident.PodName, 
					"error", err)
			}
		}(sink)
	}
}
