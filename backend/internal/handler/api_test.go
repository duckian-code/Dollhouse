package handler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestObserveAPIAddsCorrelationHeaderAndStructuredLifecycleLog(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	wrapped := ObserveAPI(func(_ context.Context, _ events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return events.APIGatewayV2HTTPResponse{StatusCode: 204}, nil
	}, logger)
	request := events.APIGatewayV2HTTPRequest{RouteKey: "GET /profile", Headers: map[string]string{"x-correlation-id": "trace-me"}}
	request.RequestContext.RequestID = "gateway-request"
	request.RequestContext.HTTP.Method = "GET"
	response, err := wrapped(context.Background(), request)
	if err != nil || response.Headers["x-correlation-id"] != "trace-me" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	for _, field := range []string{`"msg":"request completed"`, `"correlationId":"trace-me"`, `"requestId":"gateway-request"`, `"route":"GET /profile"`, `"status":204`} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("logs missing %s: %s", field, logs.String())
		}
	}
}

func TestObserveAPIRecordsHandlerErrors(t *testing.T) {
	var logs bytes.Buffer
	wrapped := ObserveAPI(func(_ context.Context, _ events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500}, errors.New("failure")
	}, slog.New(slog.NewJSONHandler(&logs, nil)))
	_, _ = wrapped(context.Background(), events.APIGatewayV2HTTPRequest{})
	if !strings.Contains(logs.String(), `"handlerError":true`) {
		t.Fatalf("logs = %s", logs.String())
	}
}
