package promocollection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"promocollection/internal/pkg/consts"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	notifmodel "promocollection/internal/pkg/models"
	"promocollection/internal/pkg/store/models"
	"promocollection/internal/service/interfaces"
	"promocollection/utils"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type BusinessLevelRulesService struct {
	loanRepo                             interfaces.LoanRepositoryInterface
	loanProductsRepo                     interfaces.LoanProductsRepositoryInterface
	messagesRepo                         interfaces.MessagesRepositoryInterface
	redisRepo                            interfaces.RedisStoreOperations
	collectionTransactionsInProgressRepo interfaces.CollectionTransactionsInProgressRepositoryInterface
	whitelistedForDataCollectionRepo     interfaces.WhitelistedForDataCollectionRepositoryInterface
	systemRulesRepo                      interfaces.SystemLevelRulesRepository
	pubsub                               interfaces.PubSubPublisherInterface
	pubsubnotifclient                    interfaces.PubSubPublisherInterface
	payload                              *notifmodel.PromoEventMessage
	unpaidRepo                           interfaces.UnpaidLoansRepositoryInterface
}

func NewBusinessLevelRulesService(
	loanRepo interfaces.LoanRepositoryInterface,
	messagesRepo interfaces.MessagesRepositoryInterface,
	loanProductsRepo interfaces.LoanProductsRepositoryInterface,
	redisRepo interfaces.RedisStoreOperations,
	collectionTransactionsInProgressRepo interfaces.CollectionTransactionsInProgressRepositoryInterface,
	whitelistedForDataCollectionRepo interfaces.WhitelistedForDataCollectionRepositoryInterface,
	systemRulesRepo interfaces.SystemLevelRulesRepository,
	pubsub, pubsubnotifclient interfaces.PubSubPublisherInterface,
	payload *notifmodel.PromoEventMessage,
	unpaidRepo interfaces.UnpaidLoansRepositoryInterface,
) *BusinessLevelRulesService {
	return &BusinessLevelRulesService{
		loanRepo:                             loanRepo,
		messagesRepo:                         messagesRepo,
		loanProductsRepo:                     loanProductsRepo,
		redisRepo:                            redisRepo,
		collectionTransactionsInProgressRepo: collectionTransactionsInProgressRepo,
		whitelistedForDataCollectionRepo:     whitelistedForDataCollectionRepo,
		systemRulesRepo:                      systemRulesRepo,
		pubsub:                               pubsub,
		pubsubnotifclient:                    pubsubnotifclient,
		payload:                              payload,
		unpaidRepo:                           unpaidRepo,
	}
}

