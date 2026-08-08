package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("AWS_REGION", "")

	got := Load()
	if got.Environment != "local" {
		t.Fatalf("Environment = %q, want local", got.Environment)
	}
	if got.AWSRegion != "us-west-2" {
		t.Fatalf("AWSRegion = %q, want us-west-2", got.AWSRegion)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("USERS_TABLE_NAME", "users-test")
	t.Setenv("NOTIFICATION_QUEUE_URL", "https://example.test/queue")

	got := Load()
	if got.UsersTableName != "users-test" {
		t.Fatalf("UsersTableName = %q, want users-test", got.UsersTableName)
	}
	if got.NotificationQueueURL != "https://example.test/queue" {
		t.Fatalf("NotificationQueueURL = %q, want test URL", got.NotificationQueueURL)
	}
}
