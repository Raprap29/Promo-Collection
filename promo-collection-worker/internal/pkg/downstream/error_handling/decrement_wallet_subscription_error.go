package error_handling

import "fmt"

// Error Response (HTTP 4xx/5xx)
type DecrementWalletSubscriptionError struct {
	ResultCode   string `json:"resultCode,omitempty"`
	Description  string `json:"description,omitempty"`
	ErrorCode    string `json:"ErrorCode,omitempty"`
	ErrorDetails string `json:"ErrorDetails,omitempty"`
	StatusCode   int    `json:"-"` // HTTP status code (e.g., 200, 400, 500)
	Err          error
}

func NewDecrementWalletSubscriptionError(err error, statusCode ...int) *DecrementWalletSubscriptionError {
	errObj := &DecrementWalletSubscriptionError{Err: err}

	// If status code is provided, use it
	if len(statusCode) > 0 {
		errObj.StatusCode = statusCode[0]
	}

	return errObj
}

func (e *DecrementWalletSubscriptionError) Error() string {
	tag := "API error:"
	if e.ErrorCode != "" {
		tag = "HTTP error:"
	}
	return fmt.Sprintf("%s statusCode=%d, err:%v, resultCode=%v, errorCode=%v, desc=%s",
		tag,
		e.StatusCode,
		e.Err,
		e.ResultCode,
		e.ErrorCode,
		e.ErrorDetails,
	)
}

func (e *DecrementWalletSubscriptionError) IsNonAPIError() bool {
	return e.ErrorCode == "" &&
		e.ErrorDetails == "" &&
		e.ResultCode == "" &&
		e.Description == "" &&
		e.Err != nil
}
