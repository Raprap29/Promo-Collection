// impl/system_level_rules_test.go
package system_level_rules

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mock store for testing
type mockSystemLevelRulesStore struct {
	doc        models.SystemLevelRules
	err        error
	called     int
	lastFilter interface{}
	lastOpts   *options.FindOneOptions
}

func (m *mockSystemLevelRulesStore) FindOne(_ context.Context, filter interface{}, opt *options.FindOneOptions) (models.SystemLevelRules, error) {
	m.called++
	m.lastFilter = filter
	m.lastOpts = opt
	return m.doc, m.err
}

func TestFetchDocument_Success(t *testing.T) {
	expected := models.SystemLevelRules{
		CreditScoreThreshold:               650,
		PartialCollectionEnabled:           true,
		ReservedAmountForPartialCollection: 100,
		EducationPeriod:                    30,
		DefermentPeriod:                    5,
		OverdueThreshold:                   10,
		UpdatedAt:                          time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC),
		MaxDataCollectionPercent:           0.4,
		PesoToDataConversionMatrix: []models.PesoToDataConversion{
			{LoanType: "standard", ConversionRate: 1.25},
			{LoanType: "promo", ConversionRate: 1.50},
		},
		MinCollectionAmount:    25.0,
		PromoCollectionEnabled: true,
		WalletExclusionList:    []string{"walletA", "walletB"},
		PromoEducationPeriod:   7,
		GracePeriod:            3,
		LoanLoadPeriod:         2,
	}

	mockRepo := &mockSystemLevelRulesStore{doc: expected}
	repo := NewSystemLevelRulesRepositoryWithInterface(mockRepo)

	got, err := repo.FetchSystemLevelRulesConfiguration(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("FetchSystemLevelRulesConfiguration() mismatch\nexpected: %#v\n     got: %#v", expected, got)
	}

	// Verify the repository called FindOne with empty filter and non-nil options.
	if mockRepo.called != 1 {
		t.Errorf("expected FindOne to be called once, got %d", mockRepo.called)
	}
	if !reflect.DeepEqual(mockRepo.lastFilter, bson.M{}) {
		t.Errorf("expected empty filter bson.M{}, got %#v", mockRepo.lastFilter)
	}
	if mockRepo.lastOpts == nil {
		t.Errorf("expected non-nil FindOne options, got nil")
	}
}

func TestFetchDocument_Error(t *testing.T) {
	mockRepo := &mockSystemLevelRulesStore{err: errors.New("db error")}
	repo := NewSystemLevelRulesRepositoryWithInterface(mockRepo)

	got, err := repo.FetchSystemLevelRulesConfiguration(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !reflect.DeepEqual(got, models.SystemLevelRules{}) {
		t.Errorf("expected zero value rules on error, got %#v", got)
	}
}
