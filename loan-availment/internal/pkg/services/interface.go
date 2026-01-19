package services

import (
	"bytes"
	"context"
	"globe/dodrio_loan_availment/internal/pkg/models"
	storeModel "globe/dodrio_loan_availment/internal/pkg/store/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SFTPClientInterface interface {
	PullCSVFromSFTP() (string, error)
	MoveFileOnSFTP(srcPath, destPath string) error
	DeleteFileOnSFTP(filepath string) error
	DeleteLocalFile(filePath string) error
	UploadFileToSFTP(localFilePath, remoteFilePath string) error
}

type LoanStatusServiceInterface interface {
	LoanStatus(*gin.Context, models.AppCreationRequest) error
}

type EligibilityCheckServiceInterface interface {
	EligibilityCheck(*gin.Context, string) error
}

type LoanAvailmentServiceInterface interface {
	AvailLoan(*gin.Context, models.AppCreationRequest) error
}

type ManageDatabaseServiceInterface interface {
	ManageDb(*gin.Context, models.ManageDbRequest) (any, error)
}

type AvailmentReportInterface interface {
	AvailmentDetailsReports() ([]models.AvailmentReport, error)
	GenerateLoanCache(startTime time.Time, endTime time.Time) (*models.ArrowCacheTable, error)
	GenerateCSV(arrowCacheTable models.ArrowCacheTable, startTime time.Time, endTime time.Time) (*bytes.Buffer, error)
	TransferCSVToSFTP(csvData *bytes.Buffer) error
}

type ArrowLoansCacheInterface interface {
	ArrowLoansCacheTable(endTime time.Time) (*models.ArrowCacheTable, error)
	GenerateLoanCache(startTime time.Time, endTime time.Time) (*models.ArrowCacheTable, error)
	GenerateCSV(arrowCacheTable models.ArrowCacheTable, startTime time.Time, endTime time.Time) (*bytes.Buffer, error)
}
type AvailmentStoreInterface interface {
	GetFailedKafkaEntries(context.Context, int32) ([]storeModel.AvailmentTransaction, error)
	SetKafkaFlag(context.Context, []string) ([]string, error)
}

// loan_status_service interface
type LoanRepo interface {
	ActiveLoanByFilter(filter interface{}) (*models.Loans, error)
}
type SubscribersRepository interface {
	AddUpdateSubscriber(MSISDN string, subscriber *models.Subscribers) error
}
type ProductsNamesInAmaxRepository interface {
	ProductNameInAmaxsByFilter(context.Context, interface{}) (*models.ProductsNamesInAmax, error)
}

type LoanProductRepo interface {
	LoanProductByFilter(filter interface{}) (*models.LoanProduct, error)
}

type Helper interface {
	GetCreditScoreCustomerTypeBrandType(ctx context.Context, msisdn string) (float64, string, string, string, *time.Time, error)
}

type BrandExist interface {
	CheckIfBrandExistsAndActive(ctx context.Context, brand string) (bool, primitive.ObjectID, error)
}

type NotificationServiceInterface interface {
	NotifyUser(ctx context.Context, MSISDN string, event string, brand primitive.ObjectID, product *models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) error
}

// loan_availment_interface
type LoanAvailmentServicesInt interface {
	InsertFailedAvailment(ctx context.Context, msisdn string, keyword string, channel string, systemClient string, errMsg string, errCode string, result bool, creditScore float64, loanProduct *models.LoanProduct, brandCode string) (models.Availments, *models.LoanProduct, string, string, error)
	UpdateAvailment(ctx context.Context, availmentID primitive.ObjectID, updateField bson.M)
	GetAvailmentById(id primitive.ObjectID) (*models.Availments, error)
}

type KafkaServiceInterface interface {
	PublishAvailmentStatusToKafka(ctx context.Context, transactionID string, availmentChannel string, msisdn string, brandType string, loanType string, loanProductName string, loanProductKeyword string, servicingPartner string, loanAmount float64, serviceFeeAmount float64, availmentResult string, availmentErrorText string, loanAgeing int32, statusOfProvisionedLoan string) error
}

type CheckEligibilityServiceInterface interface {
	CheckEligibility(ctx context.Context, msisdn string, keyword string, channel string, customer_type string, credit_score_value float64, brand_type string, brandId primitive.ObjectID) (bool, string, *models.LoanProduct, primitive.ObjectID, time.Time, error)
}