// nolint:funlen
func (s *BusinessLevelRulesService) CheckBusinessLevelRulesForPromoCollection(
	ctx context.Context,
	msisdn string,
) (bool, error) {

	// Step 1: Feature enablement
	isEnabled, err := s.isPromoCollectionEnabled(ctx)
	if err != nil {
		logger.CtxError(ctx, "SkippingCollection: Error in PromoCollectionEnabledCheck", err, slog.String("msisdn", msisdn))
		return false, err
	}
	if !isEnabled {
		logger.CtxInfo(ctx, "SkippingCollection: Promo Collection disabled", slog.String("msisdn", msisdn))
		return false, nil
	}

	// Step 2: Existing transaction check
	exists, err := s.isEntryInCollectionTransactionInProgress(ctx, msisdn)
	if err != nil {
		logger.CtxError(ctx, "SkippingCollection: Error in InCollectionTransactionInProgressCheck",
			err, slog.String("msisdn", msisdn))
		return false, err
	}
	if exists {
		logger.CtxInfo(ctx, "SkippingCollection: entry already in progress", slog.String("msisdn", msisdn))
		return false, nil
	}

	// Step 3: Create entry
	if err := s.createEntryInCollectionTransactionInProgress(ctx, msisdn); err != nil {
		logger.CtxError(ctx, "SkippingCollection: failed to create CollectionTransactionInProgress",
			err, slog.String("msisdn", msisdn))
		return false, err
	}

	shouldCleanup := true

	defer func() {
		if shouldCleanup {
			if err := s.collectionTransactionsInProgressRepo.DeleteEntry(ctx, msisdn); err != nil {
				logger.CtxError(ctx, "Failed to delete entry from CollectionTransactionInProgress",
					err, slog.String("msisdn", msisdn))
			}
		}
	}()

	// Step 4: Redis check
	isActivePeriod, err := s.checkRedisForActivePeriod(ctx, msisdn)
	if err != nil {
		logger.CtxError(ctx, "SkippingCollection: Error in RedisActivePeriodCheck", err, slog.String("msisdn", msisdn))
		return false, err
	}
	if isActivePeriod {
		logger.CtxInfo(ctx, "SkippingCollection: MSISDN in active period", slog.String("msisdn", msisdn))
		return false, nil
	}

	// Step 5: Get loan
	loan, err := s.getActiveLoan(ctx, msisdn)
	if err != nil {
		logger.CtxError(ctx, "SkippingCollection: Error during loan check", err, slog.String("msisdn", msisdn))
		return false, err
	}
	if loan == nil {
		logger.CtxInfo(ctx, "SkippingCollection: No active loan found", slog.String("msisdn", msisdn))
		return false, nil
	}

	// Step 6: Determine loan type
	isNewLoan := s.loanRepo.CheckIfOldOrNewLoan(ctx, loan)

	// Step 7: Old loan flow
	if !isNewLoan {
		logger.CtxInfo(ctx, "OldLoanDetected: Checking whitelist...", slog.String("msisdn", msisdn))

		isWhitelisted, err := s.whitelistedForDataCollectionRepo.IsMSISDNWhitelisted(ctx, msisdn)
		if err != nil {
			logger.CtxError(ctx, "SkippingCollection: whitelist check failed", err, slog.String("msisdn", msisdn))
			return false, err
		}

		if !isWhitelisted {
			logger.CtxInfo(ctx, "SkippingCollection: not whitelisted", slog.String("msisdn", msisdn))
			return false, nil
		}

		shouldProceed, err := s.processOldLoanForPromoCollection(ctx, msisdn, loan)
		if err != nil {
			logger.CtxError(ctx, "SkippingCollection: error processing old loan", err)
			return false, err
		}

		if !shouldProceed {
			logger.CtxInfo(ctx, "SkippingCollection: old loan not eligible for collection at this time")
			return false, nil
		}
	}

	// Step 8: New loan flow
	if isNewLoan {
		isInSkipPeriod := utils.IsSkipPeriodActive(
			loan.PromoEducationPeriodTimestamp,
			loan.LoanLoadPeriodTimestamp,
			loan.EducationPeriodTimestamp,
		)
		if isInSkipPeriod {
			if err := s.createSkippingCacheEntry(ctx, loan, msisdn); err != nil {
				return false, err
			}
			logger.CtxInfo(ctx, "SkippingCollection: Loan is in Active Skipping Period",
				slog.String("msisdn", msisdn),
			)
			return false, nil
		}

		logger.CtxInfo(ctx, "Loan is NOT in Active Skipping Period, proceeding to collection calculation",
			slog.String("msisdn", msisdn),
		)
	}

	// Step 9: Perform collection calculation
	if err := CollectionRules(
		ctx,
		s.payload,
		s.systemRulesRepo,
		s.loanRepo,
		s.pubsub,
		s.collectionTransactionsInProgressRepo,
		s.unpaidRepo,
		s.loanProductsRepo,
	); err != nil {
		logger.CtxError(ctx, "Error while performing data collection", err, slog.String("msisdn", msisdn))
		return false, err
	}

	shouldCleanup = false

	logger.CtxInfo(ctx, "Finished CheckBusinessLevelRulesForPromoCollection",
		slog.String("msisdn", msisdn))

	return true, nil
}

// checks if promo-based collection is enabled at the system level.
func (s *BusinessLevelRulesService) isPromoCollectionEnabled(ctx context.Context) (bool, error) {
	logger.CtxInfo(ctx, "Started PromoCollectionEnabledCheck...")

	// Step 1: Fetch system-level rules configuration from DB
	rules, err := s.systemRulesRepo.FetchSystemLevelRulesConfiguration(ctx)
	if err != nil {
		logger.CtxError(ctx, "PromoCollectionEnabledCheck: Failed to fetch system rules configuration", err)
		return false, err
	}

	// Step 2: Check promo collection flag in the rules config
	if rules.PromoCollectionEnabled {
		logger.CtxInfo(ctx, "PromoCollectionEnabledCheck: true")
		return true, nil
	}

	logger.CtxInfo(ctx, "PromoCollectionEnabledCheck: false")
	return false, nil
}

