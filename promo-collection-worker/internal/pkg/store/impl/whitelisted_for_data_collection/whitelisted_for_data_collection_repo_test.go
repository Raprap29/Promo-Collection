package whitelisted_for_data_collection

import (
	"context"
	"errors"
	"testing"

	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mockFinder struct {
	result models.WhitelistedForDataCollection
	err    error
}

func (m *mockFinder) FindOne(ctx context.Context,
	filter interface{}, opts *options.FindOneOptions) (models.WhitelistedForDataCollection, error) {
	return m.result, m.err
}

func TestIsMSISDNWhitelisted_Found(t *testing.T) {
	ctx := context.Background()
	mock := &mockFinder{
		result: models.WhitelistedForDataCollection{MSISDN: "111", Brand: "x", Active: true},
		err:    nil,
	}
	repo := &WhitelistedRepository{repo: mock}

	found, err := repo.IsMSISDNWhitelisted(ctx, "111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("expected true when document exists")
	}
}

func TestIsMSISDNWhitelisted_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockFinder{
		result: models.WhitelistedForDataCollection{},
		err:    mongo.ErrNoDocuments,
	}
	repo := &WhitelistedRepository{repo: mock}

	found, err := repo.IsMSISDNWhitelisted(ctx, "222")
	if err != nil {
		t.Fatalf("unexpected error on not found: %v", err)
	}
	if found {
		t.Fatalf("expected false when document does not exist")
	}
}

func TestIsMSISDNWhitelisted_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("database failure")
	mock := &mockFinder{
		result: models.WhitelistedForDataCollection{},
		err:    expectedErr,
	}
	repo := &WhitelistedRepository{repo: mock}

	found, err := repo.IsMSISDNWhitelisted(ctx, "333")
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error to be %v, got %v", expectedErr, err)
	}
	if found {
		t.Fatalf("expected false when underlying error occurs")
	}
}
