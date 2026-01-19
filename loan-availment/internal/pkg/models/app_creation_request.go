package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AppCreationRequest struct {
	MSISDN        string `json:"msisdn" binding:"required"`
	TransactionId string `json:"transactionId" binding:"required"`
	Keyword       string `json:"kw" binding:"required"`
	SystemClient  string `json:"systemClient" binding:"required"`
}
type AppCreationProcessRequest struct {
	MSISDN        string             `json:"msisdn" binding:"required"`
	Channel       string             `json:"channel" binding:"channel"`
	Keyword       string             `json:"kw" binding:"required"`
	SystemClient  string             `json:"systemClient" binding:"required"`
	TransactionID string             `json:"transactionId" `
	CreditScore   float64            `json:"creditScore" `
	CustomerType  string             `json:"customerType" `
	BrandCode     string             `json:"brandCode" `
	StartTime     time.Time          `json:"StartTime" `
	BrandId       primitive.ObjectID `json:"brandId" `
}
