package services

import (
	// "errors"

	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	// "globe/dodrio_loan_availment/internal/pkg/store"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ValidationService struct {
	brandRepo               BrandByFilterService
	keywordRepo             KeywordByFilterService
	subscribersRepo         CheckSubscriberBlacklistInterface
	loansProductsRepo       LoanProductRepo
	activeLoanRepo          ActiveLaoneService
	whitelistedSubsRepo     IsWhitelistedService
	channelRepo             ChannelByFilterServcie
	loanProductsBrandsRepo  LoanProductChannelService
	loanProductsChannelRepo LoanProductsChannelByFilterService
}

func NewValidationService(brandRepo BrandByFilterService, keywordRepo KeywordByFilterService, subscribersRepo CheckSubscriberBlacklistInterface, loansProductsRepo LoanProductRepo, activeLoanRepo ActiveLaoneService, whitelistedSubsRepo IsWhitelistedService, channelRepo ChannelByFilterServcie, loanProductsBrandsRepo LoanProductChannelService, loanProductsChannelRepo LoanProductsChannelByFilterService) *ValidationService {
	return &ValidationService{
		brandRepo:               brandRepo,
		keywordRepo:             keywordRepo,
		subscribersRepo:         subscribersRepo,
		loansProductsRepo:       loansProductsRepo,
		activeLoanRepo:          activeLoanRepo,
		whitelistedSubsRepo:     whitelistedSubsRepo,
		channelRepo:             channelRepo,
		loanProductsBrandsRepo:  loanProductsBrandsRepo,
		loanProductsChannelRepo: loanProductsChannelRepo,
	}
}

func (v *ValidationService) CheckKeyword(ctx context.Context, keyword string) (*models.Keyword, error) {
	keywordFilter := bson.M{
		"name": bson.M{
			"$regex":   "^" + keyword + "$",
			"$options": "i",
		}, "isDeleted": bson.M{"$ne": true},
	}

	keywordResult, err := v.keywordRepo.KeywordByFilter(keywordFilter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Error(ctx, "No Keyword found")
			return nil, consts.ErrorInvalidKeyword
		}
		logger.Error(ctx, "Error fetching keyword: %v", err)
		return nil, err
	}

	if keywordResult.IsDeleted {
		logger.Warn(ctx, "Keyword is deleted")
		return nil, consts.ErrorInvalidKeyword
	}

	return keywordResult, nil
}

func (v *ValidationService) CheckSubscriberBlacklisitng(ctx context.Context, msisdn string) (*models.Subscribers, error) {
	subscriberFilter := bson.M{"MSISDN": msisdn}
	subscriber, err := v.subscribersRepo.SubscribersByFilter(subscriberFilter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Error(ctx, "Subscriber not found in database, assumed to not be blacklisted.")
			return nil, nil
		} else {
			logger.Error(ctx, "Error in checking blacklisted status: %v", err)
			return nil, err
		}
	}

	if subscriber.Blacklisted {
		logger.Info(ctx, "User is blacklisted")
		return nil, consts.ErrorUserBlacklisted
	}

	return subscriber, nil
}

func (v *ValidationService) CheckLoanProduct(ctx context.Context, productId primitive.ObjectID) (*models.LoanProduct, error) {
	loanProductFilter := bson.M{"_id": productId, "isDeleted": bson.M{"$ne": true}}
	loanProductsResult, err := v.loansProductsRepo.LoanProductByFilter(loanProductFilter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Info(ctx, "No Loan Product found.")
			return nil, consts.ErrorLoanProductNotFound
		}
		logger.Error(ctx, "Error in Loan Product entry: %v", err)
		return nil, err
	}

	if loanProductsResult.IsDeleted {
		logger.Warn(ctx, "Loan Product not found (deleted).")
		return nil, consts.ErrorLoanProductNotFound
	}

	currentDate := time.Now()

	switch loanProductsResult.ActivationMethod {
	case "promoDates":
		if loanProductsResult.PromoStartDate != nil && loanProductsResult.PromoEndDate != nil {
			if currentDate.After(*loanProductsResult.PromoStartDate) && currentDate.Before(*loanProductsResult.PromoEndDate) {
				logger.Info(ctx, "Today's date is within the promo period.")
			} else {
				logger.Info(ctx, "Today's date is outside the promo period.")
				return nil, consts.ErrorLoanProductNotFound
			}
		} else {
			logger.Info(ctx, "Promo start or end date is not set.")
			return nil, consts.ErrorLoanProductNotFound
		}

	case "manual":
		if !loanProductsResult.Active {
			logger.Warn(ctx, "Loan product is not active.")
			return nil, consts.ErrorLoanProductNotFound
		}

	default:
		logger.Warn(ctx, "Invalid activation method.")
	}

	return loanProductsResult, nil
}

