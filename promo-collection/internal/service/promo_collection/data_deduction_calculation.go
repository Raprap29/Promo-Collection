package promocollection

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"promocollection/internal/pkg/consts"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/models"
	storeModels "promocollection/internal/pkg/store/models"
	"promocollection/internal/service/interfaces"
	pubsubsvc "promocollection/internal/service/pubsub"
)

type DeductionParams struct {
	MinCollectionAmount  float64
	ConversionRate       float64
	LoanUnpaidAmount     float64
	DeductableDataAmount float64
}

func DataToBeDeductedCalculation(
	ctx context.Context,
	payload *models.PromoEventMessage,
	params DeductionParams,
	pubsubPublisher interfaces.PubSubPublisherInterface,
	collectionTransactionsInProgressRepo interfaces.CollectionTransactionsInProgressRepositoryInterface,
	loansDocument *storeModels.Loans,
	unpaidDocument *storeModels.UnpaidLoans,
) error {

	logger.Info("Deductable Amount", slog.Float64("deductableAmount", params.DeductableDataAmount))
	outstandingDataEquivalent := params.LoanUnpaidAmount * params.ConversionRate

	switch {
	case params.DeductableDataAmount == outstandingDataEquivalent:
		return handleFullCollection(ctx,
			payload,
			params,
			pubsubPublisher,
			loansDocument,
			unpaidDocument,
		)

	case params.DeductableDataAmount < outstandingDataEquivalent:
		return handlePartialOrSkipCollection(
			ctx,
			payload,
			params,
			pubsubPublisher,
			collectionTransactionsInProgressRepo,
			loansDocument,
			unpaidDocument,
		)

	default:
		return nil
	}
}

func handleFullCollection(
	ctx context.Context,
	payload *models.PromoEventMessage,
	params DeductionParams,
	pubsubPublisher interfaces.PubSubPublisherInterface,
	loansDocument *storeModels.Loans,
	unpaidDocument *storeModels.UnpaidLoans,
) error {
	msg := buildPubSubMsg(ctx, payload, params.DeductableDataAmount, loansDocument, unpaidDocument, consts.FullCollection)
	if _, err := pubsubsvc.PubSubPublisher(ctx, pubsubPublisher, msg); err != nil {
		logger.CtxError(ctx, log_messages.ErrorInMessagePublishing, err)
		return err
	}
	logger.CtxInfo(ctx, "Published full collection payload",
		slog.Float64("deductableDataAmount", params.DeductableDataAmount))
	return nil
}

func handlePartialOrSkipCollection(
	ctx context.Context,
	payload *models.PromoEventMessage,
	params DeductionParams,
	pubsubPublisher interfaces.PubSubPublisherInterface,
	collectionTransactionsInProgressRepo interfaces.CollectionTransactionsInProgressRepositoryInterface,
	loansDocument *storeModels.Loans,
	unpaidDocument *storeModels.UnpaidLoans,
) error {
	deductionPesoEquivalent := params.DeductableDataAmount / params.ConversionRate
	if deductionPesoEquivalent >= params.MinCollectionAmount {
		finalDataDeductionAmount := math.Floor(deductionPesoEquivalent) * params.ConversionRate
		msg := buildPubSubMsg(ctx, payload, finalDataDeductionAmount, loansDocument, unpaidDocument, consts.PartialCollection)
		if _, err := pubsubsvc.PubSubPublisher(ctx, pubsubPublisher, msg); err != nil {
			logger.CtxError(ctx, log_messages.ErrorInMessagePublishing, err)
			return err
		}
		logger.CtxInfo(ctx, "Published partial collection payload",
			slog.Float64("finalDataDeductionAmount", finalDataDeductionAmount),
			slog.Float64("minCollectionAmount", params.MinCollectionAmount),
			slog.Any("message", msg))
		return nil
	}

	msisdnStr := strconv.FormatInt(payload.MSISDN, 10)
	if collectionTransactionsInProgressRepo != nil {
		if err := collectionTransactionsInProgressRepo.DeleteEntry(ctx, msisdnStr); err != nil {
			logger.CtxError(ctx, "failed to delete CollectionTransactionInProgress entry",
				err, slog.String("msisdn", msisdnStr))
			return err
		}
		logger.CtxInfo(ctx,
			"deductionPesoEquivalent < minCollectionAmount Deleted CollectionTransactionInProgress entry",
			slog.String("msisdn", msisdnStr))
		return nil
	}

	logger.CtxInfo(ctx, "Skipping collection, no collectionTransactionsInProgressRepo provided",
		slog.Float64("deductionPesoEquivalent", deductionPesoEquivalent),
		slog.Float64("minCollectionAmount", params.MinCollectionAmount))
	return nil
}

func buildPubSubMsg(
	ctx context.Context,
	payload *models.PromoEventMessage,
	amount float64, loan *storeModels.Loans,
	unpaidDocument *storeModels.UnpaidLoans,
	collectionType string,
) models.PublishtoWorkerMsgFormat {
	// Calculate ageing as days since loan creation
	var ageing int32
	if !loan.CreatedAt.IsZero() {
		ageing = int32(time.Since(loan.CreatedAt).Hours() / 24)
	}

	return models.PublishtoWorkerMsgFormat{
		Msisdn:     strconv.FormatInt(payload.MSISDN, 10),
		Unit:       consts.Unit,
		IsRollBack: consts.IsRollBack,
		Duration:   consts.Duration,
		Channel:    consts.Channel,

		SvcId:            payload.SVCID,
		Denom:            payload.DENOM,
		WalletKeyword:    payload.WALLETKEYWORD,
		WalletAmount:     fmt.Sprintf("%d", payload.WALLETAMOUNT),
		SvcDenomCombined: fmt.Sprintf("%s-%s", payload.DENOM, payload.WALLETKEYWORD),

		KafkaId:        payload.ID,
		CollectionType: collectionType,
		Ageing:         ageing,

		AvailmentTransactionId: loan.AvailmentTransactionID,
		LoanId:                 loan.LoanID,
		UnpaidLoanId:           unpaidDocument.LoanId,
		LastCollectionId:       unpaidDocument.LastCollectionID,
		BrandId:                loan.BrandID,
		LoanProductId:          loan.LoanProductID,

		ServiceFee:               float64(loan.ServiceFee),
		TotalLoanAmountInPeso:    float64(loan.TotalLoanAmount),
		TotalUnpaidAmountInPeso:  loan.TotalUnpaidLoan,
		AmountToBeDeductedInPeso: amount / loan.ConversionRate,
		UnpaidServiceFee:         float64(unpaidDocument.UnpaidServiceFee),
		DataToBeDeducted:         amount,

		StartDate:              loan.CreatedAt,
		LastCollectionDateTime: unpaidDocument.LastCollectionDateTime,

		LoanType:                     string(loan.LoanType),
		DataCollectionRequestTraceId: logger.GetTraceID(ctx),
		GUID:                         loan.GUID,
		Version:                      unpaidDocument.Version,
	}
}
