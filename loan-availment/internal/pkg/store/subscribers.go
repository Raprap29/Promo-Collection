package store

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SubscribersRepository struct {
	repo *MongoRepository[models.Subscribers]
}

func NewSubscribersRepository() *SubscribersRepository {
	collection := db.MDB.Database.Collection(consts.SubscribersCollection)
	mrepo := NewMongoRepository[models.Subscribers](collection)
	return &SubscribersRepository{repo: mrepo}
}

// IsMsisdnBlacklisted
func (r *SubscribersRepository) SubscribersByFilter(filter interface{}) (*models.Subscribers, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

//Add new subscriber or update new subscriber

func (r *SubscribersRepository) AddUpdateSubscriber(MSISDN string, subscriber *models.Subscribers) error {

	filter := bson.M{"MSISDN": MSISDN}

	var existingSubscriber models.Subscribers
	existingSubscriber, err := r.repo.Read(filter)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			subscriber.ID = primitive.NewObjectID()
			subscriber.CreatedAt = time.Now()
			subscriber.UpdatedAt = nil
			_, err = r.repo.Create(subscriber)
			if err != nil {
				return fmt.Errorf("failed to insert new subscriber: %v", err)
			}
			return nil
		}
		return fmt.Errorf("failed to fetch existing subscriber: %v", err)
	}

	if !hasChanges(existingSubscriber, *subscriber) {

		return nil
	}

	update := bson.M{
		"subscriberType":               subscriber.SubscriberType,
		"customerType":                 subscriber.CustomerType,
		"creditLoanLimitAmount":        subscriber.CreditLoanLimitAmount,
		"creditLoanLimitAmountGenDate": subscriber.CreditLoanLimitAmountGenDate,
		"updatedAt":                    time.Now(),
	}
	logger.Info(context.Background(), "Update %v", update)

	err = r.repo.Update(filter, update)
	if err != nil {
		return fmt.Errorf("failed to update subscriber: %v", err)
	}

	return nil
}

func hasChanges(existing, new models.Subscribers) bool {
	return existing.SubscriberType != new.SubscriberType ||
		existing.CustomerType != new.CustomerType ||
		existing.CreditLoanLimitAmount != new.CreditLoanLimitAmount ||
		!datesEqual(existing.CreditLoanLimitAmountGenDate, new.CreditLoanLimitAmountGenDate)
}

func datesEqual(date1, date2 *time.Time) bool {
	if date1 == nil && date2 == nil {
		return true
	}
	if (date1 == nil && date2 != nil) || (date1 != nil && date2 == nil) {
		return false
	}
	return date1.Equal(*date2)
}
