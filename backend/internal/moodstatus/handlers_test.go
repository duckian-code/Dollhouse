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
	moodItems       []MoodEntry
	moodNextToken   string
	nextToken       string
	requestedToken  string
	err             error
	recipients      []string
	recipientErr    error
}

func (s *fakeStore) PublishMood(_ context.Context, userID, eventID string, state MoodState) error {
	s.publishedUserID, s.publishedEvent, s.publishedState = userID, eventID, state
	return s.err
}

func (s *fakeStore) ListFriendStatuses(_ context.Context, userID, token string) ([]FriendStatus, string, error) {
	s.publishedUserID, s.requestedToken = userID, token
	return s.items, s.nextToken, s.err
}

func (s *fakeStore) ListMoods(_ context.Context, userID, token string) ([]MoodEntry, string, error) {
	s.publishedUserID, s.requestedToken = userID, token
	return s.moodItems, s.moodNextToken, s.err
}

func (s *fakeStore) ListNotificationRecipientIDs(_ context.Context, _ string) ([]string, error) {
	return s.recipients, s.recipientErr
}

type fakePublisher struct {
	jobs []NotificationJob
	err  error
}

func (p *fakePublisher) Publish(_ context.Context, job NotificationJob) error {
	p.jobs = append(p.jobs, job)
	return p.err
}

func authenticatedRequest(body string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{Body: body}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": "user-123"}},
	}
	return request
}

func fixedHandlers(store Store) *Handlers {
	handlers := NewHandlers(store, &fakePublisher{})
	handlers.now = func() time.Time { return time.Date(2026, 8, 16, 12, 30, 45, 0, time.FixedZone("test", -6*60*60)) }
	handlers.newID = func() (string, error) { return "event-123", nil }
	return handlers
}

func TestPublishMoodReturnsContractShapeAndPreservesNullSliders(t *testing.T) {
	store := &fakeStore{recipients: []string{"friend-1", "friend-2"}}
	publisher := &fakePublisher{}
	handlers := fixedHandlers(store)
	handlers.publisher = publisher
	request := authenticatedRequest(`{"status":" Feeling okay ","stress":3,"fatigue":null}`)
	request.Headers = map[string]string{"X-Correlation-Id": "request-456"}
	got, err := handlers.PublishMood(context.Background(), request)
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
	if len(publisher.jobs) != 1 {
		t.Fatalf("jobs=%#v", publisher.jobs)
	}
	job := publisher.jobs[0]
	if job.SchemaVersion != 1 || job.EventID != "event-123" || job.SenderUserID != "user-123" || job.CorrelationID != "request-456" || job.CreatedAt != "2026-08-16T18:30:45Z" || len(job.RecipientUserIDs) != 2 {
		t.Fatalf("job=%#v", job)
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

func TestEveryMoodAndStatusRouteRequiresAuthentication(t *testing.T) {
	handlers := fixedHandlers(&fakeStore{})
	tests := map[string]func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error){
		"POST /moods":          handlers.PublishMood,
		"GET /moods":           handlers.GetMoods,
		"GET /friend-statuses": handlers.GetFriendStatuses,
	}
	for name, invoke := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := invoke(context.Background(), events.APIGatewayV2HTTPRequest{})
			if err != nil || got.StatusCode != http.StatusUnauthorized || errorCode(got.Body) != "unauthenticated" {
				t.Fatalf("response = %#v, err = %v", got, err)
			}
		})
	}
}

func TestGetMoodsReturnsPaginationShape(t *testing.T) {
	stress := 2
	store := &fakeStore{moodNextToken: "next", moodItems: []MoodEntry{{
		EventID: "event-1",
		MoodState: MoodState{
			Status: "okay", Stress: &stress, UpdatedAt: "2026-08-16T18:30:45Z",
		},
	}}}
	request := authenticatedRequest("")
	request.QueryStringParameters = map[string]string{"nextToken": "prior"}
	got, _ := fixedHandlers(store).GetMoods(context.Background(), request)
	if got.StatusCode != http.StatusOK || store.publishedUserID != "user-123" || store.requestedToken != "prior" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	var body struct {
		Data struct {
			Items     []MoodEntry `json:"items"`
			NextToken *string     `json:"nextToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].EventID != "event-1" || body.Data.NextToken == nil || *body.Data.NextToken != "next" {
		t.Fatalf("body=%#v", body)
	}
}

func TestGetMoodsReturnsNullNextTokenWhenHistoryEnds(t *testing.T) {
	got, _ := fixedHandlers(&fakeStore{}).GetMoods(context.Background(), authenticatedRequest(""))
	var body struct {
		Data struct {
			NextToken *string `json:"nextToken"`
		} `json:"data"`
	}
	if got.StatusCode != http.StatusOK || json.Unmarshal([]byte(got.Body), &body) != nil || body.Data.NextToken != nil {
		t.Fatalf("response=%#v body=%#v", got, body)
	}
}

func TestGetFriendStatusesReturnsPaginationShape(t *testing.T) {
	stress := 2
	doll := &DollConfiguration{BodyAssetID: "body", ClothingAssetIDs: []string{}}
	store := &fakeStore{nextToken: "next", items: []FriendStatus{{
		Friend: UserSummary{UserID: "friend-1", Username: "alex", DisplayName: "Alex"},
		Doll:   doll,
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

func TestGetFriendStatusesOmitsMissingDollConfiguration(t *testing.T) {
	store := &fakeStore{items: []FriendStatus{{
		Friend: UserSummary{UserID: "friend-1", Username: "alex", DisplayName: "Alex"},
	}}}

	got, err := fixedHandlers(store).GetFriendStatuses(context.Background(), authenticatedRequest(""))
	if err != nil || got.StatusCode != http.StatusOK {
		t.Fatalf("response=%#v err=%v", got, err)
	}
	var body struct {
		Data struct {
			Items []map[string]json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 {
		t.Fatalf("body=%#v", body)
	}
	if _, exists := body.Data.Items[0]["doll"]; exists {
		t.Fatalf("doll should be omitted: %s", got.Body)
	}
	if status, exists := body.Data.Items[0]["status"]; !exists || string(status) != "null" {
		t.Fatalf("status should remain explicit null: %s", got.Body)
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

func TestPublishMoodReturnsInternalErrorWhenQueueingFails(t *testing.T) {
	store := &fakeStore{recipients: []string{"friend-1"}}
	publisher := &fakePublisher{err: errors.New("queue secret")}
	handlers := fixedHandlers(store)
	handlers.publisher = publisher
	got, err := handlers.PublishMood(context.Background(), authenticatedRequest(`{"status":"okay"}`))
	if err != nil || got.StatusCode != http.StatusInternalServerError || errorCode(got.Body) != "internal_error" || len(publisher.jobs) != 1 {
		t.Fatalf("response=%#v err=%v jobs=%#v", got, err, publisher.jobs)
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
