package models

import (
	"fmt"
	"promo-collection-worker/internal/pkg/consts"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Context key for publish time
type publishTimeKey struct{}

var PublishTimeKey = publishTimeKey{}

type PromoCollectionPublishedMessage struct {
	Msisdn                       string             `json:"msisdn" validate:"required"`
	Unit                         string             `json:"unit" validate:"required"`    // always MB
	IsRollBack                   bool               `json:"isRollBack"`                  // always false
	Duration                     string             `json:"duration"`                    // always 200
	Channel                      string             `json:"channel" validate:"required"` // always Dodrio
	SvcId                        int64              `json:"svc_id"`
	Denom                        string             `json:"denom"`
	WalletKeyword                string             `json:"walletKeyword"`
	WalletAmount                 string             `json:"walletAmount"`
	SvcDenomCombined             string             `json:"svcDenomCombined"`
	KafkaId                      int64              `json:"kafkaId"`
	CollectionType               string             `json:"collectionType"`
	Ageing                       int32              `json:"ageing"`
	AvailmentTransactionId       primitive.ObjectID `json:"availmentTransactionId"`
	LoanId                       primitive.ObjectID `json:"loanId"`
	UnpaidLoanId                 primitive.ObjectID `json:"unpaidLoanId"`
	ServiceFee                   float64            `json:"serviceFee"`
	TotalLoanAmountInPeso        float64            `json:"totalLoanAmountInPeso"`
	TotalUnpaidAmountInPeso      float64            `json:"totalUnpaidAmountInPeso"`
	AmountToBeDeductedInPeso     float64            `json:"amountToBeDeductedInPeso"`
	UnpaidServiceFee             float64            `json:"unpaidServiceFee"`
	DataToBeDeducted             float64            `json:"dataToBeDeducted"`
	LastCollectionDateTime       time.Time          `json:"lastCollectionDateTime"`
	LastCollectionId             primitive.ObjectID `json:"lastCollectionId"`
	StartDate                    time.Time          `json:"startDate"`
	BrandId                      primitive.ObjectID `json:"brandId"`
	LoanProductId                primitive.ObjectID `json:"loanProductId"`
	LoanType                     consts.LoanType    `json:"loanType"`
	DataCollectionRequestTraceId string             `json:"dataCollectionRequestTraceId"`
	GUID                         string             `json:"loanGuid"`
	Version                      int32              `json:"oldUnpaidLoanVersion"`
	PublishTime                 time.Time
}

func (p PromoCollectionPublishedMessage) String() string {
	return fmt.Sprintf(
		"Msisdn: %s, Wallet: %s, Amount: %s, Unit: %s, IsRollBack: %t, Duration: %s, Channel: %s",
		p.Msisdn,
		p.WalletKeyword,
		p.WalletAmount,
		p.Unit,
		p.IsRollBack,
		p.Duration,
		p.Channel,
	)
}
