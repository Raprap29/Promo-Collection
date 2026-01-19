package interfaces

import (
	"context"
)

// CollectionTransactionsInProgressRepoInterface defines the interface for collection
// transactions in progress repository operations
type CollectionTransactionsInProgressRepoInterface interface {
	CheckEntryExists(ctx context.Context, msisdn string) (bool, error)
	CreateEntry(ctx context.Context, msisdn string) error
	DeleteEntry(ctx context.Context, msisdn string) error
}
