package model

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

// MemoryStore is an in-memory implementation of Store. It uses a map for O(1)
// device lookup. All exported methods are safe for concurrent use.
type MemoryStore struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

// Compile-time assertion that *MemoryStore satisfies the Store interface.
var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates a MemoryStore pre-populated with the supplied device
// IDs. This is the preferred constructor for tests because it requires no
// filesystem access.
func NewMemoryStore(ids []string) *MemoryStore {
	devices := make(map[string]*Device, len(ids))
	for _, id := range ids {
		devices[id] = &Device{ID: id}
	}
	return &MemoryStore{devices: devices}
}

// NewMemoryStoreFromCSV creates a MemoryStore by reading device IDs from the
// first column of a CSV file. It returns an error if the file cannot be opened
// or parsed.
func NewMemoryStoreFromCSV(filePath string) (*MemoryStore, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	devices := make(map[string]*Device, len(records))
	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		id := record[0]
		devices[id] = &Device{ID: id}
	}
	return &MemoryStore{devices: devices}, nil
}

// GetDevice returns the Device with the given ID and true, or nil and false if
// no device with that ID exists in the store.
func (s *MemoryStore) GetDevice(id string) (*Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[id]
	return d, ok
}
