package domain

type CacheManager interface {
	// Set sets the value for the given key.
	SetBucktValue(key string, value any) error

	// Get retrieves the value for the given key.
	GetBucktValue(key string) (any, error)

	// Delete deletes the value for the given key.
	DeleteBucktValue(key string) error
}

type LRUCache interface {
	Add(key string, value []byte) (evicted bool)
	Get(key string) (value []byte, ok bool)
	Hits() uint64
	Misses() uint64
	Close()
}
