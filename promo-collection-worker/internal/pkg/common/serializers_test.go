package common

import (
	"testing"
	"time"

	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSerializeCollections_PartialSuccess(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:                   "639171000000",
		CollectionType:           consts.PartialCollectionType,
		AmountToBeDeductedInPeso: 20,
		TotalLoanAmountInPeso:    100,
		TotalUnpaidAmountInPeso:  50,
		UnpaidServiceFee:         1,
		ServiceFee:               5,
		Channel:                  "SMS",
		LoanId:                   primitive.NewObjectID(),
		UnpaidLoanId:             primitive.NewObjectID(),
	}

	coll := SerializeCollections(msg, true, "", "")
	if coll.MSISDN != msg.Msisdn {
		t.Fatalf("MSISDN mismatch")
	}
	if coll.CollectedAmount != msg.AmountToBeDeductedInPeso {
		t.Fatalf("CollectedAmount mismatch")
	}
	if !coll.Result {
		t.Fatalf("expected result true")
	}
}

func TestSerializeCollections_FullSuccess(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:                   "639171111111",
		CollectionType:           consts.FullCollectionType,
		AmountToBeDeductedInPeso: 100,
		TotalLoanAmountInPeso:    100,
		TotalUnpaidAmountInPeso:  100,
		ServiceFee:               10,
		Channel:                  "APP",
		LoanId:                   primitive.NewObjectID(),
		UnpaidLoanId:             primitive.NewObjectID(),
	}

	coll := SerializeCollections(msg, true, "", "")
	if coll.CollectionType != consts.FullCollectionType {
		t.Fatalf("CollectionType mismatch")
	}
	if coll.TotalCollectedAmount != msg.TotalLoanAmountInPeso {
		t.Fatalf("TotalCollectedAmount mismatch")
	}
}

func TestSerializeCollections_Failure(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:                  "639172222222",
		CollectionType:          consts.PartialCollectionType,
		TotalLoanAmountInPeso:   200,
		TotalUnpaidAmountInPeso: 150,
		UnpaidServiceFee:        3,
	}
	coll := SerializeCollections(msg, false, "err", "E1")
	if coll.Result {
		t.Fatalf("expected result false")
	}
	if coll.ErrorText != "err" {
		t.Fatalf("error text mismatch")
	}
}

func TestSerializeKafkaMessage(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{
		DataCollectionRequestTraceId: "tx-1",
		Msisdn:                       "639173333333",
		AmountToBeDeductedInPeso:     30,
		TotalLoanAmountInPeso:        100,
		TotalUnpaidAmountInPeso:      70,
		ServiceFee:                   4,
		UnpaidServiceFee:             1,
		SvcId:                        123,
		Denom:                        "P",
	}

	km := SerializeKafkaMessage(msg, true, "", time.Now())
	if km.TransactionId != msg.DataCollectionRequestTraceId {
		t.Fatalf("transaction id mismatch")
	}
	if km.CollectedAmount != msg.AmountToBeDeductedInPeso {
		t.Fatalf("collected amount mismatch")
	}
}

func TestSerializeKafkaMessage_FailureBranch(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{
		DataCollectionRequestTraceId: "tx1",
		Msisdn:                       "639171234999",
		AvailmentTransactionId:       primitive.NewObjectID(),
		CollectionType:               "partial",
		Channel:                      "SMS",
		SvcId:                        123,
		Denom:                        "PHP",
		TotalLoanAmountInPeso:        100,
		TotalUnpaidAmountInPeso:      50,
		ServiceFee:                   5,
		UnpaidServiceFee:             0,
	}
	km := SerializeKafkaMessage(msg, false, "err", time.Now())
	if km.Result != "Fail" {
		t.Fatalf("expected result Fail, got %s", km.Result)
	}
	if km.DataCollected != 0 {
		t.Fatalf("expected DataCollected 0, got %v", km.DataCollected)
	}
}
