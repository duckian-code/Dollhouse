package assetcatalog

import (
	"context"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/dollhouse-app/dollhouse/backend/internal/config"
)

// NewRuntimeHandler wires the route to the deployed private asset bucket.
func NewRuntimeHandler(ctx context.Context, cfg appconfig.Config) (*Handler, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	client := s3.NewFromConfig(awsCfg)
	service := NewService(client, s3.NewPresignClient(client), cfg.AssetsBucketName, cfg.AssetCatalogKey, time.Duration(cfg.AssetURLTTLSeconds)*time.Second)
	return NewHandler(service), nil
}