// checks if an CollectionTransactionsInProgress collection exists for the MSISDN to prevent duplicate processing.
func (s *BusinessLevelRulesService) isEntryInCollectionTransactionInProgress(
	ctx context.Context,
	msisdn string,
) (bool, error) {
	logger.CtxInfo(ctx, "Started CollectionTransactionInProgressCheck...")

	exists, err := s.collectionTransactionsInProgressRepo.CheckEntryExists(ctx, msisdn)
	if err != nil {
		logger.CtxError(ctx, "CollectionTransactionInProgressCheck: Error while checking Repository",
			err, slog.String("msisdn", msisdn))
		return false, err
	}

	if exists {
		logger.CtxInfo(ctx, "CollectionTransactionInProgressCheck: true",
			slog.String("msisdn", msisdn))
		return true, nil
	}

	logger.CtxInfo(ctx, "CollectionTransactionInProgressCheck: false", slog.String("msisdn", msisdn))
	return false, nil
}

// creates a new entry in CollectionTransactionsInProgress for MSISDN to mark a transaction as in-progress.
func (s *BusinessLevelRulesService) createEntryInCollectionTransactionInProgress(
	ctx context.Context,
	msisdn string,
) error {
	logger.CtxInfo(ctx, "Started CreateEntryInCollectionTransactionInProgress...")

	if err := s.collectionTransactionsInProgressRepo.CreateEntry(ctx, msisdn); err != nil {
		logger.CtxError(ctx, "CreateEntryInCollectionTransactionInProgress: Failed to create new entry in Repository", err,
			slog.String("msisdn", msisdn))
		return err
	}

	logger.CtxInfo(ctx, "CreateEntryInCollectionTransactionInProgress: Success",
		slog.String("msisdn", msisdn))
	return nil
}

