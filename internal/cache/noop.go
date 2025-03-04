package cache

import (
	"errors"

	"github.com/Rhaqim/buckt/internal/domain"
)

type NoOpCache struct{}

func NewNoOpCache() domain.CacheManager {
	return &NoOpCache{}
}

func (n *NoOpCache) Set(key string, value any) error { return nil }
func (n *NoOpCache) Get(key string) (any, error)     { return nil, errors.New("cache miss") }
func (n *NoOpCache) Delete(key string) error         { return nil }
