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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/store"
	"globe/dodrio_loan_availment/internal/pkg/utils"
)

func ProcessCSVAndUpdateDBForWhitelistProductId(ctx context.Context,filePath string, productId primitive.ObjectID) error {

	database := db.MDB.Database

	logger.Debug(ctx,"ProductId is: %v", productId)

	whitelistedSubsCollection := database.Collection(consts.WhitelistedSubsCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	msisdnIndex := findColumnIndicesProductWhitelist(records[0], "MSISDN")
	if msisdnIndex == -1 {
		logger.Error(ctx,"invalid header: missing 'MSISDN'")
		return fmt.Errorf("invalid header: missing 'MSISDN'")
	}

	// Create a set of MSISDNs from the CSV file
	msisdnSet := make(map[string]struct{})
	for _, record := range records[1:] { // Skip header row
		if len(record) <= msisdnIndex {
			continue // Skip rows that don't have enough columns
		}

		msisdn := utils.CleanMSISDN(record[msisdnIndex])
		msisdnSet[msisdn] = struct{}{}
	}

	// Insert new entries into the WhitelistedSubs collection
	for msisdn := range msisdnSet {
		filter := bson.M{
			"MSISDN":    msisdn,
			"productId": productId,
		}

		update := bson.M{
			"$setOnInsert": bson.M{
				"MSISDN":    msisdn,
				"productId": productId,
			},
		}

		opts := options.Update().SetUpsert(true)
		_, err := whitelistedSubsCollection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return err
		}
	}

	// Remove old entries for the productId that are not in the CSV
	filter := bson.M{
		"productId": productId,
		"MSISDN":    bson.M{"$nin": keysProductWhitelist(msisdnSet)},
	}

	_, err = whitelistedSubsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}

func UpdateAndValidateCSVFileForWhitelisting(ctx context.Context,filename string) (primitive.ObjectID, error) {

	loansProductsRepo := store.NewLoansProductsRepository()

	productName := strings.TrimSuffix(filepath.Base(filename), ".csv")

	// check if productId exists in the database
	filter := bson.M{"name": productName}
	loansProductsResult, err := loansProductsRepo.LoanProductByFilter(filter)
	if err != nil {
		logger.Error(ctx,"Failed to get LoanProduct ID for %v Product Name: %v", productName, err)
	}
	logger.Debug(ctx,"productId from CSV is: %v", loansProductsResult.ProductId)

	file, err := os.Open(filename)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read the header row
	headers, err := reader.Read()
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("could not read header: %v", err)
	}

	msisdnIndex := -1
	for i, header := range headers {
		if strings.TrimSpace(header) == "MSISDN" {
			msisdnIndex = i
		}
	}

	if msisdnIndex == -1 {
		return primitive.NilObjectID, fmt.Errorf("invalid header: missing 'MSISDN'")
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
			return primitive.NilObjectID, fmt.Errorf("could not read record: %v", err)
		}

		// Ensure there is exactly one column in each row
		if len(record) < 1 {
			return primitive.NilObjectID, fmt.Errorf("invalid record: expected 1 column but got %d", len(record))
		}

		msisdn := strings.TrimSpace(record[msisdnIndex])

		// Validate the MSISDN
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
		return primitive.NilObjectID, fmt.Errorf("could not write updated CSV file: %v", err)
	}

	// Write the failed records to the failed csv file
	if len(failedRecords) > 1 {

		timestamp := time.Now().Format("20060102150405")
		failedFileName := productName + "_" + timestamp + ".csv"
		failedFilePath := filepath.Join("temp", failedFileName)
		if err := common.WriteCSVFile(failedFilePath, failedRecords); err != nil {
			return primitive.NilObjectID, fmt.Errorf("could not write failed CSV file: %v", err)
		}
	}

	return loansProductsResult.ProductId, nil
}

// findColumnIndices finds the indices of the columns with the given headers.
func findColumnIndicesProductWhitelist(headerRow []string, headerNames ...string) int {
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
	return indices["MSISDN"]
}

func keysProductWhitelist(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
