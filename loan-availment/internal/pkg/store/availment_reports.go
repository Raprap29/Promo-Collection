package store

import (
	"context"
	"fmt"
	"time"

	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AvailmentReportRepository struct {
	repo *MongoRepository[[]models.AvailmentReport]
}

func NewAvailmentReportRepository() *AvailmentReportRepository {
	collection := db.MDB.Database.Collection(consts.AvailmentCollection)
	mrepo := NewMongoRepository[[]models.AvailmentReport](collection)
	return &AvailmentReportRepository{repo: mrepo}
}

func (r *AvailmentReportRepository) AvailmentReports(ctx context.Context, dynamicStartDay string) ([]models.AvailmentReport, error) {

	chunkSize := 24 / configs.REPORT_CHUNKS
	var utcStartDay time.Time
	if dynamicStartDay != "" {
		parsedTime, err := time.Parse(time.RFC3339, dynamicStartDay)
		if err == nil && parsedTime.Before(time.Now()) {
			utcStartDay = parsedTime
		} else {
			utcStartDay = getMidnightYesterdayUTC()
		}
	} else {
		utcStartDay = getMidnightYesterdayUTC()
	}
	startTime := utcStartDay.Add(-time.Duration(8) * time.Hour)
	endTime := startTime.Add(time.Duration(configs.REPORT_EVERY_X_HOURS) * time.Hour)

	var availmentReportResult []models.AvailmentReport
	logger.Info(ctx, "Query execution started ")
	chunkStart := startTime

	// Max retry limit for chunk size increase to avoid infinite loop
	maxRetries := 5
	retries := 0

	for {
		tempResult := []models.AvailmentReport{} // Temporary slice to hold chunk results

		// Loop to process chunks until the end time is reached
		for chunkStart.Before(endTime) {
			chunkEnd := chunkStart.Add(time.Duration(chunkSize) * time.Hour)
			if chunkEnd.After(endTime) {
				chunkEnd = endTime
			}

			var chunkResults []models.AvailmentReport

			pipeline := mongo.Pipeline{
				bson.D{{Key: "$match", Value: bson.M{
					"createdAt": bson.M{
						"$gte": chunkStart,
						"$lt":  chunkEnd,
					},
				}}},
				bson.D{{Key: "$lookup", Value: bson.M{
					"from":         consts.LoansCollection,
					"localField":   "MSISDN",
					"foreignField": "MSISDN",
					"as":           "loanStatus",
				}}},
				bson.D{{Key: "$project", Value: bson.M{
					"AvailmentId": "$_id",
					"TransactionDatetime": bson.M{
						"$add": bson.A{
							"$createdAt",
							bson.M{"$multiply": bson.A{8, 60, 60, 1000}}, // Convert createdAt to PHT (UTC + 8 hours)
						},
					},
					"AvailmentChannel":   "$channel",
					"MSISDN":             "$MSISDN",
					"BrandType":          "$brand",
					"LoanType":           "$loanType",
					"LoanProductName":    "$productName",
					"LoanProductKeyword": "$keyword",
					"ServicingPartner":   "$servicingPartner",
					"LoanAmount": bson.M{
						"$cond": bson.M{
							"if": bson.M{"$or": []interface{}{
								bson.M{"$eq": []interface{}{"$totalLoanAmount", nil}},
								bson.M{"$eq": []interface{}{"$totalLoanAmount", 0}},
							}},
							"then": 0,
							"else": bson.M{
								"$subtract": []interface{}{
									"$totalLoanAmount",
									bson.M{"$ifNull": []interface{}{"$serviceFee", 0}},
								},
							},
						},
					},
					"ServiceFeeAmount": bson.M{
						"$cond": bson.M{
							"if": bson.M{"$or": []interface{}{
								bson.M{"$in": []interface{}{bson.M{"$ifNull": []interface{}{"$totalLoanAmount", nil}}, []interface{}{nil, 0}}},
								bson.M{"$eq": []interface{}{bson.M{"$ifNull": []interface{}{"$serviceFee", nil}}, nil}},
							}},
							"then": 0,
							"else": "$serviceFee",
						},
					},
					"TransactionType":    consts.TransactionTypeForAvailmentReport,
					"AvailmentResult": bson.M{
						"$cond": bson.M{
							"if":   "$result",
							"then": consts.Success,
							"else": consts.Fail,
						},
					},
					"AvailmentErrorText": "$errorCode",
					"LoanAgeing": bson.M{
						"$cond": bson.M{
							"if": "$result",
							"then": bson.M{
								"$trunc": bson.M{
									"$divide": bson.A{
										bson.M{"$subtract": bson.A{"$$NOW", "$createdAt"}},
										consts.AgeingTimeInMilliSeconds,
									},
								},
							},
							"else": nil,
						},
					},
					"CreditScoreAtTime": "$creditScore",
					"StatusOfProvisionedLoan": bson.M{
						"$cond": bson.M{
							"if": "$result",
							"then": bson.M{
								"$cond": bson.M{
									"if":   bson.M{"$gt": bson.A{bson.M{"$size": "$loanStatus"}, 0}},
									"then": consts.Active,
									"else": consts.Inactive,
								},
							},
							"else": consts.Inactive,
						},
					},
					"FromMigration": bson.M{
						"$cond": bson.M{
							"if":   "$migrated",
							"then": consts.True,
							"else": consts.False,
						},
					},
				}}},
			}

			logger.Info(ctx, "Processing chunk from %v to %v with chunk size %v hours", chunkStart, chunkEnd, chunkSize)

			err := r.repo.AggregateAll(pipeline, &chunkResults)
			if err != nil {
				logger.Error(ctx, "Error processing chunk from %v to %v: %v", chunkStart, chunkEnd, err)

				// Increase the chunk size by 1 if it's not already at the max
				chunkSize++
				retries++
				logger.Info(ctx, "Increased total chunk size to %v hours. Retrying from the first chunk.", chunkSize)

				// Exit if we have exceeded the maximum retries
				if retries > maxRetries {
					return nil, fmt.Errorf("max retries reached. Unable to process the chunks due to errors")
				}

				// Reset chunkStart to the original start time and retry the process with the updated chunk size
				chunkStart = startTime
				tempResult = nil // Clear the temporary results
				break            // Exit the inner loop to retry the entire process
			}

			// Append successful chunk results and move to the next chunk
			tempResult = append(tempResult, chunkResults...)
			chunkStart = chunkEnd // Move to the next chunk
		}

		// If all chunks in the loop were processed successfully, break the outer loop
		if len(tempResult) > 0 {
			availmentReportResult = append(availmentReportResult, tempResult...)
			break
		}

		// If chunkStart has reached or passed endTime, break the loop to avoid infinite processing
		if chunkStart.Equal(endTime) || chunkStart.After(endTime) {
			break
		}
	}

	logger.Info(ctx, "Query execution completed")
	logger.Info(ctx, "Records fetched %v", len(availmentReportResult))
	return availmentReportResult, nil
}

// Function to return midnight of yesterday in UTC
func getMidnightYesterdayUTC() time.Time {
	location := time.FixedZone("Asia/Manila", 8*60*60) //PHT = UTC+8

	nowInPHT := time.Now().In(location)
	utcStartDay := getYesterdaysMidnightUTCFromPHT(nowInPHT)
	return utcStartDay
}

// Function to return yesterday's midnight in UTC based on a given PHT date
func getYesterdaysMidnightUTCFromPHT(phtDate time.Time) time.Time {
	// Subtract one day (24 hours) from the UTC date to get yesterday
	yesterday := phtDate.AddDate(0, 0, -1)

	// Set the time to midnight (00:00:00) in UTC
	yesterdayMidnight := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)

	// Return the result
	return yesterdayMidnight
	//return yesterdayMidnight.Format("2006-01-02T15:04:05.000+00:00")
}
