package loans

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"
	"promo-collection-worker/internal/service/interfaces"
	"time"

	"promo-collection-worker/internal/pkg/logger"

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
			return &models.Loans{}, err
		}
		logger.CtxError(ctx, "Error finding loan by MSISDN", err, slog.String("MSISDN", msisdn))
		return &models.Loans{}, err
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

func (lr *LoanRepository) UpdateLoanDocument(ctx context.Context,
	loan *models.Loans,
	totalUnpaidLoan float64,
	gracePeriodTimestamp time.Time) (primitive.ObjectID, error) {
	update := bson.M{
		"totalUnpaidLoan":      totalUnpaidLoan,
		"gracePeriodTimestamp": gracePeriodTimestamp,
	}

	filter := bson.M{"_id": loan.LoanID}
	err := lr.repo.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorUpdatingLoansMongoDbDocument, err)
		return primitive.NilObjectID, err
	}
	logger.CtxInfo(ctx, log_messages.SuccessLoansDocumentUpdationMongoDb, slog.Any("LoanId", loan.LoanID))

	loan.TotalLoanAmount = int32(totalUnpaidLoan)

	return loan.LoanID, nil
}
