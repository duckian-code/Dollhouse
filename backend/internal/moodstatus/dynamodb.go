package moodstatus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	userStatusIndex = "UserStatusIndex"
	statusPageSize  = 25
)

type dynamodbAPI interface {
	BatchGetItem(context.Context, *dynamodb.BatchGetItemInput, ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// ListNotificationRecipientIDs returns every accepted friend ID. DynamoDB may
// return the same relationship more than once across retried pages, so IDs are
// deduplicated before they enter a notification job.
func (s *DynamoDBStore) ListNotificationRecipientIDs(ctx context.Context, userID string) ([]string, error) {
	if s.friendshipsTable == "" {
		return nil, errors.New("friendship table configuration is incomplete")
	}
	input := &dynamodb.QueryInput{
		TableName: &s.friendshipsTable, IndexName: strptr(userStatusIndex),
		KeyConditionExpression: strptr("userId = :userId AND begins_with(statusRelatedUserId, :status)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: userID},
			":status": &types.AttributeValueMemberS{Value: acceptedStatus + "#"},
		},
		ProjectionExpression: strptr("relatedUserId"),
	}
	seen := map[string]struct{}{}
	recipients := make([]string, 0)
	for {
		output, err := s.client.Query(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("query notification recipients: %w", err)
		}
		for _, item := range output.Items {
			id, ok := stringAttribute(item["relatedUserId"])
			if !ok || id == "" {
				return nil, errors.New("accepted friendship has an invalid recipient ID")
			}
			if _, exists := seen[id]; !exists {
				seen[id] = struct{}{}
				recipients = append(recipients, id)
			}
		}
		if len(output.LastEvaluatedKey) == 0 {
			return recipients, nil
		}
		input.ExclusiveStartKey = output.LastEvaluatedKey
	}
}

// DynamoDBStore persists mood events and joins accepted relationships to users.
type DynamoDBStore struct {
	client           dynamodbAPI
	usersTable       string
	friendshipsTable string
	moodEventsTable  string
}

// NewDynamoDBStore creates a mood/status store over the existing tables.
func NewDynamoDBStore(client dynamodbAPI, usersTable, friendshipsTable, moodEventsTable string) *DynamoDBStore {
	return &DynamoDBStore{client: client, usersTable: usersTable, friendshipsTable: friendshipsTable, moodEventsTable: moodEventsTable}
}

type moodEvent struct {
	UserID     string `dynamodbav:"userId"`
	OccurredAt string `dynamodbav:"occurredAt"`
	EventID    string `dynamodbav:"eventId"`
	MoodState
	Visibility string `dynamodbav:"visibility"`
}

type relationshipItem struct {
	UserID              string `dynamodbav:"userId"`
	RelatedUserID       string `dynamodbav:"relatedUserId"`
	StatusRelatedUserID string `dynamodbav:"statusRelatedUserId"`
}

type userItem struct {
	UserID            string             `dynamodbav:"userId"`
	Username          string             `dynamodbav:"username"`
	DisplayName       string             `dynamodbav:"displayName"`
	DollConfiguration *DollConfiguration `dynamodbav:"dollConfiguration"`
	CurrentStatus     *MoodState         `dynamodbav:"currentStatus"`
}

type pageToken struct {
	UserID        string `json:"u"`
	RelatedUserID string `json:"r"`
}

// PublishMood atomically appends a history event and replaces currentStatus.
func (s *DynamoDBStore) PublishMood(ctx context.Context, userID, eventID string, state MoodState) error {
	if s.usersTable == "" || s.moodEventsTable == "" {
		return errors.New("mood table configuration is incomplete")
	}
	encodedState, err := attributevalue.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal current status: %w", err)
	}
	event := moodEvent{
		UserID: userID, OccurredAt: state.UpdatedAt + "#" + eventID, EventID: eventID,
		MoodState: state, Visibility: "FRIENDS",
	}
	encodedEvent, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("marshal mood event: %w", err)
	}
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Update: &types.Update{
			TableName:                 &s.usersTable,
			Key:                       map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
			UpdateExpression:          strptr("SET currentStatus = :status, updatedAt = :updatedAt"),
			ConditionExpression:       strptr("attribute_exists(userId)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":status": encodedState, ":updatedAt": &types.AttributeValueMemberS{Value: state.UpdatedAt}},
		}},
		{Put: &types.Put{TableName: &s.moodEventsTable, Item: encodedEvent, ConditionExpression: strptr("attribute_not_exists(userId) AND attribute_not_exists(occurredAt)")}},
	}})
	if err != nil {
		if transactionConditionFailed(err) {
			return &DomainError{Status: http.StatusNotFound, Code: "not_found", Message: "profile not found"}
		}
		return fmt.Errorf("publish mood transaction: %w", err)
	}
	return nil
}

