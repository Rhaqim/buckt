package cache

import (
	"sync/atomic"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/dgraph-io/ristretto/v2"
)

type FileCache struct {
	cache *ristretto.Cache[string, []byte]

	hits   atomic.Uint64
	misses atomic.Uint64
}

func NewFileCache(numCounters, maxCost, bufferItems int64) (domain.LRUCache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: numCounters, // number of keys to track frequency of (10M).
		MaxCost:     maxCost,     // maximum cost of cache (1GB).
		BufferItems: bufferItems, // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &FileCache{
		cache: cache,

		hits:   atomic.Uint64{},
		misses: atomic.Uint64{},
	}, nil
}

func (fc *FileCache) Add(key string, value []byte) (evicted bool) {
	if fc.cache != nil {
		cost := int64(len(value))

		_ = fc.cache.Set(key, value, cost)
	}
	return false
}

func (fc *FileCache) Get(key string) (value []byte, ok bool) {
	if fc.cache != nil {
		value, ok = fc.cache.Get(key)
		if ok {
			fc.hits.Add(1)
		} else {
			fc.misses.Add(1)
		}
		return value, ok
	}

	return nil, false
}

func (fc *FileCache) Hits() uint64 {
	return fc.hits.Load()
}

func (fc *FileCache) Misses() uint64 {
	return fc.misses.Load()
}

func (fc *FileCache) Close() {
	if fc.cache != nil {
		fc.cache.Close()
	}
}
