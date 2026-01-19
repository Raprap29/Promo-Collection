package models

import "github.com/google/uuid"

type LoanAvailmentMessage struct {
	// ID           string    `json:"_id"`
	Keyword      string    `json:"Keyword"`
	MSISDN       string    `json:"MSISDN"`
	DodrioDomain string    `json:"DodrioDomain"`
	SystemClient string    `json:"SystemClient"`
	ProcessId    uuid.UUID `json:"ProcessId"`
	Attribute    string    `json:"Attribute"`
}
