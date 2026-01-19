package promocollection

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"promocollection/internal/pkg/consts"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/models"
	storeModels "promocollection/internal/pkg/store/models"
	"promocollection/internal/service/interfaces"
	"promocollection/utils"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
)

// deduction formula
// min(M*P, L*C)
// M = "maxDataCollectionPercent" in "SystemLevelRules" collection
// P = In terms of MB; walletAmount from payload
// L = TotalUnpaidAmount from "Loans" collection
// C = peso-to-MB conversion rate; Priority - "Loans" collection if not present -> "SystemLevelRules" collection
func CollectionRules(
	ctx context.Context,
	payload *models.PromoEventMessage,
	systemRulesRepo interfaces.SystemLevelRulesRepository,
	loansRepo interfaces.LoanRepositoryInterface,
	pubsubPublisher interfaces.PubSubPublisherInterface,
	collectionTransactionsInProgressRepo interfaces.CollectionTransactionsInProgressRepositoryInterface,
	unpaidRepo interfaces.UnpaidLoansRepositoryInterface,
	loanProductRepo interfaces.LoanProductsRepositoryInterface,
) error {
	systemLevelDocument, err := fetchSystemRules(ctx, systemRulesRepo)
	if err != nil {
		return err
	}

	var totalUnpaidLoan float64
	loansDocument, err := fetchLoans(ctx, loansRepo, payload)
	if err != nil {
		return err
	}

	unpaidDocument, totalUnpaidLoan, err := getTotalUnpaidLoan(ctx, loansDocument, loansRepo, unpaidRepo)
	if err != nil {
		return err
	}

	conversionRate, err := getConversionRate(ctx, loansDocument, systemLevelDocument, loanProductRepo)
	if err != nil {
		return err
	}

	// Compute deductable amount
	deductableAmount := utils.ComputeMaxDeductibleAmount(
		systemLevelDocument.MaxDataCollectionPercent,
		float64(payload.WALLETAMOUNT),
		totalUnpaidLoan,
		conversionRate,
	)

	params := DeductionParams{
		MinCollectionAmount:  systemLevelDocument.MinCollectionAmount,
		ConversionRate:       conversionRate,
		LoanUnpaidAmount:     totalUnpaidLoan,
		DeductableDataAmount: deductableAmount,
	}

	return DataToBeDeductedCalculation(
		ctx, payload, params, pubsubPublisher, collectionTransactionsInProgressRepo, loansDocument, unpaidDocument)
}

func fetchSystemRules(
	ctx context.Context,
	repo interfaces.SystemLevelRulesRepository,
) (storeModels.SystemLevelRules, error) {
	systemLevelDocument, err := repo.FetchSystemLevelRulesConfiguration(ctx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxError(ctx, log_messages.EmptyDocumentFoundFromDb, err,
				slog.String(consts.CollectionsCollection, consts.SystemLevelRulesCollection))
		} else {
			logger.CtxError(ctx, log_messages.ErrorFetchingSystemLevelRulesMongoDBDoc, err)
		}
		return storeModels.SystemLevelRules{}, err
	}
	return systemLevelDocument, nil
}

func fetchLoans(
	ctx context.Context,
	repo interfaces.LoanRepositoryInterface,
	payload *models.PromoEventMessage,
) (*storeModels.Loans, error) {
	loansDocument, err := repo.GetLoanByMSISDN(ctx, strconv.Itoa(int(payload.MSISDN)))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxError(ctx, log_messages.EmptyDocumentFoundFromDb, err,
				slog.String(consts.CollectionsCollection, consts.LoanCollection))
		} else {
			logger.CtxError(ctx, log_messages.ErrorFetchingLoansMongoDBDoc, err)
		}
		return nil, err
	}
	return loansDocument, nil
}

func getTotalUnpaidLoan(ctx context.Context,
	loansDocument *storeModels.Loans,
	loansRepo interfaces.LoanRepositoryInterface,
	unpaidRepo interfaces.UnpaidLoansRepositoryInterface,
) (*storeModels.UnpaidLoans, float64, error) {
	totalUnpaidLoan := loansDocument.TotalUnpaidLoan
	unpaidDocument, err := unpaidRepo.GetUnpaidLoansByLoanId(ctx, loansDocument.LoanID)

	if totalUnpaidLoan == 0 {

		if err != nil {
			return unpaidDocument, 0, err
		}
		if unpaidDocument.TotalUnpaidAmount == 0 {
			return unpaidDocument, 0, fmt.Errorf(log_messages.NoUnpaidBalanceFound, unpaidDocument.ID)
		}
		totalUnpaidLoan = float64(unpaidDocument.TotalUnpaidAmount)
		loansDocument.TotalUnpaidLoan = totalUnpaidLoan

		// Update the loans document in the database with the new totalUnpaidLoan
		_, err = loansRepo.UpdateOldLoanDocument(ctx, loansDocument,
			loansDocument.ConversionRate,
			loansDocument.EducationPeriodTimestamp,
			loansDocument.LoanLoadPeriodTimestamp,
			loansDocument.MaxDataCollectionPercent,
			loansDocument.MinCollectionAmount,
			totalUnpaidLoan)
		if err != nil {
			return unpaidDocument, 0, fmt.Errorf("error updating loans document with totalUnpaidLoan: %w", err)
		}
	}
	return unpaidDocument, totalUnpaidLoan, nil
}

func getConversionRate(ctx context.Context,
	loansDocument *storeModels.Loans,
	systemLevelDocument storeModels.SystemLevelRules,
	loanProductRepo interfaces.LoanProductsRepositoryInterface,
) (float64, error) {
	if loansDocument.ConversionRate != 0.0 {
		return loansDocument.ConversionRate, nil
	}
	logger.CtxInfo(ctx, log_messages.NoConversionRateInLoans)
	if len(systemLevelDocument.PesoToDataConversionMatrix) == 0 {
		return 0, fmt.Errorf(log_messages.NoConversionRateInSystemLevelRules)
	}
	loanProdDoc, err := loanProductRepo.GetProductNameById(ctx, loansDocument.LoanProductID)
	if err != nil {
		logger.CtxError(ctx, "Error while fetching LoanProducts doc", err)
		return 0, fmt.Errorf("error while fetching LoanProducts doc: %w", err)
	}
	isScored := loanProdDoc.IsScoredProduct

	var loanType string
	if isScored {
		loanType = string(consts.LoanTypeScored)
	} else {
		loanType = string(consts.LoanTypeUnScored)
	}

	for _, conv := range systemLevelDocument.PesoToDataConversionMatrix {
		if conv.LoanType == loanType {
			return conv.ConversionRate, nil
		}
	}

	logger.CtxError(ctx, "No conversionRate found for loanType",
		fmt.Errorf("LoanType: %v", loanType))
	return 0, fmt.Errorf("conversionRate not found for loanType %s", loanType)
}
