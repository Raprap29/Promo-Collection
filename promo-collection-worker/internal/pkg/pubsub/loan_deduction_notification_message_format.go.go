package pubsub

type LoanDeductionNotificationFormat struct {
	MSISDN          string                     `json:"msisdn"`
	SmsDbEventName  string                     `json:"sms_db_event_name"`
	NotifParameters []SmsNotificationParameter `json:"notif_parameters"`
	PatternId       int32                      `json:"notification_pattern_id"`
}

type SmsNotificationParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
