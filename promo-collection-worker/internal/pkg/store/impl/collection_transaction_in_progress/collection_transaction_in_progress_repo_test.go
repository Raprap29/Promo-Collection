package collection_transaction_in_progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCheckEntryExists_Found(t *testing.T) {
	ctx := context.Background()

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{MSISDN: "100", CreatedAt: time.Now()}, nil
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: mockFind,
		create:  nil,
		delete:  nil,
	}

	ok, err := repo.CheckEntryExists(ctx, "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected found == true")
	}
}

func TestCheckEntryExists_NotFound(t *testing.T) {
	ctx := context.Background()

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{}, mongo.ErrNoDocuments
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: mockFind,
		create:  nil,
		delete:  nil,
	}

	ok, err := repo.CheckEntryExists(ctx, "200")
	if err != nil {
		t.Fatalf("unexpected error for not found: %v", err)
	}
	if ok {
		t.Fatalf("expected found == false when not found")
	}
}

func TestCheckEntryExists_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db failure")

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{}, expectedErr
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: mockFind,
		create:  nil,
		delete:  nil,
	}

	ok, err := repo.CheckEntryExists(ctx, "300")
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error to be %v, got %v", expectedErr, err)
	}
	if ok {
		t.Fatalf("expected found == false when error occurs")
	}
}

func TestCreateEntry_Success(t *testing.T) {
	ctx := context.Background()

	mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
		return (*mongo.InsertOneResult)(nil), nil
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: nil,
		create:  mockCreate,
		delete:  nil,
	}

	if err := repo.CreateEntry(ctx, "400"); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}
}

func TestCreateEntry_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("create failed")

	mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
		return nil, expectedErr
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: nil,
		create:  mockCreate,
		delete:  nil,
	}

	if err := repo.CreateEntry(ctx, "500"); err == nil {
		t.Fatalf("expected error on create but got nil")
	}
}

func TestDeleteEntry_Success(t *testing.T) {
	ctx := context.Background()

	mockDelete := func(ctx context.Context, filter interface{}) error {
		return nil
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: nil,
		create:  nil,
		delete:  mockDelete,
	}

	if err := repo.DeleteEntry(ctx, "600"); err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}
}

func TestDeleteEntry_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("delete failed")

	mockDelete := func(ctx context.Context, filter interface{}) error {
		return expectedErr
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: nil,
		create:  nil,
		delete:  mockDelete,
	}

	if err := repo.DeleteEntry(ctx, "700"); err == nil {
		t.Fatalf("expected error on delete but got nil")
	}
}

func TestMockRepo_CheckCreateDeleteCycle_InMemory(t *testing.T) {
	ctx := context.Background()

	mem := map[string]models.CollectionTransactionsInProgress{}

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		switch f := filter.(type) {
		case bson.M:
			if v, ok := f["MSISDN"]; ok {
				if s, ok := v.(string); ok {
					if e, found := mem[s]; found {
						return e, nil
					}
				}
			}
		case map[string]interface{}:
			if v, ok := f["MSISDN"]; ok {
				if s, ok := v.(string); ok {
					if e, found := mem[s]; found {
						return e, nil
					}
				}
			}
		}
		return models.CollectionTransactionsInProgress{}, mongo.ErrNoDocuments
	}

	mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
		if e, ok := document.(models.CollectionTransactionsInProgress); ok {
			mem[e.MSISDN] = e
		}
		return (*mongo.InsertOneResult)(nil), nil
	}

	mockDelete := func(ctx context.Context, filter interface{}) error {
		switch f := filter.(type) {
		case bson.M:
			if v, ok := f["MSISDN"]; ok {
				if s, ok := v.(string); ok {
					delete(mem, s)
					return nil
				}
			}
		case map[string]interface{}:
			if v, ok := f["MSISDN"]; ok {
				if s, ok := v.(string); ok {
					delete(mem, s)
					return nil
				}
			}
		}
		return nil
	}

	repo := &CollectionTransactionsInProgressRepository{
		findOne: mockFind,
		create:  mockCreate,
		delete:  mockDelete,
	}

	// not exists
	ok, err := repo.CheckEntryExists(ctx, "800")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected not exists initially")
	}

	// create
	if err := repo.CreateEntry(ctx, "800"); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}

	// exists
	ok, err = repo.CheckEntryExists(ctx, "800")
	if err != nil {
		t.Fatalf("unexpected error after create: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists after create")
	}

	// delete
	if err := repo.DeleteEntry(ctx, "800"); err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}

	// not exists
	ok, err = repo.CheckEntryExists(ctx, "800")
	if err != nil {
		t.Fatalf("unexpected error after delete: %v", err)
	}
	if ok {
		t.Fatalf("expected not exists after delete")
	}
}
