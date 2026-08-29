package notificationconsumer

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	appconfig "github.com/dollhouse-app/dollhouse/backend/internal/config"
)

// NewRuntimeHandler wires the production DynamoDB and Expo clients.
func NewRuntimeHandler(ctx context.Context, cfg appconfig.Config) (func(context.Context, events.SQSEvent) (events.SQSEventResponse, error), error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	consumer := New(
		NewDynamoDBStore(dynamodb.NewFromConfig(awsCfg), cfg.DevicesTableName),
		NewHTTPPushService(httpClient, cfg.ExpoPushAccessToken),
		logger,
	)
	return consumer.Handle, nil
}
