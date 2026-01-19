package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LoanRepositoryInterface interface {
	GetLoanByMSISDN(ctx context.Context, msisdn string) (*models.Loans, error)
	GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]models.Loans, error)
	GetLoanByAvailmentID(ctx context.Context, availmentId string) (*models.Loans, error)
	DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error
	CheckIfOldOrNewLoan(ctx context.Context, loan *models.Loans) bool
	UpdateOldLoanDocument(ctx context.Context,
		loan *models.Loans,
		conversionRate float64,
		promoEducationPeriodTimestamp time.Time,
		loanLoadPeriodTimestamp time.Time,
		maxDataCollectionPercent float64,
		minCollectionAmount float64,
		totalUnpaidLoan float64) (primitive.ObjectID, error)
}

type LoanStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.Loans, error)
	Find(ctx context.Context, filter interface{}) ([]models.Loans, error)
	Delete(ctx context.Context, filter interface{}) error
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) error
}
