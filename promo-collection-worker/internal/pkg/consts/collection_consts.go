package consts

type LoanType string

const (
	LoanTypeGES           LoanType = "GES"
	LoanTypePromo         LoanType = "PROMO SKU"
	LoanTypeLoad          LoanType = "LOAD SKU"
	CollectionCategory    string   = "Data"
	Method                string   = "Auto"
	BusinessErrorCodeMin  int      = 1000
	BusinessErrorCodeMax  int      = 3014
	SystemErrorCode       int      = 9000
	ActionAck                      = "ACK"
	ActionNack                     = "NACK"
	ActionIgnore                   = "IGNORE"
	PartialCollectionType string   = "P"
	FullCollectionType    string   = "F"
	StatusChurned         string   = "Churned"
	StatusAuto            string   = "Auto"
	StatusManual          string   = "Manual"
	StatusSweep           string   = "Sweep"
	StatusManualWriteOff  string   = "Manual_Writeoff"
	StatusAutoWriteOff    string   = "Auto_Writeoff"
	ReasonCampaign        string   = "Campaign"
	ReasonCleanUp         string   = "Clean-Up"
	ReasonAmnesty         string   = "Amnesty"
	LoadTypeData          string   = "Data"
)
