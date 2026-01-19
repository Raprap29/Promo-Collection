package common

import (
	"testing"
	"time"

	"globe/dodrio_loan_availment/internal/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestConversionRateByType(t *testing.T) {
	matrix := []models.PesoToDataConversion{
		{LoanType: "SCORED", ConversionRate: 1.5},
		{LoanType: "UNSCORED", ConversionRate: 2.5},
	}

	t.Run("found", func(t *testing.T) {
		rate, err := ConversionRateByType(matrix, true)
		assert.NoError(t, err)
		assert.Equal(t, 1.5, rate)
	})

	t.Run("unscored found", func(t *testing.T) {
		rate, err := ConversionRateByType(matrix, false)
		assert.NoError(t, err)
		assert.Equal(t, 2.5, rate)
	})

	t.Run("not found", func(t *testing.T) {
		incompleteMatrix := []models.PesoToDataConversion{
			{LoanType: "SCORED", ConversionRate: 1.5},
		}
		rate, err := ConversionRateByType(incompleteMatrix, false)
		assert.Error(t, err)
		assert.Equal(t, 0.0, rate)
		assert.Contains(t, err.Error(), "conversion rate not found for loanType: UNSCORED")
	})

	t.Run("empty matrix", func(t *testing.T) {
		rate, err := ConversionRateByType([]models.PesoToDataConversion{}, true)
		assert.Error(t, err)
		assert.Equal(t, 0.0, rate)
	})
}

func TestCalculateTimestamp(t *testing.T) {
	days := int32(2)
	expected := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	got := CalculateTimestamp(days).Time()

	assert.WithinDuration(t, expected, got, time.Second)
}
