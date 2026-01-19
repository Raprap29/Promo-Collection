package services

import (
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/store"
	"globe/dodrio_loan_availment/internal/pkg/utils"
	"globe/dodrio_loan_availment/internal/pkg/utils/worker"
)

func init() {
	gob.Register(primitive.ObjectID{})
	gob.Register(int32(0))
}

// var channels = map[string]bool{
// 	"cxs":         true,
// 	"rt":          true,
// 	"USSD":        true,
// 	"FB":          true,
// 	"DPA":         true,
// 	"G1TESTLOAN":  true,
// 	"testonly":    true,
// 	"SMStestonly": true,
// }

type LoanAvailmentService struct {
	workerPool                *worker.WorkerPool
	brandExist                BrandExist
	availmentRepo             LoanAvailmentServicesInt
	producer                  KafkaServiceInterface
	processReq                ProcessRequestService
	transactionInProgressRepo TransactionInProgressInterface
	keywordRepo               ProductNamesInAmaxService
	amaxService               ProvisionLoanAmaxService
	subscriberRepo            SubscribersRepository
	productsNameInAmaxrepo    ProductsNamesInAmaxRepository
	notificationService       NotificationServiceInterface
}

type ProcessRequestServiceImpl struct {
	helper                    Helper
	brandExist                BrandExist
	availmentRepo             LoanAvailmentServicesInt
	producer                  KafkaServiceInterface
	transactionInProgressRepo TransactionInProgressInterface
	checkElgibRepo            CheckEligibilityServiceInterface
	keywordRepo               ProductNamesInAmaxService
	amaxService               ProvisionLoanAmaxService
	subscriberRepo            SubscribersRepository
	productsNameInAmaxrepo    ProductsNamesInAmaxRepository
	systemRulesRepo           SystemLevelRulesRepository
	notificationService       NotificationServiceInterface
}

func NewProcessRequestServiceImpl(helper Helper, brandExist BrandExist, availmentRepo LoanAvailmentServicesInt, producer KafkaServiceInterface, transactionInProgressRepo TransactionInProgressInterface, checkElgibRepo CheckEligibilityServiceInterface, keywordRepo ProductNamesInAmaxService, amaxService ProvisionLoanAmaxService, subscriberRepo SubscribersRepository, productsNameInAmaxrepo ProductsNamesInAmaxRepository, systemLevelRulesRepo SystemLevelRulesRepository, notificationService NotificationServiceInterface) *ProcessRequestServiceImpl {
	return &ProcessRequestServiceImpl{
		helper:                    helper,
		brandExist:                brandExist,
		availmentRepo:             availmentRepo,
		producer:                  producer,
		transactionInProgressRepo: transactionInProgressRepo,
		checkElgibRepo:            checkElgibRepo,
		keywordRepo:               keywordRepo,
		amaxService:               amaxService,
		subscriberRepo:            subscriberRepo,
		productsNameInAmaxrepo:    productsNameInAmaxrepo,
		systemRulesRepo:           systemLevelRulesRepo,
		notificationService:       notificationService,
	}
}

var availmentReportRepo *store.AvailmentReportRepository

func NewLoanAvailmentService(workerPool *worker.WorkerPool, availmentRepo LoanAvailmentServicesInt, producer KafkaServiceInterface, processReq ProcessRequestService, transactionInProgressRepo TransactionInProgressInterface, brandExist BrandExist, keywordRepo ProductNamesInAmaxService, amaxService ProvisionLoanAmaxService, subscriberRepo SubscribersRepository, productsNameInAmaxrepo ProductsNamesInAmaxRepository, notificationService NotificationServiceInterface) *LoanAvailmentService {
	return &LoanAvailmentService{
		workerPool:                workerPool,
		brandExist:                brandExist,
		availmentRepo:             availmentRepo,
		producer:                  producer,
		processReq:                processReq,
		transactionInProgressRepo: transactionInProgressRepo,
		keywordRepo:               keywordRepo,
		amaxService:               amaxService,
		subscriberRepo:            subscriberRepo,
		productsNameInAmaxrepo:    productsNameInAmaxrepo,
		notificationService:       notificationService,
	}
}

