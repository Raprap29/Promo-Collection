package models

import "time"

type EducationPeriodNotificationPubSubMsgFormat struct {
	RemainingLoanAmount float64   `json:"REMAINING_LOAN_AMOUNT"`
	SkuName             string    `json:"SKU_NAME"`
	ServiceFee          float64   `json:"SERVICE_FEE"`
	LoanDate            time.Time `json:"LOAN_DATE"`
	DataCollectionDate  time.Time `json:"DATA_COLLECTION_DATE"`
	DataMb              float64   `json:"DATA_MB"`
	DataConversion      float64   `json:"DATA_CONVERSION"`
	PatternId           int32     `json:"NOTIFICATION_EVENT"`
}

type NotificationParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PubSubNotificationMessage struct {
	Msisdn          string                  `json:"msisdn"`
	SmsDbEventName  string                  `json:"sms_db_event_name"`
	NotifParameters []NotificationParameter `json:"notif_parameters"`
	PatternID       int32                   `json:"notification_pattern_id"`
}
