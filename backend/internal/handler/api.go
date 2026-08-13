// Package handler provides the common Lambda runtime adapters used by commands.
package handler

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

// API is the request shape accepted by API Gateway v2 Lambda integrations.
type API func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

// StartAPI starts a named API Gateway Lambda. Feature tickets replace the
// placeholder behavior by passing a domain handler through this adapter.
func StartAPI(name string) {
	lambda.Start(func(_ context.Context, _ events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return response.Error(http.StatusNotImplemented, "not_implemented", name+" is not implemented"), nil
	})
}

// StartAPIHandler starts a fully implemented API Gateway Lambda handler.
func StartAPIHandler(api API) { lambda.Start(api) }
