package messages

import (
	"context"
	"errors"
	"log/slog"
	"promocollection/internal/pkg/consts"
	mongodb "promocollection/internal/pkg/db/mongo"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/store/models"
	"promocollection/internal/pkg/store/repository"
	"promocollection/internal/service/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessagesRepository struct {
	repo interfaces.MessagesStoreInterface
}

func NewMessagesRepository(client *mongodb.MongoClient) *MessagesRepository {
	collection := client.Database.Collection(consts.Messages)
	repo := repository.NewMongoRepository[models.Messages](collection)
	return &MessagesRepository{repo: repo}
}

func NewMessagesRepositoryWithInterface(repo interfaces.MessagesStoreInterface) *MessagesRepository {
	return &MessagesRepository{repo: repo}
}

func (mr *MessagesRepository) GetPatternIdByEventandBrandId(ctx context.Context,
	event string,
	brandId primitive.ObjectID,
) (*models.Messages, error) {
	filter := bson.M{"event": event, "brandId": brandId}
	messages, err := mr.repo.FindOne(ctx, filter, options.FindOne())
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxWarn(ctx, "No patternId found for event and brandId", slog.String("event", event))
			return nil, err
		}
		logger.CtxError(ctx, "Error finding patternId by event and brandId", err, slog.String("event", event))
		return nil, err
	}

	logger.CtxDebug(ctx,
		"Fetched patternId by event and brandId",
		slog.String("event", event),
		slog.Any("pattern_id", messages.PatternId),
	)
	return &messages, nil
}
