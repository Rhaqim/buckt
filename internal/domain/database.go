package domain

import "github.com/google/uuid"

type Repository[T any] interface {
	Create(*T) error
	FindAll() ([]T, error)
	FindByID(uuid.UUID) (T, error)
	Delete(uuid.UUID) error
	GetBy(string, string) (T, error)
	GetMany(string, string) ([]T, error)
}
