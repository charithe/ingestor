package update

import (
	"context"
	"sync"
)

// InMemDB implements a primitive in-memory database using a hashmap
type InMemDB struct {
	mu      sync.RWMutex
	records map[string]*Record
}

// NewInMemDB creates a new instance of the database
func NewInMemDB() *InMemDB {
	return &InMemDB{

		records: make(map[string]*Record),
	}
}

// Upsert inserts or updates the given record into the database
func (db *InMemDB) Upsert(_ context.Context, rec *Record) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.records[rec.Email] = rec
	return nil
}

// List returns all the records in the database
func (db *InMemDB) List(_ context.Context) ([]*Record, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]*Record, len(db.records))
	i := 0

	for _, rec := range db.records {
		result[i] = rec
		i++
	}

	return result, nil
}
