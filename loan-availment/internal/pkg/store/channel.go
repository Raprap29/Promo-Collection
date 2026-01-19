package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

)

type ChannelRepository struct {
	repo *MongoRepository[models.Channels]
}

func NewChannelsRepository() *ChannelRepository {
	collection := db.MDB.Database.Collection(consts.ChannelsCollection)
	mrepo := NewMongoRepository[models.Channels](collection)
	return &ChannelRepository{repo: mrepo}
}

func (r *ChannelRepository) ChannelsByFilter(ctx context.Context,filter interface{}) (*models.Channels, error) {

	logger.Debug(ctx,filter)
	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil

}
