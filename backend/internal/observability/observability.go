// Package observability provides privacy-safe structured logging, request
// correlation, and CloudWatch Embedded Metric Format output.
package observability

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	metricNamespace       = "Dollhouse"
	maxCorrelationIDBytes = 128
)

type contextKey struct{}

// RequestContext contains identifiers that are safe to include in operational
// logs. It deliberately excludes request bodies, identity claims, and tokens.
type RequestContext struct {
	CorrelationID string
	RequestID     string
}

// WithRequest stores request identifiers and a request-scoped logger.
func WithRequest(ctx context.Context, request RequestContext, logger *slog.Logger) context.Context {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("correlationId", request.CorrelationID, "requestId", request.RequestID)
	return context.WithValue(ctx, contextKey{}, struct {
		request RequestContext
		logger  *slog.Logger
	}{request: request, logger: logger})
}

// Request returns the operational identifiers associated with ctx.
func Request(ctx context.Context) RequestContext {
	value, _ := ctx.Value(contextKey{}).(struct {
		request RequestContext
		logger  *slog.Logger
	})
	return value.request
}

// Logger returns the request-scoped logger when one is available.
func Logger(ctx context.Context) *slog.Logger {
	value, ok := ctx.Value(contextKey{}).(struct {
		request RequestContext
		logger  *slog.Logger
	})
	if ok && value.logger != nil {
		return value.logger
	}
	return slog.Default()
}

// CorrelationID prefers a caller-supplied ID, then API Gateway's request ID.
// IDs are length-limited before they are echoed or logged.
func CorrelationID(request events.APIGatewayV2HTTPRequest) string {
	for name, value := range request.Headers {
		if strings.EqualFold(name, "x-correlation-id") {
			if value = strings.TrimSpace(value); value != "" {
				return truncate(value, maxCorrelationIDBytes)
			}
		}
	}
	return truncate(strings.TrimSpace(request.RequestContext.RequestID), maxCorrelationIDBytes)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

// Metric is one value in a CloudWatch Embedded Metric Format event.
type Metric struct {
	Name  string
	Value float64
	Unit  string
}

// Emit writes an EMF event. Properties must contain only non-sensitive
// operational identifiers or aggregate counts.
func Emit(ctx context.Context, metrics ...Metric) {
	_ = WriteMetrics(os.Stdout, os.Getenv("APP_ENV"), Request(ctx), metrics...)
}

// WriteMetrics writes a single EMF event and is exported for focused tests.
func WriteMetrics(writer io.Writer, environment string, request RequestContext, metrics ...Metric) error {
	definitions := make([]map[string]string, 0, len(metrics))
	payload := map[string]any{
		"environment":   environment,
		"correlationId": request.CorrelationID,
		"requestId":     request.RequestID,
	}
	for _, metric := range metrics {
		definitions = append(definitions, map[string]string{"Name": metric.Name, "Unit": metric.Unit})
		payload[metric.Name] = metric.Value
	}
	payload["_aws"] = map[string]any{
		"Timestamp": time.Now().UnixMilli(),
		"CloudWatchMetrics": []any{map[string]any{
			"Namespace":  metricNamespace,
			"Dimensions": [][]string{{"environment"}},
			"Metrics":    definitions,
		}},
	}
	return json.NewEncoder(writer).Encode(payload)
}
