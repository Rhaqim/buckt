package cache

import (
	"errors"

	"github.com/Rhaqim/buckt/internal/domain"
)

type NoOpCache struct{}

func NewNoOpCache() domain.CacheManager {
	return &NoOpCache{}
}

func (n *NoOpCache) SetBucktValue(key string, value any) error { return nil }
func (n *NoOpCache) GetBucktValue(key string) (any, error)     { return nil, errors.New("cache miss") }
func (n *NoOpCache) DeleteBucktValue(key string) error         { return nil }

func NewNoOpFileCache() domain.LRUCache {
	return &NoOpCache{}
}

func (n *NoOpCache) Add(key string, value []byte) (evicted bool) { return false }
func (n *NoOpCache) Get(key string) (value []byte, ok bool)      { return nil, false }
func (n *NoOpCache) Hits() uint64                                { return 0 }
func (n *NoOpCache) Misses() uint64                              { return 0 }
func (n *NoOpCache) Close()                                      {}
