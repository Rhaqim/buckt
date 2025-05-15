package main

import (
	"fmt"
	"sync"

	"github.com/Rhaqim/buckt"
)

func main() {
	fileCacheConfig := buckt.FileCacheConfig{
		NumCounters: 1e7,     // 10M
		MaxCost:     1 << 30, // 1GB
		BufferItems: 64,
	}

	cacheConfig := buckt.CacheConfig{
		Manager:         NewCache(),
		FileCacheConfig: fileCacheConfig,
	}

	bucktInstance, err := buckt.Default(buckt.WithCache(cacheConfig))
	if err != nil {
		panic(err)
	}

	defer bucktInstance.Close()
}

type Cache struct {
	// Cache
	mu    sync.RWMutex
	store map[string]any
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[string]any),
	}
}

func (c *Cache) GetBucktValue(key string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println("Cache get", key)

	if val, ok := c.store[key]; ok {

		fmt.Println("Cache hit", key)

		return val, nil
	}

	return nil, nil
}

func (c *Cache) SetBucktValue(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Println("Cache set", key)

	c.store[key] = value
	return nil
}

func (c *Cache) DeleteBucktValue(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}