// checks whether the MSISDN is currently within any active skip period by
// reading SkipTimestamps from Redis. Returns true if the promo education,
// loan load, or grace period is still active; triggers notification during
// an active promo education period; clears the Redis key when all periods
// have expired.
// nolint:funlen
func (s *BusinessLevelRulesService) checkRedisForActivePeriod(
	ctx context.Context,
	msisdn string,
) (bool, error) {
	logger.CtxInfo(ctx, "Started RedisActivePeriodCheck...")

	fetchingKey := models.SkipTimestampsKeyBuilder(msisdn)
	redisValue, err := s.redisRepo.Get(ctx, fetchingKey)

	// Condition 1: If Redis key not found → not in active period
	if errors.Is(err, redis.Nil) {
		logger.CtxInfo(ctx, "RedisActivePeriodCheck: No Redis entry found", slog.String("msisdn", msisdn))
		return false, nil
	}

	// Condition 2: If other Redis error → return error
	if err != nil {
		logger.CtxError(ctx, "RedisActivePeriodCheck: Error checking Redis for MSISDN",
			err, slog.String("msisdn", msisdn))
		return false, err
	}

	// Condition 3: Redis key found -> Parse existing Redis data
	logger.CtxInfo(ctx, "RedisActivePeriodCheck: Redis entry found for MSISDN", slog.String("msisdn", msisdn))

	var skipTimestamps models.SkipTimestamps
	if redisValueBytes, ok := redisValue.([]byte); ok {
		if err := json.Unmarshal(redisValueBytes, &skipTimestamps); err != nil {
			logger.CtxError(ctx, "RedisActivePeriodCheck: Failed to unmarshal SkipTimestamps",
				err, slog.String("msisdn", msisdn))
			return true, err
		}
	}

	now := time.Now()

	if now.Before(skipTimestamps.PromoEducationPeriodTimestamp) {
		logger.CtxInfo(ctx, "RedisActivePeriodCheck: MSISDN is in promo education period",
			slog.String("msisdn", msisdn),
			slog.Time("promoEducationPeriodEndsAt", skipTimestamps.PromoEducationPeriodTimestamp),
		)

		loan, err := s.loanRepo.GetLoanByMSISDN(ctx, msisdn)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxError(ctx, "RedisActivePeriodCheck: Failed to fetch active loans for MSISDN",
				err, slog.String("msisdn", msisdn))
			return true, err
		}

		if err := s.publishEducationNotification(ctx, loan); err != nil {
			logger.CtxError(ctx, "RedisActivePeriodCheck: Failed to publish education notification",
				err, slog.String("msisdn", msisdn))
			return true, err
		}

		logger.CtxInfo(ctx, "RedisActivePeriodCheck: Notification sent successfully", slog.String("msisdn", msisdn))
		return true, nil
	}

	if now.Before(skipTimestamps.LoanLoadPeriodTimestamp) {
		logger.CtxInfo(ctx, "RedisActivePeriodCheck: MSISDN still in loan load period",
			slog.String("msisdn", msisdn),
			slog.Time("loanLoadPeriodEndsAt", skipTimestamps.LoanLoadPeriodTimestamp),
		)
		return true, nil
	}

	if now.Before(skipTimestamps.GracePeriodTimestamp) {
		logger.CtxInfo(ctx, "RedisActivePeriodCheck: MSISDN still in grace period",
			slog.String("msisdn", msisdn),
			slog.Time("gracePeriodEndsAt", skipTimestamps.GracePeriodTimestamp),
		)
		return true, nil
	}

	// All periods expired → Delete the Redis key
	if err := s.redisRepo.Delete(ctx, fetchingKey); err != nil {
		logger.CtxError(ctx, "RedisActivePeriodCheck: Failed to delete expired Redis entry",
			err, slog.String("msisdn", msisdn))
		return true, err
	}
	logger.CtxInfo(ctx, "RedisActivePeriodCheck: Deleted expired Redis entry", slog.String("msisdn", msisdn))
	logger.CtxInfo(ctx, "RedisActivePeriodCheck: All skip periods expired", slog.String("msisdn", msisdn))

	return false, nil
}

// retrieves the active loan for the MSISDN from the Loans collection.
// Returns the loan if found, nil if no active loan exists,
func (s *BusinessLevelRulesService) getActiveLoan(
	ctx context.Context,
	msisdn string,
) (*models.Loans, error) {
	logger.CtxInfo(ctx, "Started GetActiveLoan...")

	loan, err := s.loanRepo.GetLoanByMSISDN(ctx, msisdn)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxInfo(ctx, "GetActiveLoan: No active loan found", slog.String("msisdn", msisdn))
			return nil, nil
		}

		logger.CtxError(ctx, "GetActiveLoan: Error fetching loan", err, slog.String("msisdn", msisdn))
		return nil, err
	}

	if loan == nil {
		logger.CtxError(ctx, "GetActiveLoan: GetLoanByMSISDN returned nil unexpectedly",
			nil, slog.String("msisdn", msisdn))
		return nil, fmt.Errorf("loan is nil for msisdn=%s", msisdn)
	}

	logger.CtxInfo(ctx, "GetActiveLoan: Active loan found", slog.String("msisdn", msisdn))
	return loan, nil
}

