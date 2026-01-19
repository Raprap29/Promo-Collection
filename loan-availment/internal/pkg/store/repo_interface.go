package store

type RepositoryInterface[T any] interface {
	Create(entity T) (interface{}, error)
	Read(filter interface{}) (T, error)
	Update(filter interface{}, update T) error
	Delete(filter interface{}) error
	FindAll(filter interface{}) ([]T, error)
	Aggregate(pipeline interface{}, result interface{}) error
	CountDocuments(filter interface{}) (int64, error)
}
