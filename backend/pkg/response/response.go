// Package response creates consistent API Gateway responses.
package response

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

const contentType = "application/json"

// ErrorBody is the public error envelope returned by API handlers.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains a stable machine-readable code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON serializes body and creates an API Gateway v2 response.
func JSON(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"content-type": contentType},
		Body:       string(encoded),
	}, nil
}

// Error creates a structured error response.
func Error(statusCode int, code, message string) events.APIGatewayV2HTTPResponse {
	result, err := JSON(statusCode, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
	if err != nil {
		panic(err)
	}
	return result
}
