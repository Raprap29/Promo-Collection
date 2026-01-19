package router

import (
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/app/handlers"
	"globe/dodrio_loan_availment/internal/app/middleware"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/downstreams"
	"globe/dodrio_loan_availment/internal/pkg/notification"
	"globe/dodrio_loan_availment/internal/pkg/pubsub"
	"globe/dodrio_loan_availment/internal/pkg/store/repository"

	"globe/dodrio_loan_availment/internal/pkg/kafka/producer"
	kafkaServices "globe/dodrio_loan_availment/internal/pkg/kafka/producer"
	"globe/dodrio_loan_availment/internal/pkg/services"
	"globe/dodrio_loan_availment/internal/pkg/store"
	"globe/dodrio_loan_availment/internal/pkg/utils/worker"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

func SetupRouter(workerPool *worker.WorkerPool, redisClient *redis.Client, pubsubPublisher *pubsub.PubSubPublisher) *gin.Engine {

	r := gin.Default()
	meter := otel.Meter(configs.SERVICE_NAME)
	r.Use(otelgin.Middleware(configs.SERVICE_NAME))
	r.Use(middleware.NewMetricMiddleware(meter))
	r.Use(middleware.AttachRequestDetails())

	// Create Redis repository wrapper
	redisAdapter := repository.NewRedisStoreAdapter(redisClient)
	// App Creation
	amaxService := downstreams.NewAmaxService(redisAdapter, pubsubPublisher)
	brandRepo := store.NewBrandRepository()
	keywordRepo := store.NewKeywordRepository()
	subscribersRepo := store.NewSubscribersRepository()
	loansProductsRepo := store.NewLoansProductsRepository()
	activeLoanRepo := store.NewActiveLoanRepository()
	unpaidLoanRepo := store.NewUnpaidLoanRepository()
	whitelistedSubsRepo := store.NewWhitelistedSubsRepository()
	channelRepo := store.NewChannelsRepository()
	loanProductsBrandsRepo := store.NewLoanProductsBrandsRepository()
	loanProductsChannelRepo := store.NewLoanProductsChannelsRepository()
	availmentRepo := store.NewAvailmentRepository(redisAdapter)
	transactionInProgressRepo := store.NewTransactionInProgressRepository()
	walletRepo := store.NewWalletTypesRepository()
	productNamesInAmax := store.NewProductNameInAmaxsRepository()
	SubscribersRepository := store.NewSubscribersRepository()
	systemLevelRulesRepo := store.NewSystemLevelRulesRepository()
	notificationService := notification.NewNotificationService(pubsubPublisher)

	// store.
	brandExist := services.NewValidationService(brandRepo, keywordRepo, subscribersRepo, loansProductsRepo, activeLoanRepo, whitelistedSubsRepo, channelRepo, loanProductsBrandsRepo, loanProductsChannelRepo)
	checkSubsRepo := brandExist
	checkLoanRepo := brandExist
	checkWhiteListRepo := brandExist
	checkChannelRepo := brandExist
	checkChannelandProductRepo := brandExist
	checkBrandandProductRepo := brandExist
	checkScoreRepo := brandExist
	checkActiveRepo := brandExist
	checkEligb := brandExist

	helper := services.NewEligibilityCheckService(nil, checkEligb, activeLoanRepo, checkSubsRepo, checkLoanRepo, checkActiveRepo, checkWhiteListRepo, checkChannelRepo, checkChannelandProductRepo, checkBrandandProductRepo, checkScoreRepo, keywordRepo, brandRepo, SubscribersRepository, availmentRepo, unpaidLoanRepo)
	checkElgibRepo := services.NewEligibilityCheckService(helper, checkEligb, activeLoanRepo, checkSubsRepo, checkLoanRepo, checkActiveRepo, checkWhiteListRepo, checkChannelRepo, checkChannelandProductRepo, checkBrandandProductRepo, checkScoreRepo, keywordRepo, brandRepo, SubscribersRepository, availmentRepo, unpaidLoanRepo)

	producer := producer.NewKafkaService()
	processReq := services.NewProcessRequestServiceImpl(helper, brandExist, availmentRepo, producer, transactionInProgressRepo, checkElgibRepo, keywordRepo, amaxService, subscribersRepo, productNamesInAmax, systemLevelRulesRepo, notificationService)
	appCreationLoanAvailmentService := services.NewLoanAvailmentService(workerPool, availmentRepo, producer, processReq, transactionInProgressRepo, brandExist, keywordRepo, amaxService, subscribersRepo, productNamesInAmax, notificationService)
	appCreationLoanStatusService := services.NewLoanStatusService(workerPool, activeLoanRepo, loansProductsRepo, helper, brandExist, notificationService)
	appCreationHandler := handlers.NewAppCreationHandler(appCreationLoanAvailmentService, appCreationLoanStatusService)

	// Eligibility Check
	eligibilityCheckService := services.NewEligibilityCheckService(helper, checkEligb, activeLoanRepo, checkSubsRepo, checkLoanRepo, checkActiveRepo, checkWhiteListRepo, checkChannelRepo, checkChannelandProductRepo, checkBrandandProductRepo, checkScoreRepo, keywordRepo, brandRepo, SubscribersRepository, availmentRepo, unpaidLoanRepo)
	eligibilityCheckHandler := handlers.NewEligibilityCheckHandler(eligibilityCheckService)

	// Blacklist and Whitelist Subs
	blacklistAndWhitelistSubsService := services.NewBlacklistAndWhitelistFromSftpService()
	blacklistAndWhitelistSubsHandler := handlers.NewSftpHandler(blacklistAndWhitelistSubsService)

	// Arrow Loans Cache Table
	arrowLoansCacheHandler := handlers.NewArrowLoansCacheHandler()

	// Availment reports
	bucketName := configs.BUCKET_NAME

	availmentHandler := handlers.NewAvailmentHandler(bucketName)

	var kafkaRetryService *kafkaServices.KafkaRetryService
	// Availment Store
	if db.MDB != nil {
		availmentStore := store.NewAvailmentStore(*db.MDB.Client, db.MDB.Database.Name())
		kafkaRetryService = kafkaServices.NewKafkaRetryService(availmentStore, walletRepo)
		// KafkaRetryHandler

	}
	// kafkaRetryService
	kafkaRetryHandler := handlers.NewKafkaRetryHandler(kafkaRetryService)

	r.GET("/IntegrationServices/Dodrio/AppCreation/:MSISDN/:TransactionId/:Keyword", appCreationHandler.AppCreation)
	r.POST("/IntegrationServices/Dodrio/EligibilityCheck", eligibilityCheckHandler.EligibilityCheck)
	r.GET("/IntegrationServices/Dodrio/BlacklistSubs", blacklistAndWhitelistSubsHandler.BlacklistSubs)
	r.GET("/IntegrationServices/Dodrio/WhitelistSubsForProductId", blacklistAndWhitelistSubsHandler.WhitelistSubsForProductId)

	r.GET("/IntegrationServices/Dodrio/kafkaRetry", kafkaRetryHandler.RetryKafkaAvailmentMessage)
	r.GET("/IntegrationServices/Dodrio/AvailmentReports", availmentHandler.AvailmentsReports)
	r.POST("/ArrowLoansCache", arrowLoansCacheHandler.ArrowLoansCache)

	r.GET("IntegrationServices/Dodrio/Test", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Health Check"})
	})

	return r
}
