package mocks

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/stretchr/testify/mock"
)

type CacheManager struct {
	mock.Mock
}

var _ domain.CacheManager = (*CacheManager)(nil)

func (m *CacheManager) SetBucktValue(key string, value any) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *CacheManager) GetBucktValue(key string) (any, error) {
	args := m.Called(key)
	return args.Get(0), args.Error(1)
}

func (m *CacheManager) DeleteBucktValue(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

type LRUCache struct{}

var _ domain.LRUCache = (*LRUCache)(nil)

func (m *LRUCache) Close()                            {}
func (M *LRUCache) Get(key string) ([]byte, bool)     { return nil, false }
func (M *LRUCache) Add(key string, value []byte) bool { return true }
func (M *LRUCache) Hits() uint64                      { return 0 }
func (M *LRUCache) Misses() uint64                    { return 0 }
