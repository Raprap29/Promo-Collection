package store

import (
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type SystemLevelRulesRepository struct {
	repo *MongoRepository[models.SystemLevelRules]
}

func NewSystemLevelRulesRepository() *SystemLevelRulesRepository {
	collection := db.MDB.Database.Collection(consts.SystemLevelRulesCollection)
	mrepo := NewMongoRepository[models.SystemLevelRules](collection)
	return &SystemLevelRulesRepository{repo: mrepo}
}

func (r *SystemLevelRulesRepository) SystemLevelRules(filter interface{}) (models.SystemLevelRules, error) {
	systemLevelRulesResult, err := r.repo.Read(filter)
	if err != nil {
		return models.SystemLevelRules{}, err
	}
	return systemLevelRulesResult, nil
}
