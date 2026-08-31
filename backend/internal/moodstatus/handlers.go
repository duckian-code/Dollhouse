package moodstatus

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/internal/authorization"
	"github.com/dollhouse-app/dollhouse/backend/internal/observability"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

const maxRequestBodyBytes = 32 * 1024

// DomainError maps an expected storage/domain failure to an API response.
type DomainError struct {
	Status  int
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }

// Handlers implements the mood publishing and friend-status routes.
type Handlers struct {
	store     Store
	publisher NotificationPublisher
	now       func() time.Time
	newID     func() (string, error)
}

// NewHandlers creates handlers with production clock and identifier behavior.
func NewHandlers(store Store, publisher NotificationPublisher) *Handlers {
	return &Handlers{store: store, publisher: publisher, now: time.Now, newID: randomID}
}

type publishMoodRequest struct {
	Status     json.RawMessage `json:"status"`
	Stress     json.RawMessage `json:"stress"`
	Fatigue    json.RawMessage `json:"fatigue"`
	Discomfort json.RawMessage `json:"discomfort"`
}

// PublishMood validates and saves a new current status and history event.
func (h *Handlers) PublishMood(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return *failure, nil
	}
	var input publishMoodRequest
	if failure = decodeBody(request, &input); failure != nil {
		return *failure, nil
	}
	state, message := validateMood(input)
	if message != "" {
		return response.Error(http.StatusBadRequest, "validation_failed", message), nil
	}
	eventID, err := h.newID()
	if err != nil {
		return internalError(ctx, "generate mood event ID", err), nil
	}
	state.UpdatedAt = h.timestamp()
	if err := h.store.PublishMood(ctx, userID, eventID, state); err != nil {
		return h.failure(ctx, "publish mood", err), nil
	}
	recipientIDs, err := h.store.ListNotificationRecipientIDs(ctx, userID)
	if err != nil {
		return internalError(ctx, "list notification recipients", err), nil
	}
	job := NotificationJob{
		SchemaVersion: 1, EventID: eventID, SenderUserID: userID,
		RecipientUserIDs: recipientIDs, CorrelationID: correlationID(request, eventID),
		CreatedAt: state.UpdatedAt,
	}
	if err := h.publisher.Publish(ctx, job); err != nil {
		return internalError(ctx, "publish notification job", err), nil
	}
	observability.Emit(ctx,
		observability.Metric{Name: "MoodUpdatesPublished", Value: 1, Unit: "Count"},
		observability.Metric{Name: "NotificationJobsCreated", Value: 1, Unit: "Count"},
		observability.Metric{Name: "NotificationRecipientsQueued", Value: float64(len(recipientIDs)), Unit: "Count"},
	)
	return response.JSON(http.StatusCreated, map[string]any{"data": map[string]any{"eventId": eventID, "status": state}})
}

// GetFriendStatuses returns the current state of accepted friends only.
func (h *Handlers) GetFriendStatuses(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return *failure, nil
	}
	items, nextToken, err := h.store.ListFriendStatuses(ctx, userID, request.QueryStringParameters["nextToken"])
	if err != nil {
		return h.failure(ctx, "get friend statuses", err), nil
	}
	var token any
	if nextToken != "" {
		token = nextToken
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"items": items, "nextToken": token}})
}

func validateMood(input publishMoodRequest) (MoodState, string) {
	var status string
	if len(input.Status) == 0 || bytes.Equal(bytes.TrimSpace(input.Status), []byte("null")) {
		return MoodState{}, "status is required and must be a non-empty string"
	}
	if err := json.Unmarshal(input.Status, &status); err != nil || strings.TrimSpace(status) == "" {
		return MoodState{}, "status is required and must be a non-empty string"
	}
	status = strings.TrimSpace(status)
	stress, message := parseSlider("stress", input.Stress)
	if message != "" {
		return MoodState{}, message
	}
	fatigue, message := parseSlider("fatigue", input.Fatigue)
	if message != "" {
		return MoodState{}, message
	}
	discomfort, message := parseSlider("discomfort", input.Discomfort)
	if message != "" {
		return MoodState{}, message
	}
	return MoodState{Status: status, Stress: stress, Fatigue: fatigue, Discomfort: discomfort}, ""
}

func parseSlider(name string, raw json.RawMessage) (*int, string) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, ""
	}
	value, err := strconv.Atoi(string(trimmed))
	if err != nil {
		return nil, name + " must be an integer between 0 and 10"
	}
	if value < 0 || value > 10 {
		return nil, name + " must be between 0 and 10"
	}
	return &value, ""
}

func authenticatedUserID(request events.APIGatewayV2HTTPRequest) (string, *events.APIGatewayV2HTTPResponse) {
	principal, failure := authorization.Authenticate(request)
	if failure != nil {
		return "", failure
	}
	return principal.Subject, nil
}

func decodeBody(request events.APIGatewayV2HTTPRequest, target any) *events.APIGatewayV2HTTPResponse {
	body := []byte(request.Body)
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			failure := response.Error(http.StatusBadRequest, "invalid_request", "request body is not valid base64")
			return &failure
		}
		body = decoded
	}
	if len(body) == 0 || len(body) > maxRequestBodyBytes {
		message := "request body is required"
		if len(body) > maxRequestBodyBytes {
			message = "request body is too large"
		}
		failure := response.Error(http.StatusBadRequest, "invalid_request", message)
		return &failure
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body must be valid JSON: "+err.Error())
		return &failure
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return &failure
	}
	return nil
}

func (h *Handlers) failure(ctx context.Context, operation string, err error) events.APIGatewayV2HTTPResponse {
	var domain *DomainError
	if errors.As(err, &domain) {
		return response.Error(domain.Status, domain.Code, domain.Message)
	}
	return internalError(ctx, operation, err)
}

func internalError(ctx context.Context, operation string, err error) events.APIGatewayV2HTTPResponse {
	observability.Logger(ctx).ErrorContext(ctx, "mood/status request failed", "operation", operation, "error", err)
	return response.Error(http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func (h *Handlers) timestamp() string {
	return h.now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func correlationID(request events.APIGatewayV2HTTPRequest, fallback string) string {
	if value := observability.CorrelationID(request); value != "" {
		return value
	}
	return fallback
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
