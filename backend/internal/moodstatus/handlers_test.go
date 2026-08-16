package moodstatus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type fakeStore struct {
	publishedUserID string
	publishedEvent  string
	publishedState  MoodState
	items           []FriendStatus
	nextToken       string
	requestedToken  string
	err             error
}

func (s *fakeStore) PublishMood(_ context.Context, userID, eventID string, state MoodState) error {
	s.publishedUserID, s.publishedEvent, s.publishedState = userID, eventID, state
	return s.err
}

func (s *fakeStore) ListFriendStatuses(_ context.Context, userID, token string) ([]FriendStatus, string, error) {
	s.publishedUserID, s.requestedToken = userID, token
	return s.items, s.nextToken, s.err
}

func authenticatedRequest(body string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{Body: body}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": "user-123"}},
	}
	return request
}

func fixedHandlers(store Store) *Handlers {
	handlers := NewHandlers(store)
	handlers.now = func() time.Time { return time.Date(2026, 8, 16, 12, 30, 45, 0, time.FixedZone("test", -6*60*60)) }
	handlers.newID = func() (string, error) { return "event-123", nil }
	return handlers
}

func TestPublishMoodReturnsContractShapeAndPreservesNullSliders(t *testing.T) {
	store := &fakeStore{}
	got, err := fixedHandlers(store).PublishMood(context.Background(), authenticatedRequest(`{"status":" Feeling okay ","stress":3,"fatigue":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusCreated || store.publishedUserID != "user-123" || store.publishedEvent != "event-123" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	if store.publishedState.Status != "Feeling okay" || store.publishedState.Stress == nil || *store.publishedState.Stress != 3 || store.publishedState.Fatigue != nil || store.publishedState.Discomfort != nil || store.publishedState.UpdatedAt != "2026-08-16T18:30:45Z" {
		t.Fatalf("state=%#v", store.publishedState)
	}
	var body struct {
		Data struct {
			EventID string    `json:"eventId"`
			Status  MoodState `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.EventID != "event-123" || body.Data.Status.Stress == nil || body.Data.Status.Fatigue != nil {
		t.Fatalf("body=%#v", body)
	}
}

func TestPublishMoodValidatesAuthenticationBodyAndSliders(t *testing.T) {
	handlers := fixedHandlers(&fakeStore{})
	tests := []struct {
		name string
		req  events.APIGatewayV2HTTPRequest
		code string
	}{
		{name: "authentication", req: events.APIGatewayV2HTTPRequest{}, code: "unauthenticated"},
		{name: "missing status", req: authenticatedRequest(`{"stress":1}`), code: "validation_failed"},
		{name: "blank status", req: authenticatedRequest(`{"status":"  "}`), code: "validation_failed"},
		{name: "fractional slider", req: authenticatedRequest(`{"status":"ok","stress":1.5}`), code: "validation_failed"},
		{name: "high slider", req: authenticatedRequest(`{"status":"ok","fatigue":11}`), code: "validation_failed"},
		{name: "unknown field", req: authenticatedRequest(`{"status":"ok","moodId":"retired"}`), code: "invalid_request"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := handlers.PublishMood(context.Background(), test.req)
			if got.StatusCode != http.StatusBadRequest && test.code != "unauthenticated" || errorCode(got.Body) != test.code {
				t.Fatalf("response=%#v", got)
			}
		})
	}
}

func TestGetFriendStatusesReturnsPaginationShape(t *testing.T) {
	stress := 2
	store := &fakeStore{nextToken: "next", items: []FriendStatus{{
		Friend: UserSummary{UserID: "friend-1", Username: "alex", DisplayName: "Alex"},
		Doll:   DollConfiguration{BodyAssetID: "body", ClothingAssetIDs: []string{}},
		Status: &MoodState{Status: "okay", Stress: &stress, UpdatedAt: "2026-08-16T18:30:45Z"},
	}}}
	request := authenticatedRequest("")
	request.QueryStringParameters = map[string]string{"nextToken": "prior"}
	got, _ := fixedHandlers(store).GetFriendStatuses(context.Background(), request)
	if got.StatusCode != http.StatusOK || store.publishedUserID != "user-123" || store.requestedToken != "prior" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	var body struct {
		Data struct {
			Items     []FriendStatus `json:"items"`
			NextToken *string        `json:"nextToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 || body.Data.NextToken == nil || *body.Data.NextToken != "next" {
		t.Fatalf("body=%#v", body)
	}
}

func TestExpectedAndUnexpectedStoreErrorsAreSafe(t *testing.T) {
	store := &fakeStore{err: &DomainError{Status: http.StatusBadRequest, Code: "invalid_request", Message: "bad token"}}
	got, _ := fixedHandlers(store).GetFriendStatuses(context.Background(), authenticatedRequest(""))
	if got.StatusCode != http.StatusBadRequest || errorCode(got.Body) != "invalid_request" {
		t.Fatalf("response=%#v", got)
	}
	store.err = errors.New("database secret")
	got, _ = fixedHandlers(store).PublishMood(context.Background(), authenticatedRequest(`{"status":"okay"}`))
	if got.StatusCode != http.StatusInternalServerError || errorCode(got.Body) != "internal_error" {
		t.Fatalf("response=%#v", got)
	}
}

func errorCode(body string) string {
	var decoded struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(body), &decoded)
	return decoded.Error.Code
}
