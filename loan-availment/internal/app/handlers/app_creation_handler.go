package handlers

import (
	"net/http"
	"strings"

	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/services"

	"github.com/gin-gonic/gin"
)

type AppCreationHandler struct {
	loanAvailmentService services.LoanAvailmentServiceInterface
	loanStatusService    services.LoanStatusServiceInterface
}

func NewAppCreationHandler(loanAvailmentService services.LoanAvailmentServiceInterface, loanStatusService services.LoanStatusServiceInterface) *AppCreationHandler {
	return &AppCreationHandler{
		loanAvailmentService: loanAvailmentService,
		loanStatusService:    loanStatusService,
	}
	
}

// includes checks if a given slice contains a specific element
func includes(arr []string, str string) bool {
	for _, item := range arr {
		if strings.ToLower(item) == strings.ToLower(str) {
			return true
		}
	}
	return false
}

func (h *AppCreationHandler) AppCreation(c *gin.Context) {
	msisdn := c.Param("MSISDN")
	transactionId := c.Param("TransactionId")
	keyword := c.Param("Keyword")
	systemClient := c.Query("SystemClient")

	req := common.SerializeAppCreationRequest(msisdn, transactionId, keyword, systemClient)
	//TODO: TBC if this how we store multiple keywords.

	statusesArray := configs.LOAN_STATUS_KEYWORD
	isLoanStatus := includes(statusesArray, req.Keyword)

	if isLoanStatus {
		err := h.loanStatusService.LoanStatus(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		err := h.loanAvailmentService.AvailLoan(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
}
