package whitelisted_for_data_collection

import (
	"context"
	"errors"
	"testing"

	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mockFinder struct {
	result    models.WhitelistedForDataCollection
	err       error
	deleteErr error
}

func (m *mockFinder) FindOne(ctx context.Context, filter interface{}, opts *options.FindOneOptions) (models.WhitelistedForDataCollection, error) {
	return m.result, m.err
}

func (m *mockFinder) Delete(ctx context.Context, filter interface{}) error {
	return m.deleteErr
}

func TestIsMSISDNWhitelisted(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		mock      *mockFinder
		msisdn    string
		wantFound bool
		wantErr   error
	}{
		{
			name: "Document Found",
			mock: &mockFinder{
				result: models.WhitelistedForDataCollection{MSISDN: "111", Brand: "x", Active: true},
				err:    nil,
			},
			msisdn:    "111",
			wantFound: true,
			wantErr:   nil,
		},
		{
			name: "Document Not Found",
			mock: &mockFinder{
				result: models.WhitelistedForDataCollection{},
				err:    mongo.ErrNoDocuments,
			},
			msisdn:    "222",
			wantFound: false,
			wantErr:   nil,
		},
		{
			name: "Database Error",
			mock: &mockFinder{
				result: models.WhitelistedForDataCollection{},
				err:    errors.New("db failure"),
			},
			msisdn:    "333",
			wantFound: false,
			wantErr:   errors.New("db failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &WhitelistedForDataCollectionRepository{repo: tt.mock}
			found, err := repo.IsMSISDNWhitelisted(ctx, tt.msisdn)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if found != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, found)
			}
		})
	}
}

func TestDeleteByMSISDN(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    *mockFinder
		msisdn  string
		wantErr error
	}{
		{
			name: "Delete Success",
			mock: &mockFinder{
				deleteErr: nil,
			},
			msisdn:  "111",
			wantErr: nil,
		},
		{
			name: "Delete Not Found (ErrNoDocuments)",
			mock: &mockFinder{
				deleteErr: mongo.ErrNoDocuments,
			},
			msisdn:  "222",
			wantErr: nil, // ErrNoDocuments should be ignored
		},
		{
			name: "Delete Error",
			mock: &mockFinder{
				deleteErr: errors.New("delete failed"),
			},
			msisdn:  "333",
			wantErr: errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &WhitelistedForDataCollectionRepository{repo: tt.mock}
			err := repo.DeleteByMSISDN(ctx, tt.msisdn)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
