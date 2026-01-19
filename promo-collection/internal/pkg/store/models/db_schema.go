package models

import (
	"promocollection/internal/pkg/consts"
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
	LoanID                        primitive.ObjectID `bson:"_id"`
	MSISDN                        string             `bson:"MSISDN"`
	TotalLoanAmount               int32              `bson:"totalLoanAmount"`
	GUID                          string             `bson:"GUID"`
	ServiceFee                    float64            `bson:"serviceFee"`
	LoanProductID                 primitive.ObjectID `bson:"loanProductId"`
	BrandID                       primitive.ObjectID `bson:"brandId"`
	AvailmentTransactionID        primitive.ObjectID `bson:"availmentTransactionId"`
	LoanType                      consts.LoanType    `bson:"loanType"`
	CreatedAt                     time.Time          `bson:"createdAt"`
	Migrated                      bool               `bson:"migrated"`
	OldBrandType                  string             `bson:"oldBrandType,omitempty"`
	OldAvailmentID                int64              `bson:"oldAvailmentId,omitempty"`
	OldProductID                  int32              `bson:"oldProductId,omitempty"`
	GracePeriodTimestamp          time.Time          `bson:"gracePeriodTimestamp,omitempty"`
	PromoEducationPeriodTimestamp time.Time          `bson:"promoEducationPeriodTimestamp,omitempty"`
	EducationPeriodTimestamp      time.Time          `bson:"educationPeriodTimestamp,omitempty"`
	LoanLoadPeriodTimestamp       time.Time          `bson:"loanLoadPeriodTimestamp,omitempty"`
	ConversionRate                float64            `bson:"conversionRate,omitempty"`
	Version                       int32              `bson:"version,omitempty"`
	TotalUnpaidLoan               float64            `bson:"totalUnpaidLoan"`
	MinCollectionAmount           float64            `bson:"minCollectionAmount"`
	MaxDataCollectionPercent      float64            `bson:"maxDataCollectionPercent"`
}

type PesoToDataConversion struct {
	LoanType       string  `bson:"loanType"`
	ConversionRate float64 `bson:"conversionRate"`
}

type SystemLevelRules struct {
	ID                                 primitive.ObjectID `bson:"_id,omitempty"`
	Migrated                           bool               `bson:"migrated"`
	CreditScoreThreshold               float64            `bson:"creditScoreThreshold"`
	OverdueThreshold                   int32              `bson:"overdueThreshold"`
	UpdatedAt                          time.Time          `bson:"updatedAt"`
	EducationPeriod                    int32              `bson:"educationPeriod"`
	DefermentPeriod                    int32              `bson:"defermentPeriod"`
	ReservedAmountForPartialCollection float64            `bson:"reservedAmountForPartialCollection"`
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

type UnpaidLoans struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty"`
	LastCollectionID       primitive.ObjectID `bson:"lastCollectionId"`
	Version                int32              `bson:"version"`
	ValidTo                time.Time          `bson:"validTo"`
	UnpaidServiceFee       int32              `bson:"unpaidServiceFee"`
	Migrated               bool               `bson:"migrated"`
	ValidFrom              time.Time          `bson:"validFrom"`
	TotalLoanAmount        int32              `bson:"totalLoanAmount"`
	TotalUnpaidAmount      int32              `bson:"totalUnpaidAmount"`
	LastCollectionDateTime time.Time          `bson:"lastCollectionDateTime"`
	LoanId                 primitive.ObjectID `bson:"loanId"`
	DueDate                time.Time          `bson:"dueDate"`
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
