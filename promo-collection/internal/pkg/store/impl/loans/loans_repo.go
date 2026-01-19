package loans

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"promocollection/internal/pkg/consts"
	mongodb "promocollection/internal/pkg/db/mongo"
	"promocollection/internal/pkg/store/models"
	"promocollection/internal/pkg/store/repository"
	"promocollection/internal/service/interfaces"
	"time"

	"promocollection/internal/pkg/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LoanRepository struct {
	repo interfaces.LoanStoreInterface
}

func NewLoansRepository(client *mongodb.MongoClient) *LoanRepository {
	collection := client.Database.Collection(consts.LoanCollection)
	repo := repository.NewMongoRepository[models.Loans](collection)
	return &LoanRepository{repo: repo}
}

func NewLoanRepositoryWithInterface(repo interfaces.LoanStoreInterface) *LoanRepository {
	return &LoanRepository{repo: repo}
}

func (lr *LoanRepository) GetLoanByMSISDN(ctx context.Context, msisdn string) (*models.Loans, error) {
	var loan models.Loans

	filter := bson.M{"MSISDN": msisdn}
	loan, err := lr.repo.FindOne(ctx, filter, options.FindOne())

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxWarn(ctx, "No loan found for MSISDN", slog.String("MSISDN", msisdn))
			return nil, err
		}
		logger.CtxError(ctx, "Error finding loan by MSISDN", err, slog.String("MSISDN", msisdn))
		return nil, err
	}

	logger.CtxDebug(ctx, "Fetched loan by MSISDN", slog.String("MSISDN", msisdn), slog.Any("loan_id", loan.LoanID.Hex()))
	return &loan, nil
}

func (lr *LoanRepository) GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]models.Loans, error) {
	// Find loans that match the given MSISDNs and are active
	filter := bson.M{
		"MSISDN": bson.M{"$in": msisdns},
	}
	var loans []models.Loans

	loans, err := lr.repo.Find(ctx, filter)
	if err != nil {
		logger.CtxError(ctx, "Error fetching loans for MSISDN list", err, slog.Any("msisdns", msisdns))
		return nil, err
	}

	logger.CtxDebug(ctx, "Fetched loans by MSISDN list", slog.Int("count", len(loans)))

	return loans, nil
}

func (lr *LoanRepository) DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error {

	filter := bson.M{"_id": loanID}

	err := lr.repo.Delete(ctx, filter)
	if err != nil {
		logger.CtxError(ctx, "Error deleting active loan", err, slog.String("loan_id", loanID.Hex()))
		return err
	}

	logger.CtxInfo(ctx, "Successfully deleted active loan", slog.String("loan_id", loanID.Hex()))
	return nil
}

func (lr *LoanRepository) GetLoanByAvailmentID(ctx context.Context, availmentId string) (*models.Loans, error) {
	// Find loans that match the given availmentId
	if len(availmentId) == 0 {
		logger.CtxWarn(ctx, "Availment ID is empty")
		return nil, fmt.Errorf("no availmentId provided")
	}

	objectID, err := primitive.ObjectIDFromHex(availmentId)
	if err != nil {
		logger.CtxError(ctx, "Invalid ObjectID for availmentId", err, slog.String("availment_id", availmentId))
		return nil, err
	}

	filter := bson.M{"availmentTransactionId": objectID}

	loan, err := lr.repo.FindOne(ctx, filter, options.FindOne())
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxWarn(ctx, "No loan found for availment ID", slog.String("availment_id", availmentId))
			return nil, err
		}
		logger.CtxError(ctx, "Error finding loan by availment ID", err, slog.String("availment_id", availmentId))
		return nil, err
	}

	logger.CtxDebug(
		ctx,
		"Fetched loan by availment ID",
		slog.String("availment_id", availmentId),
		slog.String("loan_id", loan.LoanID.Hex()),
	)

	return &loan, nil
}

func (lr *LoanRepository) CheckIfOldOrNewLoan(ctx context.Context, loan *models.Loans) bool {
	// if ConversionRate > 0 OR either timestamp is non-zero, it's a new loan
	// NOTE: The timestamps here by loanavailment are at the start of UNIX epoch by default (i.e. 1970-01-01T00:00:00Z).
	//   IsZero might always return false, so this function might always say all loans are new.
	if loan.ConversionRate != 0 ||
		!loan.PromoEducationPeriodTimestamp.IsZero() ||
		!loan.LoanLoadPeriodTimestamp.IsZero() {
		return true
	}
	return false
}

// FIXME: loanLoad period should be set only if loanType is "LOAD SKU"
func (lr *LoanRepository) UpdateOldLoanDocument(ctx context.Context,
	loan *models.Loans,
	conversionRate float64,
	promoEducationPeriodTimestamp time.Time,
	loanLoadPeriodTimestamp time.Time,
	maxDataCollectionPercent float64,
	minCollectionAmount float64,
	totalUnpaidLoan float64) (primitive.ObjectID, error) {
	update := bson.M{
		"conversionRate":                conversionRate,
		"promoEducationPeriodTimestamp": promoEducationPeriodTimestamp,
		"loanLoadPeriodTimestamp":       loanLoadPeriodTimestamp,
		"maxDataCollectionPercent":      maxDataCollectionPercent,
		"minCollectionAmount":           minCollectionAmount,
		"totalUnpaidLoan":               totalUnpaidLoan,
	}

	filter := bson.M{"_id": loan.LoanID}
	err := lr.repo.UpdateOne(ctx, filter, update)
	if err != nil {
		return primitive.NilObjectID, err
	}

	loan.ConversionRate = conversionRate
	loan.PromoEducationPeriodTimestamp = promoEducationPeriodTimestamp
	loan.LoanLoadPeriodTimestamp = loanLoadPeriodTimestamp
	loan.MaxDataCollectionPercent = maxDataCollectionPercent
	loan.MinCollectionAmount = minCollectionAmount
	loan.TotalUnpaidLoan = totalUnpaidLoan

	return loan.LoanID, nil
}
