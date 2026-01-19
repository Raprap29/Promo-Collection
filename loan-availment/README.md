# About the Project
This backend service is used to handle Globe prepaid subscribers when they will avail a loan

# Built With
Go

# More information here
https://globetelecom.atlassian.net/wiki/spaces/SDSE/pages/5110334145/Dodrio+v2

# How to setup local environment
To setup a local environment for this service you may reuse some of the steps for loan-collection
https://globetelecom.atlassian.net/wiki/spaces/SDSE/pages/5424185464/How+to+setup+Dodrio+2+Local+Environment

<br />

# API List
1. Eligibility Check API  
3. Assurance Availment report API  *( Cron Triggered )*

<br />

# API IFS

<br />

#### **Eligibility Check API**

##### **Endpoint:** /IntegrationServices/Dodrio/EligibilityCheck

##### **Method:** `POST`

##### Description:

This endpoint fetches the credit score of the user and checks whether the MSISDN is eligible for loan availment or not.

##### Request Body:

* **Content-Type:** `application/json`

##### Example Request Body:

{  
   "MSISDN":"1068030297"  
}

##### Parameters:

* **msisdn** (string): A MSISDN (Mobile Station International Subscriber Directory Numbers) for which we need to check the loan eligibility

Response : 

| HTTP Status Code | Description | Response Body Example |
| ----- | ----- | ----- |
| 200 OK | The channel passed (transactionId) is not present in the Collections table | {     "MSISDN": "6789012345",     "SubscriberScore": "50",     "ExistingLoan": false,     "LoanKeyWord": "",     "LoanAmount": 0,     "Brand": "",     "SKU availed": "",     "Date of Availment": "",     "Outstanding Principal Amount": 0,     "Outstanding Service Fee": 0 }  |
| 400 Bad Request | MSISDN is not in correct format or we are not sending it in the request body | {    "error": "Key: 'EligibilityCheckRequest.MSISDN' Error:Field validation for 'MSISDN' failed on the 'required' tag" } |



<br /><br /><br />



#### **Availment Assurance report**

##### **Endpoint:** `/IntegrationServices/Dodrio/AvailmentReports`

##### **Method:** `GET`

##### Description:

This endpoint is to get the details of availments transactions happened in 24 hours , it would generate a report for all availments for the previous day .Once the report generation is completed a report would be uploaded in Google Cloud storage bucket in the specified format.

Request Body:

* **Content-Type:** `application/json`

##### Response : 

| HTTP Status Code | Description | Response Body Example |
| ----- | ----- | ----- |
| 200 OK | Report generation request received successfully  | Loan not found case: `{ "result":"Report Generation request recieved" }`   |
| 500 Internal Server Error | An unexpected error occurred on the server while processing the request. |  `{ "error": "Error Details" }  `  |



# Release
---
Anything commited / merged to DEV branch will initiate:
- auto-release as newly created TAG
- auto-deploy into the argonaut-helm-chart for this specific Platform
- auto-testing based on Jira-Xray-defined Test Cases found in https://gitlab.com/globetelecom/platforms/reloads-service/testing/dodrio-test-tools
---

 
