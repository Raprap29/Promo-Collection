package models

type CustomError struct {
	Code    string
	Message string
}

func (e CustomError) Error() string {
	return e.Message
}
func (e CustomError) ErrorCode() string {
	return e.Code
}
