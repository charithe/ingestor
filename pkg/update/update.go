package update

import "context"

// Record describes a customer record to be processed
type Record struct {
	Id           int64
	Name         string
	Email        string
	MobileNumber string
}

// Updater is the interface exposed by the update service
type Updater interface {
	Update(ctx context.Context, record *Record) error
}

// Database is the interface exposed by the underlying storage service
type Database interface {
	Upsert(ctx context.Context, record *Record) error
	List(ctx context.Context) ([]*Record, error)
}
