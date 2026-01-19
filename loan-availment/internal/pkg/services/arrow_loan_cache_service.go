package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"net"
	"strconv"
	"time"

	"github.com/pkg/sftp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/ssh"
)

type ArrowLoansCacheService struct {
}

func NewArrowLoansCacheService() *ArrowLoansCacheService {
	return &ArrowLoansCacheService{
	}
}



func (s *ArrowLoansCacheService) ArrowLoansCacheTable(ctx context.Context, endTime time.Time) (*models.ArrowCacheTable, error) {
	reportForLastXHours, err := strconv.ParseInt(configs.REPORT_FOR_LAST_X_HOURS, 10, 32)

	if err != nil {
		return nil, err
	}

	startTime := endTime.Add(-1 * time.Duration(reportForLastXHours) * time.Hour)

	logger.Debug(ctx,startTime, " ", endTime)
	result, err := s.GenerateLoanCache(ctx, startTime, endTime)

	if err != nil {
		return nil, err
	} else if result == nil {
		logger.Error(ctx, consts.NoDataInDatabase)
	} else {
		logger.Info(ctx, string(consts.DataFetchedFromDatabase))
	}

	csvData, err := s.GenerateCSV(ctx, *result, startTime, endTime)

	if err != nil {
		return nil, err
	}

	logger.Debug("csvData: %v", csvData)

	err = s.TransferCSVToSFTP(ctx, csvData)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *ArrowLoansCacheService) GenerateLoanCache(ctx context.Context, startTime time.Time, endTime time.Time) (*models.ArrowCacheTable, error) {
	database := db.MDB.Database
	availmentsCollection := database.Collection(consts.AvailmentCollection)
	collectionsCollection := database.Collection(consts.CollectionsCollection)

	var result models.ArrowCacheTable

	filterByPublishedToKafka := true
	filterByResult := true

	count, err := s.CountDocumentsWithOptionalFields(availmentsCollection, startTime, endTime, &filterByPublishedToKafka, nil)
	if err != nil {
		logger.Error(ctx, "Error counting documents:%v", err)
		return nil, errors.New(string(consts.ErrorQueryingDatabase))
	}
	result.AvailmentRecordsPublished = count

	count, err = s.CountDocumentsWithOptionalFields(availmentsCollection, startTime, endTime, nil, &filterByResult)
	if err != nil {
		logger.Error(ctx, "Error counting documents:%v", err)
		return nil, errors.New(string(consts.ErrorQueryingDatabase))
	}
	result.SuccessfulAvailments = count

	count, err = s.CountDocumentsWithOptionalFields(collectionsCollection, startTime, endTime, &filterByPublishedToKafka, nil)
	if err != nil {
		logger.Error(ctx, "Error counting documents:%v", err)
		return nil, errors.New(string(consts.ErrorQueryingDatabase))
	}
	result.CollectionRecordsPublished = count

	count, err = s.CountDocumentsWithOptionalFields(collectionsCollection, startTime, endTime, nil, &filterByResult)
	if err != nil {
		logger.Error(ctx, "Error counting documents:%v", err)
		return nil, errors.New(string(consts.ErrorQueryingDatabase))
	}
	result.SuccessfulCollections = count

	return &result, nil
}

// kept private because this function will not be used outside
func (s *ArrowLoansCacheService) CountDocumentsWithOptionalFields(collection *mongo.Collection, startTime time.Time, endTime time.Time, filterByPublishedToKafka *bool, filterByResult *bool) (int64, error) {
	//TODO: Create a context with a timeout (optional)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define the base filter with the createdAt condition
	filter := bson.M{
		"createdAt": bson.M{
			"$gte": startTime,
			"$lt":  endTime,
		},
	}

	// Add publishedToKafka to the filter if provided
	if filterByPublishedToKafka != nil {
		filter["publishedToKafka"] = *filterByPublishedToKafka
	}

	// Add result to the filter if provided
	if filterByResult != nil {
		filter["result"] = *filterByResult
	}

	// Count the documents matching the filter
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *ArrowLoansCacheService) GenerateCSV(ctx context.Context, arrowCacheTable models.ArrowCacheTable, startTime time.Time, endTime time.Time) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	writer := csv.NewWriter(buffer)

	header := []string{
		"no. of collection records published",
		"no. of availment records published",
		"total successful collections",
		"total successful availments",
		"start date time",
		"end date time",
	}

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	row := []string{
		fmt.Sprintf("%d", arrowCacheTable.CollectionRecordsPublished),
		fmt.Sprintf("%d", arrowCacheTable.AvailmentRecordsPublished),
		fmt.Sprintf("%d", arrowCacheTable.SuccessfulCollections),
		fmt.Sprintf("%d", arrowCacheTable.SuccessfulAvailments),
		fmt.Sprint(startTime.Format(consts.ReportDateFormat)),
		fmt.Sprint(endTime.Format(consts.ReportDateFormat)),
	}

	if err := writer.Write(row); err != nil {
		return nil, err
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (s *ArrowLoansCacheService) TransferCSVToSFTP(ctx context.Context, csvData *bytes.Buffer) error {
	config := &ssh.ClientConfig{
		User: configs.SFTP_USER,
		Auth: []ssh.AuthMethod{
			ssh.Password(configs.SFTP_PASSWORD),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(configs.SFTP_HOST, configs.SFTP_PORT), config)
	if err != nil {
		return err
	}

	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer client.Close()

	dstFile, err := client.Create(configs.SFTP_REMOTE_FILE_PATH)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.Write(csvData.Bytes()); err != nil {
		return err
	}

	logger.Info(ctx, "File sent to SFTP server")
	return nil
}
