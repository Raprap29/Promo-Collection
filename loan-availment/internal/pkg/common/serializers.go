package common

import (
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SerializeAvailments(msisdn string, channel string, brand string, loanType string, productId primitive.ObjectID, keyword string, servicingPartner string, totalLoanAmount float64, serviceFee float64, systemClient string, errorText string, errorCode string, result bool, creditScore float64, productName string) models.Availments {

	return models.Availments{
		ID:               primitive.NewObjectID(),
		MSISDN:           msisdn,
		Channel:          channel,
		Brand:            brand,
		LoanType:         loanType,
		ProductID:        productId,
		Keyword:          keyword,
		ServicingPartner: servicingPartner,
		Result:           result,
		ErrorText:        errorText,
		ErrorCode:        errorCode,
		TotalLoanAmount:  totalLoanAmount,
		ServiceFee:       serviceFee,
		GUID:             uuid.NewString(),
		CreatedAt:        time.Now(),
		PublishedToKafka: false,
		CreditScore:      creditScore,
		ProductName:      productName,
	}

}
func SerializeAvailmentLoanLog(msisdn string, status string, startTime time.Time, Channel string, GUID string, ErrorCode string) models.LoanAvaimentLog {

	EndTime := time.Now()
	TAT := EndTime.Sub(startTime).Seconds()

	return models.LoanAvaimentLog{
		TypeOfTransaction: consts.TransactionTypeForAvailmentReport,
		Status:            status,
		Channel:           Channel,
		TAT:               TAT,
		StartTime:         startTime,
		EndTime:           EndTime,
		TransactionID:     GUID,
		MSISDN:            msisdn,
		ErrorCode:         ErrorCode,
	}

}

func SerializeLoans(msisdn string, totalLoanAmount float64, guid string, serviceFee float64, loanProduct models.LoanProduct, brandId primitive.ObjectID, availmentTransactionId primitive.ObjectID, systemLevelRules models.SystemLevelRules) models.Loans {

	conversionRate, _ := ConversionRateByType(systemLevelRules.PesoToDataConversion, loanProduct.IsScoredProduct)

	// Calculate LoanLoadPeriodTimestamp only for LOAD SKU loan type
	var loanLoadPeriodTimestamp primitive.DateTime
	if loanProduct.LoanType == consts.LoanLoadType {
		loanLoadPeriodTimestamp = CalculateTimestamp(systemLevelRules.LoanLoadPeriod)
	} else {
		// For other loan types , set to current time
		loanLoadPeriodTimestamp = primitive.NewDateTimeFromTime(time.Now())
	}

	return models.Loans{
		LoanId:                        primitive.NewObjectID(),
		MSISDN:                        msisdn,
		TotalLoanAmount:               totalLoanAmount,
		GUID:                          guid,
		ServiceFee:                    serviceFee,
		LoanProductId:                 loanProduct.ProductId,
		BrandId:                       brandId,
		AvailmentTransactionId:        availmentTransactionId,
		LoanType:                      loanProduct.LoanType,
		CreatedAt:                     time.Now(),
		Migrated:                      false,
		TotalUnpaidLoan:               totalLoanAmount,
		PromoEducationPeriodTimestamp: CalculateTimestamp(systemLevelRules.PromoEducationPeriod),
		LoanLoadPeriodTimestamp:       loanLoadPeriodTimestamp,
		ConversionRate:                conversionRate,
		MinCollectionAmount:           systemLevelRules.MinCollectionAmount,
		MaxDataCollectionPercent:      systemLevelRules.MaxDataCollectionPercent,
		Version:                       1,
	}

}

func SerializeUnpaidLoans(loanId primitive.ObjectID, totalLoanAmount float64, serviceFee float64) models.UnpaidLoans {

	return models.UnpaidLoans{
		ID:                 primitive.NewObjectID(),
		LoanId:             loanId,
		Version:            1,
		ValidFrom:          time.Now(),
		ValidTo:            nil,
		TotalLoanAmount:    totalLoanAmount,
		UnpaidAmount:       totalLoanAmount,
		UnpaidServiceFee:   serviceFee,
		LastCollectionId:   nil,
		LastCollectionDate: nil,
		Migrated:           false,
	}

}

func SerializeTransactionInProgress(msisdn string, keyword string) models.TransactionInProgress {

	return models.TransactionInProgress{
		Id:        primitive.NewObjectID(),
		MSISDN:    msisdn,
		CreatedAt: time.Now(),
	}

}

func SerializeAvailmentKafkaMessage(transactionID string, availmentChannel string, msisdn string, brandType string, loanType string, loanProductName string, loanProductKeyword string, servicingPartner string, loanAmount float64, serviceFeeAmount float64, availmentResult string, availmentErrorCode string, loanAgeing int32, statusOfProvisionedLoan string) string {
	// if servicingPartner != "" {
	// 	servicingPartner = consts.ServicingPartner
	// }
	values := []string{
		transactionID,
		time.Now().Format("01/02/2006 15:04"), // Format the current time as a string
		availmentChannel,
		msisdn,
		brandType,
		loanType,
		loanProductName,
		loanProductKeyword,
		servicingPartner,
		fmt.Sprintf("%.0f", loanAmount),       // Format float as a string
		fmt.Sprintf("%.0f", serviceFeeAmount), // Format float as a string
		consts.TransactionTypeForAvailmentReport,
		availmentResult,
		availmentErrorCode,
		fmt.Sprintf("%d", loanAgeing), // Convert int32 to string
		statusOfProvisionedLoan,
	}
	return strings.Join(values, ",")

}

func SerializeEligibilityCheckResponse(msisdn string, subscriberScore *string, remainingScore *string, existingLoan *bool, loanKeyWord *string, loanAmount *float64, brand *string, result string, statusCode string) models.EligibilityCheckResponse {

	return models.EligibilityCheckResponse{
		MSISDN:          msisdn,
		SubscriberScore: subscriberScore,
		RemainingScore:  remainingScore,
		ExistingLoan:    existingLoan,
		LoanKeyWord:     loanKeyWord,
		LoanAmount:      loanAmount,
		// Brand:           brand,
		Result:     result,
		StatusCode: statusCode,
	}

}

func SerializeAppCreationRequest(msisdn string, transactionId string, keyword string, systemClient string) models.AppCreationRequest {
	return models.AppCreationRequest{
		MSISDN:        msisdn,
		TransactionId: transactionId,
		Keyword:       keyword,
		SystemClient:  systemClient,
	}
}

func SerializeAppCreationProcessRequest(msisdn string, channel string, keyword string, systemClient string, transactionId string, creditscore float64, customer_type string, brand_type string, brandId primitive.ObjectID, startTime time.Time) models.AppCreationProcessRequest {
	return models.AppCreationProcessRequest{
		MSISDN:        msisdn,
		Channel:       channel,
		Keyword:       keyword,
		SystemClient:  systemClient,
		TransactionID: transactionId,
		CreditScore:   creditscore,
		CustomerType:  customer_type,
		BrandCode:     brand_type,
		StartTime:     startTime,
		BrandId:       brandId,
	}
}
func SerializeSubscriber(msisdn string, blacklisted bool, migrated bool, exclusionReason string, subscriberType string, customerType string) models.Subscribers {
	return models.Subscribers{
		MSISDN:          msisdn,
		Blacklisted:     blacklisted,
		Migrated:        migrated,
		ExclusionReason: exclusionReason,
		SubscriberType:  subscriberType,
		CustomerType:    customerType,
	}
}
func SerializeSubscriberEntry(msisdn string, blacklisted bool, exclusionReason string, subscriberType string, customerType string, creditAmount float64, creditDate *time.Time) models.Subscribers {
	var creditDatePtr *time.Time
	if !creditDate.IsZero() {
		creditDatePtr = creditDate
	} else {
		creditDatePtr = nil
	}
	return models.Subscribers{
		MSISDN:                       msisdn,
		Blacklisted:                  blacklisted,
		ExclusionReason:              exclusionReason,
		SubscriberType:               subscriberType,
		CustomerType:                 customerType,
		CreditLoanLimitAmountGenDate: creditDatePtr,
		CreditLoanLimitAmount:        creditAmount,
	}
}