// ListFriendStatuses reads only ACCEPTED relationships, then hydrates their users.
func (s *DynamoDBStore) ListFriendStatuses(ctx context.Context, userID, token string) ([]FriendStatus, string, error) {
	if s.usersTable == "" || s.friendshipsTable == "" {
		return nil, "", errors.New("friend status table configuration is incomplete")
	}
	input := &dynamodb.QueryInput{
		TableName: &s.friendshipsTable, IndexName: strptr(userStatusIndex), Limit: int32ptr(statusPageSize),
		KeyConditionExpression: strptr("userId = :userId AND begins_with(statusRelatedUserId, :status)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: userID},
			":status": &types.AttributeValueMemberS{Value: acceptedStatus + "#"},
		},
	}
	if token != "" {
		decoded, err := decodePageToken(token)
		if err != nil || decoded.UserID != userID {
			return nil, "", &DomainError{Status: http.StatusBadRequest, Code: "invalid_request", Message: "nextToken is invalid"}
		}
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"userId":              &types.AttributeValueMemberS{Value: userID},
			"relatedUserId":       &types.AttributeValueMemberS{Value: decoded.RelatedUserID},
			"statusRelatedUserId": &types.AttributeValueMemberS{Value: acceptedStatus + "#" + decoded.RelatedUserID},
		}
	}
	output, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("query accepted friendships: %w", err)
	}
	var relationships []relationshipItem
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &relationships); err != nil {
		return nil, "", fmt.Errorf("decode accepted friendships: %w", err)
	}
	users, err := s.getUsers(ctx, relationships)
	if err != nil {
		return nil, "", err
	}
	items := make([]FriendStatus, 0, len(relationships))
	for _, relationship := range relationships {
		user, ok := users[relationship.RelatedUserID]
		if !ok {
			return nil, "", fmt.Errorf("accepted friendship references missing user %q", relationship.RelatedUserID)
		}
		items = append(items, FriendStatus{
			Friend: UserSummary{UserID: user.UserID, Username: user.Username, DisplayName: user.DisplayName},
			Doll:   user.DollConfiguration, Status: user.CurrentStatus,
		})
	}
	nextToken := ""
	if len(output.LastEvaluatedKey) > 0 {
		relatedUserID, ok := stringAttribute(output.LastEvaluatedKey["relatedUserId"])
		if !ok || relatedUserID == "" {
			return nil, "", errors.New("friend status query returned an invalid pagination key")
		}
		nextToken, err = encodePageToken(pageToken{UserID: userID, RelatedUserID: relatedUserID})
		if err != nil {
			return nil, "", fmt.Errorf("encode friend status token: %w", err)
		}
	}
	return items, nextToken, nil
}

func (s *DynamoDBStore) getUsers(ctx context.Context, relationships []relationshipItem) (map[string]userItem, error) {
	result := make(map[string]userItem, len(relationships))
	if len(relationships) == 0 {
		return result, nil
	}
	keys := make([]map[string]types.AttributeValue, 0, len(relationships))
	for _, relationship := range relationships {
		keys = append(keys, map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: relationship.RelatedUserID}})
	}
	requestItems := map[string]types.KeysAndAttributes{s.usersTable: {Keys: keys, ConsistentRead: boolptr(true)}}
	for attempts := 0; len(requestItems) > 0 && attempts < 5; attempts++ {
		output, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{RequestItems: requestItems})
		if err != nil {
			return nil, fmt.Errorf("batch get accepted friends: %w", err)
		}
		for _, encoded := range output.Responses[s.usersTable] {
			var user userItem
			if err := attributevalue.UnmarshalMap(encoded, &user); err != nil {
				return nil, fmt.Errorf("decode accepted friend: %w", err)
			}
			result[user.UserID] = user
		}
		requestItems = output.UnprocessedKeys
	}
	if len(requestItems) > 0 {
		return nil, errors.New("accepted friend reads remained unprocessed")
	}
	return result, nil
}

func encodePageToken(token pageToken) (string, error) {
	encoded, err := json.Marshal(token)
	return base64.RawURLEncoding.EncodeToString(encoded), err
}

func decodePageToken(value string) (pageToken, error) {
	encoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return pageToken{}, err
	}
	var token pageToken
	if err := json.Unmarshal(encoded, &token); err != nil || token.UserID == "" || token.RelatedUserID == "" {
		return pageToken{}, errors.New("invalid token")
	}
	return token, nil
}

func transactionConditionFailed(err error) bool {
	var canceled *types.TransactionCanceledException
	if !errors.As(err, &canceled) {
		return false
	}
	for _, reason := range canceled.CancellationReasons {
		if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
			return true
		}
	}
	return false
}

func strptr(value string) *string { return &value }
func int32ptr(value int32) *int32 { return &value }
func boolptr(value bool) *bool    { return &value }

func stringAttribute(value types.AttributeValue) (string, bool) {
	attribute, ok := value.(*types.AttributeValueMemberS)
	return attribute.Value, ok
}
