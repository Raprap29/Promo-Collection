package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TestModel struct {
	Name string
	Age  int
}

type MockMongoRepo struct {
	mock.Mock
}

func (m *MockMongoRepo) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockMongoRepo) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.SingleResult)
}

func (m *MockMongoRepo) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, pipeline, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockMongoRepo) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockMongoRepo) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockMongoRepo) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func (m *MockMongoRepo) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockMongoRepo) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(int64), args.Error(1)
}

func TestCreate(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	doc := TestModel{Name: "abcdef", Age: 25}
	expectedResult := &mongo.InsertOneResult{}

	mockRepo.On("InsertOne", mock.Anything, doc, mock.Anything).Return(expectedResult, nil)

	result, err := repo.Create(context.Background(), doc)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	mockRepo.AssertExpectations(t)
}

func TestUpdateOne(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	filter := bson.M{"name": "abcdef"}
	update := bson.M{"age": 30}
	expectedResult := &mongo.UpdateResult{}

	mockRepo.On("UpdateOne", mock.Anything, filter, bson.M{"$set": update}, mock.Anything).Return(expectedResult, nil)

	err := repo.UpdateOne(context.Background(), filter, update)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateMany(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	filter := bson.M{"active": true}
	update := bson.M{"age": 40}
	expectedResult := &mongo.UpdateResult{MatchedCount: 2}

	mockRepo.On("UpdateMany", mock.Anything, filter, update, mock.Anything).Return(expectedResult, nil)

	result, err := repo.UpdateMany(context.Background(), filter, update)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	mockRepo.AssertExpectations(t)
}

func TestDelete(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	filter := bson.M{"name": "abcdef"}
	expected := &mongo.DeleteResult{DeletedCount: 1}

	mockRepo.On("DeleteOne", mock.Anything, filter, mock.Anything).Return(expected, nil)

	err := repo.Delete(context.Background(), filter)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCountDocuments(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	filter := bson.M{"age": 25}
	expected := int64(3)

	mockRepo.On("CountDocuments", mock.Anything, filter, mock.Anything).Return(expected, nil)

	count, err := repo.CountDocuments(context.Background(), filter)

	assert.NoError(t, err)
	assert.Equal(t, expected, count)
	mockRepo.AssertExpectations(t)
}

func TestCreate_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	doc := TestModel{Name: "errcase"}
	mockRepo.On("InsertOne", mock.Anything, doc, mock.Anything).Return((*mongo.InsertOneResult)(nil), assert.AnError)

	result, err := repo.Create(context.Background(), doc)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestFindOne_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	singleResult := mongo.NewSingleResultFromDocument(nil, assert.AnError, nil)
	mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(singleResult)

	res, err := repo.FindOne(context.Background(), bson.M{}, nil)
	assert.Error(t, err)
	assert.Equal(t, TestModel{}, res)
}

func TestAggregate_ErrorBeforeCursor(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("Aggregate", mock.Anything, mock.Anything, mock.Anything).Return((*mongo.Cursor)(nil), assert.AnError)

	err := repo.Aggregate(context.Background(), bson.M{}, &TestModel{})
	assert.Error(t, err)
}

func TestUpdateOne_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*mongo.UpdateResult)(nil), assert.AnError)

	err := repo.UpdateOne(context.Background(), bson.M{}, bson.M{})
	assert.Error(t, err)
}

func TestDelete_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("DeleteOne", mock.Anything, mock.Anything, mock.Anything).Return((*mongo.DeleteResult)(nil), assert.AnError)

	err := repo.Delete(context.Background(), bson.M{})
	assert.Error(t, err)
}

func TestCountDocuments_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("CountDocuments", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), assert.AnError)

	count, err := repo.CountDocuments(context.Background(), bson.M{})
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
}

func TestUpdateMany_Error(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("UpdateMany", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*mongo.UpdateResult)(nil), assert.AnError)

	result, err := repo.UpdateMany(context.Background(), bson.M{}, bson.M{})
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestFind_ErrorBeforeCursor(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("Find", mock.Anything, mock.Anything, mock.Anything).Return((*mongo.Cursor)(nil), assert.AnError)

	res, err := repo.Find(context.Background(), bson.M{})
	assert.Nil(t, res)
	assert.Error(t, err)
}

