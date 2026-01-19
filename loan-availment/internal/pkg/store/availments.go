package store

import (
	// "context"
	"context"
	"encoding/json"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/services/interfaces"
	storeModel "globe/dodrio_loan_availment/internal/pkg/store/models"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AvailmentRepository struct {
	repo                      *MongoRepository[models.Availments]
	keywordRepo               *KeywordRepository
	transactionInProgressRepo *TransactionInProgressRepository
	loansProductsRepo         *LoansProductsRepository
	activeLoanRepo            *ActiveLoanRepository
	unpaidLoanRepo            *UnpaidLoanRepository
	brandRepo                 *BrandRepository
	systemLevelRulesRepo      *SystemLevelRulesRepository
	redisClient               interfaces.RedisStoreOperations
}

func NewAvailmentRepository(redisClient interfaces.RedisStoreOperations) *AvailmentRepository {
	collection := db.MDB.Database.Collection(consts.AvailmentCollection)
	mrepo := NewMongoRepository[models.Availments](collection)
	return &AvailmentRepository{
		repo:                      mrepo,
		keywordRepo:               NewKeywordRepository(),
		transactionInProgressRepo: NewTransactionInProgressRepository(),
		loansProductsRepo:         NewLoansProductsRepository(),
		activeLoanRepo:            NewActiveLoanRepository(),
		unpaidLoanRepo:            NewUnpaidLoanRepository(),
		brandRepo:                 NewBrandRepository(),
		systemLevelRulesRepo:      NewSystemLevelRulesRepository(),
		redisClient:               redisClient,
	}
}

func (r *AvailmentRepository) InsertAvailment(availmentsDB models.Availments) (bool, error) {

	_, err := r.repo.Create(availmentsDB)

	if err != nil {
		logger.Error("availment : Error while inserting %v", err.Error())
		return false, fmt.Errorf("availment : error while inserting %v", err.Error())
	}

	return true, nil

}

func (r *AvailmentRepository) UpdateAvailment(ctx context.Context, availmentID primitive.ObjectID, updateField bson.M) {

	filter := bson.M{"_id": availmentID}
	err := r.repo.Update(filter, updateField)
	if err != nil {
		logger.Error(ctx, "availment : Error while updating %v", err.Error())
	}
}

func (r *AvailmentRepository) InsertFailedAvailment(ctx context.Context, msisdn string, keyword string, channel string, systemClient string, errMsg string, errCode string, result bool, creditScore float64, loanProduct *models.LoanProduct, brandCode string) (models.Availments, *models.LoanProduct, string, string, error) {

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var totalLoanAmount float64
	var serviceFee float64
	var availmentsDB models.Availments
	var keywordResult *models.Keyword
	var loanType string
	var servicingPartner string
	var servicingPartnerNumberString string
	var productId primitive.ObjectID
	var loanProductName string
	if errMsg == "" {
		errMsg = "Unknown Error"
	}

	if errMsg == consts.ErrorKeywordFormatValidationFailed.Error() {
		availmentsDB = common.SerializeAvailments(msisdn, channel, "", "", primitive.NilObjectID, keyword, "", 0, 0, systemClient, errMsg, errCode, result, creditScore, "")
		loanProduct = nil
	} else {
		if loanProduct == nil {
			keywordFilter := bson.M{"name": bson.M{
				"$regex":   "^" + keyword + "$",
				"$options": "i",
			}, "isDeleted": bson.M{"$ne": true}}
			keywordResult, _ = r.keywordRepo.KeywordByFilter(keywordFilter)
			if keywordResult == nil {
				productId = primitive.NilObjectID
				loanType = ""
			} else {
				loanProductName = ""
				productId = keywordResult.ProductId
				loanProductFilter := bson.M{"_id": productId}
				loanProduct, _ = r.loansProductsRepo.LoanProductByFilter(loanProductFilter)
				loanProduct, errMsg := r.loansProductsRepo.LoanProductByFilter(loanProductFilter)
				if errMsg != nil {
					totalLoanAmount = 0
					serviceFee = 0
					loanProductName = ""
					loanType = ""
				} else {
					loanProductName = loanProduct.Name
					if strings.EqualFold(loanProduct.FeeType, string(consts.Percent)) {
						serviceFee = (loanProduct.Price * loanProduct.ServiceFee) / 100
					} else {
						serviceFee = loanProduct.ServiceFee
					}
					totalLoanAmount = serviceFee + loanProduct.Price
					loanType = loanProduct.LoanType
				}
			}

		} else {
			productId = loanProduct.ProductId
			loanType = loanProduct.LoanType
			if errMsg != "" {
				loanProductName = loanProduct.Name
				totalLoanAmount = 0
				serviceFee = 0
			} else {
				if strings.EqualFold(loanProduct.FeeType, string(consts.Percent)) {
					serviceFee = (loanProduct.Price * loanProduct.ServiceFee) / 100
				} else {
					serviceFee = loanProduct.ServiceFee
				}
				totalLoanAmount = serviceFee + loanProduct.Price
				loanProductName = loanProduct.Name
			}
		}

		// if brandCode == "" && loanProduct != nil {
		// 	brandCode, _ = r.brandRepo.BrandsByProductId(ctx, loanProduct.ProductId)
		// }
		servicingPartner, servicingPartnerNumberString, _ = r.loansProductsRepo.ServicingPartnerByProductId(ctx, productId)

		// Availments Table Serialization
		availmentsDB = common.SerializeAvailments(msisdn, channel, brandCode, loanType, productId, keyword, servicingPartner, totalLoanAmount, serviceFee, systemClient, errMsg, errCode, result, creditScore, loanProductName)
	}

	// Start session
	session, err := db.MDB.Client.StartSession()

	if err != nil {
		return models.Availments{}, &models.LoanProduct{}, servicingPartnerNumberString, "", err
	}
	defer session.EndSession(ctx)

	txnErr := mongo.WithSession(ctx, session, func(sessCtx mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}

		// Insert availmentsDB
		_, err := r.InsertAvailment(availmentsDB)
		if err != nil {
			session.AbortTransaction(sessCtx)
			return err
		}

		// Delete transactionInProgress By MSISDN
		// _, err = r.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(msisdn)
		// if err != nil {
		// 	session.AbortTransaction(sessCtx)
		// 	return err
		// }

		session.CommitTransaction(sessCtx)
		return err
	})
	if txnErr != nil {
		return models.Availments{}, &models.LoanProduct{}, servicingPartnerNumberString, "", txnErr
	}

	return availmentsDB, loanProduct, servicingPartnerNumberString, "Inserted Successfully", nil

}

