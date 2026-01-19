package common

import (
	"encoding/csv"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// writeCSVFile writes records to a CSV file.
func WriteCSVFile(filename string, records [][]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("could not write record: %v", err)
		}
	}

	return nil
}

// ConvertUTCToPHT converts a UTC time to Philippine Time (PHT, UTC+8)
func ConvertUTCToPHT(utcTime time.Time) time.Time {
	// Define the UTC+8 offset directly
	loc := time.FixedZone("Asia/Manila", 8*60*60) // PHT is UTC+8

	// Convert the time to the target timezone
	return utcTime.In(loc)
}

func ConversionRateByType(pesoToDataConversion []models.PesoToDataConversion, isScored bool) (float64, error) {
	if len(pesoToDataConversion) == 0 {
		return 0, fmt.Errorf("conversion matrix is empty")
	}

	targetLoanType := consts.UnscoredLoanType
	if isScored {
		targetLoanType = consts.ScoredLoanType
	}

	for _, conversion := range pesoToDataConversion {
		if conversion.LoanType == targetLoanType {
			return conversion.ConversionRate, nil
		}
	}

	return 0, fmt.Errorf("conversion rate not found for loanType: %s", targetLoanType)
}

func CalculateTimestamp(days int32) primitive.DateTime {
	return primitive.NewDateTimeFromTime(time.Now().Add(time.Duration(days) * 24 * time.Hour))
}
