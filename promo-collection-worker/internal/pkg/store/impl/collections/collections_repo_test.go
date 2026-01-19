package collections

import (
	"context"
	"errors"
	"fmt"
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/store/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCreateEntry(t *testing.T) {
	ctx := context.Background()
	model := &models.Collections{
		MSISDN:           "1234567890",
		CollectionType:   "Data",
		CollectedAmount:  50.0,
		PublishedToKafka: false,
	}

	t.Run("Success", func(t *testing.T) {
		mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			objID := primitive.NewObjectID()
			return &mongo.InsertOneResult{InsertedID: objID}, nil
		}

		repo := &CollectionsRepository{
			create: mockCreate,
		}

		_, err := repo.CreateEntry(ctx, model)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return nil, expectedErr
		}

		repo := &CollectionsRepository{
			create: mockCreate,
		}

		_, err := repo.CreateEntry(ctx, model)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestUpdatePublishToKafka(t *testing.T) {
	ctx := context.Background()
	id := primitive.NewObjectID()

	t.Run("Success", func(t *testing.T) {
		mockUpdateOne := func(ctx context.Context, filter interface{}, update interface{}) error {
			// Verify filter contains the correct ID
			filterMap, ok := filter.(bson.M)
			assert.True(t, ok)
			assert.Equal(t, id, filterMap["_id"])

			// Verify update sets publishedToKafka to true
			updateMap, ok := update.(bson.M)
			assert.True(t, ok)
			assert.Equal(t, true, updateMap["publishedToKafka"])

			return nil
		}

		repo := &CollectionsRepository{
			updateOne: mockUpdateOne,
		}

		err := repo.UpdatePublishToKafka(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("update error")
		mockUpdateOne := func(ctx context.Context, filter interface{}, update interface{}) error {
			return expectedErr
		}

		repo := &CollectionsRepository{
			updateOne: mockUpdateOne,
		}

		err := repo.UpdatePublishToKafka(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

// Mock for mongo.Collection to test GetFailedKafkaEntriesCursor
type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, pipeline, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

// MockRepo for testing GetFailedKafkaEntriesCursor
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) GetCollection() interface{} {
	args := m.Called()
	return args.Get(0)
}

// MockMongoRepository implements the repository interface for testing
type MockMongoRepository struct {
	mock.Mock
}

func (m *MockMongoRepository) GetCollection() interface{} {
	args := m.Called()
	return args.Get(0)
}

func TestGetFailedKafkaEntriesCursor_InvalidDateFormats(t *testing.T) {
	ctx := context.Background()
	repo := &CollectionsRepository{}

	invalidDates := []string{
		"",                     // Empty string
		"invalid-date",         // Not a date
		"not-a-date",           // Not a date
		"2025/01/01",           // Wrong separator
		"01-01-2025",           // Wrong order
		"2025-13-01",           // Invalid month
		"2025-01-32",           // Invalid day
		"2025-02-30",           // Invalid day for February
		"2025-01-01T10:00:00Z", // With time (should fail with DateFormat)
	}

	for _, invalidDate := range invalidDates {
		t.Run(fmt.Sprintf("InvalidDate_%s", invalidDate), func(t *testing.T) {
			cursor, err := repo.GetFailedKafkaEntriesCursor(ctx, invalidDate, 100)
			assert.Error(t, err, "Expected error for date: %s", invalidDate)
			assert.Nil(t, cursor)
		})
	}
}

func TestGetFailedKafkaEntriesCursor_ValidInputs(t *testing.T) {
	ctx := context.Background()
	repo := &CollectionsRepository{}

	validDates := []string{
		"2025-01-01",
		"2024-12-31",
		"2023-02-28",
		"2024-02-29",
		"2025-06-15",
	}

	batchSizes := []int32{1, 100}

	for _, validDate := range validDates {
		for _, batchSize := range batchSizes {
			t.Run(fmt.Sprintf("ValidDate_%s_BatchSize_%d", validDate, batchSize), func(t *testing.T) {
				// This will panic because the repo is nil, but we can test that the input validation succeeds
				// by checking that we get a nil pointer panic (not an input validation error)
				defer func() {
					if r := recover(); r != nil {
						// Expected panic due to nil repo
						assert.Contains(t, fmt.Sprintf("%v", r), "nil pointer dereference")
					} else {
						t.Error("Expected panic due to nil repository")
					}
				}()

				cursor, err := repo.GetFailedKafkaEntriesCursor(ctx, validDate, batchSize)
				// This line should not be reached due to panic
				assert.Error(t, err)
				assert.Nil(t, cursor)
			})
		}
	}
}

func TestUpdatePublishedToKafkaInBulk(t *testing.T) {
	ctx := context.Background()
	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	collectionIds := []string{id1.Hex(), id2.Hex()}

	t.Run("Success", func(t *testing.T) {
		mockUpdateMany := func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{
				MatchedCount:  2,
				ModifiedCount: 2,
			}, nil
		}

		repo := &CollectionsRepository{
			updateMany: mockUpdateMany,
		}

		failedIds, err := repo.UpdatePublishedToKafkaInBulk(ctx, collectionIds)

		assert.NoError(t, err)
		assert.Empty(t, failedIds)
	})

	t.Run("Partial success", func(t *testing.T) {
		mockUpdateMany := func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{
				MatchedCount:  2,
				ModifiedCount: 1, // Only one document was modified
			}, nil
		}

		mockFind := func(ctx context.Context, filter interface{}) ([]models.Collections, error) {
			// Return the document that wasn't updated
			return []models.Collections{
				{ID: id2},
			}, nil
		}

		repo := &CollectionsRepository{
			updateMany: mockUpdateMany,
			find:       mockFind,
		}

		failedIds, err := repo.UpdatePublishedToKafkaInBulk(ctx, collectionIds)

		assert.NoError(t, err)
		assert.Equal(t, []string{id2.Hex()}, failedIds)
	})

	t.Run("Invalid ObjectID", func(t *testing.T) {
		invalidIds := []string{"invalid-id"}
		repo := &CollectionsRepository{}

		failedIds, err := repo.UpdatePublishedToKafkaInBulk(ctx, invalidIds)

		assert.Error(t, err)
		assert.Nil(t, failedIds)
	})

	t.Run("UpdateMany error", func(t *testing.T) {
		expectedErr := errors.New("update error")
		mockUpdateMany := func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
			return nil, expectedErr
		}

		repo := &CollectionsRepository{
			updateMany: mockUpdateMany,
		}

		failedIds, err := repo.UpdatePublishedToKafkaInBulk(ctx, collectionIds)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, failedIds)
	})

	t.Run("Find error", func(t *testing.T) {
		expectedErr := errors.New("find error")
		mockUpdateMany := func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
			return &mongo.UpdateResult{
				MatchedCount:  2,
				ModifiedCount: 1, // Only one document was modified
			}, nil
		}

		mockFind := func(ctx context.Context, filter interface{}) ([]models.Collections, error) {
			return nil, expectedErr
		}

		repo := &CollectionsRepository{
			updateMany: mockUpdateMany,
			find:       mockFind,
		}

		failedIds, err := repo.UpdatePublishedToKafkaInBulk(ctx, collectionIds)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, failedIds)
	})
}
func bsonDToMap(d bson.D) (bson.M, error) {
	data, err := bson.Marshal(d)
	if err != nil {
		return nil, err
	}
	var m bson.M
	err = bson.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func TestGetFailedKafkaEntriesAggregationPipeline(t *testing.T) {
	threshold := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pipeline := getFailedKafkaEntriesAggregationPipeline(threshold)

	if len(pipeline) != 5 {
		t.Errorf("expected 5 stages, got %d", len(pipeline))
	}

	// Stage 1: $match
	matchStageMap, err := bsonDToMap(pipeline[0])
	if err != nil {
		t.Fatal(err)
	}

	match, ok := matchStageMap["$match"].(bson.M)
	if !ok {
		t.Fatal("$match stage not found or wrong type")
	}

	if match["publishedToKafka"] != false {
		t.Errorf("expected publishedToKafka=false, got %v", match["publishedToKafka"])
	}

	createdAtCond, ok := match["createdAt"].(bson.M)
	if !ok {
		t.Errorf("expected createdAt to be bson.M, got %T", match["createdAt"])
	}

	// Check that $gte exists and is a valid timestamp
	gteValue, exists := createdAtCond["$gte"]
	if !exists {
		t.Errorf("expected createdAt to have $gte condition")
	}

	// The value should be a primitive.DateTime representing the threshold time
	if gteDateTime, ok := gteValue.(primitive.DateTime); ok {
		// Convert threshold to primitive.DateTime for comparison
		expectedDateTime := primitive.NewDateTimeFromTime(threshold)
		if gteDateTime != expectedDateTime {
			t.Errorf("expected createdAt $gte to be %v (DateTime), got %v", expectedDateTime, gteDateTime)
		}
	} else {
		t.Errorf("expected createdAt $gte to be primitive.DateTime, got %T", gteValue)
	}

	// Stage 2: $lookup
	lookupStageMap, err := bsonDToMap(pipeline[1])
	if err != nil {
		t.Fatal(err)
	}
	lookup := lookupStageMap["$lookup"].(bson.M)
	if lookup["from"] != consts.LoanCollection {
		t.Errorf("expected $lookup from %s, got %v", consts.LoanCollection, lookup["from"])
	}

	// Stage 5: $project
	projectStageMap, err := bsonDToMap(pipeline[4])
	if err != nil {
		t.Fatal(err)
	}
	project := projectStageMap["$project"].(bson.M)

	// Check a few keys exist
	expectedKeys := []string{"_id", "MSISDN", "loanGUID"}
	for _, key := range expectedKeys {
		if _, ok := project[key]; !ok {
			t.Errorf("$project missing expected key %s", key)
		}
	}
}
