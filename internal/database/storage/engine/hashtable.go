package engine

import "sync"

// HashTable — потокобезопасная хэш-таблица для одной партиции.
type HashTable struct {
	mu   sync.RWMutex
	data map[string]string
}

func newHashTable() *HashTable {
	return &HashTable{data: make(map[string]string)}
}

func (h *HashTable) Set(key, val string) {
	h.mu.Lock()
	h.data[key] = val
	h.mu.Unlock()
}

func (h *HashTable) Get(key string) (string, bool) {
	h.mu.RLock()
	v, ok := h.data[key]
	h.mu.RUnlock()
	return v, ok
}

func (h *HashTable) Del(key string) {
	h.mu.Lock()
	delete(h.data, key)
	h.mu.Unlock()
}
