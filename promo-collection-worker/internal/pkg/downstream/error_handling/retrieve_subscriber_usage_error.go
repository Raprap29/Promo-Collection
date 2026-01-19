package error_handling

import (
	"fmt"
)

type RetrieveSubscriberUsageError struct {
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StatusCode   int    `json:"-"` // HTTP status code (e.g., 200, 400, 500)
	Err          error
}

func NewRetrieveSubscriberUsageError(err error, statusCode ...int) *RetrieveSubscriberUsageError {
	errObj := &RetrieveSubscriberUsageError{Err: err}

	if len(statusCode) > 0 {
		errObj.StatusCode = statusCode[0]
	}

	return errObj
}

func (e *RetrieveSubscriberUsageError) Error() string {
	tag := "API error:"
	if e.ErrorCode != "" {
		tag = "HTTP error:"
	}
	return fmt.Sprintf("%s statusCode=%d, err:%v, errorCode=%v, errorMessage=%s",
		tag,
		e.StatusCode,
		e.Err,
		e.ErrorCode,
		e.ErrorMessage,
	)
}
