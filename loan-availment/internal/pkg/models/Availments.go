package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Availments struct {
	ID               primitive.ObjectID `bson:"_id"`
	MSISDN           string             `bson:"MSISDN"`
	Channel          string             `bson:"channel"`
	Brand            string             `bson:"brand"`
	LoanType         string             `bson:"loanType"`
	ProductID        primitive.ObjectID `bson:"productId"`
	Keyword          string             `bson:"keyword"`
	ServicingPartner string             `bson:"servicingPartner"`
	Result           bool               `bson:"result"`
	ErrorText        string             `bson:"errorText"`
	ErrorCode        string             `bson:"errorCode"`
	CreditScore      float64            `bson:"creditScore"`
	TotalLoanAmount  float64            `bson:"totalLoanAmount,omitempty"`
	ServiceFee       float64            `bson:"serviceFee,omitempty"`
	GUID             string             `bson:"GUID"`
	CreatedAt        time.Time          `bson:"createdAt"`
	PublishedToKafka bool               `bson:"publishedToKafka"`
	ProductName      string             `bson:"productName"`
}
