package store

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type LoansProductsRepository struct {
	repo *MongoRepository[models.LoanProduct]
}

func NewLoansProductsRepository() *LoansProductsRepository {
	collection := db.MDB.Database.Collection(consts.LoansProductsCollection)
	mrepo := NewMongoRepository[models.LoanProduct](collection)
	return &LoansProductsRepository{repo: mrepo}
}

func (r *LoansProductsRepository) LoanProductByFilter(filter interface{}) (*models.LoanProduct, error) {
	loanProductResult, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}
	return &loanProductResult, nil
}

func (r *LoansProductsRepository) ServicingPartnerByProductId(ctx context.Context, productId primitive.ObjectID) (string, string, error) {
	servicingPartnerPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "_id", Value: productId}}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.WalletTypes},
			{Key: "localField", Value: "walletId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "walletType"},
		}}},
		bson.D{{Key: "$unwind", Value: "$walletType"}},
		bson.D{{Key: "$match", Value: bson.D{{Key: "walletType.isDeleted", Value: bson.M{"$ne": true}}}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "name", Value: "$walletType.name"},
			{Key: "number", Value: "$walletType.number"},
			{Key: "isDeleted", Value: "$walletType.isDeleted"},
		}}},
	}

	var servicingPartnerResult struct {
		ServicingPartner          string `bson:"name"`
		ServicingPartnerNumber    int32  `bson:"number"`
		ServicingPartnerIsDeleted bool   `bson:"isDeleted"`
	}

	if err := r.repo.Aggregate(servicingPartnerPipeline, &servicingPartnerResult); err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn(ctx, "No Servicing Partner found")
		} else {
			logger.Error(ctx, err)
		}
		return "", "", err
	}

	if servicingPartnerResult.ServicingPartnerIsDeleted {
		logger.Warn(ctx, "Servicing partner is marked as deleted")
		return "", "", mongo.ErrNoDocuments
	}

	return servicingPartnerResult.ServicingPartner, fmt.Sprintf("%d", servicingPartnerResult.ServicingPartnerNumber), nil
}
