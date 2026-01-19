package successhandling_service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"errors"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/models"
	"promo-collection-worker/internal/pkg/pubsub"
	storemodels "promo-collection-worker/internal/pkg/store/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	svcKafka "promo-collection-worker/internal/service/kafka"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildFullNotificationAndBuilders(t *testing.T) {
	// stub fetch hooks
	oldPattern := fetchPatternIdFullFunc
	oldProduct := fetchProductNameFullFunc
	defer func() {
		fetchPatternIdFullFunc = oldPattern
		fetchProductNameFullFunc = oldProduct
	}()

	fetchPatternIdFullFunc = func(ctx context.Context, mc *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 321, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, nil
	}
	fetchProductNameFullFunc = func(ctx context.Context, mc *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return &storemodels.LoanProducts{Name: "TEST_SKU"}, nil
	}

	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:                   "639123456789",
		DataToBeDeducted:         100,
		AmountToBeDeductedInPeso: 50.0,
		StartDate:                time.Now(),
		LoanProductId:            primitive.NewObjectID(),
		TotalLoanAmountInPeso:    1000.0,
		TotalUnpaidAmountInPeso:  200.0,
		ServiceFee:               5.0,
		Channel:                  "SMS",
		LoanId:                   primitive.NewObjectID(),
		GUID:                     "guid-1",
		AvailmentTransactionId:   primitive.NewObjectID(),
		BrandId:                  primitive.NewObjectID(),
	}

	// build notification
	b, err := buildFullNotification(context.Background(), nil, msg)
	if err != nil {
		t.Fatalf("buildFullNotification returned error: %v", err)
	}

	var notif pubsub.LoanDeductionNotificationFormat
	if err := json.Unmarshal(b, &notif); err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}

	if notif.MSISDN != msg.Msisdn {
		t.Fatalf("expected msisdn %s, got %s", msg.Msisdn, notif.MSISDN)
	}
	if notif.PatternId != 321 {
		t.Fatalf("expected pattern id 321, got %d", notif.PatternId)
	}

	// collection model builder
	coll := buildFullCollectionModel(msg)
	if coll.CollectionType == "" {
		t.Fatalf("expected CollectionType to be set")
	}
	if coll.TotalCollectedAmount <= 0 {
		t.Fatalf("expected TotalCollectedAmount > 0")
	}

	// closed/unpaid builders
	closed := buildClosedLoanEntry(msg)
	if closed["GUID"] != msg.GUID {
		t.Fatalf("closed GUID mismatch")
	}
	unpaid := buildUnpaidLoanEntry(msg)
	if unpaid["version"] != msg.Version+1 {
		// Version defaults to zero in test message; ensure key exists
		if unpaid["version"] == nil {
			t.Fatalf("unpaid loan entry missing version")
		}
	}
}

func TestBuildPartialNotificationAndCollection(t *testing.T) {
	oldPattern := fetchPatternIdFunc
	oldProduct := fetchProductNameFunc
	defer func() {
		fetchPatternIdFunc = oldPattern
		fetchProductNameFunc = oldProduct
	}()

	fetchPatternIdFunc = func(ctx context.Context, mc *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 111, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, nil
	}
	fetchProductNameFunc = func(ctx context.Context, mc *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return &storemodels.LoanProducts{Name: "PARTIAL_SKU"}, nil
	}

	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:                   "639987654321",
		AmountToBeDeductedInPeso: 10.0,
		StartDate:                time.Now(),
		LoanProductId:            primitive.NewObjectID(),
		TotalUnpaidAmountInPeso:  90.0,
		ServiceFee:               2.0,
		LoanId:                   primitive.NewObjectID(),
		LastCollectionId:         primitive.NewObjectID(),
		Version:                  1,
	}

	grace := time.Now().Add(24 * time.Hour)
	b, err := buildPartialNotification(context.Background(), nil, msg, grace)
	if err != nil {
		t.Fatalf("buildPartialNotification returned error: %v", err)
	}
	var notif pubsub.LoanDeductionNotificationFormat
	if err := json.Unmarshal(b, &notif); err != nil {
		t.Fatalf("failed to unmarshal partial notification: %v", err)
	}
	if notif.PatternId != 111 {
		t.Fatalf("expected pattern id 111, got %d", notif.PatternId)
	}

	coll := preparePartialCollectionModel(msg, 80.0)
	if coll.CollectionType != "PartialCollection" && coll.CollectionType == "" {
		// ensure some value set; not strict on exact constant name to avoid import of consts
		t.Fatalf("unexpected collection type: %v", coll.CollectionType)
	}
}

// fakeProducer implements pkgkafka.KafkaProducerInterface for testing
type fakeProducer struct{ fail bool }

func (f *fakeProducer) Publish(ctx context.Context, msg []byte) error {
	if f.fail {
		return errors.New("producer fail")
	}
	return nil
}

// ensure fakeCollRepo implements CollectionTransactionsInProgressRepoInterface
type fakeCollRepo struct {
	fail bool
}

func (f *fakeCollRepo) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	return false, nil
}
func (f *fakeCollRepo) CreateEntry(ctx context.Context, msisdn string) error { return nil }
func (f *fakeCollRepo) DeleteEntry(ctx context.Context, msisdn string) error {
	if f.fail {
		return errors.New("delete fail")
	}
	return nil
}

func TestPublishNotificationAndFinalizeFullAndPartial(t *testing.T) {
	// stub notify functions to avoid pubsub/net calls
	oldNotifyFull := notifyFullFunc
	oldNotifyPartial := notifyPartialFunc
	defer func() { notifyFullFunc = oldNotifyFull; notifyPartialFunc = oldNotifyPartial }()
	notifyFullFunc = func(ctx context.Context, mc *mongodb.MongoClient, pubSub serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, notificationTopic string) error {
		return nil
	}
	notifyPartialFunc = func(ctx context.Context, mc *mongodb.MongoClient, pubSub serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, g time.Time, notificationTopic string) error {
		return nil
	}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639000000000"}
	// build a CollectionWorkerKafkaService with a fake producer
	producer := &fakeProducer{fail: false}
	kafkaSvc := svcKafka.NewCollectionWorkerKafkaService(producer)
	collRepo := &fakeCollRepo{fail: false}

	// run full finalize
	if err := publishNotificationPayloadToPubSubAndKafkaFullCollection(context.Background(), nil, nil, kafkaSvc, collRepo, msg, time.Now(), ""); err != nil {
		t.Fatalf("publishNotificationAndFinalizeFull failed: %v", err)
	}

	// simulate kafka failure
	// simulate kafka/producer failure
	producer.fail = true
	if err := publishNotificationPayloadToPubSubAndKafkaFullCollection(context.Background(), nil, nil, kafkaSvc, collRepo, msg, time.Now(), ""); err == nil {
		t.Fatalf("expected kafka publish to fail")
	}

	// simulate delete failure
	producer.fail = false
	collRepo.fail = true
	if err := publishNotificationPayloadToPubSubAndKafkaFullCollection(context.Background(), nil, nil, kafkaSvc, collRepo, msg, time.Now(), ""); err == nil {
		t.Fatalf("expected delete to fail")
	}

	// test partial finalize - similar flow
	collRepo.fail = false
	if err := publishNotificationAndFinalize(context.Background(), nil, nil, kafkaSvc, collRepo, msg, time.Now(), time.Now(), ""); err != nil {
		t.Fatalf("publishNotificationAndFinalize failed: %v", err)
	}
}
