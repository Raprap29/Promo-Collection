package interfaces

import (
	"context"
)

type PubSubPublisherInterface interface {
	Close()
	PublishMessage(context.Context, any) (string, error)
}
