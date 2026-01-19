
package models

type RequestDetails struct {
    RequestID      string                 `json:"requestId"`
    IP             string                 `json:"ip"`
    UserAgent      string                 `json:"userAgent"`
    HTTPMethod     string                 `json:"httpMethod"`
    Path           string                 `json:"path"`
    OperationName  string                 `json:"operationName"`
    Status         int                    `json:"status,omitempty"`
    RequestTime    string                 `json:"requestTime"`
    ResponseTime   string                 `json:"responseTime,omitempty"`
    RequestParams  map[string]interface{} `json:"requestParameters"`
    ResponseParams map[string]interface{} `json:"responseParameters,omitempty"`
}