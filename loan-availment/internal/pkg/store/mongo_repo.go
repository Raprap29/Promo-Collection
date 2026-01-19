package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository[T any] struct {
	collection *mongo.Collection
}

func NewMongoRepository[T any](collection *mongo.Collection) *MongoRepository[T] {
	return &MongoRepository[T]{collection: collection}
}

func (r *MongoRepository[T]) Create(document interface{}) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Read a document by filter
func (r *MongoRepository[T]) Read(filter interface{}) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result T
	err := r.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *MongoRepository[T]) Aggregate(pipeline interface{}, result interface{}) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		return cursor.Decode(result)
	}
	return mongo.ErrNoDocuments
}

func (r *MongoRepository[T]) AggregateAll(pipeline interface{}, result interface{}) error {

	ctx := context.Background()

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// if cursor.All(ctx) {
	return cursor.All(ctx, result)
	// }
	// return mongo.ErrNoDocuments
}

// Update a document
func (r *MongoRepository[T]) Update(filter interface{}, update interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return err
	}

	return nil
}

// Delete a document
func (r *MongoRepository[T]) Delete(filter interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}

func (r *MongoRepository[T]) FindAll(filter interface{}) ([]T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []T
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var entity T
		if err := cursor.Decode(&entity); err != nil {
			return nil, err
		}
		results = append(results, entity)
	}
	return results, nil
}

func (r *MongoRepository[T]) CountDocuments(filter interface{}) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *MongoRepository[T]) Upsert(filter interface{}, update interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	result, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if result.UpsertedCount > 0 {
		println("Subscriber inserted with ID:", result.UpsertedID)
		return nil
	} else if result.MatchedCount > 0 {
		println("Subscriber updated.")
		return nil
	}
	return nil
}
