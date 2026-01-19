package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CollectionsRepoInterface defines the interface for collections repository operations
type CollectionsRepoInterface interface {
	CreateEntry(ctx context.Context, model *models.Collections) (primitive.ObjectID, error)
	UpdatePublishToKafka(ctx context.Context, id primitive.ObjectID) error
	UpdatePublishedToKafkaInBulk(ctx context.Context, collectionIds []string) ([]string, error)
	GetFailedKafkaEntriesCursor(ctx context.Context, duration string, batchSize int32) (*mongo.Cursor, error)
}
