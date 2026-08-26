// Package friendship implements user discovery and the friendship lifecycle.
package friendship

import "context"

const (
	StatusPendingIncoming = "PENDING_INCOMING"
	StatusPendingOutgoing = "PENDING_OUTGOING"
	StatusAccepted        = "ACCEPTED"
)

// UserSummary is the public user shape used by friendship APIs.
type UserSummary struct {
	UserID      string `json:"userId" dynamodbav:"userId"`
	Username    string `json:"username" dynamodbav:"username"`
	DisplayName string `json:"displayName" dynamodbav:"displayName"`
}

// FriendRequest describes one incoming or outgoing pending relationship.
type FriendRequest struct {
	RequestID   string      `json:"requestId"`
	User        UserSummary `json:"user"`
	Status      string      `json:"status"`
	RequestedAt string      `json:"requestedAt"`
}

// Friendship describes an accepted relationship.
type Friendship struct {
	Friend     UserSummary `json:"friend"`
	Status     string      `json:"status"`
	AcceptedAt string      `json:"acceptedAt"`
}

// Store is the persistence boundary used by friendship handlers.
type Store interface {
	SearchUsers(context.Context, string, string, string) ([]UserSummary, string, error)
	SendRequest(context.Context, string, string, string, string) (FriendRequest, error)
	ListRequests(context.Context, string) ([]FriendRequest, []FriendRequest, error)
	AcceptRequest(context.Context, string, string, string) (Friendship, error)
	DeclineRequest(context.Context, string, string) error
	RemoveFriend(context.Context, string, string) error
}

// DomainError maps an expected persistence/domain failure to an API response.
type DomainError struct {
	Status  int
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }
