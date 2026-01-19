package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/mongo/options"
)

type SystemLevelRulesRepository interface {
	FetchSystemLevelRulesConfiguration(ctx context.Context) (models.SystemLevelRules, error)
}

type SystemLevelRulesStore interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.SystemLevelRules, error)
}
