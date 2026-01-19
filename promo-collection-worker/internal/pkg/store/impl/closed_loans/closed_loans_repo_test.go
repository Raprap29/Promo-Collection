package closedloans

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mock implementation of ClosedLoansStoreInterface for testing
type mockClosedLoansStore struct {
	createFunc  func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	findOneFunc func(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.ClosedLoans, error)
	findFunc    func(ctx context.Context, filter interface{}) ([]models.ClosedLoans, error)
}

func (m *mockClosedLoansStore) Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, document)
	}
	return nil, errors.New("mock create not implemented")
}

func (m *mockClosedLoansStore) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.ClosedLoans, error) {
	if m.findOneFunc != nil {
		return m.findOneFunc(ctx, filter, opt)
	}
	return models.ClosedLoans{}, errors.New("mock findOne not implemented")
}

func (m *mockClosedLoansStore) Find(ctx context.Context, filter interface{}) ([]models.ClosedLoans, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, filter)
	}
	return nil, errors.New("mock find not implemented")
}

// Test data fixtures
func createTestClosedLoansEntry() bson.M {
	return bson.M{
		"GUID":                           "test-guid-123",
		"MSISDN":                         "639123456789",
		"availmentTransactionId":         primitive.NewObjectID(),
		"brandId":                        primitive.NewObjectID(),
		"endDate":                        time.Now(),
		"loanId":                         primitive.NewObjectID(),
		"loanProductId":                  primitive.NewObjectID(),
		"loanType":                       consts.LoanTypeGES,
		"startDate":                      time.Now().Add(-24 * time.Hour),
		"status":                         consts.StatusAuto,
		"totalCollectedAmount":           1000.50,
		"totalLoanAmount":                1000.00,
		"totalWrittenOffOrChurnedAmount": 0.0,
		"writeOffReason":                 "",
	}
}

// Test NewClosedLoansRepositoryWithInterface with valid interface
func TestNewClosedLoansRepositoryWithInterface(t *testing.T) {
	mockRepo := &mockClosedLoansStore{}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	if repo == nil {
		t.Fatal("Expected non-nil repository")
	}

	if repo.repo != mockRepo {
		t.Error("Expected repository to use provided interface")
	}
}

// Test NewClosedLoansRepositoryWithInterface with nil interface
func TestNewClosedLoansRepositoryWithInterfaceNilInterface(t *testing.T) {
	repo := NewClosedLoansRepositoryWithInterface(nil)

	if repo == nil {
		t.Fatal("Expected non-nil repository even with nil interface")
	}

	if repo.repo != nil {
		t.Error("Expected nil internal repository when interface is nil")
	}
}

// Test CreateClosedLoansEntry with mock interface - success
func TestClosedLoansRepositoryCreateClosedLoansEntryMockSuccess(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{
				InsertedID: primitive.NewObjectID(),
			}, nil
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	testEntry := createTestClosedLoansEntry()

	err := repo.CreateClosedLoansEntry(context.Background(), testEntry)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// Test CreateClosedLoansEntry with mock interface - error
func TestClosedLoansRepositoryCreateClosedLoansEntryMockError(t *testing.T) {
	expectedError := errors.New("mock repository error")

	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return nil, expectedError
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	testEntry := createTestClosedLoansEntry()

	err := repo.CreateClosedLoansEntry(context.Background(), testEntry)

	if err != expectedError {
		t.Fatalf("Expected error %v, got: %v", expectedError, err)
	}
}

// Test CreateClosedLoansEntry with mock interface - panic
func TestClosedLoansRepositoryCreateClosedLoansEntryMockPanic(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			panic("mock panic")
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	testEntry := createTestClosedLoansEntry()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic, but none occurred")
		}
	}()

	repo.CreateClosedLoansEntry(context.Background(), testEntry)
}

// Test CreateClosedLoansEntry with empty entry
func TestClosedLoansRepositoryCreateClosedLoansEntryEmptyEntry(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{
				InsertedID: primitive.NewObjectID(),
			}, nil
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	emptyEntry := bson.M{}

	err := repo.CreateClosedLoansEntry(context.Background(), emptyEntry)

	if err != nil {
		t.Fatalf("Expected no error for empty entry, got: %v", err)
	}
}

// Test CreateClosedLoansEntry with large entry
func TestClosedLoansRepositoryCreateClosedLoansEntryLargeEntry(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{
				InsertedID: primitive.NewObjectID(),
			}, nil
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	// Create a large entry with many fields
	largeEntry := bson.M{}
	for i := 0; i < 100; i++ {
		largeEntry[fmt.Sprintf("field%d", i)] = fmt.Sprintf("value%d", i)
	}

	err := repo.CreateClosedLoansEntry(context.Background(), largeEntry)

	if err != nil {
		t.Fatalf("Expected no error for large entry, got: %v", err)
	}
}

// Test CreateClosedLoansEntry with context cancellation
func TestClosedLoansRepositoryCreateClosedLoansEntryContextCancelled(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			// Simulate context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mongo.InsertOneResult{
					InsertedID: primitive.NewObjectID(),
				}, nil
			}
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	testEntry := createTestClosedLoansEntry()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := repo.CreateClosedLoansEntry(ctx, testEntry)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

// Test concurrent access to CreateClosedLoansEntry
func TestClosedLoansRepositoryCreateClosedLoansEntryConcurrent(t *testing.T) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return &mongo.InsertOneResult{
				InsertedID: primitive.NewObjectID(),
			}, nil
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	const numGoroutines = 10
	const numOperations = 5

	errorsChan := make(chan error, numGoroutines*numOperations)

	// Start multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numOperations; j++ {
				entry := bson.M{
					"GUID":     fmt.Sprintf("concurrent-guid-%d-%d", goroutineID, j),
					"MSISDN":   fmt.Sprintf("639%d%d", goroutineID, j),
					"loanType": consts.LoanTypeGES,
				}

				err := repo.CreateClosedLoansEntry(context.Background(), entry)
				errorsChan <- err
			}
		}(i)
	}

	// Collect all errors
	for i := 0; i < numGoroutines*numOperations; i++ {
		err := <-errorsChan
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

// Benchmark tests
func BenchmarkClosedLoansRepositoryCreateClosedLoansEntry(b *testing.B) {
	mockRepo := &mockClosedLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{
				InsertedID: primitive.NewObjectID(),
			}, nil
		},
	}

	repo := NewClosedLoansRepositoryWithInterface(mockRepo)
	testEntry := createTestClosedLoansEntry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.CreateClosedLoansEntry(context.Background(), testEntry)
	}
}

// Test repository field access (for coverage)
func TestClosedLoansRepositoryFields(t *testing.T) {
	mockRepo := &mockClosedLoansStore{}
	repo := NewClosedLoansRepositoryWithInterface(mockRepo)

	// Test that the repo field is accessible (for coverage)
	if repo.repo == nil {
		t.Error("Expected repo field to be set")
	}
}

// Test NewClosedLoansRepository with nil client (for 100% coverage)
func TestNewClosedLoansRepositoryNilClient(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when client is nil")
		}
	}()

	// This will panic due to nil pointer dereference, but that's expected behavior
	// and we need to test that the constructor is called
	NewClosedLoansRepository(nil)
}
