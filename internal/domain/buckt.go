package domain

import (
	"context"
	"io"
	"log"
)

type CacheManager interface {
	// Set sets the value for the given key.
	SetBucktValue(ctx context.Context, key string, value any) error

	// Get retrieves the value for the given key.
	GetBucktValue(ctx context.Context, key string) (any, error)

	// Delete deletes the value for the given key.
	DeleteBucktValue(ctx context.Context, key string) error
}

type LRUCache interface {
	Add(key string, value []byte) (evicted bool)
	Get(key string) (value []byte, ok bool)
	Hits() uint64
	Misses() uint64
	Close()
}

type BucktLogger interface {
	Errorf(format string, args ...any)
	Info(message string)
	Infof(format string, args ...any)
	Warn(message string)
	WrapError(message string, err error) error
	WrapErrorf(message string, err error, args ...any) error
	AddLogger(logger *log.Logger)
	GetLogger() *log.Logger
	Writer() io.Writer
}
