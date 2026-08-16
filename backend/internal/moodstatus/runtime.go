package moodstatus

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	appconfig "github.com/dollhouse-app/dollhouse/backend/internal/config"
)

// NewRuntimeHandlers wires mood/status handlers to DynamoDB.
func NewRuntimeHandlers(ctx context.Context, cfg appconfig.Config) (*Handlers, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	store := NewDynamoDBStore(dynamodb.NewFromConfig(awsCfg), cfg.UsersTableName, cfg.FriendshipsTableName, cfg.MoodEventsTableName)
	return NewHandlers(store), nil
}
