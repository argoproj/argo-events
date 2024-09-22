package naivewatcher

import (
	"sync"
)

// Mutex is the extended mutex
type Mutex struct {
	in      sync.RWMutex
	locking bool
}

// Lock locks the mutex
func (m *Mutex) Lock() {
	m.in.Lock()
	defer m.in.Unlock()

	m.locking = true
}

// Unlock unlocks the mutex
func (m *Mutex) Unlock() {
	m.in.Lock()
	defer m.in.Unlock()

	m.locking = false
}

// TryLock returns true if it gets the lock, otherwise returns false
func (m *Mutex) TryLock() bool {
	m.in.Lock()
	defer m.in.Unlock()

	if m.locking {
		return false
	}
	m.locking = true
	return true
}

// IsLocked returs whether the mutex is locked
func (m *Mutex) IsLocked() bool {
	m.in.RLock()
	defer m.in.RUnlock()

	return m.locking
}
