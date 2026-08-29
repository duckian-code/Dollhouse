package notificationconsumer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

type fakeStore struct {
	devices    []Device
	listErr    error
	disabled   []Device
	disableErr error
}

func (f *fakeStore) ListEnabledDevices(_ context.Context, _ []string) ([]Device, error) {
	return f.devices, f.listErr
}

func (f *fakeStore) DisableDevice(_ context.Context, device Device) error {
	f.disabled = append(f.disabled, device)
	return f.disableErr
}

type fakePushService struct {
	batches [][]Message
	send    func([]Message) ([]Ticket, error)
}

func (f *fakePushService) Send(_ context.Context, messages []Message) ([]Ticket, error) {
	f.batches = append(f.batches, append([]Message(nil), messages...))
	if f.send != nil {
		return f.send(messages)
	}
	tickets := make([]Ticket, len(messages))
	for i := range tickets {
		tickets[i] = Ticket{Status: "ok", ID: "ticket"}
	}
	return tickets, nil
}

func TestHandleReportsOnlyFailedSQSRecords(t *testing.T) {
	store := &fakeStore{devices: []Device{{UserID: "recipient", DeviceID: "device", PushToken: "ExpoPushToken[secret]"}}}
	push := &fakePushService{}
	response, err := New(store, push, nil).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "good", Body: validJobBody()},
		{MessageId: "bad", Body: `{}`},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.BatchItemFailures) != 1 || response.BatchItemFailures[0].ItemIdentifier != "bad" {
		t.Fatalf("failures=%#v", response.BatchItemFailures)
	}
	if len(push.batches) != 1 || len(push.batches[0]) != 1 {
		t.Fatalf("batches=%#v", push.batches)
	}
	message := push.batches[0][0]
	if message.Body != "A friend shared a new mood." || message.Data["eventId"] != "event-1" {
		t.Fatalf("message=%#v", message)
	}
}

func TestHandleStopsAfterFailureToPreserveFIFOOrder(t *testing.T) {
	store := &fakeStore{devices: []Device{{UserID: "recipient", DeviceID: "device", PushToken: "token"}}}
	push := &fakePushService{}
	response, err := New(store, push, nil).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{
		{MessageId: "failed", Body: `{}`},
		{MessageId: "unprocessed", Body: validJobBody()},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.BatchItemFailures) != 2 || response.BatchItemFailures[0].ItemIdentifier != "failed" || response.BatchItemFailures[1].ItemIdentifier != "unprocessed" {
		t.Fatalf("failures=%#v", response.BatchItemFailures)
	}
	if len(push.batches) != 0 {
		t.Fatalf("later FIFO record was processed: %#v", push.batches)
	}
}

func TestHandleDisablesDeviceNotRegisteredWithoutLoggingToken(t *testing.T) {
	const token = "ExpoPushToken[do-not-log]"
	store := &fakeStore{devices: []Device{{UserID: "recipient", DeviceID: "device", PushToken: token}}}
	push := &fakePushService{send: func([]Message) ([]Ticket, error) {
		return []Ticket{{Status: "error", Details: TicketDetails{Error: "DeviceNotRegistered"}}}, nil
	}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	response, err := New(store, push, logger).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "message", Body: validJobBody()}}})
	if err != nil || len(response.BatchItemFailures) != 0 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if len(store.disabled) != 1 || store.disabled[0].PushToken != token {
		t.Fatalf("disabled=%#v", store.disabled)
	}
	if strings.Contains(logs.String(), token) {
		t.Fatalf("logs contain push token: %s", logs.String())
	}
	for _, field := range []string{`"lifecycle":"started"`, `"lifecycle":"completed"`, `"invalidated":1`} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("logs missing %s: %s", field, logs.String())
		}
	}
}

func TestHandleChunksExpoRequestsAtOneHundred(t *testing.T) {
	store := &fakeStore{}
	for i := 0; i < 101; i++ {
		store.devices = append(store.devices, Device{UserID: "recipient", DeviceID: "device", PushToken: "token"})
	}
	push := &fakePushService{}
	response, err := New(store, push, nil).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "message", Body: validJobBody()}}})
	if err != nil || len(response.BatchItemFailures) != 0 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if len(push.batches) != 2 || len(push.batches[0]) != 100 || len(push.batches[1]) != 1 {
		t.Fatalf("batch sizes=%v", []int{len(push.batches[0]), len(push.batches[1])})
	}
}

func TestHandleRetriesStorageAndExpoFailures(t *testing.T) {
	tests := []struct {
		name  string
		store *fakeStore
		push  *fakePushService
	}{
		{name: "device lookup", store: &fakeStore{listErr: errors.New("unavailable")}, push: &fakePushService{}},
		{name: "Expo call", store: &fakeStore{devices: []Device{{UserID: "u", DeviceID: "d", PushToken: "t"}}}, push: &fakePushService{send: func([]Message) ([]Ticket, error) { return nil, errors.New("unavailable") }}},
		{name: "Expo ticket", store: &fakeStore{devices: []Device{{UserID: "u", DeviceID: "d", PushToken: "t"}}}, push: &fakePushService{send: func([]Message) ([]Ticket, error) {
			return []Ticket{{Status: "error", Details: TicketDetails{Error: "InvalidCredentials"}}}, nil
		}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := New(test.store, test.push, nil).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "retry", Body: validJobBody()}}})
			if err != nil || len(response.BatchItemFailures) != 1 || response.BatchItemFailures[0].ItemIdentifier != "retry" {
				t.Fatalf("response=%#v err=%v", response, err)
			}
		})
	}
}

func TestDecodeJobRejectsInvalidAndDeduplicatesRecipients(t *testing.T) {
	for _, body := range []string{
		`not-json`,
		`{"schemaVersion":2,"eventId":"e","senderUserId":"s","recipientUserIds":[],"correlationId":"c","createdAt":"now"}`,
		`{"schemaVersion":1,"eventId":"e","senderUserId":"s","recipientUserIds":[""],"correlationId":"c","createdAt":"now"}`,
		validJobBody() + `{}`,
	} {
		if _, err := decodeJob(body); err == nil {
			t.Fatalf("expected body to fail: %s", body)
		}
	}
	job, err := decodeJob(`{"schemaVersion":1,"eventId":"e","senderUserId":"s","recipientUserIds":["u","u"],"correlationId":"c","createdAt":"now"}`)
	if err != nil || len(job.RecipientUserIDs) != 1 {
		t.Fatalf("job=%#v err=%v", job, err)
	}
}

func validJobBody() string {
	return `{"schemaVersion":1,"eventId":"event-1","senderUserId":"sender","recipientUserIds":["recipient"],"correlationId":"request-1","createdAt":"2026-08-29T20:00:00Z"}`
}
