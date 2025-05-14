package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DiskStorage implements the Storage interface using files on disk
type DiskStorage struct {
	dirPath string
	memory  *MemoryStorage // In-memory cache
	mutex   sync.RWMutex
}

// NewDiskStorage creates a new disk storage
func NewDiskStorage(dirPath string) (*DiskStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}

	ds := &DiskStorage{
		dirPath: dirPath,
		memory:  NewMemoryStorage(),
	}

	// Load existing data from disk
	if err := ds.loadFromDisk(); err != nil {
		return nil, err
	}

	return ds, nil
}

// loadFromDisk loads all keys from disk into memory
func (d *DiskStorage) loadFromDisk() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	files, err := os.ReadDir(d.dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(d.dirPath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Skip files that can't be read
			continue
		}

		var value Value
		if err := json.Unmarshal(data, &value); err != nil {
			// Skip files that can't be unmarshaled
			continue
		}

		// Check if the key has expired
		if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
			// Key has expired, delete the file
			os.Remove(filePath)
			continue
		}

		// Store in memory
		d.memory.Set(file.Name(), value)
	}

	return nil
}

// Get retrieves a value for the given key
func (d *DiskStorage) Get(key string) (Value, error) {
	// Try to get from memory first
	val, err := d.memory.Get(key)
	if err == nil {
		return val, nil
	}

	// If not in memory or expired, try to get from disk
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	filePath := filepath.Join(d.dirPath, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Value{}, ErrKeyNotFound
		}
		return Value{}, err
	}

	var value Value
	if err := json.Unmarshal(data, &value); err != nil {
		return Value{}, err
	}

	// Check for expiration
	if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
		// Key has expired, delete it
		d.mutex.RUnlock()
		d.Delete(key)
		d.mutex.RLock()
		return Value{}, ErrKeyExpired
	}

	// Update memory cache
	d.memory.Set(key, value)

	return value, nil
}

// Set stores a value for the given key
func (d *DiskStorage) Set(key string, value Value) error {
	// Set in memory
	if err := d.memory.Set(key, value); err != nil {
		return err
	}

	// Set on disk
	d.mutex.Lock()
	defer d.mutex.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	filePath := filepath.Join(d.dirPath, key)
	return os.WriteFile(filePath, data, 0644)
}

// Delete removes a key from the storage
func (d *DiskStorage) Delete(key string) error {
	// Delete from memory
	d.memory.Delete(key)

	// Delete from disk
	d.mutex.Lock()
	defer d.mutex.Unlock()

	filePath := filepath.Join(d.dirPath, key)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Has checks if a key exists in the storage
func (d *DiskStorage) Has(key string) bool {
	// Check in memory first
	if d.memory.Has(key) {
		return true
	}

	// Check on disk
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	filePath := filepath.Join(d.dirPath, key)
	_, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// Load the key into memory for future access
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	var value Value
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}

	// Check for expiration
	if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
		// Key has expired, delete it
		d.mutex.RUnlock()
		d.Delete(key)
		d.mutex.RLock()
		return false
	}

	// Update memory cache
	d.memory.Set(key, value)

	return true
}

// Keys returns all keys in the storage
func (d *DiskStorage) Keys() []string {
	// Refresh from disk first
	if err := d.loadFromDisk(); err != nil {
		return d.memory.Keys()
	}
	return d.memory.Keys()
}

// Clear removes all keys from the storage
func (d *DiskStorage) Clear() error {
	// Clear memory
	d.memory.Clear()

	// Clear disk
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Remove all files in the directory
	files, err := os.ReadDir(d.dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(d.dirPath, file.Name())
		if err := os.Remove(filePath); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the storage
func (d *DiskStorage) Close() error {
	// No specific close action needed for disk storage
	return nil
}