package ops

import "sync"

// Registry maps op names to their Op implementations.
// It is safe for concurrent use after all Register calls complete.
type Registry struct {
	mu  sync.RWMutex
	ops map[string]Op
}

func NewRegistry() *Registry {
	return &Registry{ops: make(map[string]Op)}
}

// Register adds op under name. Panics on duplicate (intended for init-time setup).
func (r *Registry) Register(name string, op Op) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.ops[name]; exists {
		panic("ops.Registry: duplicate registration for " + name)
	}
	r.ops[name] = op
}

// Get returns the Op registered under name, or nil if not found.
func (r *Registry) Get(name string) Op {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ops[name]
}

// All returns a snapshot of the registered ops keyed by name.
func (r *Registry) All() map[string]Op {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Op, len(r.ops))
	for k, v := range r.ops {
		out[k] = v
	}
	return out
}