func TestAggregateAll_ErrorBeforeCursor(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	mockRepo.On("Aggregate", mock.Anything, mock.Anything, mock.Anything).Return((*mongo.Cursor)(nil), assert.AnError)

	err := repo.AggregateAll(context.Background(), bson.M{}, &[]TestModel{})
	assert.Error(t, err)
}

func TestAggregate_Success(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	// Create a real cursor with mock data using mongo.NewCursorFromDocuments
	docs := []interface{}{bson.M{"Name": "John", "Age": 30}}
	cursor, _ := mongo.NewCursorFromDocuments(docs, nil, nil)

	mockRepo.On("Aggregate", mock.Anything, mock.Anything, mock.Anything).
		Return(cursor, nil)

	var result TestModel
	err := repo.Aggregate(context.Background(), bson.M{}, &result)
	assert.NoError(t, err)
	assert.Equal(t, "John", result.Name)
}

func TestAggregateAll_Success(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	docs := []interface{}{
		bson.M{"Name": "A", "Age": 20},
		bson.M{"Name": "B", "Age": 25},
	}
	cursor, _ := mongo.NewCursorFromDocuments(docs, nil, nil)

	mockRepo.On("Aggregate", mock.Anything, mock.Anything, mock.Anything).
		Return(cursor, nil)

	var results []TestModel
	err := repo.AggregateAll(context.Background(), bson.M{}, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestFind_Success(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	docs := []interface{}{
		bson.M{"Name": "X", "Age": 50},
	}
	cursor, _ := mongo.NewCursorFromDocuments(docs, nil, nil)

	mockRepo.On("Find", mock.Anything, mock.Anything, mock.Anything).
		Return(cursor, nil)

	results, err := repo.Find(context.Background(), bson.M{})
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "X", results[0].Name)
}

func TestFindAll(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockMongoRepo)
		repo := NewMongoRepository[TestModel](mockRepo)

		docs := []interface{}{
			bson.M{"Name": "User1", "Age": 25},
			bson.M{"Name": "User2", "Age": 30},
		}
		cursor, _ := mongo.NewCursorFromDocuments(docs, nil, nil)

		mockRepo.On("Find", mock.Anything, bson.M{}, mock.Anything).
			Return(cursor, nil)

		results, err := repo.FindAll(context.Background(), nil)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "User1", results[0].Name)
		assert.Equal(t, "User2", results[1].Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockMongoRepo)
		repo := NewMongoRepository[TestModel](mockRepo)

		mockRepo.On("Find", mock.Anything, bson.M{}, mock.Anything).
			Return((*mongo.Cursor)(nil), assert.AnError)

		results, err := repo.FindAll(context.Background(), nil)
		assert.Error(t, err)
		assert.Nil(t, results)
		mockRepo.AssertExpectations(t)
	})
}

func TestFindOne_Success(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	expectedModel := TestModel{Name: "Test", Age: 30}
	singleResult := mongo.NewSingleResultFromDocument(expectedModel, nil, nil)

	mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).Return(singleResult)

	result, err := repo.FindOne(context.Background(), bson.M{"name": "Test"}, nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedModel, result)
	mockRepo.AssertExpectations(t)
}

func TestGetCollection(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	collection := repo.GetCollection()

	assert.Equal(t, mockRepo, collection)
}

func TestAggregate_NoDocuments(t *testing.T) {
	mockRepo := new(MockMongoRepo)
	repo := NewMongoRepository[TestModel](mockRepo)

	cursor, _ := mongo.NewCursorFromDocuments([]interface{}{}, nil, nil)

	mockRepo.On("Aggregate", mock.Anything, mock.Anything, mock.Anything).
		Return(cursor, nil)

	var result TestModel
	err := repo.Aggregate(context.Background(), bson.M{}, &result)

	assert.Equal(t, mongo.ErrNoDocuments, err)
}
