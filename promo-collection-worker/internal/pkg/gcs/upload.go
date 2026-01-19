package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"

	"cloud.google.com/go/storage"
)

type GCSClient struct {
	Client     *storage.Client
	BucketName string
	FolderName string
}

type GcsInterface interface {
	Upload(ctx context.Context, msg *models.PromoCollectionPublishedMessage) error
	Close(ctx context.Context)
}

func NewGCSClient(ctx context.Context, bucketName string) (GcsInterface, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSClient{
		Client:     client,
		BucketName: bucketName,
		FolderName: consts.GCSFolderName,
	}, nil
}

func (g *GCSClient) Close(ctx context.Context) {
	if g.Client == nil {
		return
	}
	if err := g.Client.Close(); err != nil {
		logger.CtxError(ctx, log_messages.ErrorClosingGCSClient, err)
	}
}

func (g *GCSClient) Upload(ctx context.Context, msg *models.PromoCollectionPublishedMessage) error {
	objectName := fmt.Sprintf("%s/%d_%s.json", g.FolderName, msg.PublishTime.Unix(), msg.Msisdn)
	object := g.Client.Bucket(g.BucketName).Object(objectName)
	jsonData, err := json.Marshal(msg)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorMarshallingJSON, err)
		return err
	}
	writer := object.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	writer.ContentType = "application/json"
	_, err = writer.Write(jsonData)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorUploadingToGCSBucket, err)
		return err
	}
	if err := writer.Close(); err != nil {
		logger.CtxError(ctx, log_messages.ErrorClosingGCSWriter, err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.UploadedToGCSBucket, slog.String("objectName", objectName))
	return nil
}
