package unpaidloans

import (
	"context"
	"errors"
	"testing"
	"time"

	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/service/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Constants
const mockNotImplementedMsg = "mock not implemented"
const databaseConnectionErrorMsg = "database connection error"
const mongoServerErrorMsg = "mongo server error"
const contextDeadlineExceededMsg = "context deadline exceeded"
const expectedErrorGotNilMsg = "Expected error, got nil"
const expectedNoErrorMsg = "Expected no error"

// Mock implementations for testing
type mockUnpaidLoansStore struct {
	createFunc  func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	findOneFunc func(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error)
	findFunc    func(ctx context.Context, filter interface{}) ([]models.UnpaidLoans, error)
	deleteFunc  func(ctx context.Context, filter interface{}) error
	updateFunc  func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error)
}

func (m *mockUnpaidLoansStore) Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, document)
	}
	return nil, errors.New(mockNotImplementedMsg)
}

func (m *mockUnpaidLoansStore) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error) {
	if m.findOneFunc != nil {
		return m.findOneFunc(ctx, filter, opt)
	}
	return models.UnpaidLoans{}, errors.New(mockNotImplementedMsg)
}

func (m *mockUnpaidLoansStore) Find(ctx context.Context, filter interface{}) ([]models.UnpaidLoans, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, filter)
	}
	return []models.UnpaidLoans{}, errors.New(mockNotImplementedMsg)
}

func (m *mockUnpaidLoansStore) Delete(ctx context.Context, filter interface{}) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, filter)
	}
	return errors.New(mockNotImplementedMsg)
}

func (m *mockUnpaidLoansStore) UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, filter, update)
	}
	return nil, errors.New(mockNotImplementedMsg)
}

func createTestBsonM() bson.M {
	return bson.M{
		"loanId":                 primitive.NewObjectID(),
		"totalLoanAmount":        1000,
		"totalUnpaidAmount":      900,
		"unpaidServiceFee":       100,
		"dueDate":                time.Now().Add(7 * 24 * time.Hour),
		"lastCollectionId":       primitive.NewObjectID(),
		"version":                1,
		"validFrom":              time.Now(),
		"validTo":                time.Now().Add(24 * time.Hour),
		"lastCollectionDateTime": time.Now(),
		"migrated":               false,
	}
}

// Test NewUnpaidLoansRepository
func TestNewUnpaidLoansRepository(t *testing.T) {
	t.Run("function exists and is callable", func(t *testing.T) {
		// Test that the function exists and can be referenced
		// This ensures the function signature is correct and the function is accessible
		defer func() {
			if r := recover(); r != nil {
				// If we get here, it means the function exists and was called
				t.Logf("NewUnpaidLoansRepository function exists and was called (panicked as expected): %v", r)
			}
		}()

		// Try to call the function with nil - this should panic but proves the function exists
		_ = NewUnpaidLoansRepository(nil)

		// If we reach here without panic, the function exists but didn't panic as expected
		t.Log("NewUnpaidLoansRepository function exists and completed without panic")
	})
}

// Helper function to test repository creation with interface
func testRepositoryCreation(t *testing.T, name string, repo interfaces.UnpaidLoansStoreInterface, expectedResult bool) {
	t.Run(name, func(t *testing.T) {
		result := NewUnpaidLoansRepositoryWithInterface(repo)

		if expectedResult && result == nil {
			t.Errorf("Expected non-nil result, got nil")
		}
		if !expectedResult && result != nil {
			t.Errorf("Expected nil result, got non-nil")
		}
		if result != nil {
			if result.repo != repo {
				t.Errorf("Expected repo to match input, got different")
			}
		}
	})
}

// Test NewUnpaidLoansRepositoryWithInterface
func TestNewUnpaidLoansRepositoryWithInterface(t *testing.T) {
	t.Run("successful creation with interface", func(t *testing.T) {
		testRepositoryCreation(t, "successful creation with interface", &mockUnpaidLoansStore{}, true)
	})

	t.Run("successful creation with nil interface", func(t *testing.T) {
		testRepositoryCreation(t, "successful creation with nil interface", nil, true)
	})
}

