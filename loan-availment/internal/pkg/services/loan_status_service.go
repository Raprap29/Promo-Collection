package services

import (
	"encoding/gob"
	"time"

	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/notification"
	"globe/dodrio_loan_availment/internal/pkg/utils"
	"globe/dodrio_loan_availment/internal/pkg/utils/worker"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	gob.Register(primitive.ObjectID{})
	gob.Register(int32(0))
}

type LoanStatusService struct {
	workerPool          *worker.WorkerPool
	activeLoanRepo      LoanRepo
	loansProductsRepo   LoanProductRepo
	helper              Helper
	brandExist          BrandExist
	notificationService *notification.NotificationService
}

func NewLoanStatusService(workerPool *worker.WorkerPool, activeLoanRepo LoanRepo, loansProductsRepo LoanProductRepo, helper Helper, brandExist BrandExist, notificationService *notification.NotificationService) *LoanStatusService {
	return &LoanStatusService{
		workerPool:          workerPool,
		activeLoanRepo:      activeLoanRepo,
		loansProductsRepo:   loansProductsRepo,
		helper:              helper,
		brandExist:          brandExist,
		notificationService: notificationService,
	}
}

func (h *LoanStatusService) LoanStatus(c *gin.Context, request models.AppCreationRequest) error {
	ctx := c.Request.Context()
	cleanedMSISDN := utils.CleanMSISDN(request.MSISDN)
	isMsisdnValid, msisdn_err := utils.IsValidMSISDN(cleanedMSISDN)

	if msisdn_err != nil {
		logger.Error(ctx, "MSISDN format not valid")

		c.JSON(http.StatusInternalServerError, gin.H{"message": consts.ErrorMSISDNNotValid.Error()})
		return nil
	}
	if !isMsisdnValid {
		logger.Error(ctx, "MSISDN format not valid")

		c.JSON(http.StatusInternalServerError, gin.H{"message": consts.ErrorMSISDNNotValid.Error()})
		return nil
	}

	logger.Info(ctx, "CHECK LOAN STATUS FOR MSISDN: %v", request.MSISDN)
	msisdn := cleanedMSISDN[len(cleanedMSISDN)-10:]
	filter := bson.M{"MSISDN": msisdn}

	loanFound := true
	activeLoanResult, err := h.activeLoanRepo.ActiveLoanByFilter(filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusOK, gin.H{"message": consts.NoLoanFound})
			loanFound = false
		} else {
			return err
		}
	}

	if loanFound {
		c.JSON(http.StatusOK, gin.H{"message": consts.LoanFound})
		loansProductsFilter := bson.M{"_id": activeLoanResult.LoanProductId}
		loansProductsResult, err := h.loansProductsRepo.LoanProductByFilter(loansProductsFilter)
		if err != nil {
			return err
		}

		phtTime := common.ConvertUTCToPHT(activeLoanResult.CreatedAt)
		go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanStatusLoanFound, activeLoanResult.BrandId, loansProductsResult, activeLoanResult.LoanId, phtTime)

	} else {
		_, _, brandCode, _, _, err := h.helper.GetCreditScoreCustomerTypeBrandType(ctx, request.MSISDN)
		if err != nil {
			logger.Error(ctx, consts.LoanAvailmentFailedDownstreamFailed)
			go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedDownstreamFailed, primitive.NilObjectID, nil, primitive.NilObjectID, time.Time{})
		}
		_, brandId, err := h.brandExist.CheckIfBrandExistsAndActive(ctx, brandCode)
		if err != nil {
			logger.Error(ctx, consts.LoanAvailmentFailedBrandDoesNotExist)
			go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedBrandDoesNotExist, primitive.NilObjectID, nil, primitive.NilObjectID, time.Time{})
		}
		go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanStatusLoanNotFound, brandId, nil, primitive.NilObjectID, time.Time{})
		logger.Info(ctx, consts.LoanStatusLoanNotFound)

	}
	return nil
}
