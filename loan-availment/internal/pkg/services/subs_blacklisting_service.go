package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/utils"
)

func ProcessCSVAndUpdateDBForBlacklistedSubs(ctx context.Context,filePath string) error {

	database := db.MDB.Database

	subscribersCollection := database.Collection(consts.SubscribersCollection)
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Open and read the CSV file
	csvFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Find the indices for MSISDN and Reason columns
	msisdnIndex, reasonIndex := findColumnIndicesSubsBlacklist(records[0], "MSISDN", "Reason")
	if msisdnIndex == -1 || reasonIndex == -1 {
		return fmt.Errorf("invalid header: missing 'MSISDN' or 'Reason'")
	}

	// Create a set of MSISDNs from the CSV file
	msisdnSet := make(map[string]string)
	for _, record := range records[1:] { // Skip header row
		if len(record) <= msisdnIndex || len(record) <= reasonIndex {
			continue // Skip rows that don't have enough columns
		}

		msisdn := utils.CleanMSISDN(record[msisdnIndex])
		reason := strings.TrimSpace(record[reasonIndex])

		msisdnSet[msisdn] = reason
	}

	// Update MongoDB collection based on MSISDNs in the CSV
	for msisdn, reason := range msisdnSet {
		filter := bson.M{"MSISDN": msisdn}
		update := bson.M{"$set": bson.M{
			"blacklisted":     true,
			"exclusionReason": reason,
		}}
		_, err := subscribersCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			logger.Error(ctx,"Error updating MongoDB collection: %v",err)
			return err
		}

		// Check if MSISDN exists; if not, create a new entry
		result := subscribersCollection.FindOne(ctx, filter)
		var existingSubscriber struct{ MSISDN string }
		err = result.Decode(&existingSubscriber)
		if err != nil && err == mongo.ErrNoDocuments {
			// MSISDN does not exist, create a new entry
			newSubscriber := common.SerializeSubscriber(msisdn, true, false, reason, "", "")
			_, err = subscribersCollection.InsertOne(ctx, newSubscriber)
			if err != nil {
				logger.Error(ctx,"Error creating new entry: %v",err)
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// Set blacklisted to false for MSISDNs not present in the CSV
	filter := bson.M{"MSISDN": bson.M{"$nin": keysSubsBlacklist(msisdnSet)}}
	update := bson.M{
		"$set": bson.M{
			"blacklisted":     false,
			"exclusionReason": nil,
		},
	}
	_, err = subscribersCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func UpdateAndValidateCSVFileForBlacklisting(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read the header row
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("could not read header: %v", err)
	}

	if len(headers) < 2 {
		return fmt.Errorf("invalid header: expected at least two columns")
	}

	msisdnIndex, reasonIndex := -1, -1
	for i, header := range headers {
		if strings.TrimSpace(header) == "MSISDN" {
			msisdnIndex = i
		} else if strings.TrimSpace(header) == "Reason" {
			reasonIndex = i
		}
	}

	if msisdnIndex == -1 || reasonIndex == -1 {
		return fmt.Errorf("invalid header: missing 'MSISDN' or 'Reason'")
	}

	var updatedRecords [][]string
	var failedRecords [][]string

	// Add headers to the new files
	updatedRecords = append(updatedRecords, headers)
	failedRecords = append(failedRecords, []string{"MSISDN", "failedReason"})
	// failedRecords = [][]string{{"MSISDN", "failedReason"}}

	// Read and validate each row
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("could not read record: %v", err)
		}

		if len(record) < len(headers) {
			return fmt.Errorf("invalid record: expected %d columns but got %d", len(headers), len(record))
		}

		msisdn := strings.TrimSpace(record[msisdnIndex])
		reason := strings.TrimSpace(record[reasonIndex])

		// Skip empty MSISDN or Reason fields
		if msisdn == "" || reason == "" {
			continue
		}

		cleanedMSISDN := utils.CleanMSISDN(msisdn)
		_, err = utils.IsValidMSISDN(cleanedMSISDN)
		if errors.Is(err, consts.ErrorMsisdnFormatValidationFailed) {
			failedRecords = append(failedRecords, []string{msisdn, err.Error()})
			continue
		}

		// Update the MSISDN in the record
		record[msisdnIndex] = cleanedMSISDN
		updatedRecords = append(updatedRecords, record)
	}

	// Write the updated records back to the original file
	if err := common.WriteCSVFile(filename, updatedRecords); err != nil {
		return fmt.Errorf("could not write updated CSV file: %v", err)
	}

	fileNameBlacklist := strings.TrimSuffix(filepath.Base(filename), ".csv")
	// Write the failed records to the ".csv" file
	if len(failedRecords) > 1 {
		timestamp := time.Now().Format("20060102150405")
		failedFileName := fileNameBlacklist + "_" + timestamp + ".csv"
		failedFilePath := filepath.Join("temp", failedFileName)
		if err := common.WriteCSVFile(failedFilePath, failedRecords); err != nil {
			return fmt.Errorf("could not write failed CSV file: %v", err)
		}
	}

	return nil
}

// findColumnIndices finds the indices of the columns with the given headers.
func findColumnIndicesSubsBlacklist(headerRow []string, headerNames ...string) (int, int) {
	indices := make(map[string]int)
	for i, header := range headerRow {
		header = strings.TrimSpace(header)
		for _, name := range headerNames {
			if strings.EqualFold(header, name) {
				indices[name] = i
				break
			}
		}
	}
	return indices["MSISDN"], indices["Reason"]
}

func keysSubsBlacklist(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
