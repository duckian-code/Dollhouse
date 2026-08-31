// Package handler provides the common Lambda runtime adapters used by commands.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dollhouse-app/dollhouse/backend/internal/observability"
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

// ObserveAPI adds privacy-safe structured lifecycle logs and correlation IDs.
func ObserveAPI(api API, logger *slog.Logger) API {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		started := time.Now()
		requestID := request.RequestContext.RequestID
		correlationID := observability.CorrelationID(request)
		ctx = observability.WithRequest(ctx, observability.RequestContext{
			CorrelationID: correlationID,
			RequestID:     requestID,
		}, logger)

		result, err := api(ctx, request)
		if result.Headers == nil {
			result.Headers = make(map[string]string)
		}
		if correlationID != "" {
			result.Headers["x-correlation-id"] = correlationID
		}
		observability.Logger(ctx).InfoContext(ctx, "request completed",
			"method", request.RequestContext.HTTP.Method,
			"route", request.RouteKey,
			"status", result.StatusCode,
			"durationMs", time.Since(started).Milliseconds(),
			"handlerError", err != nil)
		return result, err
	}
}

// StartAPIHandler starts a fully implemented, observed API Gateway handler.
func StartAPIHandler(api API) { lambda.Start(ObserveAPI(api, nil)) }
