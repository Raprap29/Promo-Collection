package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MessagesRepository struct {
	repo *MongoRepository[models.Messages]
}

func NewMessagesRepository() *MessagesRepository {
	collection := db.MDB.Database.Collection(consts.Messages)
	mrepo := NewMongoRepository[models.Messages](collection)

	return &MessagesRepository{repo: mrepo}
}

func (r *MessagesRepository) MessageByFilter(filter interface{}) ([]models.Messages, error) {

	messageResult, err := r.repo.FindAll(filter)
	if err != nil {
		return nil, err
	}
	return messageResult, nil
}


func (r *MessagesRepository) GetMessageID(ctx context.Context, event string, brandID primitive.ObjectID, promoCollectionEnabled bool) (*models.MessageResponse, error) {
	if event == "" {
		return nil, consts.ErrorMissingRequiredInputs
	}

	// Initialize the filter with the event
	messageFilter := bson.M{"event": event, "isDeleted": bson.M{"$ne": true}}

	// Conditionally add brandID if it's not a NilObjectID
	if brandID != primitive.NilObjectID {
		messageFilter["brandId"] = brandID
	}

	// Get all matching messages
	messages, err := r.MessageByFilter(messageFilter)
	if err != nil {
		logger.Error(ctx, "Error while fetching messages for event %v and brand %v: %v", event, brandID, err)
		return nil, err
	}

	if len(messages) == 0 {
		logger.Warn(ctx, "No documents for event %v and brand %v found", event, brandID)
		return nil, mongo.ErrNoDocuments
	}

	// Select the appropriate message based on promoCollectionEnabled flag
	var selectedMessage models.Messages

	if len(messages) == 1 {
		// If only one message, use it regardless of labels
		selectedMessage = messages[0]
	} else {
		// Multiple messages found, filter by labels based on promoCollectionEnabled
		found := false

		for _, msg := range messages {
			// Check if message has DC_ON label
			hasDCONLabel := false
			if msg.Labels != nil {
				for _, label := range msg.Labels {
					if label == "DC_ON" {
						hasDCONLabel = true
						break
					}
				}
			}

			// Select message based on promoCollectionEnabled and label presence
			if (promoCollectionEnabled && hasDCONLabel) || (!promoCollectionEnabled && !hasDCONLabel) {
				selectedMessage = msg
				found = true
				break
			}
		}

		// If no matching message found based on labels, use the first one as fallback
		if !found && len(messages) > 0 {
			selectedMessage = messages[0]
		}
	}

	logger.Info(ctx, "Selected Message: %v", selectedMessage)
	if selectedMessage.IsDeleted {
		logger.Info(ctx, "Document is deleted")
		return nil, consts.ErrorNoDocumentFound
	}
	// Create and return the response with either the found parameters or nil
	response := models.MessageResponse{
		MessageID:  selectedMessage.PatternId,
		Parameters: selectedMessage.Parameters,
	}

	return &response, nil
}
