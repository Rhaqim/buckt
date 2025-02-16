package domain_old

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BucktRepository[T any] interface {
	Create(*T) error
	Update(*T) error
	Delete(uuid.UUID) error
	GetAll() ([]T, error)
	GetByID(uuid.UUID) (T, error)
	GetBy(interface{}, ...interface{}) (T, error)
	GetMany(interface{}, ...interface{}) ([]T, error)
	RawQuery(string, ...interface{}) *gorm.DB
}
