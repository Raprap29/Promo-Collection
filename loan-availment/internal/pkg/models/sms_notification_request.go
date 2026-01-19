package models

type NotificationParameter struct {
	Name  string
	Value string
}

type SmsNotificationRequestPayload struct {
	NotificationParameter []NotificationParameter `json:"notificationParameter"`
	PatternID                 int32                       `json:"patternId"`
}

// Protobuf-compatible models for PubSub messages
type SmsNotificationParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SmsNotificationRequest struct {
	Msisdn          string                     `json:"msisdn"`
	SmsDbEventName  string                     `json:"sms_db_event_name"`
	NotifParameters []SmsNotificationParameter `json:"notif_parameters"`
	PatternID       int32                      `json:"notification_pattern_id"`
}
