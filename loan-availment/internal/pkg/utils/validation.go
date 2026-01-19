package utils

import (
	"errors"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"regexp"
	"strings"
	"unicode"
)

func ValidateAndFormat(MSISDN string) (string, error) {
	// Trim the input to last 10 characters
	if len(MSISDN) < 10 {
		logger.Error("Invalid MSISDN")
		return "", errors.New("invalid MSISDN")
	}
	trimmed := MSISDN[len(MSISDN)-10:]

	// Check if the last 10 characters are digits
	for _, char := range trimmed {
		if !unicode.IsDigit(char) {
			logger.Error("Invalid MSISDN")
			return "", errors.New("invalid MSISDN")
		}
	}

	return trimmed, nil
}

// CleanMSISDN extracts the last 10, 11, or 12 digits from the MSISDN string.
func CleanMSISDN(msisdn string) string {
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(msisdn, "")

	// Extract the last 10, 11, or 12 digits if they exist
	if len(cleaned) >= 12 {
		cleaned = cleaned[len(cleaned)-12:]
	} else if len(cleaned) >= 11 {
		cleaned = cleaned[len(cleaned)-11:]
	} else if len(cleaned) >= 10 {
		cleaned = cleaned[len(cleaned)-10:]
	}

	return cleaned
}

// IsValidMSISDN checks if the MSISDN is valid based on the regex pattern and length.
func IsValidMSISDN(cleanedMSISDN string) (bool, error) {
	// Regex to check for valid prefixes (9, 09, 639, 632) followed by 9 digits
	regex := regexp.MustCompile(consts.ValidPrefixForMSISDN)

	// Validate the cleaned MSISDN using the regex
	if !regex.MatchString(cleanedMSISDN) {
		return false, consts.ErrorMsisdnFormatValidationFailed
	}

	// Check if the length is between 10 and 12 digits
	if len(cleanedMSISDN) < 10 || len(cleanedMSISDN) > 12 {
		return false, consts.ErrorMsisdnFormatValidationFailed
	}

	return true, nil
}

func IsValidTransactionID(transactionID string) (bool, string, string, error) {
	// Valid channel codes
	validChannelCodes := consts.Channels

	// Rule 1: Check if the total length exceeds 70 characters
	if len(transactionID) > 70 {
		return false, "", "", consts.ErrorTransactionIdFormatValidationFailed
	}

	// Rule 2: Check if the TransactionID starts with a valid channel code
	var validCode string
	for _, code := range validChannelCodes {
		if strings.HasPrefix(strings.ToLower(transactionID), strings.ToLower(code)) {
			validCode = code
			break
		}
	}

	if validCode == "" {
		return false, "", "", consts.ErrorTransactionIdFormatValidationFailed
	}

	// Remove the valid channel code from the start of the TransactionID
	transactionID = strings.TrimPrefix(transactionID, validCode)

	// Rule 3: Check if the remaining part (after the channel code) contains only valid characters
	// Allow only alphabets, numbers, dash (-), and underscore (_)
	validGeneratedPart := regexp.MustCompile(consts.ValidChannelCode).MatchString
	if !validGeneratedPart(transactionID) {
		return false, "", "", consts.ErrorTransactionIdFormatValidationFailed
	}

	return true, validCode, transactionID, nil
}

func IsValidKeyword(keyword string) (bool, error) {

	// Rule 1: Keyword should be case insensitive
	keyword = strings.ToLower(keyword)

	// Rule 2: Check if the keyword exceeds 160 characters
	if len(keyword) > 160 {
		return false, consts.ErrorKeywordFormatValidationFailed
	}

	// Rule 3: Check if the keyword contains only allowed characters (alphabets, numbers, and space)
	// The regular expression allows alphabets, numbers, and spaces only
	validKeyword := regexp.MustCompile(consts.ValidKeywordCode).MatchString
	if !validKeyword(keyword) {
		return false, consts.ErrorKeywordFormatValidationFailed
	}

	return true, nil
}
