package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cloud.google.com/go/storage"

	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/store"
)

type AvailmentReportService struct {
	gcsClient           *storage.Client
	bucketName          string
	availmentReportRepo *store.AvailmentReportRepository
}

func NewAvailmentService(gcsClient *storage.Client, bucketName string, availmentReportRepo *store.AvailmentReportRepository) *AvailmentReportService {
	return &AvailmentReportService{
		gcsClient:           gcsClient,
		bucketName:          bucketName,
		availmentReportRepo: availmentReportRepo,
	}
}

func (s *AvailmentReportService) AvailmentDetailsReports(ctx context.Context, dynamicStartDay string) ([]models.AvailmentReport, error) {

	results, err := s.availmentReportRepo.AvailmentReports(ctx, dynamicStartDay)

	if err != nil {
		return nil, err
	} else if results == nil {
		logger.Error("No data found in MongoDB")
	} else {
		logger.Info("data fetched from mongodb successfully")
	}

	// Write availment reports to txt
	filename, err := s.WriteAvailmentReportsToTxtFile(ctx, results)
	if err != nil {
		return nil, err
	}

	// Upload the file to Google Cloud Storage
	err = s.UploadAvailmentReportsTxtFileToGCS(ctx, filename)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *AvailmentReportService) WriteAvailmentReportsToTxtFile(ctx context.Context, results []models.AvailmentReport) (string, error) {

	now := time.Now()
	startTime := now.Add(-time.Duration(configs.REPORT_EVERY_X_HOURS) * time.Hour).Truncate(24 * time.Hour)

	// Generate the filename based on the startTime
	startDate := startTime.Format(consts.ReportFileNameDateFormat)

	txtFilename := fmt.Sprintf("availment_report_%s.txt", startDate)

	directoryPath := configs.DIRECTORY_PATH

	fullFilePath := filepath.Join(directoryPath, txtFilename)
	logger.Info("fullFilePath = ", fullFilePath)

	//create the directory if it doesn't exist
	if err := os.MkdirAll(directoryPath, os.ModePerm); err != nil {
		logger.Error("failed at os.MkdirAll > ", err.Error())
		return "", err
	}

	// Create a new TXT file
	file, err := os.Create(fullFilePath)
	if err != nil {
		logger.Error("failed at os.Create > ", err.Error())
		return "", err
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	header := []string{
		"AvailmentId", "TransactionDatetime", "AvailmentChannel", "MSISDN", "BrandType", "LoanType",
		"LoanProductName", "LoanProductKeyword", "ServicingPartner", "LoanAmount", "ServiceFeeAmount",
		"TransactionType", "AvailmentResult", "AvailmentErrorText", "LoanAgeing", "CreditScoreAtTime",
		"StatusOfProvisionedLoan", "FromMigration",
	}
	if err := writer.Write(header); err != nil {
		logger.Error("failed at writer.Write header> ", err.Error())
		return "", err
	}

	// Write the data
	for _, result := range results {
		var loanAging string
		if result.LoanAgeing != nil {
			loanAging = strconv.Itoa(*result.LoanAgeing)
		}
		record := []string{
			result.AvailmentId,
			result.TransactionDatetime.Format(consts.ReportDateTimeFormat),
			result.AvailmentChannel,
			result.MSISDN,
			result.BrandType,
			result.LoanType,
			result.LoanProductName,
			result.LoanProductKeyword,
			result.ServicingPartner,
			fmt.Sprintf("%.2f", result.LoanAmount),
			fmt.Sprintf("%.2f", result.ServiceFeeAmount),
			result.TransactionType,
			result.AvailmentResult,
			result.AvailmentErrorText,
			loanAging,
			fmt.Sprintf("%d", result.CreditScoreAtTime),
			result.StatusOfProvisionedLoan,
			result.FromMigration,
		}
		if err := writer.Write(record); err != nil {
			logger.Error("failed at writer.Write records > ", err.Error())
			return "", err
		}
	}

	logger.Info(ctx, "Data successfully written to %s", txtFilename)
	return txtFilename, nil
}

func (s *AvailmentReportService) UploadAvailmentReportsTxtFileToGCS(ctx context.Context, filename string) error {

	// Open the file
	directoryPath := configs.DIRECTORY_PATH
	fullFilePath := filepath.Join(directoryPath, filename)

	file, err := os.Open(fullFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bucket := s.gcsClient.Bucket(s.bucketName)
	folderPath := configs.ASSURANCE_REPORT_DESTINATION_FOLDER
	objectPath := filepath.Join(folderPath, filename)
	object := bucket.Object(objectPath)
	writer := object.NewWriter(ctx)

	// Upload the file to GCS
	if _, err := io.Copy(writer, file); err != nil {
		logger.Error("failed at io.Copy > ", err.Error())
		return fmt.Errorf("failed to copy file to GCS: %v", err)
	}

	if err := writer.Close(); err != nil {
		logger.Error("failed at writer.Close > ", err.Error())
		return fmt.Errorf("failed to close GCS writer: %v", err)
	}

	// Delete the local file
	if err := os.Remove(fullFilePath); err != nil {
		logger.Error("failed at os.Remove > ", err.Error())
		return fmt.Errorf("failed to delete local file: %v", err)
	}

	logger.Info(ctx, "File %s successfully uploaded to Google Cloud Storage bucket %s", filename, s.bucketName)
	return nil
}