type ProcessRequestService interface {
	ProcessRequest(keyword string, msisdn string, systemClient string, channel string, transactionId string, creditScore float64, startTime time.Time, ctx context.Context)
}

type TransactionInProgressInterface interface {
	DeleteTransactionInProgressByMsisdn(ctx context.Context, msisdn string) (bool, error)
	IsAvailmentTransactionInProgress(ctx context.Context, msisdn string) (bool, error)
	CreateTransactionInProgressEntry(ctx context.Context, transactionInProgressDB models.TransactionInProgress) (bool, error)
}

type TransactionInprogressAndBrandCheckService interface {
	transactionInprogressAndBrandCheck(ctx context.Context, msisdn string) (float64, string, string, primitive.ObjectID, string, error)
}

type BrandByFilterService interface {
	BrandsByFilter(filter interface{}) (*models.Brand, error)
}

type KeywordByFilterService interface {
	KeywordByFilter(filter interface{}) (*models.Keyword, error)
}

type CheckKeywordInterface interface {
	CheckKeyword(ctx context.Context, keyword string) (*models.Keyword, error)
}

type ActiveFilterInterface interface {
	ActiveLoanByFilter(filter interface{}) (*models.Loans, error)
}

type UnpaidLoanFilterInterface interface {
	UnpaidLoansByFilter(filter interface{}) (*models.UnpaidLoans, error)
}

type ProductNamesInAmaxService interface {
	ProductNamesInAmaxByKeyword(keywordName string, brandId primitive.ObjectID) (string, error)
}

type ProvisionLoanAmaxService interface {
	ProvisionLoanAmax(ctx context.Context, msisdn string, transactionId string, loanProduct *models.LoanProduct, keyword string, channel string, systemClient string, productsNameInAmax string, creditScore float64, brand_type string, brandId primitive.ObjectID, startTime time.Time) error
}

// eligibility_interface
type CheckSubsInterface interface {
	CheckSubscriberBlacklisitng(ctx context.Context, msisdn string) (*models.Subscribers, error)
}

type CheckSubscriberBlacklistInterface interface {
	SubscribersByFilter(filter interface{}) (*models.Subscribers, error)
	AddUpdateSubscriber(MSISDN string, subscriber *models.Subscribers) error
}

type CheckLoanProductService interface {
	CheckLoanProduct(ctx context.Context, productId primitive.ObjectID) (*models.LoanProduct, error)
}

type CheckActiveLoanService interface {
	CheckActiveLoan(ctx context.Context, msisdn string) (bool, primitive.ObjectID, time.Time, error)
}

type ActiveLaoneService interface {
	IsActiveLoan(ctx context.Context, msisdn string) (bool, *models.Loans, error)
}

type CheckWhitelistSErvice interface {
	CheckWhitelist(ctx context.Context, msisdn string, loanProduct *models.LoanProduct) (bool, error)
}

type IsWhitelistedService interface {
	IsWhitelisted(msisdn string, productId primitive.ObjectID) (bool, error)
}

type CheckChannelExistService interface {
	CheckIfChannelExistsAndActive(ctx context.Context, channel string) (bool, primitive.ObjectID, error)
}

type ChannelByFilterServcie interface {
	ChannelsByFilter(ctx context.Context, filter interface{}) (*models.Channels, error)
}

type CheckChannelAndProductService interface {
	CheckIfChannelAndProductMapped(ctx context.Context, channelId, productId primitive.ObjectID) (bool, error)
}

type LoanProductChannelService interface {
	LoanProductsBrandsByFilter(ctx context.Context, filter interface{}) (*models.LoanProductsBrands, error)
}

type CheckBrandandProductService interface {
	CheckIfBrandAndProductMapped(ctx context.Context, brandId, productId primitive.ObjectID) (bool, error)
}

type CheckScoreEligService interface {
	CheckCreditScoreEligibility(ctx context.Context, creditScore float64, loanProductResult *models.LoanProduct) bool
}

type LoanProductsChannelByFilterService interface {
	LoanProductsChannelsByFilter(filter interface{}) (*models.LoanProductsChannels, error)
}

type SystemLevelRulesRepository interface {
	SystemLevelRules(filter interface{}) (models.SystemLevelRules, error)
}

type RedisStoreOperations interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Time-based operations
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}
