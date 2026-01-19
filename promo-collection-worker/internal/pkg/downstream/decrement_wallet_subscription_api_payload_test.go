package downstream

import (
	"promo-collection-worker/internal/pkg/models"
	"strings"
	"testing"
)

func TestPromoCollectionPublishedMessageString(t *testing.T) {
	msg := models.PromoCollectionPublishedMessage{
		Msisdn:        "639171234567",
		WalletKeyword: "WAL",
		WalletAmount:  "10",
		Unit:          "MB",
		IsRollBack:    false,
		Duration:      "200",
		Channel:       "SMS",
	}

	s := msg.String()

	if !strings.Contains(s, msg.Msisdn) {
		t.Fatalf("expected string to contain msisdn %s, got %s", msg.Msisdn, s)
	}
	if !strings.Contains(s, msg.Channel) {
		t.Fatalf("expected string to contain channel %s, got %s", msg.Channel, s)
	}
}
