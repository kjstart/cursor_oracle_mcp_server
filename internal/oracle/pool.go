// Package oracle: ExecutorPool manages multiple named Oracle connections.
package oracle

import (
	"context"
	"fmt"
	"sync"
)

// ExecutorPool holds multiple Executors by name (e.g. "source", "target").
type ExecutorPool struct {
	executors map[string]*Executor
	names     []string
	mu        sync.RWMutex
}

// NewExecutorPool creates a pool of executors from a name -> DSN map.
func NewExecutorPool(connections map[string]string) (*ExecutorPool, error) {
	if len(connections) == 0 {
		return nil, fmt.Errorf("at least one connection is required")
	}

	pool := &ExecutorPool{
		executors: make(map[string]*Executor),
		names:     make([]string, 0, len(connections)),
	}

	for name, dsn := range connections {
		ex, err := NewExecutor(dsn)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("connection %q: %w", name, err)
		}
		pool.executors[name] = ex
		pool.names = append(pool.names, name)
	}

	return pool, nil
}

// Close closes all connections in the pool.
func (p *ExecutorPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ex := range p.executors {
		ex.Close()
	}
	p.executors = nil
	p.names = nil
}

// Names returns the list of configured connection names.
func (p *ExecutorPool) Names() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, len(p.names))
	copy(out, p.names)
	return out
}

// Execute runs SQL on the named connection. If connectionName is "" and there is exactly one connection, that one is used.
func (p *ExecutorPool) Execute(ctx context.Context, connectionName string, sqlText string, statementType string) (*ExecutionResult, error) {
	name := connectionName
	if name == "" {
		p.mu.RLock()
		n := len(p.names)
		if n == 1 {
			name = p.names[0]
		}
		p.mu.RUnlock()
		if name == "" {
			return nil, fmt.Errorf("connection name is required when multiple databases are configured; use list_connections to see names")
		}
	}

	p.mu.RLock()
	ex, ok := p.executors[name]
	p.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown connection %q; use list_connections to see configured names", name)
	}

	return ex.Execute(ctx, sqlText, statementType)
}
