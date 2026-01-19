package publisher

import (
	"context"
	"log/slog"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/service/interfaces"
)

type PubSubPublisherService struct {
	publisher interfaces.PubSubPublisherInterface
}

func NewPubSubPublisherService(publisher interfaces.PubSubPublisherInterface) *PubSubPublisherService {
	return &PubSubPublisherService{publisher: publisher}
}

func (p *PubSubPublisherService) PubSubPublisher(ctx context.Context,
	msg any) (string, error) {
	return PubSubPublisher(ctx, p.publisher, msg)
}

func PubSubPublisher(ctx context.Context,
	publisher interfaces.PubSubPublisherInterface,
	msg any) (string, error) {
	pubSubMsgId, err := publisher.PublishMessage(ctx, msg)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorInMessagePublishing, err)
		return "", err
	}
	logger.CtxInfo(ctx, log_messages.SuccessPubSubPublisher, 
		slog.String("PubSubMsgId: ", pubSubMsgId),
		slog.Any("PubSubMsg", msg),
	)

	return pubSubMsgId, nil
}
