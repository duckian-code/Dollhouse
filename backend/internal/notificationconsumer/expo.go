package notificationconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	expoPushEndpoint     = "https://exp.host/--/api/v2/push/send"
	maxExpoResponseBytes = 1 << 20
)

// HTTPPushService sends messages through the Expo Push Service HTTP API.
type HTTPPushService struct {
	client      *http.Client
	endpoint    string
	accessToken string
}

// NewHTTPPushService creates an Expo client. accessToken may be empty when
// enhanced push security is not enabled for the Expo project.
func NewHTTPPushService(client *http.Client, accessToken string) *HTTPPushService {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPPushService{client: client, endpoint: expoPushEndpoint, accessToken: accessToken}
}

type expoResponse struct {
	Data   []Ticket    `json:"data"`
	Errors []expoError `json:"errors"`
}

type expoError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Send submits one Expo-sized batch and returns ordered push tickets.
func (s *HTTPPushService) Send(ctx context.Context, messages []Message) ([]Ticket, error) {
	if len(messages) == 0 || len(messages) > maxExpoBatchSize {
		return nil, fmt.Errorf("Expo batch size must be between 1 and %d", maxExpoBatchSize)
	}
	body, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("encode Expo messages: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Expo request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if s.accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+s.accessToken)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call Expo Push Service: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxExpoResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Expo response: %w", err)
	}
	if len(responseBody) > maxExpoResponseBytes {
		return nil, errors.New("Expo response is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Expo returned HTTP %d", response.StatusCode)
	}
	var decoded expoResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode Expo response: %w", err)
	}
	if len(decoded.Errors) > 0 {
		codes := make([]string, 0, len(decoded.Errors))
		for _, item := range decoded.Errors {
			codes = append(codes, item.Code)
		}
		return nil, fmt.Errorf("Expo request errors: %s", strings.Join(codes, ","))
	}
	return decoded.Data, nil
}
