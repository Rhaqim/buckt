package domain

import "github.com/google/uuid"

type BucktRepository[T any] interface {
	Create(*T) error
	Update(*T) error
	Delete(uuid.UUID) error
	GetAll() ([]T, error)
	GetByID(uuid.UUID) (T, error)
	GetBy(interface{}, ...interface{}) (T, error)
	GetMany(interface{}, ...interface{}) ([]T, error)
}
