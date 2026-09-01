// Package moodstatus implements mood publishing and accepted-friend status reads.
package moodstatus

import "context"

const acceptedStatus = "ACCEPTED"

// MoodState is the current mood and optional disclosed state values.
type MoodState struct {
	Status     string `json:"status" dynamodbav:"status"`
	Stress     *int   `json:"stress" dynamodbav:"stress"`
	Fatigue    *int   `json:"fatigue" dynamodbav:"fatigue"`
	Discomfort *int   `json:"discomfort" dynamodbav:"discomfort"`
	UpdatedAt  string `json:"updatedAt" dynamodbav:"updatedAt"`
}

// UserSummary is the public identity included with a friend's status.
type UserSummary struct {
	UserID      string `json:"userId" dynamodbav:"userId"`
	Username    string `json:"username" dynamodbav:"username"`
	DisplayName string `json:"displayName" dynamodbav:"displayName"`
}

// DollConfiguration contains the approved asset IDs used to render a doll.
type DollConfiguration struct {
	BodyAssetID      string   `json:"bodyAssetId" dynamodbav:"bodyAssetId"`
	HairAssetID      string   `json:"hairAssetId" dynamodbav:"hairAssetId"`
	EyesAssetID      string   `json:"eyesAssetId" dynamodbav:"eyesAssetId"`
	NoseAssetID      string   `json:"noseAssetId" dynamodbav:"noseAssetId"`
	MouthAssetID     string   `json:"mouthAssetId" dynamodbav:"mouthAssetId"`
	ClothingAssetIDs []string `json:"clothingAssetIds" dynamodbav:"clothingAssetIds"`
	UpdatedAt        string   `json:"updatedAt" dynamodbav:"updatedAt"`
}

// FriendStatus combines the friend's public identity, optional doll, and current status.
type FriendStatus struct {
	Friend UserSummary        `json:"friend"`
	Doll   *DollConfiguration `json:"doll,omitempty"`
	Status *MoodState         `json:"status"`
}

// Store is the persistence boundary used by mood and status handlers.
type Store interface {
	PublishMood(context.Context, string, string, MoodState) error
	ListNotificationRecipientIDs(context.Context, string) ([]string, error)
	ListFriendStatuses(context.Context, string, string) ([]FriendStatus, string, error)
}

// NotificationJob is the identifier-only message sent after a mood is saved.
// It intentionally excludes the free-form status and slider values.
type NotificationJob struct {
	SchemaVersion    int      `json:"schemaVersion"`
	EventID          string   `json:"eventId"`
	SenderUserID     string   `json:"senderUserId"`
	RecipientUserIDs []string `json:"recipientUserIds"`
	CorrelationID    string   `json:"correlationId"`
	CreatedAt        string   `json:"createdAt"`
}

// NotificationPublisher queues asynchronous notification work.
type NotificationPublisher interface {
	Publish(context.Context, NotificationJob) error
}
