package models

import "time"

type PromoEventMessage struct {
	ID            int64   `json:"ID"`
	MSISDN        int64   `json:"MSISDN"`
	MESSAGE       *string `json:"MESSAGE"` // nullable
	SVCID         int64   `json:"SVC_ID"`
	OPID          int64   `json:"OP_ID"`
	DENOM         string  `json:"DENOM"`
	CHARGEDAMT    float64 `json:"CHARGED_AMT"`
	OPDATETIME    string  `json:"OP_DATETIME"` // could be time.Time if parsed
	EXPIRYDATE    string  `json:"EXPIRY_DATE"`
	STATUS        int     `json:"STATUS"`
	ERRCODE       int     `json:"ERR_CODE"`
	ORIGIN        int64   `json:"ORIGIN"`
	STARTTIME     string  `json:"START_TIME"`
	HPLMN         *string `json:"HPLMN"`       // nullable
	ORIGEXPIRY    *string `json:"ORIG_EXPIRY"` // nullable
	EXPIRY        int     `json:"EXPIRY"`
	WALLETAMOUNT  int     `json:"WALLET_AMOUNT"`
	WALLETKEYWORD string  `json:"WALLET_KEYWORD"`
}

type KafkaMessageForPublishing struct {
	TransactionId       string    `json:"transactionId"`
	AvailmentId         string    `json:"availmentId"`
	CollectionDateTime  time.Time `json:"collectionDateTime"`
	Method              string    `json:"method"`
	LoanAgeing          int32     `json:"loanAgeing"`
	CollectedLoanAmount float64   `json:"collectedLoanAmount"`
	CollectedServiceFee float64   `json:"collectedServiceFee"`
	UnpaidLoanAmount    float64   `json:"unpaidLoanAmount"`
	UnpaidServiceFee    float64   `json:"unpaidServiceFee"`
	Result              string    `json:"result"`
	CollectionErrorText string    `json:"collectionErrorText"`
	MSISDN              string    `json:"msisdn"`
	CollectionCategory  string    `json:"collectionCategory"`
	PaymentChannel      string    `json:"paymentChannel"`
	CollectionType      string    `json:"collectionType"`
	CollectedAmount     float64   `json:"collectedAmount"`
	TotalUnpaid         float64   `json:"totalUnpaid"`
	TokenPaymentId      string    `json:"tokenPaymentId"`
	DataCollected       float64   `json:"dataCollected"`
}
