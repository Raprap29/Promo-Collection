package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/pubsub"
	"globe/dodrio_loan_availment/internal/pkg/store"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Repository interfaces
type MessagesRepo interface {
	GetMessageID(ctx context.Context, event string, brand primitive.ObjectID, promoEnabled bool) (*models.MessageResponse, error)
}

type SystemLevelRulesRepo interface {
	SystemLevelRules(filter interface{}) (models.SystemLevelRules, error)
}

type ActiveLoanRepo interface {
	ActiveLoanByFilter(filter interface{}) (*models.Loans, error)
}

type LoansProductsRepo interface {
	LoanProductByFilter(filter interface{}) (*models.LoanProduct, error)
}

type UnpaidLoanRepo interface {
	UnpaidLoansByFilter(filter interface{}) (*models.UnpaidLoans, error)
}

// NotificationService handles all notification-related operations
type NotificationService struct {
	messageRepo             MessagesRepo
	activeLoanRepo          ActiveLoanRepo
	loansProductsRepository LoansProductsRepo
	unpaidLoanRepo          UnpaidLoanRepo
	systemLevelRulesRepo    SystemLevelRulesRepo
	pubsubPublisher         pubsub.PubSubPublisherInterface
}

// NewNotificationService creates a new NotificationService with default implementations
func NewNotificationService(pubsubPublisher pubsub.PubSubPublisherInterface) *NotificationService {
	return &NotificationService{
		messageRepo:             store.NewMessagesRepository(),
		activeLoanRepo:          store.NewActiveLoanRepository(),
		loansProductsRepository: store.NewLoansProductsRepository(),
		unpaidLoanRepo:          store.NewUnpaidLoanRepository(),
		systemLevelRulesRepo:    store.NewSystemLevelRulesRepository(),
		pubsubPublisher:         pubsubPublisher,
	}
}

// NotifyUser sends a notification to a user based on the event and other parameters
func (h *NotificationService) NotifyUser(ctx context.Context, MSISDN string, event string, brand primitive.ObjectID, product *models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) error {
	// Fetch system level rules to check if promo collection is enabled
	systemRules, err := h.systemLevelRulesRepo.SystemLevelRules(bson.M{})
	if err != nil {
		logger.Error(ctx, "Failed to fetch system level rules: %v", err)
		return err
	}

	response, err := h.messageRepo.GetMessageID(ctx, event, brand, systemRules.PromoCollectionEnabled)
	if err != nil {
		return err
	}

	// Convert pointer to value for the struct approach
	var productValue models.LoanProduct
	if product != nil {
		productValue = *product
	}
	params := h.getValuesOfParameters(response.Parameters, MSISDN, productValue, loanId, loanDate)

	payload := models.SmsNotificationRequestPayload{
		NotificationParameter: params,
		PatternID:             response.MessageID,
	}

	logger.Info(ctx, "NotifyUser MSISDN: %v PatternID: %v LoanID: %v LoanDate: %v", MSISDN, response.MessageID, loanId, loanDate)

	// Send notification to PubSub
	err = h.sendNotificationToPubSub(ctx, payload, MSISDN, event)
	if err != nil {
		logger.Error(ctx, "Failed to send notification to PubSub: %v", err)
		return err
	}

	logger.Info(ctx, "NotifyUser payload: %v", payload)

	return nil
}

// sendNotificationToPubSub sends the notification payload to PubSub
func (h *NotificationService) sendNotificationToPubSub(ctx context.Context, payload models.SmsNotificationRequestPayload, msisdn string, event string) error {
	// Convert the current payload to the protobuf-compatible format
	var notifParameters []models.SmsNotificationParameter
	for _, param := range payload.NotificationParameter {
		notifParameters = append(notifParameters, models.SmsNotificationParameter(param))
	}

	// Create the protobuf-compatible SMS notification request
	smsRequest := models.SmsNotificationRequest{
		Msisdn:          msisdn,
		SmsDbEventName:  event,
		NotifParameters: notifParameters,
		PatternID:       payload.PatternID,
	}

	// Convert to JSON
	payloadBytes, err := json.Marshal(smsRequest)
	if err != nil {
		logger.Error(ctx, "Failed to marshal SMS notification request: %v", err)
		return fmt.Errorf("failed to marshal SMS request: %w", err)
	}

	topicName := configs.PUBSUB_TOPIC

	// Create a separate context with timeout for PubSub operation to avoid cancellation issues
	pubsubCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messageID, err := h.pubsubPublisher.Publish(pubsubCtx, topicName, payloadBytes, nil)
	if err != nil {
		logger.Error(ctx, "Failed to publish SMS notification to PubSub topic %s: %v", topicName, err)
		return fmt.Errorf("failed to publish to pubsub: %w", err)
	}

	logger.Info(ctx, "Successfully published SMS notification to PubSub topic %s with message ID: %s", topicName, messageID)
	return nil
}

func (h *NotificationService) getLoanResult(MSISDN string) *models.Loans {
	filter := bson.M{"MSISDN": MSISDN}
	loanResult, err := h.activeLoanRepo.ActiveLoanByFilter(filter)
	if err != nil {
		return nil
	}
	return loanResult
}

func (h *NotificationService) computeFeeFromProduct(product *models.LoanProduct) float64 {
	if strings.EqualFold(product.FeeType, string(consts.Percent)) {
		return (product.Price * product.ServiceFee) / 100
	}
	return product.ServiceFee
}

// Helper function to format float to 2 decimal places
func formatFloat(value float64) string {
	return fmt.Sprintf(consts.FloatTwoDecimalFormat, value)
}