func (r *AvailmentRepository) InsertSuccessfulAvailment(ctx context.Context, msisdn string, loanProduct *models.LoanProduct, keyword string, channel string, systemClient string, errMsg string, errCode string, result bool, creditScore float64, brandCode string, brandId primitive.ObjectID) (models.Availments, *models.LoanProduct, string, string, error) {

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var totalLoanAmount float64
	var serviceFee float64
	// keywordFilter := bson.D{{Key: "name", Value: keyword}}
	// keywordResult, _ := r.keywordRepo.KeywordByFilter(keywordFilter)

	// loanProductFilter := bson.M{"_id": keywordResult.ProductId}
	// loanProductsResult, _ := r.loansProductsRepo.LoanProductByFilter(loanProductFilter)

	// loanType, _ := r.keywordRepo.LoanTypeByKeyword(keyword)
	// brandCode, brandId, _, _ := r.keywordRepo.BrandCodeAndIdByKeyword(keyword)

	servicingPartner, servicingPartnerNumberString, _ := r.loansProductsRepo.ServicingPartnerByProductId(ctx, loanProduct.ProductId)

	if strings.EqualFold(loanProduct.FeeType, string(consts.Percent)) {
		serviceFee = (loanProduct.Price * loanProduct.ServiceFee) / 100
	} else {
		serviceFee = loanProduct.ServiceFee
	}
	systemLevelRules, err := r.systemLevelRulesRepo.SystemLevelRules(map[string]interface{}{})
	if err != nil {
		logger.Error(ctx, "Error fetching system level rules: %v", err)
		return models.Availments{}, nil, "", "", err
	}
	totalLoanAmount = serviceFee + loanProduct.Price
	// Availments Table Serialization
	availmentsDB := common.SerializeAvailments(msisdn, channel, brandCode, loanProduct.LoanType, loanProduct.ProductId, keyword, servicingPartner, totalLoanAmount, serviceFee, systemClient, errMsg, errCode, result, creditScore, loanProduct.Name)

	// Loans Table Serialization
	loansDB := common.SerializeLoans(msisdn, totalLoanAmount, availmentsDB.GUID, serviceFee, *loanProduct, brandId, availmentsDB.ID, systemLevelRules)

	// UnpaidLoans Table Serialization
	unpaidLoansDB := common.SerializeUnpaidLoans(loansDB.LoanId, totalLoanAmount, serviceFee)

	// Start DB session
	session, err := db.MDB.Client.StartSession()

	if err != nil {
		return models.Availments{}, &models.LoanProduct{}, servicingPartnerNumberString, "", err
	}
	defer session.EndSession(ctx)

	txnErr := mongo.WithSession(ctx, session, func(sessCtx mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}

		// Insert availmentsDB
		_, err := r.InsertAvailment(availmentsDB)
		if err != nil {
			session.AbortTransaction(sessCtx)
			return err
		}

		// Insert loansDB
		_, err = r.activeLoanRepo.CreateActiveLoanEntry(ctx, loansDB)
		if err != nil {
			session.AbortTransaction(sessCtx)
			return err
		}

		// Insert unpaidLoansDB
		_, err = r.unpaidLoanRepo.CreateUnpaidLoansEntry(ctx, unpaidLoansDB)
		if err != nil {
			session.AbortTransaction(sessCtx)
			return err
		}

		// Delete transactionInProgress By MSISDN
		_, err = r.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
		if err != nil {
			session.AbortTransaction(sessCtx)
			return err
		}

		session.CommitTransaction(sessCtx)
		biggerTimestamp := loansDB.PromoEducationPeriodTimestamp
		if loansDB.LoanLoadPeriodTimestamp.Time().After(loansDB.PromoEducationPeriodTimestamp.Time()) {
			biggerTimestamp = loansDB.LoanLoadPeriodTimestamp
		}
		duration := time.Until(biggerTimestamp.Time())
		redisData, err := json.Marshal(storeModel.RedisLoan{
			EducationPeriodTimestamp: loansDB.PromoEducationPeriodTimestamp,
			LoanLoadPeriodTimestamp:  loansDB.LoanLoadPeriodTimestamp,
		})
		if err != nil {
			logger.Error(ctx, "Error marshaling loan data: %v", err)
			return err
		}
		redisKey := fmt.Sprintf(storeModel.LoanKeyPattern, msisdn)
		err = r.redisClient.Set(context.Background(), redisKey, redisData, duration)
		if err != nil {
			logger.Error(ctx, "Error saving loan to redis: %v", err)
			return err
		} else {
			logger.Info(ctx, "Successfully saved loan to redis with key %s", redisKey)
		}
		return err
	})
	if txnErr != nil {
		return models.Availments{}, &models.LoanProduct{}, servicingPartnerNumberString, "", txnErr
	}

	return availmentsDB, loanProduct, servicingPartnerNumberString, "Inserted Successfully", nil

}
func (r *AvailmentRepository) GetAvailmentById(id primitive.ObjectID) (*models.Availments, error) {
	filter := bson.M{"_id": id}

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
