package friendship

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

const maxRequestBodyBytes = 32 * 1024

// Handlers implements the friendship and user-search Lambda routes.
type Handlers struct {
	store Store
	now   func() time.Time
	newID func() (string, error)
}

// NewHandlers creates handlers with production clock and identifier behavior.
func NewHandlers(store Store) *Handlers {
	return &Handlers{store: store, now: time.Now, newID: randomID}
}

// SearchUsers searches usernames by case-insensitive prefix.
func (h *Handlers) SearchUsers(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return *failure, nil
	}
	query := strings.TrimSpace(request.QueryStringParameters["q"])
	if query == "" {
		return response.Error(http.StatusBadRequest, "validation_failed", "q must be a non-empty string"), nil
	}
	if utf8.RuneCountInString(query) > 50 {
		return response.Error(http.StatusBadRequest, "validation_failed", "q must be 50 characters or fewer"), nil
	}
	items, nextToken, err := h.store.SearchUsers(ctx, query, request.QueryStringParameters["nextToken"], userID)
	if err != nil {
		return h.failure("search users", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"items": items, "nextToken": nullableToken(nextToken)}})
}

type sendRequestBody struct {
	UserID string `json:"userId"`
}

// SendFriendRequest creates mirrored pending relationship records.
func (h *Handlers) SendFriendRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return *failure, nil
	}
	var input sendRequestBody
	if failure = decodeBody(request, &input); failure != nil {
		return *failure, nil
	}
	input.UserID = strings.TrimSpace(input.UserID)
	if input.UserID == "" {
		return response.Error(http.StatusBadRequest, "validation_failed", "userId must be a non-empty string"), nil
	}
	if input.UserID == userID {
		return response.Error(http.StatusBadRequest, "validation_failed", "users cannot friend themselves"), nil
	}
	requestID, err := h.newID()
	if err != nil {
		return internalError("generate friend request ID", err), nil
	}
	friendRequest, err := h.store.SendRequest(ctx, userID, input.UserID, requestID, h.timestamp())
	if err != nil {
		return h.failure("send friend request", err), nil
	}
	return response.JSON(http.StatusCreated, map[string]any{"data": map[string]any{"friendRequest": friendRequest}})
}

// ListFriendRequests returns pending requests grouped by direction.
func (h *Handlers) ListFriendRequests(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return *failure, nil
	}
	incoming, outgoing, err := h.store.ListRequests(ctx, userID)
	if err != nil {
		return h.failure("list friend requests", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"incoming": incoming, "outgoing": outgoing}})
}

// AcceptFriendRequest accepts only a request addressed to the signed-in user.
func (h *Handlers) AcceptFriendRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, requestID, failure := routeIdentity(request, "requestId")
	if failure != nil {
		return *failure, nil
	}
	friendship, err := h.store.AcceptRequest(ctx, userID, requestID, h.timestamp())
	if err != nil {
		return h.failure("accept friend request", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"friendship": friendship}})
}

// DeclineFriendRequest removes both pending records atomically.
func (h *Handlers) DeclineFriendRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, requestID, failure := routeIdentity(request, "requestId")
	if failure != nil {
		return *failure, nil
	}
	if err := h.store.DeclineRequest(ctx, userID, requestID); err != nil {
		return h.failure("decline friend request", err), nil
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNoContent}, nil
}

// RemoveFriend removes both accepted records atomically.
func (h *Handlers) RemoveFriend(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, friendID, failure := routeIdentity(request, "friendId")
	if failure != nil {
		return *failure, nil
	}
	if friendID == userID {
		return response.Error(http.StatusBadRequest, "validation_failed", "friendId cannot identify the signed-in user"), nil
	}
	if err := h.store.RemoveFriend(ctx, userID, friendID); err != nil {
		return h.failure("remove friend", err), nil
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNoContent}, nil
}

func routeIdentity(request events.APIGatewayV2HTTPRequest, parameter string) (string, string, *events.APIGatewayV2HTTPResponse) {
	userID, failure := authenticatedUserID(request)
	if failure != nil {
		return "", "", failure
	}
	value := strings.TrimSpace(request.PathParameters[parameter])
	if value == "" {
		failure := response.Error(http.StatusBadRequest, "invalid_request", parameter+" is required")
		return "", "", &failure
	}
	return userID, value, nil
}

func authenticatedUserID(request events.APIGatewayV2HTTPRequest) (string, *events.APIGatewayV2HTTPResponse) {
	if request.RequestContext.Authorizer == nil || request.RequestContext.Authorizer.JWT == nil {
		failure := response.Error(http.StatusUnauthorized, "unauthenticated", "valid Cognito authentication is required")
		return "", &failure
	}
	userID := strings.TrimSpace(request.RequestContext.Authorizer.JWT.Claims["sub"])
	if userID == "" {
		failure := response.Error(http.StatusUnauthorized, "unauthenticated", "valid Cognito authentication is required")
		return "", &failure
	}
	return userID, nil
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

func (h *Handlers) failure(operation string, err error) events.APIGatewayV2HTTPResponse {
	var domain *DomainError
	if errors.As(err, &domain) {
		return response.Error(domain.Status, domain.Code, domain.Message)
	}
	return internalError(operation, err)
}

func internalError(operation string, err error) events.APIGatewayV2HTTPResponse {
	log.Printf("%s: %v", operation, err)
	return response.Error(http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func (h *Handlers) timestamp() string { return h.now().UTC().Format(time.RFC3339) }

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func nullableToken(token string) any {
	if token == "" {
		return nil
	}
	return token
}
