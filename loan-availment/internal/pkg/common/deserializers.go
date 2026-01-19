package common

import (
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/models"
	storeModels "globe/dodrio_loan_availment/internal/pkg/store/models"
	"time"
)

func PrepareAvailmentMessage(wallets []models.WalletTypes, availments []storeModels.AvailmentTransaction) ([][]string, error) {

	walletMap := make(map[string]string)
	for _, w := range wallets {
		walletMap[w.Name] = fmt.Sprintf("%.d", w.Number)
	}
	availmentMessages := make([][]string, len(availments))
	for i, message := range availments {
		availmentResult := "Fail"
		if message.Result {
			availmentResult = "Success"
		}

		loanStatus := "Active"

		if message.LoanProvisionStatus == "Closed" {
			loanStatus = "Closed"
		} else if message.LoanProvisionStatus == "Churned" {
			loanStatus = "Writeoff"
		} else if message.LoanProvisionStatus == "Auto_Writeoff" || message.LoanProvisionStatus == "Manual_Writeoff" {
			loanStatus = "Writeoff"
		} else if message.LoanProvisionStatus == "Sweep" {
			loanStatus = "Closed"
		} else if message.LoanProvisionStatus == "Auto" && message.CollectionType == "F" {
			loanStatus = "Closed"
		}

		ageing := int(time.Since(message.CreatedAt).Hours() / 24)
		availmentMessages[i] = []string{
			message.GUID,
			message.CreatedAt.Format("01/02/2006 15:04"),
			message.Channel,
			message.MSISDN,
			message.Brand,
			message.LoanType,
			//get productname by id
			message.ProductName,
			//get keyword by id
			message.Keyword,
			//get channel by id

			walletMap[message.ServicingPartner],
			fmt.Sprintf("%.0f", message.TotalLoanAmount),
			fmt.Sprintf("%.0f", message.ServiceFee),
			consts.TransactionTypeForAvailmentReport,
			//add fail or succeess for tue or false
			availmentResult,
			message.ErrorCode,
			//find difference between current date createdAt
			fmt.Sprintf("%d", ageing),
			//find loan status
			loanStatus,
		}
	}

	return availmentMessages, nil
}
