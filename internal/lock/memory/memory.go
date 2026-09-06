package memory

import (
	"context"
	"sync"

	"control-plane/internal/lock"
)

// MemoryLock is an in-process lock.Locker useful for tests or single-node
// deployments. It behaves like a sync.Mutex per key.
type MemoryLock struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// New creates an in-memory locker.
func New() *MemoryLock {
	return &MemoryLock{
		locks: make(map[string]*sync.Mutex),
	}
}

func (m *MemoryLock) Lock(ctx context.Context, key string) (lock.UnlockFunc, error) {
	m.mu.Lock()
	if m.locks[key] == nil {
		m.locks[key] = &sync.Mutex{}
	}
	mu := m.locks[key]
	m.mu.Unlock()

	mu.Lock()
	return func() error {
		mu.Unlock()
		return nil
	}, nil
}
