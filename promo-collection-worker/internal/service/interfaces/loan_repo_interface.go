package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/store/models"

	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LoanRepositoryInterface interface {
	GetLoanByMSISDN(ctx context.Context, msisdn string) (*models.Loans, error)
	GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]models.Loans, error)
	GetLoanByAvailmentID(ctx context.Context, availmentId string) (*models.Loans, error)
	DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error
	UpdateLoanDocument(
		ctx context.Context,
		loan *models.Loans,
		totalUnpaidLoan float64,
		gracePeriodTimestamp time.Time,
	) (primitive.ObjectID, error)
}

type LoanStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.Loans, error)
	Find(ctx context.Context, filter interface{}) ([]models.Loans, error)
	Delete(ctx context.Context, filter interface{}) error
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) error
}
