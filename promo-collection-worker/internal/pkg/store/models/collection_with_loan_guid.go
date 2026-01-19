package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CollectionWithLoanGUID struct {
	ID                     primitive.ObjectID `bson:"_id"`
	MSISDN                 string             `bson:"MSISDN"`
	Ageing                 int32              `bson:"ageing"`
	AvailmentTransactionId primitive.ObjectID `bson:"availmentTransactionId"`
	CollectedAmount        float64            `bson:"collectedAmount"`
	CollectionCategory     string             `bson:"collectionCategory"`
	CollectionType         string             `bson:"collectionType"`
	CreatedAt              time.Time          `bson:"createdAt"`
	ErrorText              string             `bson:"errorText"`
	LoanId                 primitive.ObjectID `bson:"loanId"`
	Method                 string             `bson:"method"`
	PaymentChannel         string             `bson:"paymentChannel"`
	PublishedToKafka       bool               `bson:"publishedToKafka"`
	Result                 bool               `bson:"result"`
	ServiceFee             float64            `bson:"serviceFee"`
	TokenPaymentId         string             `bson:"tokenPaymentId"`
	TotalCollectedAmount   float64            `bson:"totalCollectedAmount"`
	UnpaidLoanId           primitive.ObjectID `bson:"unpaidLoanId"`
	TotalUnpaid            float64            `bson:"totalUnpaid"`
	ErrorCode              string             `bson:"errorCode"`
	UnpaidServiceFee       float64            `bson:"unpaidServiceFee"`
	KafkaTransactionId     string             `bson:"kafkaTransactionId"`
	DataCollected          float64            `bson:"dataCollected"`
	TransactionId          string             `bson:"transactionId"`
	CollectedServiceFee    float64            `bson:"collectedServiceFee"`
	UnpaidLoanAmount       float64            `bson:"unpaidLoanAmount"`
	LoanGUID               string             `bson:"loanGUID"`
}
