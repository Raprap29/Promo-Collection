package models

import (
	"promo-collection-worker/internal/pkg/consts"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UnpaidBalanceChange struct {
	ID                 primitive.ObjectID `bson:"_id"`
	Version            int32              `bson:"version"`
	ValidFrom          time.Time          `bson:"validFrom"`
	ValidTo            time.Time          `bson:"validTo"`
	TotalLoanAmount    int32              `bson:"totalLoanAmount"`
	TotalUnpaidAmount  int32              `bson:"totalUnpaidAmount"`
	UnpaidServiceFee   int32              `bson:"unpaidServiceFee"`
	DueDate            time.Time          `bson:"dueDate"`
	LastCollectionID   primitive.ObjectID `bson:"lastCollectionId"`
	LastCollectionDate time.Time          `bson:"lastCollectionDateTime"`
	Migrated           bool               `bson:"migrated"`
}

type Loans struct {
	LoanID                 primitive.ObjectID `bson:"_id"`
	MSISDN                 string             `bson:"MSISDN"`
	TotalLoanAmount        int32              `bson:"totalLoanAmount"`
	GUID                   string             `bson:"GUID"`
	ServiceFee             int32              `bson:"serviceFee"`
	LoanProductID          primitive.ObjectID `bson:"loanProductId"`
	BrandID                primitive.ObjectID `bson:"brandId"`
	AvailmentTransactionID primitive.ObjectID `bson:"availmentTransactionId"`
	LoanType               consts.LoanType    `bson:"loanType"`
	CreatedAt              time.Time          `bson:"createdAt"`
	Migrated               bool               `bson:"migrated"`

	// Fields from the old system that may be retained
	OldBrandType   string `bson:"oldBrandType,omitempty"`
	OldAvailmentID int64  `bson:"oldAvailmentId,omitempty"`
	OldProductID   int32  `bson:"oldProductId,omitempty"`

	// Fields that will be added in the future
	UnpaidBalanceChanges     []UnpaidBalanceChange `bson:"unpaidBalanceChanges"`
	GracePeriodTimestamp     time.Time             `bson:"gracePeriodTimestamp,omitempty"`
	EducationPeriodTimestamp time.Time             `bson:"educationPeriodTimestamp,omitempty"`
	LoanLoadPeriodTimestamp  time.Time             `bson:"loanLoadPeriodTimestamp,omitempty"`
	ConversionRate           float64               `bson:"conversionRate,omitempty"`
	Version                  int32                 `bson:"version,omitempty"`
	TotalUnpaidLoan          float64               `bson:"totalUnpaidLoan"`
}

type PesoToDataConversion struct {
	LoanType       string  `bson:"loanType"`
	ConversionRate float64 `bson:"conversionRate"`
}

type SystemLevelRules struct {
	ID                                 primitive.ObjectID `bson:"_id,omitempty"`
	Migrated                           bool               `bson:"migrated"`
	CreditScoreThreshold               int32              `bson:"creditScoreThreshold"`
	OverdueThreshold                   int32              `bson:"overdueThreshold"`
	UpdatedAt                          time.Time          `bson:"updatedAt"`
	EducationPeriod                    int32              `bson:"educationPeriod"`
	DefermentPeriod                    int32              `bson:"defermentPeriod"`
	ReservedAmountForPartialCollection int32              `bson:"reservedAmountForPartialCollection"`
	PartialCollectionEnabled           bool               `bson:"partialCollectionEnabled"`

	MaxDataCollectionPercent   float64                `bson:"maxDataCollectionPercent,omitempty"`
	PesoToDataConversionMatrix []PesoToDataConversion `bson:"pesoToDataConversionMatrix,omitempty"`
	MinCollectionAmount        float64                `bson:"minCollectionAmount,omitempty"`
	PromoCollectionEnabled     bool                   `bson:"promoCollectionEnabled,omitempty"`
	WalletExclusionList        []string               `bson:"walletExclusionList,omitempty"`
	PromoEducationPeriod       int32                  `bson:"promoEducationPeriod,omitempty"`
	GracePeriod                int32                  `bson:"gracePeriod,omitempty"`
	LoanLoadPeriod             int32                  `bson:"loanLoadPeriod,omitempty"`
}

type CollectionTransactionsInProgress struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	MSISDN    string             `bson:"MSISDN"`
	CreatedAt time.Time          `bson:"createdAt"`
}