// processOldLoanForPromoCollection handles the full lifecycle of an existing (old) loan
// when entering the promo-collection flow. It:
//  1. Loads system-level rules
//  2. Recomputes loan parameters (conversion, education/loan-load periods)
//  3. Updates the loan document in MongoDB
//  4. Removes the MSISDN from the whitelist
//  5. Evaluates skipping periods and saves Redis timestamps when required
//  6. Publishes promo-education notifications if still within education period
//
// nolint:funlen
func (s *BusinessLevelRulesService) processOldLoanForPromoCollection(
	ctx context.Context,
	msisdn string,
	loan *models.Loans,
) (bool, error) {
	logger.CtxInfo(ctx, "Started ProcessOldLoanForPromoCollection...", slog.String("msisdn", msisdn))

	// Step 1: Fetch system-level rules
	systemLevelRulesDoc, err := s.systemRulesRepo.FetchSystemLevelRulesConfiguration(ctx)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFetchingSystemLevelRulesMongoDBDoc, err)
		return false, err
	}

	// Step 2: Compute updated loan parameters
	conversionRate, err := getConversionRateForLoanType(
		ctx, loan.LoanType, systemLevelRulesDoc.PesoToDataConversionMatrix,
	)
	if err != nil {
		return false, err
	}
	promoEducationPeriodTimestamp := loan.CreatedAt.AddDate(0, 0, int(systemLevelRulesDoc.PromoEducationPeriod))

	var loanLoadPeriodTimestamp time.Time
	if loan.LoanType == consts.LoanTypeLoad {
		loanLoadPeriodTimestamp = loan.CreatedAt.AddDate(0, 0, int(systemLevelRulesDoc.LoanLoadPeriod))
	}

	// Step 3: Update loan document
	id, err := s.loanRepo.UpdateOldLoanDocument(ctx,
		loan,
		conversionRate,
		promoEducationPeriodTimestamp,
		loanLoadPeriodTimestamp,
		systemLevelRulesDoc.MaxDataCollectionPercent,
		systemLevelRulesDoc.MinCollectionAmount,
		loan.TotalUnpaidLoan,
	)
	if err != nil {
		logger.CtxError(ctx, "OldLoanProcessing: Failed to update old loans doc",
			err, slog.String("loanId", loan.LoanID.String()))
		return false, err
	}
	logger.CtxInfo(ctx, "OldLoanProcessing: Loan document updated successfully", slog.Any("loanId", id))

	// Step 4: Delete MSISDN from whitelistedForDataCollectionRepo
	if err := s.whitelistedForDataCollectionRepo.DeleteByMSISDN(ctx, msisdn); err != nil {
		return false, fmt.Errorf("failed to delete from whitelist after period lapse: %w", err)
	}

	// Step 5: Check promo education and loan load period validity
	isInSkipPeriod := utils.IsSkipPeriodActive(promoEducationPeriodTimestamp, loanLoadPeriodTimestamp, time.Time{})
	if isInSkipPeriod {
		// 4.1 Create Redis entry
		redisEntry := models.SkipTimestamps{
			GracePeriodTimestamp:          time.Time{},
			PromoEducationPeriodTimestamp: promoEducationPeriodTimestamp,
			LoanLoadPeriodTimestamp:       loanLoadPeriodTimestamp,
		}
		if err := s.redisRepo.SaveSkipTimestamps(ctx, msisdn, redisEntry); err != nil {
			return false, fmt.Errorf("failed to save skiptimestamps: %w", err)
		}

		// 4.2 Publish education notification
		if promoEducationPeriodTimestamp.After(time.Now()) {
			if err := s.publishEducationNotification(ctx, loan); err != nil {
				return false, err
			}
		}

		logger.CtxInfo(ctx, "OldLoanProcessing: skipping further processing (active skipping period)",
			slog.String("msisdn", msisdn), slog.String("loanId", loan.LoanID.Hex()),
		)
		return false, nil
	}

	logger.CtxInfo(ctx, "Finished ProcessOldLoanForPromoCollection", slog.String("msisdn", msisdn))
	return true, nil
}

// createSkippingCacheEntry processes a newly created loan that is still within
// its skipping periods (promo education or loan-load period).
// It saves the loan's skip timestamps in Redis, sends an education notification
func (s *BusinessLevelRulesService) createSkippingCacheEntry(
	ctx context.Context,
	loan *models.Loans,
	msisdn string,
) error {
	logger.CtxInfo(ctx, "Started CreateSkippingCacheEntry...", slog.String("msisdn", msisdn))

	redisEntry := models.SkipTimestamps{
		GracePeriodTimestamp:          time.Time{},
		PromoEducationPeriodTimestamp: loan.PromoEducationPeriodTimestamp,
		LoanLoadPeriodTimestamp:       loan.LoanLoadPeriodTimestamp,
	}
	if err := s.redisRepo.SaveSkipTimestamps(ctx, msisdn, redisEntry); err != nil {
		return fmt.Errorf("failed to save skip timestamps: %w", err)
	}

	if loan.PromoEducationPeriodTimestamp.After(time.Now()) {
		if err := s.publishEducationNotification(ctx, loan); err != nil {
			return err
		}
	}

	logger.CtxInfo(ctx, "Non-lapsed loan: education notification sent, skipping collection",
		slog.String("msisdn", msisdn))
	return nil
}

