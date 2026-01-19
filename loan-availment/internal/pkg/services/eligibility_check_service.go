package services

import (
	"context"
	"fmt"

	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/downstreams"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/utils"

	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type EligibilityCheckService struct {
	helper                     Helper
	checkEligb                 CheckKeywordInterface
	activeLoanRepo             ActiveFilterInterface
	checkSubsRepo              CheckSubsInterface
	checkLoanRepo              CheckLoanProductService
	checkActiveRepo            CheckActiveLoanService
	checkWhiteListRepo         CheckWhitelistSErvice
	checkChannelRepo           CheckChannelExistService
	checkChannelandProductRepo CheckChannelAndProductService
	checkBrandandProductRepo   CheckBrandandProductService
	checkScoreRepo             CheckScoreEligService
	keywordRepo                KeywordByFilterService
	brandRepo                  BrandByFilterService
	subscriberRepo             SubscribersRepository
	availmentRepo              LoanAvailmentServicesInt
	unpaidLoanRepo             UnpaidLoanFilterInterface
}

func NewEligibilityCheckService(helper Helper, checkEligb CheckKeywordInterface, activeLoanRepo ActiveFilterInterface, checkSubsRepo CheckSubsInterface, checkLoanRepo CheckLoanProductService, checkActiveRepo CheckActiveLoanService, checkWhiteListRepo CheckWhitelistSErvice, checkChannelRepo CheckChannelExistService, checkChannelandProductRepo CheckChannelAndProductService, checkBrandandProductRepo CheckBrandandProductService, checkScoreRepo CheckScoreEligService, keywordRepo KeywordByFilterService, brandRepo BrandByFilterService, subscriberRepo SubscribersRepository, availmentRepo LoanAvailmentServicesInt, unpaidLoanRepo UnpaidLoanFilterInterface) *EligibilityCheckService {
	return &EligibilityCheckService{
		helper:                     helper,
		checkEligb:                 checkEligb,
		activeLoanRepo:             activeLoanRepo,
		availmentRepo:              availmentRepo,
		checkSubsRepo:              checkSubsRepo,
		checkLoanRepo:              checkLoanRepo,
		checkActiveRepo:            checkActiveRepo,
		checkWhiteListRepo:         checkWhiteListRepo,
		checkChannelRepo:           checkChannelRepo,
		checkChannelandProductRepo: checkChannelandProductRepo,
		checkBrandandProductRepo:   checkBrandandProductRepo,
		checkScoreRepo:             checkScoreRepo,
		keywordRepo:                keywordRepo,
		brandRepo:                  brandRepo,
		subscriberRepo:             subscriberRepo,
		unpaidLoanRepo:             unpaidLoanRepo,
	}
}

func (h *EligibilityCheckService) EligibilityCheck(c *gin.Context, MSISDN string) error {
	ctx := c.Request.Context()
	cleanedMSISDN := utils.CleanMSISDN(MSISDN)
	isMsisdnValid, msisdn_err := utils.IsValidMSISDN(cleanedMSISDN)

	if msisdn_err != nil {
		logger.Error(ctx, "Error in checking MSISDN format: %v", msisdn_err)
		response := common.SerializeEligibilityCheckResponse(MSISDN, nil, nil, nil, nil, nil, nil, consts.ErrorMSISDNNotValid.Error(), strconv.Itoa(http.StatusNotImplemented))
		c.JSON(http.StatusNotImplemented, response)
		return nil
	}
	if !isMsisdnValid {
		logger.Warn(ctx, "MSISDN format not valid")
		response := common.SerializeEligibilityCheckResponse(MSISDN, nil, nil, nil, nil, nil, nil, consts.ErrorMSISDNNotValid.Error(), strconv.Itoa(http.StatusNotImplemented))
		c.JSON(http.StatusNotImplemented, response)
		return nil
	}
	MSISDN = cleanedMSISDN[len(cleanedMSISDN)-10:]

	credit_score_val, _, brand_type, _, _, err := h.helper.GetCreditScoreCustomerTypeBrandType(ctx, MSISDN)
	credit_score_reason := ""
	if err != nil {
		if err == consts.ErrorMSISDNNotFound {
			response := common.SerializeEligibilityCheckResponse(MSISDN, nil, nil, nil, nil, nil, nil, consts.ErrorMSISDNNotFound.Error(), strconv.Itoa(http.StatusBadGateway))
			c.JSON(http.StatusBadGateway, response)
			return nil
		}
		response := common.SerializeEligibilityCheckResponse(MSISDN, nil, nil, nil, nil, nil, nil, http.StatusText(http.StatusInternalServerError), strconv.Itoa(http.StatusInternalServerError))
		c.JSON(http.StatusInternalServerError, response)
		return nil
	}

	credit_score := strconv.FormatInt(int64(credit_score_val), 10)
	if credit_score_reason != "" {
		credit_score = credit_score_reason
	}

	activeLoanFilter := bson.M{"MSISDN": MSISDN}
	active_loan, act_err := h.activeLoanRepo.ActiveLoanByFilter(activeLoanFilter)
	if act_err != nil {
		if act_err == mongo.ErrNoDocuments {
			logger.Debug(ctx, "No Active Loan")
			response := common.SerializeEligibilityCheckResponse(MSISDN, &credit_score, &credit_score, new(bool), new(string), new(float64), new(string), http.StatusText(http.StatusOK), strconv.Itoa(http.StatusOK))

			c.JSON(http.StatusOK, response)
			return nil
		}
		logger.Error(ctx, "Error in checking active loan status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": act_err.Error()})
		return nil
	} else {
		logger.Info(ctx, "EligibilityCheckService hasActiveLoan - MSISDN: %v", MSISDN)
	}

	unpaidLoanFilter := bson.M{"loanId": active_loan.LoanId}
	unpaidLoanResult, u_err := h.unpaidLoanRepo.UnpaidLoansByFilter(unpaidLoanFilter)
	loanAmount := 0.00
	if u_err == nil {
		loanAmount = unpaidLoanResult.UnpaidAmount
	}

	logger.Info(ctx, "EligibilityCheckService UnPaidLoanAmount - MSISDN: %v RemainingLoanAmount: %v", MSISDN, loanAmount)

	availments, err := h.availmentRepo.GetAvailmentById(active_loan.AvailmentTransactionId)
	fmt.Println(availments)

	keywordName := ""
	if availments != nil {
		keywordName = availments.Keyword
	}

	remainingScore := ""
	remainingScore_val := credit_score_val - (active_loan.TotalLoanAmount - active_loan.ServiceFee)
	if credit_score_reason != "" {
		remainingScore = credit_score_reason
	} else {
		remainingScore = strconv.FormatInt(int64(remainingScore_val), 10)
	}
	loanProvisionStatus := true

	response := common.SerializeEligibilityCheckResponse(MSISDN, &credit_score, &remainingScore, &loanProvisionStatus, &keywordName, &loanAmount, &brand_type, http.StatusText(http.StatusOK), strconv.Itoa(http.StatusOK))

	c.JSON(http.StatusOK, response)
	return nil
}