type WhitelistedForDataCollection struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	MSISDN string             `bson:"MSISDN"`
	Brand  string             `bson:"Brand"`
	Active bool               `bson:"Active"`
}

type Collections struct {
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
	TotalUnpaid            float64            `bson:"totalUnpaid"` // including service fee
	ErrorCode              string             `bson:"errorCode"`
	UnpaidServiceFee       float64            `bson:"unpaidServiceFee"`
	KafkaTransactionId     string             `bson:"kafkaTransactionId"`
	DataCollected          float64            `bson:"dataCollected"`
	TransactionId          string             `bson:"transactionId"`
	CollectedServiceFee    float64            `bson:"collectedServiceFee"`
	UnpaidLoanAmount       float64            `bson:"unpaidLoanAmount"` // excluding service fee
}

type UnpaidLoans struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty"`
	LastCollectionId       primitive.ObjectID `bson:"lastCollectionId"`
	Version                int32              `bson:"version"`
	ValidTo                time.Time          `bson:"validTo"`
	UnpaidServiceFee       int32              `bson:"unpaidServiceFee"`
	Migrated               bool               `bson:"migrated"`
	ValidFrom              time.Time          `bson:"validFrom"`
	TotalLoanAmount        int32              `bson:"totalLoanAmount"`
	TotalUnpaidAmount      int32              `bson:"totalUnpaidAmount"`
	LastCollectionDateTime time.Time          `bson:"lastCollectionDateTime"`
	LoanId                 primitive.ObjectID `bson:"loanId"`
}

type ClosedLoans struct {
	GUID                           string             `bson:"GUID"`
	MSISDN                         string             `bson:"MSISDN"`
	Id                             primitive.ObjectID `bson:"_id"`
	AvailmentTransactionId         primitive.ObjectID `bson:"availmentTransactionId"`
	BrandId                        primitive.ObjectID `bson:"brandId"`
	EndDate                        time.Time          `bson:"endDate"`
	LoanId                         primitive.ObjectID `bson:"loanId"`
	LoanProductId                  primitive.ObjectID `bson:"loanProductId"`
	LoanType                       consts.LoanType    `bson:"loanType"`
	StartDate                      time.Time          `bson:"startDate"`
	Status                         string             `bson:"status"`
	TotalCollectedAmount           float64            `bson:"totalCollectedAmount"`
	TotalLoanAmount                float64            `bson:"totalLoanAmount"`
	TotalWrittenOffOrChurnedAmount float64            `bson:"totalWrittenOffOrChurnedAmount"`
	WriteOffReason                 string             `bson:"writeOffReason"`
}

type Messages struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	PatternId    int32              `bson:"patternId" json:"patternId"`
	Event        string             `bson:"event" json:"event"`
	Parameters   []string           `bson:"parameters" json:"parameters"`
	OldBrandType string             `bson:"oldBrandType" json:"oldBrandType"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    *time.Time         `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	IsDeleted    bool               `bson:"isDeleted" json:"isDeleted"`
	Migrated     bool               `bson:"migrated" json:"migrated"`
	BrandId      primitive.ObjectID `bson:"brandId" json:"brandId"`
	Labels       []string           `bson:"labels,omitempty" json:"labels,omitempty"`
}

type LoanProducts struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Name             string             `bson:"name"`
	Price            int32              `bson:"price"`
	ServiceFee       int32              `bson:"serviceFee"`
	FeeType          string             `bson:"feeType"`
	LoanType         string             `bson:"loanType"`
	WalletId         primitive.ObjectID `bson:"walletId"`
	IsScoredProduct  bool               `bson:"isScoredProduct"`
	HasWhitelisting  bool               `bson:"hasWhitelisting"`
	ActivationMethod string             `bson:"activationMethod"`
	Active           bool               `bson:"active"`
	PromoStartDate   time.Time          `bson:"promoStartDate"`
	PromoEndDate     time.Time          `bson:"promoEndDate"`
	ProductGUID      string             `bson:"productGUID"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
	IsDeleted        bool               `bson:"isDeleted"`
	Migrated         bool               `bson:"migrated"`
}
