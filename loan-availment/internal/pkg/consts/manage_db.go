package consts

type Operation string

const (
	CREATE            Operation = "create"
	UPDATE            Operation = "update"
	READ              Operation = "read"
	DELETE            Operation = "delete"
	INVALID_OPERATION           = "invalid operation"

	DatetimeLayout = "2006-01-02T15:04:05.000Z07:00"
)
