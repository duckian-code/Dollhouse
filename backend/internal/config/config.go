// Package config loads backend resource names and runtime settings from the
// environment. It does not read dotenv files; local tooling may export one.
package config

import "os"

// Config contains shared settings used by Lambda handlers.
type Config struct {
	Environment          string
	AWSRegion            string
	UsersTableName       string
	MoodsTableName       string
	AssetsBucketName     string
	NotificationQueueURL string
}

// Load reads configuration from environment variables. Defaults are limited to
// values that are safe and useful for local development.
func Load() Config {
	return Config{
		Environment:          getOrDefault("APP_ENV", "local"),
		AWSRegion:            getOrDefault("AWS_REGION", "us-west-2"),
		UsersTableName:       os.Getenv("USERS_TABLE_NAME"),
		MoodsTableName:       os.Getenv("MOODS_TABLE_NAME"),
		AssetsBucketName:     os.Getenv("ASSETS_BUCKET_NAME"),
		NotificationQueueURL: os.Getenv("NOTIFICATION_QUEUE_URL"),
	}
}

func getOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
