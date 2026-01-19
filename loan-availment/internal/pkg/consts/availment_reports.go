package consts

const (
	Success  = "Success"
	Fail     = "Fail"
	Active   = "Active"
	Inactive = "Inactive"
	True     = "True"
	False    = "False"

	SuccessMessageRevAvailmentReport           = "Report uploaded to GCS Bucket"
	SuccessProcessingMessageRevAvailmentReport = "Report Generation request received"

	TransactionTypeForAvailmentReport = "Availment"
	ReportEveryXHours                 = 24 * 104
	AgeingTimeInMilliSeconds          = 86400000
	ReportDateTimeFormat              = "2006-01-02 15:04:05" //To be finalized
	ReportFileNameDateFormat          = "2006-01-02"          //"YYYY-MM-DD", to be finalized
	FloatTwoDecimalFormat             = "%.2f"
)
