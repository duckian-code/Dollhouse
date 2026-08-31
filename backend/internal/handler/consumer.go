package handler

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dollhouse-app/dollhouse/backend/internal/config"
	"github.com/dollhouse-app/dollhouse/backend/internal/notificationconsumer"
)

// StartNotificationConsumer starts the SQS-triggered notification Lambda.
func StartNotificationConsumer() {
	handler, err := notificationconsumer.NewRuntimeHandler(context.Background(), config.Load())
	if err != nil {
		log.Fatalf("initialize notification consumer: %v", err)
	}
	lambda.Start(handler)
}
