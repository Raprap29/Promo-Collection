package models

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
