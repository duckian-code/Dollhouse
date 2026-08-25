package moodstatus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsAPI interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// SQSPublisher publishes versioned notification jobs to the configured FIFO
// queue. The event ID is also the SQS deduplication ID, making SDK retries safe.
type SQSPublisher struct {
	client   sqsAPI
	queueURL string
}

func NewSQSPublisher(client sqsAPI, queueURL string) *SQSPublisher {
	return &SQSPublisher{client: client, queueURL: queueURL}
}

func (p *SQSPublisher) Publish(ctx context.Context, job NotificationJob) error {
	if p.client == nil || p.queueURL == "" {
		return errors.New("notification queue configuration is incomplete")
	}
	if job.EventID == "" || job.SenderUserID == "" || job.CorrelationID == "" || job.CreatedAt == "" {
		return errors.New("notification job identifiers are incomplete")
	}
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode notification job: %w", err)
	}
	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               &p.queueURL,
		MessageBody:            strptr(string(body)),
		MessageDeduplicationId: &job.EventID,
		MessageGroupId:         &job.SenderUserID,
	})
	if err != nil {
		return fmt.Errorf("send notification job: %w", err)
	}
	return nil
}
