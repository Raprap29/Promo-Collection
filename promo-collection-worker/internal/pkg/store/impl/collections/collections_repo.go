package collections

import (
	"context"
	"log/slog"
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"
	"promo-collection-worker/internal/service/interfaces"
	"time"

	mongodb "promo-collection-worker/internal/pkg/db/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionsRepository implements the CollectionsRepoInterface

type CollectionsRepository struct {
	repo       *repository.MongoRepository[models.Collections]
	create     func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	updateOne  func(ctx context.Context, filter interface{}, update interface{}) error
	updateMany func(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error)
	find       func(ctx context.Context, filter interface{}) ([]models.Collections, error)
}

// Ensure CollectionsRepository implements the CollectionsRepoInterface
var _ interfaces.CollectionsRepoInterface = (*CollectionsRepository)(nil)

func NewCollectionsRepository(
	client *mongodb.MongoClient) interfaces.CollectionsRepoInterface {
	collection := client.Database.Collection(consts.CollectionsCollection)
	repo := repository.NewMongoRepository[models.Collections](collection)
	r := &CollectionsRepository{
		repo: repo,
	}
	r.create = repo.Create
	r.updateOne = repo.UpdateOne
	r.updateMany = repo.UpdateMany
	r.find = repo.Find
	return r
}

func (r *CollectionsRepository) CreateEntry(ctx context.Context, model *models.Collections) (primitive.ObjectID,
	error) {
	result, err := r.create(ctx, model)
	if err != nil {
		logger.CtxError(ctx, "Failed to create collection entry", err, slog.String("msisdn", model.MSISDN))
		return primitive.ObjectID{}, err
	}
	logger.CtxInfo(ctx, "Created collection entry", slog.String("msisdn", model.MSISDN))
	return result.InsertedID.(primitive.ObjectID), nil
}

func (r *CollectionsRepository) UpdatePublishToKafka(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{"publishedToKafka": true}
	err := r.updateOne(ctx, filter, update)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorUpdatingCollectionsMongoDbDocument, err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessUpdatedCollectionsDocument, slog.Any("CollectionsId", id))

	return nil
}

func (r *CollectionsRepository) GetFailedKafkaEntriesCursor(ctx context.Context,
	duration string, batchSize int32) (*mongo.Cursor, error) {

	thresholdDate, err := time.Parse(consts.DateFormat, duration)
	if err != nil {
		logger.CtxError(ctx, log_messages.InvalidDurationFormat, err)
		return nil, err
	}

	collection := r.repo.GetCollection()

	pipeline := getFailedKafkaEntriesAggregationPipeline(thresholdDate)

	// Set options for batch processing
	opts := options.Aggregate().SetBatchSize(batchSize)

	cursor, err := collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		logger.CtxError(ctx, log_messages.FailedToGetFailedKafkaEntries, err)
		return nil, err
	}

	return cursor, nil
}

func getFailedKafkaEntriesAggregationPipeline(thresholdDate time.Time) mongo.Pipeline {
	return mongo.Pipeline{
		{
			{Key: "$match", Value: bson.D{
				{Key: "publishedToKafka", Value: false},
				{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: thresholdDate}}},
			}},
		},
		{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: consts.LoanCollection},
				{Key: "localField", Value: "loanId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "loanDetails"},
			}},
		},
		{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: consts.ClosedLoansCollection},
				{Key: "localField", Value: "loanId"},
				{Key: "foreignField", Value: "loanId"},
				{Key: "as", Value: "closedLoanDetails"},
			}},
		},
		{
			{Key: "$addFields", Value: bson.D{
				{Key: "loanGUID", Value: bson.D{
					{Key: "$ifNull", Value: []interface{}{
						bson.D{{Key: "$arrayElemAt", Value: []interface{}{"$loanDetails.GUID", 0}}},
						bson.D{{Key: "$arrayElemAt", Value: []interface{}{"$closedLoanDetails.GUID", 0}}},
					}},
				}},
			}},
		},
		{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 1},
				{Key: "MSISDN", Value: 1},
				{Key: "ageing", Value: 1},
				{Key: "collectedAmount", Value: 1},
				{Key: "collectionCategory", Value: 1},
				{Key: "collectionType", Value: 1},
				{Key: "createdAt", Value: 1},
				{Key: "errorText", Value: 1},
				{Key: "method", Value: 1},
				{Key: "paymentChannel", Value: 1},
				{Key: "result", Value: 1},
				{Key: "serviceFee", Value: 1},
				{Key: "tokenPaymentId", Value: 1},
				{Key: "totalCollectedAmount", Value: 1},
				{Key: "totalUnpaid", Value: 1},
				{Key: "unpaidServiceFee", Value: 1},
				{Key: "dataCollected", Value: 1},
				{Key: "transactionId", Value: 1},
				{Key: "kafkaTransactionId", Value: 1},
				{Key: "collectedServiceFee", Value: 1},
				{Key: "unpaidLoanAmount", Value: 1},
				{Key: "loanGUID", Value: 1},
			}},
		},
	}
}

func (r *CollectionsRepository) UpdatePublishedToKafkaInBulk(ctx context.Context,
	collectionIds []string) ([]string, error) {

	objectIDs := make([]primitive.ObjectID, len(collectionIds))
	for i, id := range collectionIds {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logger.CtxError(ctx, log_messages.InvalidObjectID, err)
			return nil, err
		}
		objectIDs[i] = objectID
	}

	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	update := bson.M{"$set": bson.M{"publishedToKafka": true}}

	updateResult, err := r.updateMany(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	failedUpdateIDs := []string{}
	if updateResult.MatchedCount != updateResult.ModifiedCount {
		filterFailed := bson.M{
			"_id":              bson.M{"$in": objectIDs},
			"publishedToKafka": bson.M{"$ne": true},
		}
		failedUpdate, err := r.find(ctx, filterFailed)
		if err != nil {
			return nil, err
		}
		for i := range failedUpdate {
			failedUpdateIDs = append(failedUpdateIDs, failedUpdate[i].ID.Hex())
		}
	}
	if len(failedUpdateIDs) > 0 {
		logger.CtxInfo(ctx, log_messages.FailedToUpdateKafkaFlag, slog.Any("failedUpdateIDs", failedUpdateIDs))
	}
	return failedUpdateIDs, nil
}
