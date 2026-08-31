// Package notificationconsumer delivers queued mood notifications to enabled
// Expo devices.
package notificationconsumer

import "context"

// Job is the identifier-only payload published after a mood update is saved.
type Job struct {
	SchemaVersion    int      `json:"schemaVersion"`
	EventID          string   `json:"eventId"`
	SenderUserID     string   `json:"senderUserId"`
	RecipientUserIDs []string `json:"recipientUserIds"`
	CorrelationID    string   `json:"correlationId"`
	CreatedAt        string   `json:"createdAt"`
}

// Device is an enabled notification destination stored in DynamoDB.
type Device struct {
	UserID    string `dynamodbav:"userId"`
	DeviceID  string `dynamodbav:"deviceId"`
	PushToken string `dynamodbav:"pushToken"`
}

// Store resolves notification destinations and disables invalid ones.
type Store interface {
	ListEnabledDevices(context.Context, []string) ([]Device, error)
	DisableDevice(context.Context, Device) error
}

// Message is the non-sensitive payload sent to Expo.
type Message struct {
	To    string         `json:"to"`
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Sound string         `json:"sound,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

// Ticket records whether Expo accepted one message.
type Ticket struct {
	Status  string        `json:"status"`
	ID      string        `json:"id,omitempty"`
	Message string        `json:"message,omitempty"`
	Details TicketDetails `json:"details,omitempty"`
}

// TicketDetails contains Expo's machine-readable delivery error.
type TicketDetails struct {
	Error string `json:"error,omitempty"`
}

// PushService submits notification messages to Expo.
type PushService interface {
	Send(context.Context, []Message) ([]Ticket, error)
}
