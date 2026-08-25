package profiledoll

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dollhouse-app/dollhouse/backend/internal/assetcatalog"
	appconfig "github.com/dollhouse-app/dollhouse/backend/internal/config"
)

// NewRuntimeHandlers wires the feature handlers to the deployed Users table.
func NewRuntimeHandlers(ctx context.Context, cfg appconfig.Config) (*Handlers, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	store := NewDynamoDBStore(dynamodb.NewFromConfig(awsCfg), cfg.UsersTableName)
	assetValidator := assetcatalog.NewValidator(s3.NewFromConfig(awsCfg), cfg.AssetsBucketName, cfg.AssetCatalogKey)
	return NewHandlers(store, assetValidator), nil
}
