package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"promocollection/internal/service/interfaces"
)

type MongoRepository[T any] struct {
	collection interfaces.MongoRepositoryInterface
}

func NewMongoRepository[T any](collection interfaces.MongoRepositoryInterface) *MongoRepository[T] {
	return &MongoRepository[T]{collection: collection}
}

func (r *MongoRepository[T]) Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {

	if result, err := r.collection.InsertOne(ctx, document); err != nil {
		return nil, err
	} else {
		return result, nil
	}

}

// Read a document by filter
func (r *MongoRepository[T]) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (T, error) {

	var result T

	if err := r.collection.FindOne(ctx, filter, opt).Decode(&result); err != nil {
		return result, err
	}

	return result, nil

}

func (r *MongoRepository[T]) Aggregate(ctx context.Context, pipeline interface{}, result interface{}) error {

	if cursor, err := r.collection.Aggregate(ctx, pipeline); err != nil {
		return err
	} else {
		defer func() {
			if err := cursor.Close(ctx); err != nil {
				_ = err
			}
		}()

		if cursor.Next(ctx) {
			return cursor.Decode(result)
		}
		return mongo.ErrNoDocuments
	}
}

// Update a document
func (r *MongoRepository[T]) UpdateOne(ctx context.Context, filter interface{}, update interface{}) error {

	if updateResult, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": update}); err != nil {
		return err
	} else {
		_ = updateResult
		return nil
	}

}

// Delete a document
func (r *MongoRepository[T]) Delete(ctx context.Context, filter interface{}) error {

	if deleteResult, err := r.collection.DeleteOne(ctx, filter); err != nil {
		return err
	} else {
		_ = deleteResult
		return nil
	}
}

func (r *MongoRepository[T]) FindAll(ctx context.Context, filter interface{}) ([]T, error) {

	if cursor, err := r.collection.Find(ctx, bson.M{}); err != nil {
		return nil, err
	} else {
		defer func() {
			if err := cursor.Close(ctx); err != nil {
				_ = err
			}
		}()

		var results []T
		for cursor.Next(ctx) {
			var entity T
			if err := cursor.Decode(&entity); err != nil {
				return nil, err
			}
			results = append(results, entity)
		}
		return results, nil
	}
}

func (r *MongoRepository[T]) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {

	if count, err := r.collection.CountDocuments(ctx, filter); err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (r *MongoRepository[T]) Find(ctx context.Context, filter interface{}) ([]T, error) {

	if cursor, err := r.collection.Find(ctx, filter); err != nil {
		return nil, err
	} else {
		defer func() {
			if err := cursor.Close(ctx); err != nil {
				_ = err
			}
		}()

		var results []T
		for cursor.Next(ctx) {
			var entity T
			if err := cursor.Decode(&entity); err != nil {
				return nil, err
			}
			results = append(results, entity)
		}
		return results, nil
	}
}

// updatemany
func (r *MongoRepository[T]) Update(
	ctx context.Context,
	filter interface{},
	update interface{},
) (*mongo.UpdateResult, error) {

	if result, err := r.collection.UpdateMany(ctx, filter, update); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func (r *MongoRepository[T]) AggregateAll(ctx context.Context, pipeline interface{}, result interface{}) error {

	if cursor, err := r.collection.Aggregate(ctx, pipeline); err != nil {
		return err
	} else {
		defer func() {
			if err := cursor.Close(ctx); err != nil {
				_ = err
			}
		}()

		return cursor.All(ctx, result)
	}

}
