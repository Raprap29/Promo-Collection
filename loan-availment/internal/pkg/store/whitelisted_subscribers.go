package store

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type WhitelistedSubsRepository struct {
	repo *MongoRepository[models.WhitelistedSubs]
}

func NewWhitelistedSubsRepository() *WhitelistedSubsRepository {
	collection := db.MDB.Database.Collection(consts.WhitelistedSubsCollection)
	mrepo := NewMongoRepository[models.WhitelistedSubs](collection)
	return &WhitelistedSubsRepository{repo: mrepo}
}

func (r *WhitelistedSubsRepository) IsWhitelisted(msisdn string, productId primitive.ObjectID,) (bool, error) {

	filter := bson.M{"MSISDN": msisdn, "productId": productId}
	count, err := r.repo.CountDocuments(filter)
	if err != nil {
		return false, fmt.Errorf("error counting documents: %w", err)
	}

	return count > 0, nil
}
