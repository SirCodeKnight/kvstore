package storage

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound is returned when a key is not found in the storage
	ErrKeyNotFound = errors.New("key not found")
	
	// ErrKeyExpired is returned when a key has expired
	ErrKeyExpired = errors.New("key expired")
)

// Value represents a value stored in the key-value store
type Value struct {
	Data        []byte
	Expiration  int64 // Unix timestamp in nanoseconds, 0 means no expiration
}

// Storage defines the interface for storage backends
type Storage interface {
	// Get retrieves a value for the given key
	Get(key string) (Value, error)
	
	// Set stores a value for the given key
	Set(key string, value Value) error
	
	// Delete removes a key from the storage
	Delete(key string) error
	
	// Has checks if a key exists in the storage
	Has(key string) bool
	
	// Keys returns all keys in the storage
	Keys() []string
	
	// Clear removes all keys from the storage
	Clear() error
	
	// Close closes the storage
	Close() error
}

// MemoryStorage implements the Storage interface using in-memory map
type MemoryStorage struct {
	data  map[string]Value
	mutex sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]Value),
	}
}

// Get retrieves a value for the given key
func (m *MemoryStorage) Get(key string) (Value, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	val, ok := m.data[key]
	if !ok {
		return Value{}, ErrKeyNotFound
	}
	
	// Check for expiration
	if val.Expiration > 0 && val.Expiration < time.Now().UnixNano() {
		// Key has expired, delete it
		m.mutex.RUnlock()
		m.Delete(key)
		m.mutex.RLock()
		return Value{}, ErrKeyExpired
	}
	
	return val, nil
}

// Set stores a value for the given key
func (m *MemoryStorage) Set(key string, value Value) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.data[key] = value
	return nil
}

// Delete removes a key from the storage
func (m *MemoryStorage) Delete(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.data, key)
	return nil
}

// Has checks if a key exists in the storage
func (m *MemoryStorage) Has(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	val, ok := m.data[key]
	if !ok {
		return false
	}
	
	// Check for expiration
	if val.Expiration > 0 && val.Expiration < time.Now().UnixNano() {
		return false
	}
	
	return true
}

// Keys returns all keys in the storage
func (m *MemoryStorage) Keys() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	keys := make([]string, 0, len(m.data))
	now := time.Now().UnixNano()
	
	for k, v := range m.data {
		// Skip expired keys
		if v.Expiration > 0 && v.Expiration < now {
			continue
		}
		keys = append(keys, k)
	}
	
	return keys
}

// Clear removes all keys from the storage
func (m *MemoryStorage) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.data = make(map[string]Value)
	return nil
}

// Close closes the storage
func (m *MemoryStorage) Close() error {
	m.Clear()
	return nil
}