func (h *EligibilityCheckService) CheckEligibility(ctx context.Context, msisdn string, keyword string, channel string, customer_type string, credit_score_value float64, brand_type string, brandId primitive.ObjectID) (bool, string, *models.LoanProduct, primitive.ObjectID, time.Time, error) {

	// Keyword valid check
	keywordResult, err := h.checkEligb.CheckKeyword(ctx, keyword)

	if err != nil {
		return false, keyword, nil, primitive.NilObjectID, time.Time{}, err
	}
	keyword = keywordResult.Name
	// Subscriber Blacklist check
	_, err = h.checkSubsRepo.CheckSubscriberBlacklisitng(ctx, msisdn)
	if err != nil {
		return false, keyword, nil, primitive.NilObjectID, time.Time{}, err
	}

	// Loan Product Validity
	loanProductResult, err := h.checkLoanRepo.CheckLoanProduct(ctx, keywordResult.ProductId)
	if err != nil {
		return false, keyword, loanProductResult, primitive.NilObjectID, time.Time{}, err
	}

	// Outstanding loan check
	isLoanActive, loanId, loanDate, err := h.checkActiveRepo.CheckActiveLoan(ctx, msisdn)
	if err != nil {
		return false, keyword, loanProductResult, loanId, loanDate, err
	}
	if isLoanActive {
		return false, keyword, loanProductResult, loanId, loanDate, consts.ErrorActiveLoan
	}

	//D20-424 Subscriber Whitelist check for product Id
	isWhitelisted, err := h.checkWhiteListRepo.CheckWhitelist(ctx, msisdn, loanProductResult)
	if !isWhitelisted {
		return false, keyword, loanProductResult, loanId, loanDate, err
	}

	// MSISDN Channel Validity
	isChannelValid, channelId, err := h.checkChannelRepo.CheckIfChannelExistsAndActive(ctx, channel)
	if !isChannelValid {
		return false, keyword, loanProductResult, loanId, loanDate, err
	}

	//D20-423 Channel Validation Against Product
	isChannelAndProductMapped, err := h.checkChannelandProductRepo.CheckIfChannelAndProductMapped(ctx, channelId, keywordResult.ProductId)
	if !isChannelAndProductMapped {
		return false, keyword, loanProductResult, loanId, loanDate, err
	}

	// credit_score_value, customer_type, brand_type, err := GetCreditScoreCustomerTypeBrandType(msisdn)
	// if err != nil {
	// 	return false, keyword, 0, "", loanProductResult, primitive.NilObjectID, loanId, loanDate, consts.ErrorGetDetailsByAttributeFailed
	// }
	// // MSISDN Brand Validity
	// isBrandValid, brandId, err := CheckIfBrandExistsAndActive(brand_type)
	// if !isBrandValid {
	// 	return false, keyword, credit_score_value, brand_type, loanProductResult, brandId, loanId, loanDate, err
	// }

	//D20-421 : UUP Customer Type Validation for Consumer
	if !isValidCustomerType(ctx, customer_type) {
		return false, keyword, loanProductResult, loanId, loanDate, consts.ErrorUupConsumerType
	}

	//D20-430 : UUP Brand Type Validation for Consumer
	if !isValidBrandType(ctx, brand_type) {
		return false, keyword, loanProductResult, loanId, loanDate, consts.ErrorUupBrandType
	}

	//D20-422 : Brand Type Validation Against Product
	isBrandAndProductMapped, err := h.checkBrandandProductRepo.CheckIfBrandAndProductMapped(ctx, brandId, keywordResult.ProductId)
	if !isBrandAndProductMapped {
		return false, keyword, loanProductResult, loanId, loanDate, err
	}

	//D20-425 Check credit score
	if !loanProductResult.IsScoredProduct {
		return true, keyword, loanProductResult, loanId, loanDate, nil
	} else {
		// credit_score_value, _, _, err := GetCreditScoreCustomerTypeBrandType(msisdn)
		// if err != nil {
		// 	credit_score_value = 0
		// }

		has_user_eligible_credit_score := h.checkScoreRepo.CheckCreditScoreEligibility(ctx, credit_score_value, loanProductResult)

		if !has_user_eligible_credit_score {
			// logger.Print("")
			logger.Info(ctx, "Not eligible for the loan as per the credit score")
			return false, keyword, loanProductResult, loanId, loanDate, consts.ErrorInEligibleCreditScore
		}
	}

	return true, keyword, loanProductResult, loanId, loanDate, nil

}

