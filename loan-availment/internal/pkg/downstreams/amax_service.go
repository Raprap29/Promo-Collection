package downstreams

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/downstreams/models"
	"globe/dodrio_loan_availment/internal/pkg/kafka/producer"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	serviceModels "globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/notification"
	"globe/dodrio_loan_availment/internal/pkg/pubsub"
	"globe/dodrio_loan_availment/internal/pkg/services/interfaces"
	"globe/dodrio_loan_availment/internal/pkg/store"
	"globe/dodrio_loan_availment/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var SessionResultCodeErrorMessages = map[int]*serviceModels.CustomError{
	1: {Message: "Session Invalid", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_INVALID_SESSION"},
	2: {Message: "Session Not Available", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_SESSION_UNAVAILABLE"},
}

type AmaxService struct {
	availmentRepo             *store.AvailmentRepository
	transactionInProgressRepo *store.TransactionInProgressRepository
	kafkaService              *producer.KafkaService
	notificationService       *notification.NotificationService
}

func NewAmaxService(redisAdapter interfaces.RedisStoreOperations, pubsubPublisher *pubsub.PubSubPublisher) *AmaxService {
	return &AmaxService{
		availmentRepo:             store.NewAvailmentRepository(redisAdapter),
		transactionInProgressRepo: store.NewTransactionInProgressRepository(),
		kafkaService:              producer.NewKafkaService(),
		notificationService:       notification.NewNotificationService(pubsubPublisher),
	}
}

func (h *AmaxService) CreateSession(ctx context.Context) (string, error) {
	//endpointURL
	endpointURL := configs.AMAX_CREATE_SESSION_URL
	payload := `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:soap="http://soap.amaretto.globe.com">
	<soapenv:Header/>
	<soapenv:Body>
	   <soap:createSession/>
	</soapenv:Body>
 </soapenv:Envelope>`
	//creating new request
	logger.Info(ctx, "Create session Api call payload :%v", payload)
	req, err := http.NewRequest("POST", endpointURL, strings.NewReader(payload))
	if err != nil {
		logger.Error(ctx, "Error while creating a create session API request: %v", err.Error())
		return "", err

	}
	logger.Info(ctx, "Create session Api call request :%v", req)
	body, resp, err := ApiCall(req)
	if err != nil {
		logger.Error(ctx, "Error while reading response from AMAX Create Session response: %v", err.Error())
		return "", err
	}
	logger.Info(ctx, "Create session Api call response Code:%v", resp.StatusCode)
	if resp.StatusCode == 200 {
		var CreateSessionResponse models.CreateSessionSuccessResponse
		err = xml.Unmarshal(body, &CreateSessionResponse)
		if err != nil {
			logger.Error(ctx, "Error while unmarshalling response from Create Session: %v", err.Error())
			return "", err
		}
		logger.Info(ctx, "Create session Api call Response in XML:%v ", CreateSessionResponse)
		//checking session result code
		if CreateSessionResponse.Body.CreateSessionResponse.Return.SessionResultCode == 0 {
			return CreateSessionResponse.Body.CreateSessionResponse.Return.Session, nil
		}
		//session result code is not 0
		errorMessage, exists := SessionResultCodeErrorMessages[CreateSessionResponse.Body.CreateSessionResponse.Return.SessionResultCode]
		if !exists {
			return "", consts.ErrorLoanAvailmentFailed
		}
		return "", errorMessage
	}

	//status code is not 200
	var CreateSessionErrorResponse models.ErrorResponse
	err = xml.Unmarshal(body, &CreateSessionErrorResponse)
	if err != nil {
		logger.Error(ctx, "Error while unmarshalling response from Create Session: %v", err.Error())
		return "", err
	}
	logger.Info(ctx, "Create session Api call error Response in XML:%v ", CreateSessionErrorResponse)
	return "", errors.New(CreateSessionErrorResponse.Body.Fault.Detail.Text)
}

var ResultErrorCodeMessages = map[int]*serviceModels.CustomError{
	1: {Message: "Session could not be generated", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_ERROR_GENERATING_SESSION"},
	2: {Message: "Username/password is invalid", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_CREDENTIAL_INVALID"},
	3: {Message: "Insufficient funds to process transaction", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_INSUFFICIENT_FUNDS"},
	4: {Message: "Invalid card number", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_INVALID_CARD_NUMBER"},
	5: {Message: "Invalid amount", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_INVALID_AMOUNT"},
	6: {Message: "Operation failed", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_OPERATION_FAILED"},
	7: {Message: "No permission for this operation", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_NO_PERMISSION"},
	8: {Message: "Invalid product keyword", Code: "DODRIO2_LOAN_AVAILMENT_REQUEST_TOPUP_INVALID_KEYWORD"},
}

func (h *AmaxService) LoginSession(ctx context.Context, session string) (int, int, error) {

	//endpointURL
	endpointURL := configs.AMAX_LOGIN_SESSION_URL

	//request payload
	requestPayload := `<soapEnv:Envelope xmlns:soapEnv="http://schemas.xmlsoap.org/soap/envelope/">									
		<soapEnv:Body>									
	  		<login xmlns="http://soap.amaretto.globe.com">									
			<session>%s</session>									
			<user>%s</user>									
			<password>%s</password>									
	  		</login>									
		</soapEnv:Body>									
   </soapEnv:Envelope>`

	user := configs.LOGIN_SESSION_USER
	password := configs.LOGIN_SESSION_PASSWORD

	//generating encrypted password
	userpass := user + password

	hash := sha1.New()
	hash.Write([]byte(userpass))
	digest := hash.Sum(nil)
	encrypyted_password := hex.EncodeToString(digest)

	userpass2 := session + strings.ToLower(encrypyted_password)

	hash = sha1.New()
	hash.Write([]byte(userpass2))
	digest = hash.Sum(nil)
	encrypyted_password2 := hex.EncodeToString(digest)

	payload := fmt.Sprintf(requestPayload, session, user, encrypyted_password2)
	logger.Info(ctx, "Login session Api call payload :%v", payload)
	//creating new request
	req, err := http.NewRequest("POST", endpointURL, strings.NewReader(payload))
	if err != nil {
		logger.Error(ctx, "Error while creating a login session API request: %v", err.Error())
		return -1, -1, err
	}
	logger.Info(ctx, "Login session Api call request :%v", req)
	body, resp, err := ApiCall(req)
	if err != nil {
		logger.Error(ctx, "Error while reading response from AMAX Login Session response: %v", err.Error())
		return -1, -1, err
	}
	logger.Info(ctx, "Login session Api call response Code:%v", resp.StatusCode)
	if resp.StatusCode == 200 {
		var LoginSessionResponse models.LoginSessionSuccessResponse
		err = xml.Unmarshal(body, &LoginSessionResponse)
		if err != nil {
			logger.Error(ctx, "Error while unmarshalling the response into struct in login session API: %v", err.Error())
			return -1, -1, err
		}
		logger.Info(ctx, "Login session Api call Response in XML:%v ", LoginSessionResponse)
		//session is valid
		if LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode == 0 {
			//Transaction is successfull
			if LoginSessionResponse.Body.LoginResponse.Return.ResultCode == 0 {
				return LoginSessionResponse.Body.LoginResponse.Return.ResultCode, LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode, nil
			}
			//result code is not zero
			errorMessage, exists := ResultErrorCodeMessages[LoginSessionResponse.Body.LoginResponse.Return.ResultCode]

			if !exists {
				return -1, LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode, consts.ErrorLoanAvailmentFailed
			}

			return -1, LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode, errorMessage
		}
		//session result code is not 0
		errorMessage, exists := SessionResultCodeErrorMessages[LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode]
		if !exists {
			return -1, LoginSessionResponse.Body.LoginResponse.Return.SessionResultCode, consts.ErrorLoanAvailmentFailed
		}
		return -1, -1, errorMessage
	}
	//status code is not 200
	var LoginSessionErrorResponse models.ErrorResponse
	err = xml.Unmarshal(body, &LoginSessionErrorResponse)
	if err != nil {
		logger.Error(ctx, "Error while unmarshalling the response into struct in login session API: %v", err.Error())
		return -1, -1, err
	}
	logger.Info(ctx, "Login session Api call error Response in XML:%v ", LoginSessionErrorResponse)
	return -1, -1, errors.New(LoginSessionErrorResponse.Body.Fault.Detail.Text)

}

func (h *AmaxService) Topup(ctx context.Context, sessionId string, extTransID int64, target string, product string, amount float64) (resultCode int, sessionResultCode int, transId int, e error) {

	requestPayload := `
	<soapEnv:Envelope xmlns:soapEnv="http://schemas.xmlsoap.org/soap/envelope/">										
  		<soapEnv:Body>										
    		<topup xmlns="http://soap.amaretto.globe.com">										
				<session>%s</session>
				<extTransID>%d</extTransID>		
				<target>%s</target>	
				<product>%s</product>	
				<amount>%.2f</amount>								
    		</topup>										
  		</soapEnv:Body>										
	</soapEnv:Envelope>`

	payload := fmt.Sprintf(requestPayload, sessionId, extTransID, target, product, amount)
	logger.Debug(ctx, "payload of topup", payload)

	endpointURL := configs.AMAX_TOPUP_URL
	logger.Info(ctx, "TopUp Api call payload :%v", payload)
	req, err := http.NewRequest("POST", endpointURL, strings.NewReader(payload))
	if err != nil {
		logger.Error(ctx, "Error while creating a topup API request: %v", err.Error())
		return -1, -1, -1, err
	}
	logger.Info(ctx, "TopUp Api call request :%v", req)
	body, resp, err := ApiCall(req)
	if err != nil {
		logger.Error(ctx, "Error while reading response from AMAX Login Session response: %v", err.Error())
		return -1, -1, -1, err
	}
	logger.Info(ctx, "TopUp Api call response Code:%v", resp.StatusCode)
	if resp.StatusCode == 200 {
		var Response models.TopUpSuccessResponse
		err = xml.Unmarshal(body, &Response)
		if err != nil {
			logger.Error(ctx, "Error while unmarshalling the response into struct in topup: %v", err.Error())
			return -1, -1, -1, err
		}
		logger.Info(ctx, "TopUp Api call Response in XML:%v ", Response)
		//check if session result code is 0
		if Response.Body.TopupResponse.Return.SessionResultCode == 0 {
			//if result code is zero
			if Response.Body.TopupResponse.Return.ResultCode == 0 {
				return 0, 0, Response.Body.TopupResponse.Return.TransId, nil
			}
			//result code is not zero
			errorMessage, exists := ResultErrorCodeMessages[Response.Body.TopupResponse.Return.ResultCode]

			if !exists {
				return -1, Response.Body.TopupResponse.Return.SessionResultCode, -1, consts.ErrorLoanAvailmentFailed
			}

			return -1, Response.Body.TopupResponse.Return.SessionResultCode, -1, errorMessage
		}
		//session result code is not zero
		logger.Info(ctx, "Response.Body.TopupResponse.Return.SessionResultCode", Response.Body.TopupResponse.Return.SessionResultCode)
		errorMessage, exists := SessionResultCodeErrorMessages[Response.Body.TopupResponse.Return.SessionResultCode]
		if !exists {
			return -1, -1, -1, consts.ErrorLoanAvailmentFailed
		}
		return -1, -1, -1, errorMessage

	}
	//status code is not 200
	var ErrorResponse models.ErrorResponse
	err = xml.Unmarshal(body, &ErrorResponse)
	if err != nil {
		logger.Error(ctx, "Error while unmarshalling the response into struct: %v", err.Error())
	}
	logger.Info(ctx, "Toup Up Api call error Response in XML:%v ", ErrorResponse)
	return -1, -1, -1, errors.New(ErrorResponse.Body.Fault.Detail.Text)
}

func ApiCall(req *http.Request) ([]byte, *http.Response, error) {

	amaxXAPIKey := configs.AMAX_X_API_KEY
	// req.Header.Set("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("Content-Type", "application/xml")
	req.Header.Add("x-api-key", amaxXAPIKey)
	// Create a context with a timeout of TimeoutInSecs seconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.TIMEOUT_IN_SECONDS)*time.Second)
	defer cancel()
	client := &http.Client{}
	// Load CA certificate (Root CA) for server validation
	logger.Debug("http amax client certificate required : %v ", configs.AMAX_CA_CERTIFICATE_REQUIRED)
	if configs.AMAX_CA_CERTIFICATE_REQUIRED {
		rootCA := configs.AMAX_CA_CERTIFICATE

		// Create a CA certificate pool and add the server's CA
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rootCA)); !ok {
			logger.Error(ctx, consts.ErrorFailedAppendCACertificate)
		}

		// Setup TLS configuration to verify the server certificate
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}

		// Create an HTTP transport with the TLS configuration
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		client = &http.Client{Transport: tr}
		logger.Debug("http amax client: %v", client)
	}

	// Create an HTTP client with the transport
	// Associate the context with the request
	req = req.WithContext(ctx)
	resp, err := client.Do(req)

	if err != nil {
		logger.Error(ctx, "Error while making API request to AMAX API: %v", err.Error())
		return []byte(""), resp, consts.ErrorLoanAvailmentTimeout
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, "Error while reading response from AMAX API response: %v", err.Error())
		return []byte(""), resp, err
	}
	return body, resp, nil
}

func (h *AmaxService) ProvisionLoanAmax(ctx context.Context, msisdn string, transactionId string, loanProduct *serviceModels.LoanProduct, keyword string, channel string, systemClient string, productsNameInAmax string, creditScore float64, brand_type string, brandId primitive.ObjectID, startTime time.Time) error {

	MsisdnStruct := serviceModels.ActiveLoanRequest{
		MSISDN: msisdn,
	}
	logger.Info(MsisdnStruct, "Create session Api call for msisdn: %v ", msisdn)
	//calling create session API
	sessionId, err := h.CreateSession(ctx)
	if err != nil {
		logger.Error(ctx, "Error in Create Session API: %v", err)
		availmentDB, loanProductDB, servicingPartnerNumberString, response, error := h.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, err.Error(), consts.ErrorLoanAvailmentFailed.Code, false, creditScore, loanProduct, brand_type)
		go func(ctx context.Context) {
			err_pubsub := h.kafkaService.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, loanProductDB.Name, keyword, servicingPartnerNumberString, loanProductDB.Price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
			if err_pubsub == nil {
				updateField := bson.M{"publishedToKafka": true}
				h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
			}
		}(ctx)
		h.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
		if error != nil {
			logger.Error(ctx, err)
		}
		logger.Info(ctx, response)
		//Notification service
		go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedSystemFailure, brandId, loanProduct, primitive.NilObjectID, time.Time{})

		transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
		logger.Info(ctx, transaction)
		return err
	}

	//calling login session API - result code, session result code response paramters
	logger.Info(MsisdnStruct, "Login session Api call for msisdn : %v", msisdn)
	_, _, err = h.LoginSession(ctx, sessionId)
	if err != nil {
		logger.Error(ctx, "Error in Login Session API: %v ", err)
		availmentDB, loanProductDB, servicingPartnerNumberString, response, error := h.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, err.Error(), consts.ErrorLoanAvailmentFailed.Code, false, creditScore, loanProduct, brand_type)
		go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedSystemFailure, brandId, loanProduct, primitive.NilObjectID, time.Time{})

		go func(ctx context.Context) {
			err_obj := h.kafkaService.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, loanProductDB.Name, keyword, servicingPartnerNumberString, loanProductDB.Price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
			if err_obj == nil {
				updateField := bson.M{"publishedToKafka": true}
				h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
			}
		}(ctx)

		go h.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
		if error != nil {
			logger.Error(ctx, error)
		}
		logger.Info(ctx, response)
		transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
		logger.Info(ctx, transaction)
		return err
	}

	//paramters for topup API - result code, session result code, trans ID response paramters
	target := msisdn
	// amount := session.Get("price").(int)
	// amount := 0 //TODO: Confirm this.
	extTransID, _ := strconv.ParseInt(transactionId, 10, 64)
	logger.Info(MsisdnStruct, "Toup Up call for msisdn : %v", msisdn)
	//calling Topup API
	resultCode, sessionResultCode, transId, errObj := h.Topup(ctx, sessionId, extTransID, target, productsNameInAmax, loanProduct.Price)
	if errObj != nil {
		logger.Error(ctx, "Error in Topup API: %v ", errObj)
		logger.Info(ctx, "Transaction Failed")
		availmentDB, _, servicingPartnerNumberString, response, error := h.availmentRepo.InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, errObj.Error(), utils.GetErrorCode(errObj), false, creditScore, loanProduct, brand_type)
		go func(ctx context.Context) {
			errOfKafka := h.kafkaService.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, loanProduct.Name, keyword, servicingPartnerNumberString, loanProduct.Price, availmentDB.ServiceFee, consts.FailedAvailmentResult, availmentDB.ErrorCode, 0, "")
			if errOfKafka == nil {
				updateField := bson.M{"publishedToKafka": true}
				h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
			}
		}(ctx)
		logger.Debug(ctx, errObj, errObj != consts.ErrorLoanAvailmentTimeout)
		if errObj != consts.ErrorLoanAvailmentTimeout {
			h.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
			go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedSystemFailure, brandId, loanProduct, primitive.NilObjectID, time.Time{})

			return errObj
		}

		if error != nil {
			logger.Error(ctx, errObj)
		}
		logger.Info(ctx, response)
		go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentFailedTimeOut, brandId, loanProduct, primitive.NilObjectID, time.Time{})

		transaction := common.SerializeAvailmentLoanLog(msisdn, consts.LoggerFailedAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
		logger.Info(ctx, transaction)
		return errObj
	}

	logger.Info(ctx, resultCode, sessionResultCode, transId)
	logger.Info(ctx, "Success")
	availmentDB, loanProductDB, servicingPartnerNumberString, response, error := h.availmentRepo.InsertSuccessfulAvailment(ctx, msisdn, loanProduct, keyword, channel, systemClient, "", "", true, creditScore, brand_type, brandId)
	go func(ctx context.Context) {
		e := h.kafkaService.PublishAvailmentStatusToKafka(ctx, availmentDB.GUID, channel, msisdn, availmentDB.Brand, availmentDB.LoanType, loanProductDB.Name, keyword, servicingPartnerNumberString, loanProductDB.Price, availmentDB.ServiceFee, consts.SuccessAvailmentResult, availmentDB.ErrorCode, 0, consts.StatusOfProvisionedLoan)
		if e == nil {
			updateField := bson.M{"publishedToKafka": true}
			h.availmentRepo.UpdateAvailment(ctx, availmentDB.ID, updateField)
		}
	}(ctx)
	h.transactionInProgressRepo.DeleteTransactionInProgressByMsisdn(ctx, msisdn)
	if error != nil {
		logger.Error(ctx, error)
	}
	logger.Info(ctx, response)
	go h.notificationService.NotifyUser(ctx, msisdn, consts.LoanAvailmentSuccess, brandId, loanProduct, primitive.NilObjectID, time.Time{})
	transaction := common.SerializeAvailmentLoanLog(msisdn, consts.SuccessAvailmentResult, startTime, channel, availmentDB.GUID, availmentDB.ErrorCode)
	logger.Info(ctx, transaction)
	return nil

}
