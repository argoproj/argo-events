package util

import "sync"

// Concurrent Safe String keyed map
type StringKeyedMap[T any] struct {
	items map[string]T
	lock  *sync.RWMutex
}

func NewStringKeyedMap[T any]() StringKeyedMap[T] {
	return StringKeyedMap[T]{
		items: make(map[string]T, 0),
		lock:  &sync.RWMutex{},
	}
}

func (sm *StringKeyedMap[T]) Store(key string, item T) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.items[key] = item
}

func (sm *StringKeyedMap[T]) Load(key string) (T, bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	item, ok := sm.items[key]
	return item, ok
}

func (sm *StringKeyedMap[T]) Delete(key string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.items, key)
}