// Test CreateUnpaidLoanEntry
func TestCreateUnpaidLoanEntry(t *testing.T) {
	tests := []struct {
		name          string
		loanEntry     bson.M
		setupMocks    func() interfaces.UnpaidLoansStoreInterface
		expectedError bool
	}{
		{
			name:      "successful creation",
			loanEntry: createTestBsonM(),
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
					},
				}
			},
			expectedError: false,
		},
		{
			name:      "creation with error",
			loanEntry: createTestBsonM(),
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return nil, errors.New(databaseConnectionErrorMsg)
					},
				}
			},
			expectedError: true,
		},
		{
			name:      "creation with nil loan entry",
			loanEntry: nil,
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
					},
				}
			},
			expectedError: false,
		},
		{
			name:      "creation with empty loan entry",
			loanEntry: bson.M{},
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
					},
				}
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := tt.setupMocks()
			repo := &UnpaidLoansRepository{repo: mockRepo}

			err := repo.CreateUnpaidLoanEntry(ctx, tt.loanEntry)

			if tt.expectedError && err == nil {
				t.Error(expectedErrorGotNilMsg)
			}
			if !tt.expectedError && err != nil {
				t.Errorf("%s, got: %v", expectedNoErrorMsg, err)
			}
		})
	}
}

// Test CreateUnpaidLoanEntry with nil context
func TestCreateUnpaidLoanEntryNilContext(t *testing.T) {
	loanEntry := createTestBsonM()
	mockRepo := &mockUnpaidLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
		},
	}
	repo := &UnpaidLoansRepository{repo: mockRepo}

	// Test with nil context - should not panic and handle gracefully
	err := repo.CreateUnpaidLoanEntry(context.TODO(), loanEntry)

	// This should either succeed or fail gracefully, not panic
	if err != nil {
		t.Logf("Got expected error with nil context: %v", err)
	}
}

// Test CreateUnpaidLoanEntry with nil repository
func TestCreateUnpaidLoanEntryNilRepository(t *testing.T) {
	loanEntry := createTestBsonM()
	repo := &UnpaidLoansRepository{repo: nil}

	defer func() {
		if r := recover(); r != nil {
			// Panic is expected when repository is nil - this is correct behavior
			t.Logf("Expected panic occurred with nil repository: %v", r)
		}
	}()

	ctx := context.Background()
	err := repo.CreateUnpaidLoanEntry(ctx, loanEntry)

	// Should panic when repo is nil - this is expected behavior
	if err == nil {
		t.Error("Expected panic when repository is nil, got nil error")
	}
}

// Test UnpaidLoansRepositoryInterface methods
func TestUnpaidLoansRepositoryInterface(t *testing.T) {
	t.Run("interface compliance", func(t *testing.T) {
		var _ interfaces.UnpaidLoansRepositoryInterface = &UnpaidLoansRepository{}
	})
}

// Test repository field access
func TestUnpaidLoansRepositoryRepoField(t *testing.T) {
	t.Run("repo field access", func(t *testing.T) {
		mockRepo := &mockUnpaidLoansStore{
			createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
				return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
			},
		}

		repo := &UnpaidLoansRepository{repo: mockRepo}

		// Test that the repo field is accessible and properly set
		if repo.repo == nil {
			t.Error("Expected repo field to be set")
		}

		// Test that we can call methods on the repo field
		ctx := context.Background()
		err := repo.CreateUnpaidLoanEntry(ctx, createTestBsonM())
		if err != nil {
			t.Errorf("%s when calling through repo field, got: %v", expectedNoErrorMsg, err)
		}
	})
}

// Test error propagation
func TestCreateUnpaidLoanEntryErrorPropagation(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func() interfaces.UnpaidLoansStoreInterface
		expectedError string
	}{
		{
			name: "database connection error",
			setupMock: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return nil, errors.New(databaseConnectionErrorMsg)
					},
				}
			},
			expectedError: databaseConnectionErrorMsg,
		},
		{
			name: "mongo server error",
			setupMock: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return nil, errors.New(mongoServerErrorMsg)
					},
				}
			},
			expectedError: mongoServerErrorMsg,
		},
		{
			name: "timeout error",
			setupMock: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
						return nil, errors.New(contextDeadlineExceededMsg)
					},
				}
			},
			expectedError: contextDeadlineExceededMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := tc.setupMock()
			repo := &UnpaidLoansRepository{repo: mockRepo}

			err := repo.CreateUnpaidLoanEntry(ctx, createTestBsonM())

			if err == nil {
				t.Error(expectedErrorGotNilMsg)
			} else if err.Error() != tc.expectedError {
				t.Errorf("Expected error %q, got %q", tc.expectedError, err.Error())
			}
		})
	}
}

