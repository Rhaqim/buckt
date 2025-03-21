package cache

import (
	"github.com/Rhaqim/buckt/internal/domain"
	lru "github.com/hashicorp/golang-lru"
)

type FileCache struct {
	cache *lru.Cache
}

func NewFileCache(size int) domain.LRUCache {
	cache, _ := lru.New(size)
	return &FileCache{
		cache: cache,
	}
}

func (fc *FileCache) Add(key any, value any) (evicted bool) {
	if fc.cache != nil {
		fc.cache.Add(key, value)
	}
	return false
}

func (fc *FileCache) Get(key any) (value any, ok bool) {
	if fc.cache != nil {
		return fc.cache.Get(key)
	}
	return nil, false
}

func (fc *FileCache) Purge() {
	if fc.cache != nil {
		fc.cache.Purge()
	}
}
