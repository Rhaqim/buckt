package domain

type CacheManager interface {
	// Set sets the value for the given key.
	Set(key string, value any) error

	// Get retrieves the value for the given key.
	Get(key string) (any, error)

	// Delete deletes the value for the given key.
	Delete(key string) error
}