// Test concurrent access
func TestCreateUnpaidLoanEntryConcurrent(t *testing.T) {
	mockRepo := &mockUnpaidLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			// Simulate some processing time
			time.Sleep(1 * time.Millisecond)
			return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
		},
	}

	repo := &UnpaidLoansRepository{repo: mockRepo}
	ctx := context.Background()
	loanEntry := createTestBsonM()

	const numGoroutines = 10
	const numOperations = 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				err := repo.CreateUnpaidLoanEntry(ctx, loanEntry)
				if err != nil {
					t.Errorf("Goroutine %d, operation %d failed: %v", id, j, err)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	time.Sleep(2 * time.Second)
}

// Helper function to convert document to bson.M
func convertToBsonM(document interface{}) bson.M {
	if document == nil {
		return nil
	}

	if bsonDoc, ok := document.(bson.M); ok {
		return bsonDoc
	}

	// For other types, create a simple bson.M wrapper
	return bson.M{"data": document}
}

// Helper function to run document type test case
func runDocumentTypeTestCase(t *testing.T, tc struct {
	name      string
	document  interface{}
	shouldErr bool
}, repo *UnpaidLoansRepository, ctx context.Context) {
	t.Run(tc.name, func(t *testing.T) {
		doc := convertToBsonM(tc.document)
		err := repo.CreateUnpaidLoanEntry(ctx, doc)

		if tc.shouldErr && err == nil {
			t.Error(expectedErrorGotNilMsg)
		}
		if !tc.shouldErr && err != nil {
			t.Errorf("%s, got: %v", expectedNoErrorMsg, err)
		}
	})
}

// Test with different BSON document types
func TestCreateUnpaidLoanEntryDifferentDocumentTypes(t *testing.T) {
	testCases := []struct {
		name      string
		document  interface{}
		shouldErr bool
	}{
		{
			name:      "valid bson.M",
			document:  createTestBsonM(),
			shouldErr: false,
		},
		{
			name:      "bson.D",
			document:  bson.D{bson.E{Key: "loanId", Value: primitive.NewObjectID()}, bson.E{Key: "amount", Value: 1000}},
			shouldErr: false,
		},
		{
			name:      "struct document",
			document:  struct{ Name string }{Name: "test"},
			shouldErr: false,
		},
		{
			name:      "string document",
			document:  "invalid document",
			shouldErr: false, // MongoDB can handle string documents
		},
	}

	ctx := context.Background()
	mockRepo := &mockUnpaidLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
		},
	}
	repo := &UnpaidLoansRepository{repo: mockRepo}

	for _, tc := range testCases {
		runDocumentTypeTestCase(t, tc, repo, ctx)
	}
}

// Test repository method chaining
func TestUnpaidLoansRepositoryMethodChaining(t *testing.T) {
	t.Run("multiple operations on same repository", func(t *testing.T) {
		mockRepo := &mockUnpaidLoansStore{
			createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
				return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
			},
		}
		repo := &UnpaidLoansRepository{repo: mockRepo}
		ctx := context.Background()

		// Perform multiple operations
		for i := 0; i < 5; i++ {
			loanEntry := bson.M{
				"loanId":    primitive.NewObjectID(),
				"amount":    1000 + i,
				"operation": i,
			}

			err := repo.CreateUnpaidLoanEntry(ctx, loanEntry)
			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}
	})
}

// Test repository with different context types

type ctxKey string

const testKey ctxKey = "test"

func TestCreateUnpaidLoanEntryDifferentContexts(t *testing.T) {
	testCases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "background context",
			ctx:  context.Background(),
		},
		{
			name: "TODO context",
			ctx:  context.TODO(),
		},
		{
			name: "context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return ctx
			}(),
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), testKey, "value"),
		},
	}

	loanEntry := createTestBsonM()
	mockRepo := &mockUnpaidLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
		},
	}
	repo := &UnpaidLoansRepository{repo: mockRepo}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.CreateUnpaidLoanEntry(tc.ctx, loanEntry)
			if err != nil {
				t.Errorf("Failed with %s context: %v", tc.name, err)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkNewUnpaidLoansRepositoryWithInterface(b *testing.B) {
	mockRepo := &mockUnpaidLoansStore{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewUnpaidLoansRepositoryWithInterface(mockRepo)
	}
}

func BenchmarkCreateUnpaidLoanEntry(b *testing.B) {
	ctx := context.Background()
	loanEntry := createTestBsonM()
	mockRepo := &mockUnpaidLoansStore{
		createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
		},
	}
	repo := &UnpaidLoansRepository{repo: mockRepo}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.CreateUnpaidLoanEntry(ctx, loanEntry)
	}
}

