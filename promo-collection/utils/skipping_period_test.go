package utils

import (
	"testing"
	"time"
)

func TestIsSkipPeriodActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		edu        time.Time
		loan       time.Time
		wantValid  bool
		wantBigger string
	}{
		{
			name:      "both before now - invalid",
			edu:       now.Add(-2 * time.Hour),
			loan:      now.Add(-1 * time.Hour),
			wantValid: false,
		},
		{
			name:      "education after now",
			edu:       now.Add(1 * time.Hour),
			loan:      now.Add(-1 * time.Hour),
			wantValid: true,
		},
		{
			name:      "loanLoad after now",
			edu:       now.Add(-1 * time.Hour),
			loan:      now.Add(1 * time.Hour),
			wantValid: true,
		},
		{
			name:      "education greater than loanLoad",
			edu:       now.Add(2 * time.Hour),
			loan:      now.Add(1 * time.Hour),
			wantValid: true,
		},
		{
			name:      "loanLoad greater than education",
			edu:       now.Add(1 * time.Hour),
			loan:      now.Add(2 * time.Hour),
			wantValid: true,
		},
		{
			name:      "both equal and after now",
			edu:       now.Add(1 * time.Hour),
			loan:      now.Add(1 * time.Hour),
			wantValid: true,
		},
		{
			name:      "both equal and before now",
			edu:       now.Add(-1 * time.Hour),
			loan:      now.Add(-1 * time.Hour),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid := IsSkipPeriodActive(tt.edu, tt.loan, time.Time{})
			if gotValid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v", tt.wantValid, gotValid)
			}
		})
	}
}
