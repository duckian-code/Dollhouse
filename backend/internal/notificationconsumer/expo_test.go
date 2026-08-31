package notificationconsumer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHTTPPushServiceSendsMessagesAndDecodesTickets(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.String() != expoPushEndpoint || request.Header.Get("Authorization") != "Bearer access-token" || request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request method=%s url=%s headers=%v", request.Method, request.URL, request.Header)
		}
		var messages []Message
		if err := json.NewDecoder(request.Body).Decode(&messages); err != nil || len(messages) != 1 || messages[0].To != "token" {
			t.Errorf("messages=%#v err=%v", messages, err)
		}
		return testResponse(http.StatusOK, `{"data":[{"status":"ok","id":"ticket-1"}]}`), nil
	})}
	tickets, err := NewHTTPPushService(httpClient, "access-token").Send(context.Background(), []Message{{To: "token", Body: "body"}})
	if err != nil || len(tickets) != 1 || tickets[0].ID != "ticket-1" {
		t.Fatalf("tickets=%#v err=%v", tickets, err)
	}
}

func TestHTTPPushServiceRejectsBadBatchAndResponses(t *testing.T) {
	client := NewHTTPPushService(http.DefaultClient, "")
	if _, err := client.Send(context.Background(), nil); err == nil {
		t.Fatal("expected empty batch to fail")
	}
	tooLarge := make([]Message, maxExpoBatchSize+1)
	if _, err := client.Send(context.Background(), tooLarge); err == nil {
		t.Fatal("expected oversized batch to fail")
	}

	responses := []struct {
		name   string
		status int
		body   string
	}{
		{name: "HTTP failure", status: http.StatusTooManyRequests, body: `{}`},
		{name: "invalid JSON", status: http.StatusOK, body: `nope`},
		{name: "request error", status: http.StatusOK, body: `{"errors":[{"code":"TOO_MANY_REQUESTS","message":"slow down"}]}`},
	}
	for _, test := range responses {
		t.Run(test.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return testResponse(test.status, test.body), nil
			})}
			if _, err := NewHTTPPushService(httpClient, "").Send(context.Background(), []Message{{To: "token"}}); err == nil {
				t.Fatal("expected response to fail")
			}
		})
	}
}
