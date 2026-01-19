package consts

type ArrowLoanCacheError string

const (
	ReportDateFormat = "2006-01-02T15:04:05.000Z"

	ErrorQueryingDatabase   ArrowLoanCacheError = "error querying the database"
	NoDataInDatabase        ArrowLoanCacheError = "No data found in MongoDB"
	DataFetchedFromDatabase ArrowLoanCacheError = "data fetched from mongodb successfully"
)
