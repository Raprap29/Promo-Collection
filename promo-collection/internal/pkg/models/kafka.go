package models

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
