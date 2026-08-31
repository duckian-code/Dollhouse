package friendship

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
	searchItems     []UserSummary
	searchToken     string
	incoming        []FriendRequest
	outgoing        []FriendRequest
	request         FriendRequest
	friendship      Friendship
	err             error
	searchQuery     string
	searchPageToken string
	searchExclude   string
	fromUserID      string
	toUserID        string
	requestID       string
	actionUserID    string
	friendID        string
}

func (s *fakeStore) SearchUsers(_ context.Context, query, token, exclude string) ([]UserSummary, string, error) {
	s.searchQuery, s.searchPageToken, s.searchExclude = query, token, exclude
	return s.searchItems, s.searchToken, s.err
}
func (s *fakeStore) SendRequest(_ context.Context, from, to, requestID, _ string) (FriendRequest, error) {
	s.fromUserID, s.toUserID, s.requestID = from, to, requestID
	return s.request, s.err
}
func (s *fakeStore) ListRequests(_ context.Context, userID string) ([]FriendRequest, []FriendRequest, error) {
	s.actionUserID = userID
	return s.incoming, s.outgoing, s.err
}
func (s *fakeStore) AcceptRequest(_ context.Context, userID, requestID, _ string) (Friendship, error) {
	s.actionUserID, s.requestID = userID, requestID
	return s.friendship, s.err
}
func (s *fakeStore) DeclineRequest(_ context.Context, userID, requestID string) error {
	s.actionUserID, s.requestID = userID, requestID
	return s.err
}
func (s *fakeStore) RemoveFriend(_ context.Context, userID, friendID string) error {
	s.actionUserID, s.friendID = userID, friendID
	return s.err
}

func authenticatedRequest(body string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{Body: body}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": "user-123"}}}
	return request
}

func fixedHandlers(store Store) *Handlers {
	h := NewHandlers(store)
	h.now = func() time.Time { return time.Date(2026, 8, 16, 12, 30, 45, 0, time.FixedZone("test", -6*60*60)) }
	h.newID = func() (string, error) { return "request-123", nil }
	return h
}

