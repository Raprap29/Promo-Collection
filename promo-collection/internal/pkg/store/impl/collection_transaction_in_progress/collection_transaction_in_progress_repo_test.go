package collection_transaction_in_progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mockStore struct {
	find   func(ctx context.Context, filter interface{}, opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error)
	create func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	delete func(ctx context.Context, filter interface{}) error
}

func (m *mockStore) FindOne(ctx context.Context, filter interface{}, opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
	if m.find != nil {
		return m.find(ctx, filter, opts)
	}
	return models.CollectionTransactionsInProgress{}, mongo.ErrNoDocuments
}

func (m *mockStore) Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	if m.create != nil {
		return m.create(ctx, document)
	}
	return (*mongo.InsertOneResult)(nil), nil
}

func (m *mockStore) Delete(ctx context.Context, filter interface{}) error {
	if m.delete != nil {
		return m.delete(ctx, filter)
	}
	return nil
}

func TestCheckEntryExistsFound(t *testing.T) {
	ctx := context.Background()

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{MSISDN: "100", CreatedAt: time.Now()}, nil
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{find: mockFind})

	ok, err := repo.CheckEntryExists(ctx, "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected found == true")
	}
}

func TestCheckEntryExistsNotFound(t *testing.T) {
	ctx := context.Background()

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{}, mongo.ErrNoDocuments
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{find: mockFind})

	ok, err := repo.CheckEntryExists(ctx, "200")
	if err != nil {
		t.Fatalf("unexpected error for not found: %v", err)
	}
	if ok {
		t.Fatalf("expected found == false when not found")
	}
}

func TestCheckEntryExistsError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db failure")

	mockFind := func(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		return models.CollectionTransactionsInProgress{}, expectedErr
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{find: mockFind})

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

func TestCreateEntrySuccess(t *testing.T) {
	ctx := context.Background()

	mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
		return (*mongo.InsertOneResult)(nil), nil
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{create: mockCreate})

	if err := repo.CreateEntry(ctx, "400"); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}
}

func TestCreateEntryError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("create failed")

	mockCreate := func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
		return nil, expectedErr
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{create: mockCreate})

	if err := repo.CreateEntry(ctx, "500"); err == nil {
		t.Fatalf("expected error on create but got nil")
	}
}

func TestDeleteEntrySuccess(t *testing.T) {
	ctx := context.Background()

	mockDelete := func(ctx context.Context, filter interface{}) error {
		return nil
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{delete: mockDelete})

	if err := repo.DeleteEntry(ctx, "600"); err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}
}

func TestDeleteEntryError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("delete failed")

	mockDelete := func(ctx context.Context, filter interface{}) error {
		return expectedErr
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{delete: mockDelete})

	if err := repo.DeleteEntry(ctx, "700"); err == nil {
		t.Fatalf("expected error on delete but got nil")
	}
}

func TestMockRepoCreateExistDeleteCycle(t *testing.T) {
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

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{find: mockFind, create: mockCreate, delete: mockDelete})

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

func TestMockRepoBasicFindCreateDelete(t *testing.T) {
	// ensure mock store methods behave as expected when used from the repo
	ctx := context.Background()

	mem := map[string]models.CollectionTransactionsInProgress{}

	mockFind := func(ctx context.Context, filter interface{}, opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error) {
		switch f := filter.(type) {
		case bson.M:
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
		if f, ok := filter.(bson.M); ok {
			if v, ok := f["MSISDN"]; ok {
				if s, ok := v.(string); ok {
					delete(mem, s)
					return nil
				}
			}
		}
		return nil
	}

	repo := NewCollectionTransactionsInProgressRepositoryWithInterface(&mockStore{find: mockFind, create: mockCreate, delete: mockDelete})

	// create + find
	if err := repo.CreateEntry(ctx, "900"); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}
	ok, err := repo.CheckEntryExists(ctx, "900")
	if err != nil || !ok {
		t.Fatalf("expected entry to exist after create, got ok=%v err=%v", ok, err)
	}

	// delete + find
	if err := repo.DeleteEntry(ctx, "900"); err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}
	ok, err = repo.CheckEntryExists(ctx, "900")
	if err != nil || ok {
		t.Fatalf("expected entry to be deleted, got ok=%v err=%v", ok, err)
	}
}
