// impl/system_level_rules.go
package system_level_rules

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"
	"promo-collection-worker/internal/service/interfaces"
)

type SystemLevelRulesRepository struct {
	repo interfaces.SystemLevelRulesStore
}

func NewSystemLevelRulesRepository(client *mongodb.MongoClient) *SystemLevelRulesRepository {
	collection := client.Database.Collection(consts.SystemLevelRulesCollection)
	repo := repository.NewMongoRepository[models.SystemLevelRules](collection)
	return &SystemLevelRulesRepository{repo: repo}
}

func NewSystemLevelRulesRepositoryWithInterface(repo interfaces.SystemLevelRulesStore) *SystemLevelRulesRepository {
	return &SystemLevelRulesRepository{repo: repo}
}

func (slr *SystemLevelRulesRepository) FetchSystemLevelRulesConfiguration(
	ctx context.Context) (models.SystemLevelRules, error) {
	filter := bson.M{}
	opts := options.FindOne()

	rules, err := slr.repo.FindOne(ctx, filter, opts)
	if err != nil {
		logger.CtxError(ctx, "Failed to fetch system level rules", err)
		return models.SystemLevelRules{}, err // *** return zero value on error ***
	}

	logger.CtxDebug(ctx, "Successfully fetched system level rules")
	return rules, nil
}
