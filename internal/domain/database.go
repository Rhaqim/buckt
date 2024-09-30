package domain

type Repository[T any] interface {
	Create(*T) error
	FindAll() ([]T, error)
	FindByID(uint) (T, error)
	Delete(uint) error
}
