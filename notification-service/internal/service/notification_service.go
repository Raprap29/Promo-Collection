package service

import (
	"context"
	"encoding/json"
	"fmt"
	"notificationservice/internal/pkg/config"
	"notificationservice/internal/pkg/downstream"
	"notificationservice/internal/pkg/logger"
	"notificationservice/internal/pkg/models"
)

// NotificationService handles Pub/Sub notifications
type NotificationService struct {
	cfg *config.AppConfig
}

func NewNotificationService(cfg *config.AppConfig) *NotificationService {
	return &NotificationService{
		cfg: cfg,
	}
}

func (s *NotificationService) HandleMessage(ctx context.Context, msg []byte) error {
	logger.CtxInfo(ctx, fmt.Sprintf("Received Message from Pubsub: %s",msg))

	// 1. Parse Pub/Sub message
	var notification models.PubSubNotificationMessage
	if err := json.Unmarshal(msg, &notification); err != nil {
		logger.CtxError(ctx, "Failed to unmarshal message", err)
		return nil
	}

	// 2. Convert to SoapRequestNotifParameter
	soapParams := make([]models.SoapRequestNotifParameter, len(notification.NotifParameters))
	for i, p := range notification.NotifParameters {
		soapParams[i] = models.SoapRequestNotifParameter(p)
	}

	// 3. Create SOAP payload
	soapPayload, err := downstream.CreateSOAPRequest(
		fmt.Sprint(notification.PatternID),
		notification.Msisdn,
		soapParams,
	)
	if err != nil {
		logger.CtxError(ctx, "Failed to create SOAP request", err)
		return nil
	}

	logger.CtxInfo(ctx, fmt.Sprintf("SOAP Request Payload for MSISDN: %s, event: %s\n%s", 
		notification.Msisdn,  notification.SmsDbEventName, soapPayload))

	// 4. Send SOAP request
	url := s.cfg.SMSNotificationConfig.SEND_SMS_NOTIFICATION_URL
	apiKey := s.cfg.SMSNotificationConfig.SEND_SMS_NOTIFICATION_X_API_KEY
	resp, err := downstream.SendSOAPRequest(url, apiKey, soapPayload)
	if err != nil {
		logger.CtxError(ctx, "SOAP request failed", err)
		return err
	}

	// 5. Log SOAP response
	respBytes, _ := json.Marshal(resp)
	logger.CtxInfo(ctx, fmt.Sprintf("SOAP Response for MSISDN: %s, event: %s\n%s", 
		notification.Msisdn, notification.SmsDbEventName, string(respBytes)))

	logger.CtxInfo(ctx, fmt.Sprintf(
		"Notification sent successfully for MSISDN: %s, event: %s", 
			notification.Msisdn, notification.SmsDbEventName))

	return nil
}