func (h *LoanAvailmentService) AvailLoan(c *gin.Context, request models.AppCreationRequest) error {
	startTime := time.Now()
	ctx := c.Request.Context()

	msisdn := request.MSISDN
	transactionId := request.TransactionId
	keyword := request.Keyword
	systemClient := request.SystemClient
	var creditScore = float64(0)

	// App creation request format validation
	isValid, cleanedMSISDN, channel, transactionId, error := h.requestValidation(ctx, msisdn, keyword, transactionId)
	if error != nil {
		if errors.Is(error, consts.ErrorMsisdnFormatValidationFailed) ||
			errors.Is(error, consts.ErrorTransactionIdFormatValidationFailed) {
			availmentDB, loanProductDB, servicingPartnerNumberString, _, _ := h.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, error.Error(), utils.GetErrorCode(error), false, creditScore, nil, "")

			loanProductName := ""
			loanProductPrice := float64(0)
			if loanProductDB != nil {
				loanProductName = loanProductDB.Name
				loanProductPrice = loanProductDB.Price
			}
			go func(ctx context.Context) {
				err := h.producer.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, loanProductName, keyword, servicingPartnerNumberString, loanProductPrice, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
				if err == nil {
					updateField := bson.M{"publishedToKafka": true}
					h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
				}
			}(ctx)
			if errors.Is(error, consts.ErrorTransactionIdFormatValidationFailed) {
				go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedInvalidTransactionId, primitive.NilObjectID, nil, primitive.NilObjectID, time.Time{})
			}

			transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
			logger.Info(ctx, transaction)
		}
		if errors.Is(error, consts.ErrorKeywordFormatValidationFailed) {
			availmentDB, _, servicingPartnerNumberString, _, _ := h.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, error.Error(), utils.GetErrorCode(error), false, creditScore, nil, "")
			go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedInvalidKeyword, primitive.NilObjectID, nil, primitive.NilObjectID, time.Time{})
			go func(ctx context.Context) {
				err := h.producer.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, "", keyword, servicingPartnerNumberString, 0, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
				if err == nil {
					updateField := bson.M{"publishedToKafka": true}
					h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
				}
			}(ctx)
			transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
			logger.Info(ctx, transaction)
			h.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
		}
		return error
	}

	if isValid {
		go h.processReq.ProcessRequest(keyword, cleanedMSISDN, systemClient, channel, transactionId, creditScore, startTime, ctx)
	}

	c.JSON(http.StatusOK, gin.H{"message": "App Creation request for loan availment created successfully"})
	return nil
}

func (p *ProcessRequestServiceImpl) AvailLoanRequest(request models.AppCreationProcessRequest, ctx context.Context) (bool, error) {

	logger.Info(ctx, "LOAN AVAILMENT STARTED....")

	var price float64
	var name string

	msisdn := request.MSISDN
	var error_obj error

	is_eligible, keyword, loanProduct, loanId, loanDate, error_obj := p.checkElgibRepo.CheckEligibility(ctx, request.MSISDN, request.Keyword, request.Channel, request.CustomerType, request.CreditScore, request.BrandCode, request.BrandId)

	phtTime := common.ConvertUTCToPHT(loanDate)

	request.Keyword = keyword
	logger.Info(ctx, "Is eligible for Loan Availment: %v", is_eligible)
	if error_obj != nil {
		var error_str string
		var messageString string
		caught, messageString := p.checkCaughtError(error_obj)
		if !caught {
			error_str = "Un caught Error:" + error_obj.Error()
		} else {
			error_str = error_obj.Error()
		}
		availmentDB, loanProductDB, servicingPartnerNumberString, _, _ := p.availmentRepo.InsertFailedAvailment(ctx, request.MSISDN, request.Keyword, request.Channel, request.SystemClient, error_str, utils.GetErrorCode(error_obj), false, request.CreditScore, loanProduct, request.BrandCode)
		if loanProductDB == nil {
			price = 0
			name = ""
		} else {
			price = loanProductDB.Price
			name = loanProductDB.Name
		}
		go func(ctx context.Context) {
			err := p.producer.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, request.Channel, request.MSISDN, availmentDB.Brand, availmentDB.LoanType, name, request.Keyword, servicingPartnerNumberString, price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
			if err == nil {
				updateField := bson.M{"publishedToKafka": true}
				p.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
			}
		}(ctx)

		p.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
		if error_obj == consts.ErrorActiveLoan {
			go p.notificationService.NotifyUser(ctx, msisdn, messageString, request.BrandId, nil, loanId, phtTime)
		} else {
			go p.notificationService.NotifyUser(ctx, msisdn, messageString, request.BrandId, loanProduct, loanId, phtTime)
		}
		transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, request.StartTime, request.Channel, availmentDB.GUID, availmentDB.ErrorCode)
		logger.Info(ctx, transaction)
		return false, error_obj

	}

	if is_eligible {
		filter := bson.M{
			"productId": loanProduct.ProductId,
			"brandId":   request.BrandId,
			"isDeleted": bson.M{"$ne": true},
		}
		logger.Debug("filter for product name in amax :%v ", filter)
		productsNameInAmax, err := p.productsNameInAmaxrepo.ProductNameInAmaxsByFilter(ctx, filter)
		if err != nil {
			logger.Debug("Error while retriving product name in amax :%v ", err)
			availmentDB, loanProductDB, servicingPartnerNumberString, _, _ := p.availmentRepo.InsertFailedAvailment(ctx, request.MSISDN, request.Keyword, request.Channel, request.SystemClient, err.Error(), utils.GetErrorCode(err), false, request.CreditScore, loanProduct, request.BrandCode)
			if loanProductDB == nil {
				price = 0
				name = ""
			} else {
				price = loanProductDB.Price
				name = loanProductDB.Name
			}
			go p.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedSystemFailure, request.BrandId, loanProduct, loanId, phtTime)
			go func(ctx context.Context) {
				err := p.producer.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, request.Channel, request.MSISDN, availmentDB.Brand, availmentDB.LoanType, name, request.Keyword, servicingPartnerNumberString, price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
				if err == nil {
					updateField := bson.M{"publishedToKafka": true}
					p.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
				}
			}(ctx)
			p.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
			transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, request.StartTime, request.Channel, availmentDB.GUID, availmentDB.ErrorCode)
			logger.Info(ctx, transaction)
			// return true, error
			return false, err
		}
		logger.Info(ctx, "product names in amax:%v", productsNameInAmax)
		err_amax := p.amaxService.ProvisionLoanAmax(ctx, request.MSISDN, request.TransactionID, loanProduct, request.Keyword, request.Channel, request.SystemClient, productsNameInAmax.Name, request.CreditScore, request.BrandCode, request.BrandId, request.StartTime)
		if err_amax != nil {
			// logger.Error(ctx, "Error in AvailLoanRequest: %v", err_amax)
			return true, err_amax
		}
		// PublishTransactionToKafka()
		return true, nil
	}

	if error_obj != nil {
		return true, error_obj
	}

	return true, nil
}