func (s *BusinessLevelRulesService) publishEducationNotification(
	ctx context.Context,
	loan *models.Loans,
) error {
	logger.CtxInfo(ctx, "Started PublishEducationNotification...")
	loanProd, err := s.loanProductsRepo.GetProductNameById(ctx, loan.LoanProductID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		logger.CtxError(ctx, "Error fetching loanProducts for id", err,
			slog.String("LoanProductId", loan.LoanProductID.String()),
		)
		return err
	}

	messages, err := s.messagesRepo.GetPatternIdByEventandBrandId(ctx, consts.EducationPeriodSpiel, loan.BrandID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		logger.CtxError(ctx, "Error fetching patternId from event", err,
			slog.String("event", consts.EducationPeriodSpiel),
			slog.Any("brandId", loan.BrandID),
		)
		return err
	}

	paramValueMap := map[string]string{
		"REMAINING_LOAN_AMOUNT": formatFloat(loan.TotalUnpaidLoan),
		"SKU_NAME":              loanProd.Name,
		"SERVICE_FEE":           formatFloat(loan.ServiceFee),
		"LOAN_DATE":             loan.CreatedAt.Format(consts.SmsDateFormat),

		"DATA_COLLECTION_DATE": loan.PromoEducationPeriodTimestamp.Format(consts.SmsDateFormat),
		"DATA_MB":              fmt.Sprintf("%s MB", formatFloat(loan.TotalUnpaidLoan*loan.ConversionRate)),
		"DATA_CONVERSION":      fmt.Sprintf("%s MB", formatFloat(loan.ConversionRate)),
	}

	notifParams := make([]notifmodel.NotificationParameter, 0, len(messages.Parameters))
	for _, paramName := range messages.Parameters {
		value, ok := paramValueMap[paramName]
		if !ok {
			logger.CtxError(ctx, "Unknown notification param",
				nil, slog.String("param", paramName), slog.Int("patternId", int(messages.PatternId)),
			)
			return errors.New("unknown notification param")
		}
		notifParams = append(notifParams, notifmodel.NotificationParameter{
			Name:  paramName,
			Value: value,
		})
	}

	// Construct Pub/Sub notification message
	notifMsg := notifmodel.PubSubNotificationMessage{
		Msisdn:          loan.MSISDN,
		SmsDbEventName:  consts.EducationPeriodSpiel,
		NotifParameters: notifParams,
		PatternID:       messages.PatternId,
	}

	// Publish the message
	if _, err = s.pubsubnotifclient.PublishMessage(ctx, notifMsg); err != nil {
		logger.CtxError(ctx, "PublishEducationNotification: Error publishing", err)
		return fmt.Errorf(log_messages.ErrorInMessagePublishing, err)
	}
	logger.CtxInfo(ctx, "PublishEducationNotification: Published", slog.Any("message", notifMsg))
	return nil
}

func formatFloat(value float64) string {
	return fmt.Sprintf(consts.FloatTwoDecimalFormat, value)
}

func getConversionRateForLoanType(
	ctx context.Context,
	loanType consts.LoanType,
	matrix []models.PesoToDataConversion,
) (float64, error) {

	logger.CtxInfo(ctx, "GetConversionRate: Starting lookup...",
		slog.String("loanType", string(loanType)))

	normalizedLoanType := strings.ToUpper(strings.TrimSpace(string(loanType)))

	for _, entry := range matrix {
		entryType := strings.ToUpper(strings.TrimSpace(entry.LoanType))

		if entryType == normalizedLoanType {
			logger.CtxInfo(ctx, "GetConversionRate: Match found",
				slog.String("loanType", entry.LoanType),
				slog.Float64("conversionRate", entry.ConversionRate))
			return entry.ConversionRate, nil
		}
	}

	logger.CtxError(ctx, "GetConversionRate: No conversion rate found",
		nil,
		slog.String("loanType", string(loanType)),
	)

	return 0, fmt.Errorf("conversion rate not found for loanType=%s", loanType)
}
