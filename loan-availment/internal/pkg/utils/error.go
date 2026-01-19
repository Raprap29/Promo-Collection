package utils

import "globe/dodrio_loan_availment/internal/pkg/models"

func GetErrorCode(err error) string {
	if customErr, ok := err.(*models.CustomError); ok {
		return customErr.ErrorCode() // Access the code from CustomError
	}
	return "DODRIO2_INTERNAL_ERROR" // Return empty or default code for non-custom errors
}
