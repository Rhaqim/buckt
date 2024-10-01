package domain

type Repository[T any] interface {
	Create(*T) error
	FindAll() ([]T, error)
	FindByID(string) (T, error)
	Delete(string) error
	GetBy(string, string) (T, error)
}
