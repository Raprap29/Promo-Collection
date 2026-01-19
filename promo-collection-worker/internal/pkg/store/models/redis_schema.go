package models

import (
	"fmt"
	"time"
)

const (
	SkipTimestampsKeyPattern = "skipTimestamps:%s" // skipTimestamps:MSISDN
)

func SkipTimestampsKeyBuilder(msisdn string) string {
	return fmt.Sprintf(SkipTimestampsKeyPattern, msisdn)
}

type SkipTimestamps struct {
	GracePeriodTimestamp          time.Time `json:"gracePeriodTimestamp"`
	PromoEducationPeriodTimestamp time.Time `json:"promoEducationPeriodTimestamp"`
	LoanLoadPeriodTimestamp       time.Time `json:"loanLoadPeriodTimestamp"`
}
