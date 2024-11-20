package domain

import "github.com/google/uuid"

type Repository[T any] interface {
	Create(*T) error
	FindAll() ([]T, error)
	FindByID(uuid.UUID) (T, error)
	Delete(uuid.UUID) error
	GetBy(interface{}, ...interface{}) (T, error)
	GetMany(interface{}, ...interface{}) ([]T, error)
}
