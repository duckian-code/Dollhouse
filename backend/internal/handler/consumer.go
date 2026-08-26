package handler

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// StartNotificationConsumer starts the SQS-triggered notification Lambda.
func StartNotificationConsumer() {
	lambda.Start(func(_ context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
		failures := make([]events.SQSBatchItemFailure, 0, len(event.Records))
		for _, record := range event.Records {
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
		return events.SQSEventResponse{BatchItemFailures: failures}, nil
	})
}
