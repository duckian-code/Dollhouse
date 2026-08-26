package moodstatus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type fakeSQS struct {
	input *sqs.SendMessageInput
	err   error
}

func (f *fakeSQS) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.input = input
	return &sqs.SendMessageOutput{}, f.err
}

func TestSQSPublisherSendsIdentifierOnlyFIFOMessage(t *testing.T) {
	client := &fakeSQS{}
	job := NotificationJob{
		SchemaVersion: 1, EventID: "event-1", SenderUserID: "sender-1",
		RecipientUserIDs: []string{"recipient-1"}, CorrelationID: "request-1", CreatedAt: "2026-08-18T16:00:00Z",
	}
	if err := NewSQSPublisher(client, "queue-url").Publish(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if client.input == nil || *client.input.QueueUrl != "queue-url" || *client.input.MessageDeduplicationId != "event-1" || *client.input.MessageGroupId != "sender-1" {
		t.Fatalf("input=%#v", client.input)
	}
	for _, sensitiveField := range []string{"status", "stress", "fatigue", "discomfort", "pushToken"} {
		if strings.Contains(*client.input.MessageBody, sensitiveField) {
			t.Fatalf("message contains sensitive field %q: %s", sensitiveField, *client.input.MessageBody)
		}
	}
	var decoded NotificationJob
	if err := json.Unmarshal([]byte(*client.input.MessageBody), &decoded); err != nil || decoded.EventID != job.EventID || len(decoded.RecipientUserIDs) != 1 {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
}

func TestSQSPublisherReturnsConfigurationAndSendFailures(t *testing.T) {
	job := NotificationJob{EventID: "event", SenderUserID: "sender", CorrelationID: "correlation", CreatedAt: "now"}
	if err := NewSQSPublisher(&fakeSQS{}, "").Publish(context.Background(), job); err == nil {
		t.Fatal("expected missing queue URL to fail")
	}
	client := &fakeSQS{err: errors.New("unavailable")}
	if err := NewSQSPublisher(client, "queue").Publish(context.Background(), job); err == nil || !strings.Contains(err.Error(), "send notification job") {
		t.Fatalf("err=%v", err)
	}
}
