package mocks

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/stretchr/testify/mock"
)

type CacheManager struct {
	mock.Mock
}

func NewCacheManager() domain.CacheManager {
	return &CacheManager{}
}

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
