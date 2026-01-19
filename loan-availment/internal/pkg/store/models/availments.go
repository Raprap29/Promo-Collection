package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AvailmentTransaction struct {
	ID                  primitive.ObjectID `bson:"_id"`
	MSISDN              string             `bson:"MSISDN"`
	Channel             string             `bson:"channel"`
	Brand               string             `bson:"brand"`
	LoanType            string             `bson:"loanType"`
	ProductId           primitive.ObjectID `bson:"productId"`
	ProductName         string             `bson:"productName"`
	Keyword             string             `bson:"keyword"`
	ServicingPartner    string             `bson:"servicingPartner"`
	Result              bool               `bson:"result"`
	ErrorText           string             `bson:"errorText"`
	ErrorCode           string             `bson:"ErrorCode"`
	CreditScore         float64            `bson:"creditScore"`
	TotalLoanAmount     float64            `bson:"totalLoanAmount"`
	ServiceFee          float64            `bson:"serviceFee"`
	GUID                string             `bson:"GUID"`
	CreatedAt           time.Time          `bson:"createdAt"`
	LoanProvisionStatus string             `bson:"loanStatus"`
	PublishedToKafka    bool               `bson:"publishedToKafka"`
	CollectionType      string             `bson:"collectionType"`
}
