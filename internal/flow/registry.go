package flow

import (
	"context"
	"fmt"
	"sync"

	"github.com/SmrutAI/ingestion-pipeline/internal/core"
)

// FlowRegistry holds named flows and runs them by name.
// It is safe for concurrent use.
type FlowRegistry struct {
	mu    sync.RWMutex
	flows map[string]*Flow
}

// NewFlowRegistry returns an empty registry.
func NewFlowRegistry() *FlowRegistry {
	return &FlowRegistry{flows: make(map[string]*Flow)}
}

// Register adds a flow to the registry under its name.
// Returns an error if a flow with that name is already registered.
func (r *FlowRegistry) Register(f *Flow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.flows[f.name]; exists {
		return fmt.Errorf("flow registry: flow %q already registered", f.name)
	}
	r.flows[f.name] = f
	return nil
}

// Run executes the named flow and returns its stats.
// Returns an error if the flow name is not found.
func (r *FlowRegistry) Run(ctx context.Context, name string) (*core.FlowStats, error) {
	r.mu.RLock()
	f, ok := r.flows[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("flow registry: flow %q not found", name)
	}
	return f.Run(ctx)
}

// List returns the names of all registered flows.
// The order of names is not guaranteed.
func (r *FlowRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.flows))
	for k := range r.flows {
		names = append(names, k)
	}
	return names
}