func (p *ProcessRequestServiceImpl) ProcessRequest(keyword string, msisdn string, systemClient string, channel string, transactionId string, creditScore float64, startTime time.Time, ctx context.Context) {

	var price float64
	var name string

	credit_score_value, customer_type, brand_type, brandId, notificationEvent, err := p.transactionInprogressAndBrandCheck(ctx, msisdn)
	if err != nil {
		availmentDB, loanProductDB, servicingPartnerNumberString, _, _ := p.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, err.Error(), utils.GetErrorCode(err), false, creditScore, nil, "")
		if loanProductDB == nil {
			price = 0
			name = ""
		} else {
			price = loanProductDB.Price
			name = loanProductDB.Name
		}
		go p.notificationService.NotifyUser(ctx, msisdn, notificationEvent, brandId, loanProductDB, primitive.NilObjectID, time.Time{})
		go func(ctx context.Context) {
			err := p.producer.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, name, keyword, servicingPartnerNumberString, price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
			if err == nil {
				updateField := bson.M{"publishedToKafka": true}
				p.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
			} else {
				logger.Error(ctx, "PublishAvailmentStatusToKafka: %v", err)
			}
		}(ctx)
		transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
		logger.Info(ctx, transaction)
		// transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(msisdn)
		return
	}

	transactionIProgressDB := common.SerializeTransactionInProgress(msisdn, keyword)
	p.transactionInProgressRepo.CreateTransactionInProgressEntry(ctx, transactionIProgressDB)

	logger.Debug(ctx, "Keyword: %v, MSISDN: %v, systemClient: %v, channel: %v", keyword, msisdn, systemClient, channel)

	request := common.SerializeAppCreationProcessRequest(msisdn, channel, keyword, systemClient, transactionId, credit_score_value, customer_type, brand_type, brandId, startTime)

	p.AvailLoanRequest(request, ctx)

	// transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(msisdn)
}