func TestSearchUsersUsesAuthenticatedUserAndPagination(t *testing.T) {
	store := &fakeStore{searchItems: []UserSummary{{UserID: "user-456", Username: "alex", DisplayName: "Alex"}}, searchToken: "next"}
	request := authenticatedRequest("")
	request.QueryStringParameters = map[string]string{"q": " Al ", "nextToken": "prior"}
	got, err := fixedHandlers(store).SearchUsers(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusOK || store.searchQuery != "Al" || store.searchPageToken != "prior" || store.searchExclude != "user-123" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	var body struct {
		Data struct {
			Items     []UserSummary `json:"items"`
			NextToken *string       `json:"nextToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 || body.Data.NextToken == nil || *body.Data.NextToken != "next" {
		t.Fatalf("body=%#v", body)
	}
}

func TestSearchUsersValidatesAuthenticationAndQuery(t *testing.T) {
	got, _ := fixedHandlers(&fakeStore{}).SearchUsers(context.Background(), events.APIGatewayV2HTTPRequest{})
	if got.StatusCode != http.StatusUnauthorized || errorCode(got.Body) != "unauthenticated" {
		t.Fatalf("response=%#v", got)
	}
	got, _ = fixedHandlers(&fakeStore{}).SearchUsers(context.Background(), authenticatedRequest(""))
	if got.StatusCode != http.StatusBadRequest || errorCode(got.Body) != "validation_failed" {
		t.Fatalf("response=%#v", got)
	}
}

func TestEveryFriendshipRouteRequiresAuthentication(t *testing.T) {
	handlers := fixedHandlers(&fakeStore{})
	tests := map[string]func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error){
		"GET /users/search":                  handlers.SearchUsers,
		"POST /friend-requests":              handlers.SendFriendRequest,
		"GET /friend-requests":               handlers.ListFriendRequests,
		"POST /friend-requests/{id}/accept":  handlers.AcceptFriendRequest,
		"POST /friend-requests/{id}/decline": handlers.DeclineFriendRequest,
		"DELETE /friends/{id}":               handlers.RemoveFriend,
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

func TestSendFriendRequestReturnsContractShape(t *testing.T) {
	store := &fakeStore{request: FriendRequest{RequestID: "request-123", User: UserSummary{UserID: "user-456"}, Status: StatusPendingOutgoing, RequestedAt: "2026-08-16T18:30:45Z"}}
	got, err := fixedHandlers(store).SendFriendRequest(context.Background(), authenticatedRequest(`{"userId":"user-456"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusCreated || store.fromUserID != "user-123" || store.toUserID != "user-456" || store.requestID != "request-123" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	var body struct {
		Data struct {
			FriendRequest FriendRequest `json:"friendRequest"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.FriendRequest.Status != StatusPendingOutgoing {
		t.Fatalf("body=%#v", body)
	}
}

func TestSendFriendRequestRejectsSelfAndDuplicate(t *testing.T) {
	store := &fakeStore{}
	got, _ := fixedHandlers(store).SendFriendRequest(context.Background(), authenticatedRequest(`{"userId":"user-123"}`))
	if got.StatusCode != http.StatusBadRequest || store.fromUserID != "" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	store.err = &DomainError{Status: http.StatusConflict, Code: "conflict", Message: "duplicate"}
	got, _ = fixedHandlers(store).SendFriendRequest(context.Background(), authenticatedRequest(`{"userId":"user-456"}`))
	if got.StatusCode != http.StatusConflict || errorCode(got.Body) != "conflict" {
		t.Fatalf("response=%#v", got)
	}
}

func TestListFriendRequestsKeepsDirectionsSeparate(t *testing.T) {
	store := &fakeStore{incoming: []FriendRequest{{RequestID: "in", Status: StatusPendingIncoming}}, outgoing: []FriendRequest{{RequestID: "out", Status: StatusPendingOutgoing}}}
	got, _ := fixedHandlers(store).ListFriendRequests(context.Background(), authenticatedRequest(""))
	if got.StatusCode != http.StatusOK || store.actionUserID != "user-123" {
		t.Fatalf("response=%#v", got)
	}
	var body struct {
		Data struct {
			Incoming []FriendRequest `json:"incoming"`
			Outgoing []FriendRequest `json:"outgoing"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Incoming) != 1 || len(body.Data.Outgoing) != 1 {
		t.Fatalf("body=%#v", body)
	}
}

func TestAcceptDeclineAndRemoveUseAuthenticatedRouteIdentity(t *testing.T) {
	store := &fakeStore{friendship: Friendship{Friend: UserSummary{UserID: "user-456"}, Status: StatusAccepted}}
	request := authenticatedRequest("")
	request.PathParameters = map[string]string{"requestId": "request-1"}
	got, _ := fixedHandlers(store).AcceptFriendRequest(context.Background(), request)
	if got.StatusCode != http.StatusOK || store.actionUserID != "user-123" || store.requestID != "request-1" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
	got, _ = fixedHandlers(store).DeclineFriendRequest(context.Background(), request)
	if got.StatusCode != http.StatusNoContent {
		t.Fatalf("response=%#v", got)
	}
	request.PathParameters = map[string]string{"friendId": "user-456"}
	got, _ = fixedHandlers(store).RemoveFriend(context.Background(), request)
	if got.StatusCode != http.StatusNoContent || store.friendID != "user-456" {
		t.Fatalf("response=%#v store=%#v", got, store)
	}
}

func TestAuthorizationAndStorageErrorsAreSafe(t *testing.T) {
	store := &fakeStore{err: &DomainError{Status: http.StatusForbidden, Code: "forbidden", Message: "recipient only"}}
	request := authenticatedRequest("")
	request.PathParameters = map[string]string{"requestId": "request-1"}
	got, _ := fixedHandlers(store).AcceptFriendRequest(context.Background(), request)
	if got.StatusCode != http.StatusForbidden || errorCode(got.Body) != "forbidden" {
		t.Fatalf("response=%#v", got)
	}
	store.err = errors.New("secret database detail")
	got, _ = fixedHandlers(store).ListFriendRequests(context.Background(), authenticatedRequest(""))
	if got.StatusCode != http.StatusInternalServerError || errorCode(got.Body) != "internal_error" || json.Valid([]byte(got.Body)) == false {
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