// ParameterRawData holds the raw numeric/data values (like your Test struct)
type ParameterRawData struct {
	SkuName              string
	ServiceFeeAmount     float64
	LoanFeeAmount        float64
	RemainingAmount      float64
	LoanDateTime         time.Time
	AmountCollectedValue float64
	DataConversionRate   float64
	DataMBValue          float64
	DataCollectionTime   time.Time
}

// ParameterFormattedData holds the formatted string values (like your TestImp struct)
type ParameterFormattedData struct {
	ParameterRawData
	// Additional formatted fields if needed
	ServiceFeeFormatted      string
	LoanFeeFormatted         string
	RemainingAmountFormatted string
	AmountCollectedFormatted string
	DataConversionFormatted  string
	DataMBFormatted          string
	LoanDateFormatted        string
	DataCollectionFormatted  string
}

// toFormattedMap converts raw data to formatted strings (like your toMap method)
func (p *ParameterRawData) toFormattedMap() *ParameterFormattedData {
	return &ParameterFormattedData{
		ParameterRawData:         *p,
		ServiceFeeFormatted:      formatFloat(p.ServiceFeeAmount),
		LoanFeeFormatted:         formatFloat(p.LoanFeeAmount),
		RemainingAmountFormatted: formatFloat(p.RemainingAmount),
		AmountCollectedFormatted: formatFloat(p.AmountCollectedValue),
		DataConversionFormatted:  fmt.Sprintf("%s MB", formatFloat(p.DataConversionRate)),
		DataMBFormatted:          fmt.Sprintf("%s MB", formatFloat(p.DataMBValue)),
		LoanDateFormatted:        p.LoanDateTime.Format(consts.DateFormat),
		DataCollectionFormatted:  p.DataCollectionTime.Format(consts.DateFormat),
	}
}

// Helper method to populate raw data from models
func (h *NotificationService) createParameterRawData(MSISDN string, product models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) *ParameterRawData {
	rawData := &ParameterRawData{
		LoanDateTime: loanDate,
	}

	// Get loan result
	loanResult := h.getLoanResult(MSISDN)

	// Set SKU Name
	if product.Name != "" { // Check if product has data instead of nil check
		rawData.SkuName = product.Name
	} else if loanResult != nil {
		if fetchedProduct, err := h.loansProductsRepository.LoanProductByFilter(bson.M{"_id": loanResult.LoanProductId}); err == nil && fetchedProduct != nil {
			rawData.SkuName = fetchedProduct.Name
		}
	}

	// Set Service Fee
	if product.Name != "" { // Check if product has data instead of nil check
		rawData.ServiceFeeAmount = h.computeFeeFromProduct(&product) // Pass address of value
	} else if loanResult != nil {
		if fetchedProduct, err := h.loansProductsRepository.LoanProductByFilter(bson.M{"_id": loanResult.LoanProductId}); err == nil && fetchedProduct != nil {
			rawData.ServiceFeeAmount = h.computeFeeFromProduct(fetchedProduct)
		}
	}

	// Set Loan Fee Amount
	if loanResult != nil {
		rawData.LoanFeeAmount = loanResult.TotalLoanAmount
	}

	// Get unpaid loan data
	if loanId == primitive.NilObjectID && loanResult != nil {
		loanId = loanResult.LoanId
	}

	if loanId != primitive.NilObjectID {
		if unpaidLoan, err := h.unpaidLoanRepo.UnpaidLoansByFilter(bson.M{"loanId": loanId}); err == nil && unpaidLoan != nil {
			rawData.RemainingAmount = unpaidLoan.UnpaidAmount
			rawData.AmountCollectedValue = unpaidLoan.TotalLoanAmount - unpaidLoan.UnpaidAmount
			if rawData.LoanFeeAmount == 0 {
				rawData.LoanFeeAmount = unpaidLoan.TotalLoanAmount
			}
		}
	}

	// Set Data values
	if loanResult != nil {
		rawData.DataConversionRate = loanResult.ConversionRate
		rawData.DataMBValue = loanResult.ConversionRate * loanResult.TotalUnpaidLoan
		rawData.DataCollectionTime = common.ConvertUTCToPHT(loanResult.PromoEducationPeriodTimestamp.Time())
	}

	return rawData
}

// Alternative getValuesOfParameters using the struct approach
func (h *NotificationService) getValuesOfParameters(parameters []string, MSISDN string, product models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) []models.NotificationParameter {
	var result []models.NotificationParameter

	// Create raw data struct
	rawData := h.createParameterRawData(MSISDN, product, loanId, loanDate)

	// Convert to formatted data using toFormattedMap (like your toMap method)
	formattedData := rawData.toFormattedMap()

	for _, param := range parameters {
		var value string

		switch param {
		case consts.SkuName, consts.LoanSku:
			value = formattedData.SkuName
		case consts.ServiceFee:
			value = formattedData.ServiceFeeFormatted
		case consts.LoanFeeAmt:
			value = formattedData.LoanFeeFormatted
		case consts.RemainingLoanAmount:
			value = formattedData.RemainingAmountFormatted
		case consts.LoanDate:
			value = formattedData.LoanDateFormatted
		case consts.AmountCollected:
			value = formattedData.AmountCollectedFormatted
		case consts.DataConversion:
			value = formattedData.DataConversionFormatted
		case consts.DataMB:
			value = formattedData.DataMBFormatted
		case consts.DataCollectionDate:
			value = formattedData.DataCollectionFormatted
		default:
			value = ""
		}

		result = append(result, models.NotificationParameter{
			Name:  param,
			Value: value,
		})
	}

	return result
}
