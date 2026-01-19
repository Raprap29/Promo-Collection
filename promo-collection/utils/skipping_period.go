package utils

import "time"

func IsSkipPeriodActive(
	educationPeriodTimestamp,
	loanLoadPeriodTimestamp,
	gracePeriodTimestamp time.Time,
) bool {
	now := time.Now()

	return !educationPeriodTimestamp.Before(now) ||
		!loanLoadPeriodTimestamp.Before(now) ||
		!gracePeriodTimestamp.Before(now)
}