func (v *ValidationService) CheckActiveLoan(ctx context.Context, msisdn string) (bool, primitive.ObjectID, time.Time, error) {
	isLoanActive, loan, err := v.activeLoanRepo.IsActiveLoan(ctx, msisdn)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn(ctx, "No Active Loan")
			isLoanActive = false
		} else {
			logger.Warn(ctx, "Mongo Error on IsActiveLoan check :%v", err)
			return false, primitive.NilObjectID, time.Time{}, err
		}
	}

	if isLoanActive {
		logger.Info(ctx, "Active loan already exists")
		return true, loan.LoanId, loan.CreatedAt, consts.ErrorActiveLoan
	}

	return false, primitive.NilObjectID, time.Time{}, nil
}

func (v *ValidationService) CheckWhitelist(ctx context.Context, msisdn string, loanProduct *models.LoanProduct) (bool, error) {

	// Check if whitelisting is enabled for the loan product
	if !loanProduct.HasWhitelisting {
		logger.Warn(ctx, "Whitelisting is not enabled for this loan product.")
		return true, nil

	}

	isWhitelisted, err := v.whitelistedSubsRepo.IsWhitelisted(msisdn, loanProduct.ProductId)
	if err != nil {
		logger.Error(ctx, "Error in checking whitelisted status: %v", err)
		return false, err
	}

	if !isWhitelisted {
		logger.Warn(ctx, "User not whitelisted for selected product")
		return false, consts.ErrorUserNotWhitelistedForProduct
	}

	return true, nil
}

func (v *ValidationService) CheckIfBrandExistsAndActive(ctx context.Context, brand string) (bool, primitive.ObjectID, error) {
	brandFilter := bson.M{
		"code": bson.M{
			"$regex":   "^" + brand + "$",
			"$options": "i",
		},
		"isDeleted": bson.M{"$ne": true},
	}
	brandResult, brandErr := v.brandRepo.BrandsByFilter(brandFilter)
	if brandErr != nil {
		logger.Info(ctx, "MSISDN's Brand not found")
		return false, primitive.NilObjectID, consts.ErrorBrandTypeNotFound
	}
	if brandResult.IsDeleted {
		logger.Info(ctx, "MSISDN's Brand is deleted")
		return false, primitive.NilObjectID, consts.ErrorBrandTypeNotFound
	}
	return true, brandResult.Id, nil

}

func (v *ValidationService) CheckIfChannelExistsAndActive(ctx context.Context, channel string) (bool, primitive.ObjectID, error) {

	channelFilter := bson.M{
		"code":      channel,
		"isDeleted": bson.M{"$ne": true},
	}
	channelResult, channelErr := v.channelRepo.ChannelsByFilter(ctx, channelFilter)
	if channelErr != nil {
		logger.Info(ctx, "MSISDN's Channel not found")
		return false, primitive.NilObjectID, consts.ErrorChannelTypeNotFound
	}
	if channelResult.IsDeleted {
		logger.Info(ctx, "MSISDN's Channel is deleted")
		return false, primitive.NilObjectID, consts.ErrorChannelTypeNotFound
	}

	return true, channelResult.Id, nil

}

func (v *ValidationService) CheckIfBrandAndProductMapped(ctx context.Context, brandId, productId primitive.ObjectID) (bool, error) {

	loanProductsBrandsFilter := bson.M{
		"brandId":   brandId,
		"productId": productId,
	}

	logger.Debug(ctx, "brandId, productId is: %v, %v", brandId, productId)
	_, lpb_err := v.loanProductsBrandsRepo.LoanProductsBrandsByFilter(ctx, loanProductsBrandsFilter)
	if lpb_err != nil {
		if lpb_err == mongo.ErrNoDocuments {
			logger.Warn(ctx, "No mapped data found for BrandId and ProductId")
			return false, consts.ErrorBrandNotAllowedToAvailLoanProduct
		} else {
			logger.Error(ctx, "Error in checking mapped data found for BrandId and ProductId: %v", lpb_err)
			return false, lpb_err
		}
		// return false, lpb_err
	}

	return true, nil

}

func (v *ValidationService) CheckIfChannelAndProductMapped(ctx context.Context, channelId, productId primitive.ObjectID) (bool, error) {

	loanProductsChannelsFilter := bson.M{
		"channelId": channelId,
		"productId": productId,
	}

	_, lpc_err := v.loanProductsChannelRepo.LoanProductsChannelsByFilter(loanProductsChannelsFilter)
	if lpc_err != nil {
		if lpc_err == mongo.ErrNoDocuments {
			logger.Warn(ctx, "No mapped data found for ChannelId and ProductId")
			return false, consts.ErrorChannelNotAllowedToAvailLoanProduct
		} else {
			logger.Error(ctx, "Error in checking mapped data found for BrandId and ProductId: %v", lpc_err)
			return false, lpc_err
		}
		// return false, lpc_err
	}

	return true, nil

}

func (v *ValidationService) CheckCreditScoreEligibility(ctx context.Context, creditScore float64, loanProductResult *models.LoanProduct) bool {
	logger.Debug(ctx, "fetched loan price is  = %v", loanProductResult.Price)
	creditScoreEligibility := v.checkCreditScoreAndLoanPrice(creditScore, loanProductResult.Price)
	return creditScoreEligibility
}
func (v *ValidationService) checkCreditScoreAndLoanPrice(creditScore float64, loanPrice float64) bool {
	return creditScore >= loanPrice
}