func (h *EligibilityCheckService) GetCreditScoreCustomerTypeBrandType(ctx context.Context, msisdn string) (float64, string, string, string, *time.Time, error) {
	result, err := downstreams.GetDetailsByAttribute(ctx, msisdn)
	if err != nil {
		logger.Error(ctx, "GetDetailsByAttribute not functioning: %v", err)
		return 0, "", "", "", nil, err
	}

	var creditScoreValue float64
	var customerType string
	var brandType string
	var subscriberType string
	var creditLimitDate time.Time

	for _, attribute := range result.AttributeValues {
		// fmt.Println(attribute)
		switch attribute.Name {
		case "clla":
			if attribute.Value != "" {
				creditScoreValueFloat, err := strconv.ParseFloat(attribute.Value, 64)
				if err != nil {
					logger.Error(ctx, "Credit score was not a valid number: %v", err)
				} else {
					creditScoreValue = float64(creditScoreValueFloat)
				}
			}
		case "btc":
			if strings.EqualFold(attribute.Value, string(consts.BrandGHP)) || strings.EqualFold(attribute.Value, string(consts.BrandGHPPrepaid)) || strings.EqualFold(attribute.Value, string(consts.BrandGP)) {
				brandType = string(consts.BrandGHPPrepaid)
			} else {
				brandType = attribute.Value
			}
		case "cfutd":
			customerType = attribute.Value
		case "prodt":
			subscriberType = attribute.Value
		case "clsd":
			if attribute.Value == "" {
				logger.Warn(ctx, "Credit limit date is empty")
			} else {
				parsedTime, err := time.Parse("2006-01-02 15:04:05", attribute.Value) // Example: for ISO8601 format
				if err != nil {
					logger.Error(ctx, "Failed to parse credit limit date: %v", err)
				} else {
					creditLimitDate = parsedTime
				}
			}

		}

	}

	if err != nil {
		logger.Error(ctx, "Error inserting or Updating subscriber %v", err)
	}

	return creditScoreValue, customerType, brandType, subscriberType, &creditLimitDate, nil
}

// Helper function to validate customer type
func isValidCustomerType(ctx context.Context, customerType string) bool {
	// Use consts.CustomerTypes slice to validate
	for _, validType := range consts.CustomerTypes {
		if strings.EqualFold(customerType, string(validType)) {
			return true
		}
	}
	logger.Error(ctx, consts.ErrorUupConsumerType)
	return false
}

// Helper function to validate brand type
func isValidBrandType(ctx context.Context, brandType string) bool {
	// Use consts.BrandTypes slice to validate
	for _, validType := range consts.BrandTypes {
		if strings.EqualFold(brandType, string(validType)) {
			return true
		}
	}
	logger.Error(ctx, consts.ErrorUupBrandType)
	return false
}