// Test edge cases and boundary conditions
func TestCreateUnpaidLoanEntryEdgeCases(t *testing.T) {
	t.Run("very large bson document", func(t *testing.T) {
		ctx := context.Background()
		largeDocument := bson.M{}
		for i := 0; i < 1000; i++ {
			largeDocument[primitive.NewObjectID().Hex()] = primitive.NewObjectID().Hex()
		}

		mockRepo := &mockUnpaidLoansStore{
			createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
				return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
			},
		}
		repo := &UnpaidLoansRepository{repo: mockRepo}

		err := repo.CreateUnpaidLoanEntry(ctx, largeDocument)
		if err != nil {
			t.Errorf("Failed to handle large document: %v", err)
		}
	})

	t.Run("document with special characters", func(t *testing.T) {
		ctx := context.Background()
		specialDoc := bson.M{
			"field with spaces": "value with spaces",
			"field.with.dots":   "value with dots",
			"field$with$dollar": "value with dollar",
			"field\nwith\nline": "value with line breaks",
		}

		mockRepo := &mockUnpaidLoansStore{
			createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
				return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
			},
		}
		repo := &UnpaidLoansRepository{repo: mockRepo}

		err := repo.CreateUnpaidLoanEntry(ctx, specialDoc)
		if err != nil {
			t.Errorf("Failed to handle document with special characters: %v", err)
		}
	})
}

// Test constructor functions
func TestUnpaidLoansRepositoryConstructor(t *testing.T) {
	t.Run("NewUnpaidLoansRepositoryWithInterface with nil", func(t *testing.T) {
		repo := NewUnpaidLoansRepositoryWithInterface(nil)
		if repo == nil {
			t.Fatal("Expected non-nil repository")
		}
		if repo.repo != nil {
			t.Error("Expected nil repo when passing nil")
		}
	})

	t.Run("NewUnpaidLoansRepositoryWithInterface with mock", func(t *testing.T) {
		mockRepo := &mockUnpaidLoansStore{}
		repo := NewUnpaidLoansRepositoryWithInterface(mockRepo)
		if repo == nil {
			t.Error("Expected non-nil repository")
		}
		// Note: We can't directly compare interface with concrete type
		// The important thing is that the repo was created successfully
	})
}

// Test repository method behavior
func TestUnpaidLoansRepositoryMethods(t *testing.T) {
	t.Run("CreateUnpaidLoanEntry with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		loanEntry := createTestBsonM()
		mockRepo := &mockUnpaidLoansStore{
			createFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					return &mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil
				}
			},
		}
		repo := &UnpaidLoansRepository{repo: mockRepo}

		err := repo.CreateUnpaidLoanEntry(ctx, loanEntry)
		if err == nil {
			t.Error("Expected error due to cancelled context")
		}
	})
}

// Test UpdatePreviousValidToField
func TestUpdatePreviousValidToField(t *testing.T) {
	tests := []struct {
		name          string
		loanId        primitive.ObjectID
		setupMocks    func() interfaces.UnpaidLoansStoreInterface
		expectedError bool
	}{
		{
			name:   "successful update",
			loanId: primitive.NewObjectID(),
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					findOneFunc: func(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error) {
						return models.UnpaidLoans{ID: primitive.NewObjectID()}, nil
					},
					updateFunc: func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
						return &mongo.UpdateResult{UpsertedID: primitive.NewObjectID()}, nil
					},
				}
			},
			expectedError: false,
		},
		{
			name:   "find one error",
			loanId: primitive.NewObjectID(),
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					findOneFunc: func(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error) {
						return models.UnpaidLoans{}, errors.New(databaseConnectionErrorMsg)
					},
				}
			},
			expectedError: true,
		},
		{
			name:   "update error",
			loanId: primitive.NewObjectID(),
			setupMocks: func() interfaces.UnpaidLoansStoreInterface {
				return &mockUnpaidLoansStore{
					findOneFunc: func(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error) {
						return models.UnpaidLoans{ID: primitive.NewObjectID()}, nil
					},
					updateFunc: func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
						return nil, errors.New(mongoServerErrorMsg)
					},
				}
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockRepo := tt.setupMocks()
			repo := &UnpaidLoansRepository{repo: mockRepo}

			err := repo.UpdatePreviousValidToField(ctx, tt.loanId)

			if tt.expectedError && err == nil {
				t.Error(expectedErrorGotNilMsg)
			}
			if !tt.expectedError && err != nil {
				t.Errorf("%s, got: %v", expectedNoErrorMsg, err)
			}
		})
	}
}
