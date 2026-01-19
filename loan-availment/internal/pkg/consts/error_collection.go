package consts

import "globe/dodrio_loan_availment/internal/pkg/models"

var (
	NOProductNameInAmax = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_INTERNAL_ERROR_PRODUCTNAMEINAMAX_NOT_FOUND",
		Message: "No product name in amax found for given product and brand",
	}
	ErrorDownstreamTimeout = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_REQUEST_UUP_SERVICE_TIMEOUT",
		Message: "Downstream service timeout",
	}
	ErrorMSISDNNotFound = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_REQUEST_UUP_MSISDN_NOT_FOUND",
		Message: "MSISDN not found",
	}
	ErrorUUPAccessTockenFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_REQUEST_UUP_INVALID_ACCESS_TOKEN",
		Message: "UUP access token failed",
	}
	ErrorMSISDNNotValid = &models.CustomError{
		Code:    "DODRIO2_VALIDATION_MSISDN_INVALID",
		Message: "MSISDN not valid",
	}
	ErrorInEligibleCreditScore = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_CREDIT_SCORE_NOT_ELIGBLE",
		Message: "MSISDN credit score not eligible",
	}
	ErrorActiveLoan = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_HAS_EXISTING_LOAN",
		Message: "MSISDN has existing loan",
	}
	ErrorUserBlacklisted = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_BLACKLISTED",
		Message: "MSISDN is blacklisted",
	}
	ErrorUserNotWhitelistedForProduct = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_NOT_WHITELISTED",
		Message: "MSISDN not part of the loan product whitelist",
	}
	ErrorNoDocumentFound = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_INTERNAL_ERROR_NO_DOCUMENTS_FOUND",
		Message: "No documents in result",
	}
	ErrorTransactionInProgress = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_DUPLICATE_REQUEST",
		Message: "Transaction in progress",
	}
	ErrorMsisdnFormatValidationFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_FORMAT_INVALID",
		Message: "Loan availment MSISDN parameter validation failed",
	}
	ErrorTransactionIdFormatValidationFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_TRANSACTION_ID_FORMAT_INVALID",
		Message: "Loan availment TransactionID parameter validation failed",
	}
	ErrorKeywordFormatValidationFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_KEYWORD_FORMAT_INVALID",
		Message: "Loan availment Keyword parameter validation failed",
	}
	ErrorInvalidKeyword = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_KEYWORD_NOT_FOUND_OR_ACTIVE",
		Message: "Keyword does not exist",
	}
	ErrorLoanProductNotFound = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_LOAN_PRODUCT_NOT_FOUND",
		Message: "Loan product not found",
	}
	ErrorGetDetailsByAttributeFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_REQUEST_GET_DETAILS_BY_ATTRIBUTE_FAILED",
		Message: "getDetailsByAttribute request failed",
	}
	ErrorBrandTypeNotFound = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_BRAND_TYPE_NOT_FOUND_OR_ACTIVE",
		Message: "Brand type not found or not active",
	}
	ErrorChannelTypeNotFound = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_CHANNEL_TYPE_NOT_FOUND_OR_ACTIVE",
		Message: "Channel type not found or not active",
	}
	ErrorUupConsumerType = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_CONSUMER_TYPE_NOT_SUPPORTED",
		Message: "Customer type not allowed",
	}
	ErrorUupBrandType = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_BRAND_TYPE_NOT_SUPPORTED",
		Message: "Brand type not allowed",
	}
	ErrorBrandNotAllowedToAvailLoanProduct = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_BRAND_CANNOT_AVAIL_LOAN_PRODUCT",
		Message: "MSISDN brand not allowed to avail specific loan product",
	}
	ErrorChannelNotAllowedToAvailLoanProduct = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_VALIDATION_MSISDN_CHANNEL_CANNOT_AVAIL_LOAN_PRODUCT",
		Message: "MSISDN channel not allowed to avail specific loan product",
	}
	ErrorLoanAvailmentFailed = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_TRANSACTION_FAILED",
		Message: "Loan availment failed",
	}
	ErrorLoanAvailmentTimeout = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_TRANSACTION_TIMEOUT",
		Message: "Loan availment timeout",
	}
	ErrorMissingRequiredInputs = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_INTERNAL_ERROR_MISSING_MESSAGE_EVENT",
		Message: "Missing required inputs for get message Id function",
	}
	ErrorFailedAppendCACertificate = &models.CustomError{
		Code:    "DODRIO2_LOAN_AVAILMENT_INTERNAL_ERROR_FAILED_APPEND_CA_CERTIFCATE",
		Message: "error while appending CA certificate",
	}
)
