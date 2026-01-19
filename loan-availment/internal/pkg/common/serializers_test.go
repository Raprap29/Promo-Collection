package common

import (
	"testing"
	"time"

	"globe/dodrio_loan_availment/internal/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSerializeAvailments(t *testing.T) {
	id := primitive.NewObjectID()
	availment := SerializeAvailments("09171234567", "APP", "GLOBE", "DATA", id, "KEY", "PARTNER", 100.0, 10.0, "APP", "No Error", "0", true, 85.5, "Product A")

	assert.Equal(t, "09171234567", availment.MSISDN)
	assert.Equal(t, "DATA", availment.LoanType)
	assert.Equal(t, true, availment.Result)
	assert.NotEmpty(t, availment.GUID)
	assert.Equal(t, id, availment.ProductID)
}

func TestSerializeAvailmentLoanLog(t *testing.T) {
	start := time.Now().Add(-5 * time.Second)
	log := SerializeAvailmentLoanLog("09171234567", "SUCCESS", start, "APP", uuid.New().String(), "200")

	assert.Equal(t, "09171234567", log.MSISDN)
	assert.Equal(t, "SUCCESS", log.Status)
	assert.True(t, log.TAT >= 5)
}

func TestSerializeLoans(t *testing.T) {
	id := primitive.NewObjectID()
	brand := primitive.NewObjectID()
	availmentId := primitive.NewObjectID()
	rules := models.SystemLevelRules{
		DefermentPeriod: 3,
		EducationPeriod: 1,
		LoanLoadPeriod:  7,
		PesoToDataConversion: []models.PesoToDataConversion{
			{LoanType: "SCORED", ConversionRate: 1.5},
		},
		MinCollectionAmount:      10,
		MaxDataCollectionPercent: 80,
	}

	loanProduct := models.LoanProduct{
		ProductId:       id,
		IsScoredProduct: true,
		LoanType:        "DATA",
	}

	loan := SerializeLoans("09171234567", 100, "guid-123", 5, loanProduct, brand, availmentId, rules)

	assert.Equal(t, "09171234567", loan.MSISDN)
	assert.Equal(t, id, loan.LoanProductId)
	assert.Equal(t, 1.5, loan.ConversionRate)
}

func TestSerializeUnpaidLoans(t *testing.T) {
	loanId := primitive.NewObjectID()
	unpaid := SerializeUnpaidLoans(loanId, 100.0, 5.0)

	assert.Equal(t, loanId, unpaid.LoanId)
	assert.Equal(t, 100.0, unpaid.TotalLoanAmount)
	assert.Equal(t, 5.0, unpaid.UnpaidServiceFee)
}

func TestSerializeTransactionInProgress(t *testing.T) {
	tip := SerializeTransactionInProgress("09171234567", "KEYWORD")

	assert.Equal(t, "09171234567", tip.MSISDN)
	assert.False(t, tip.Id.IsZero())
}

func TestSerializeAvailmentKafkaMessage(t *testing.T) {
	message := SerializeAvailmentKafkaMessage(
		"tx123", "APP", "09171234567", "GLOBE", "DATA", "Product A", "KEY", "PARTNER", 100, 5, "SUCCESS", "0", 3, "OPEN")

	assert.Contains(t, message, "tx123")
	assert.Contains(t, message, "09171234567")
	assert.Contains(t, message, "GLOBE")
	assert.Contains(t, message, "100")
}

func TestSerializeEligibilityCheckResponse(t *testing.T) {
	subscriberScore := "80"
	remainingScore := "20"
	existingLoan := true
	keyword := "DATA"
	amount := 100.0
	brand := "GLOBE"

	resp := SerializeEligibilityCheckResponse("09171234567", &subscriberScore, &remainingScore, &existingLoan, &keyword, &amount, &brand, "SUCCESS", "200")

	assert.Equal(t, "SUCCESS", resp.Result)
	assert.Equal(t, &subscriberScore, resp.SubscriberScore)
	assert.Equal(t, &keyword, resp.LoanKeyWord)
	assert.Equal(t, &amount, resp.LoanAmount)
}

func TestSerializeAppCreationRequest(t *testing.T) {
	req := SerializeAppCreationRequest("09171234567", "tx123", "DATA", "APP")

	assert.Equal(t, "09171234567", req.MSISDN)
	assert.Equal(t, "tx123", req.TransactionId)
}

func TestSerializeAppCreationProcessRequest(t *testing.T) {
	start := time.Now()
	brand := primitive.NewObjectID()

	req := SerializeAppCreationProcessRequest("09171234567", "APP", "KEY", "APP", "tx123", 80.0, "PREPAID", "GLOBE", brand, start)

	assert.Equal(t, "09171234567", req.MSISDN)
	assert.Equal(t, "KEY", req.Keyword)
	assert.Equal(t, 80.0, req.CreditScore)
}

func TestSerializeSubscriber(t *testing.T) {
	sub := SerializeSubscriber("09171234567", true, false, "fraud", "PREPAID", "NEW")

	assert.True(t, sub.Blacklisted)
	assert.Equal(t, "fraud", sub.ExclusionReason)
}

func TestSerializeSubscriberEntry(t *testing.T) {
	now := time.Now()
	sub := SerializeSubscriberEntry("09171234567", false, "", "PREPAID", "EXISTING", 100.0, &now)

	assert.Equal(t, "09171234567", sub.MSISDN)
	assert.Equal(t, 100.0, sub.CreditLoanLimitAmount)
	assert.NotNil(t, sub.CreditLoanLimitAmountGenDate)
}
