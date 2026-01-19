package common

import (
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/models"
	dbModels "promo-collection-worker/internal/pkg/store/models"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SerializeCollections(
	msg *models.PromoCollectionPublishedMessage, isSuccess bool, errorDetails string,
	errorCode string) *dbModels.Collections {

	collection := dbModels.Collections{
		ID:                     primitive.NewObjectID(),
		MSISDN:                 msg.Msisdn,
		Ageing:                 msg.Ageing,
		AvailmentTransactionId: msg.AvailmentTransactionId,
		CollectionCategory:     consts.CollectionCategory,
		CollectionType:         msg.CollectionType,
		LoanId:                 msg.LoanId,
		Method:                 consts.Method,
		PaymentChannel:         msg.Channel,
		PublishedToKafka:       false,
		ServiceFee:             msg.ServiceFee,
		TokenPaymentId:         strconv.FormatInt(msg.SvcId, 10) + "-" + msg.Denom,
		CreatedAt:              time.Now(),
		TransactionId:          msg.DataCollectionRequestTraceId,
	}

	if isSuccess {
		collection.CollectedAmount = msg.AmountToBeDeductedInPeso
		collection.Result = true
		collection.ErrorText = ""
		collection.ErrorCode = ""
		collection.UnpaidLoanId = msg.UnpaidLoanId
		collection.DataCollected = msg.DataToBeDeducted
		switch msg.CollectionType {
		case consts.PartialCollectionType:
			collection.TotalCollectedAmount = msg.AmountToBeDeductedInPeso + msg.TotalLoanAmountInPeso
			collection.TotalUnpaid = msg.TotalUnpaidAmountInPeso - msg.AmountToBeDeductedInPeso
			collection.UnpaidServiceFee = calculateUnpaidServiceFee(msg.AmountToBeDeductedInPeso, msg.UnpaidServiceFee)
			collection.CollectedServiceFee = msg.ServiceFee - collection.UnpaidServiceFee
			collection.UnpaidLoanAmount = collection.TotalUnpaid - (msg.ServiceFee - collection.CollectedServiceFee)
		case consts.FullCollectionType:
			collection.TotalCollectedAmount = msg.TotalLoanAmountInPeso
			collection.TotalUnpaid = 0
			collection.UnpaidServiceFee = 0
			collection.CollectedServiceFee = msg.ServiceFee
			collection.UnpaidLoanAmount = 0
		}
	} else {
		collection.CollectedAmount = 0
		collection.Result = false
		collection.ErrorText = errorDetails
		collection.ErrorCode = errorCode
		collection.TotalCollectedAmount = msg.TotalLoanAmountInPeso - msg.TotalUnpaidAmountInPeso
		collection.TotalUnpaid = msg.TotalUnpaidAmountInPeso
		collection.UnpaidServiceFee = msg.UnpaidServiceFee
		collection.UnpaidLoanId = msg.UnpaidLoanId
		collection.DataCollected = 0
		collection.CollectedServiceFee = 0
		collection.UnpaidLoanAmount = msg.TotalUnpaidAmountInPeso - msg.UnpaidServiceFee
	}

	return &collection

}

func SerializeKafkaMessage(msg *models.PromoCollectionPublishedMessage,
	isSuccess bool,
	errorDetails string,
	collectionDateTime time.Time) *models.KafkaMessageForPublishing {

	kafkaMessage := models.KafkaMessageForPublishing{
		TransactionId:      msg.DataCollectionRequestTraceId,
		MSISDN:             msg.Msisdn,
		LoanAgeing:         msg.Ageing,
		AvailmentId:        msg.GUID,
		CollectionCategory: msg.CollectionType,
		CollectionType:     consts.CollectionCategory,
		Method:             consts.Method,
		PaymentChannel:     msg.Channel,
		TokenPaymentId:     strconv.FormatInt(msg.SvcId, 10) + "-" + msg.Denom,
		CollectionDateTime: collectionDateTime,
	}

	if isSuccess {
		kafkaMessage.CollectedAmount = msg.AmountToBeDeductedInPeso
		kafkaMessage.Result = "Success"
		kafkaMessage.CollectionErrorText = ""
		kafkaMessage.CollectedLoanAmount = (msg.TotalLoanAmountInPeso -
			msg.TotalUnpaidAmountInPeso) + msg.AmountToBeDeductedInPeso
		var newUnpaidServiceFee float64
		if msg.AmountToBeDeductedInPeso >= msg.UnpaidServiceFee {
			newUnpaidServiceFee = 0
		} else {
			newUnpaidServiceFee = msg.UnpaidServiceFee - msg.AmountToBeDeductedInPeso
		}

		kafkaMessage.CollectedServiceFee = msg.ServiceFee - newUnpaidServiceFee
		kafkaMessage.TotalUnpaid = msg.TotalLoanAmountInPeso - kafkaMessage.CollectedLoanAmount
		kafkaMessage.UnpaidLoanAmount = msg.TotalLoanAmountInPeso - kafkaMessage.CollectedLoanAmount
		kafkaMessage.UnpaidServiceFee = newUnpaidServiceFee
		kafkaMessage.DataCollected = msg.DataToBeDeducted
	} else {
		kafkaMessage.CollectedAmount = 0
		kafkaMessage.Result = "Fail"
		kafkaMessage.CollectionErrorText = errorDetails
		kafkaMessage.CollectedLoanAmount = msg.TotalLoanAmountInPeso - msg.TotalUnpaidAmountInPeso
		kafkaMessage.CollectedServiceFee = msg.ServiceFee - msg.UnpaidServiceFee
		kafkaMessage.TotalUnpaid = msg.TotalUnpaidAmountInPeso
		kafkaMessage.UnpaidLoanAmount = msg.TotalUnpaidAmountInPeso
		kafkaMessage.UnpaidServiceFee = msg.UnpaidServiceFee
		kafkaMessage.DataCollected = 0
	}

	return &kafkaMessage
}

func calculateUnpaidServiceFee(amountToBeDeductedInPeso, unpaidServiceFee float64) float64 {
	if amountToBeDeductedInPeso >= unpaidServiceFee {
		return 0
	}
	return unpaidServiceFee - amountToBeDeductedInPeso
}