func (h *LoanAvailmentService) requestValidation(ctx context.Context, msisdn string, keyword string, transactionId string) (bool, string, string, string, error) {

	// MSISDN parameter format validation
	cleanedMSISDN := utils.CleanMSISDN(msisdn)
	isMsisdnValid, msisdn_err := utils.IsValidMSISDN(cleanedMSISDN)

	if msisdn_err != nil {
		logger.Error(ctx, "Error in checking MSISDN format")
	}
	if !isMsisdnValid {
		logger.Error(ctx, "MSISDN format not valid")
		return false, "", "", "", msisdn_err
	}
	last10MSISDN := cleanedMSISDN[len(cleanedMSISDN)-10:]
	// TranasctionID parameter format validation
	isTransactionIdValid, channel, transactionNumber, trnId_err := utils.IsValidTransactionID(transactionId)
	if trnId_err != nil {
		logger.Error(ctx, "Error in checking Transaction ID format")
	}
	if !isTransactionIdValid {
		logger.Warn(ctx, "Transaction ID format not valid")
		return false, "", "", "", trnId_err
	}

	//D20-413 Keyword parameter format validation
	isKeywordValid, keyword_err := utils.IsValidKeyword(keyword)
	if keyword_err != nil {
		logger.Error(ctx, "Error in checking Keyword format")
	}
	if !isKeywordValid {
		logger.Error(ctx, "Keyword format not valid")
		return false, "", channel, transactionNumber, keyword_err
	}

	return true, last10MSISDN, channel, transactionNumber, nil

}
func (p *ProcessRequestServiceImpl) checkCaughtError(err error) (bool, string) {
	switch {
	case errors.Is(err, consts.ErrorInvalidKeyword):
		return true, consts.LoanAvailmentFailedKeywordNotFound
	case errors.Is(err, consts.ErrorUserBlacklisted):
		return true, consts.LoanAvailmentFailedBlacklistedSubscriber
	case errors.Is(err, consts.ErrorLoanProductNotFound):
		return true, consts.LoanAvailmentFailedProductNotFound
	case errors.Is(err, consts.ErrorActiveLoan):
		return true, consts.LoanAvailmentFailedActiveLoan
	case errors.Is(err, consts.ErrorUserNotWhitelistedForProduct):
		return true, consts.LoanAvailmentFailedNotWhitelisted
	case errors.Is(err, consts.ErrorGetDetailsByAttributeFailed):
		return true, consts.LoanAvailmentFailedDownstreamFailed
	case errors.Is(err, consts.ErrorBrandTypeNotFound):
		return true, consts.LoanAvailmentFailedBrandDoesNotExist
	case errors.Is(err, consts.ErrorChannelTypeNotFound):
		return true, consts.LoanAvailmentFailedChannelNotAvailible
	case errors.Is(err, consts.ErrorBrandNotAllowedToAvailLoanProduct):
		return true, consts.LoanAvailmentFailedBrandNotMappedToProduct
	case errors.Is(err, consts.ErrorChannelNotAllowedToAvailLoanProduct):
		return true, consts.LoanAvailmentFailedChannelNotAvailible
	case errors.Is(err, consts.ErrorUupConsumerType):
		return true, consts.LoanAvailmentFailedCustomerTypeNotAllowed
	case errors.Is(err, consts.ErrorUupBrandType):
		return true, consts.LoanAvailmentFailedSystemFailure
	case errors.Is(err, consts.ErrorInEligibleCreditScore):
		return true, consts.LoanAvailmentLowUUpScore
	case errors.Is(err, consts.ErrorGetDetailsByAttributeFailed):
		return true, consts.LoanAvailmentFailedDownstreamFailed
	default:
		return false, consts.LoanAvailmentFailedSystemFailure
	}
}

func (p *ProcessRequestServiceImpl) transactionInprogressAndBrandCheck(ctx context.Context, msisdn string) (float64, string, string, primitive.ObjectID, string, error) {
	credit_score_value, customer_type, brand_type, subscriberType, creditLimitDate, err := p.helper.GetCreditScoreCustomerTypeBrandType(ctx, msisdn)
	if err != nil {
		return 0, "", "", primitive.NilObjectID, consts.LoanAvailmentFailedDownstreamFailed, err

	}
	_, brandId, err := p.brandExist.CheckIfBrandExistsAndActive(ctx, brand_type)
	if err != nil {
		return credit_score_value, customer_type, brand_type, primitive.NilObjectID, consts.LoanAvailmentFailedBrandDoesNotExist, err

	}

	isAvailmentTransactionInProgress, err := p.transactionInProgressRepo.IsAvailmentTransactionInProgress(ctx, msisdn)
	if err != nil {
		logger.Error(ctx, "isAvailmentTransactionInProgress Error: %v", err)
		return credit_score_value, customer_type, brand_type, brandId, consts.LoanAvailmentFailedSystemFailure, err
	}
	if isAvailmentTransactionInProgress {
		return credit_score_value, customer_type, brand_type, brandId, consts.LoanAvailmentTransactionInProgress, consts.ErrorTransactionInProgress
	}
	subscribers := common.SerializeSubscriberEntry(msisdn, false, "", subscriberType, customer_type, credit_score_value, creditLimitDate)
	err = p.subscriberRepo.AddUpdateSubscriber(msisdn, &subscribers)
	return credit_score_value, customer_type, brand_type, brandId, "", nil
}
