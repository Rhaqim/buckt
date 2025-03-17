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
	Add(key any, value any) (evicted bool)
	Get(key any) (value any, ok bool)
	Purge()
}
