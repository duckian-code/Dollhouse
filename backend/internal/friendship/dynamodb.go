package friendship

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	userSearchIndex = "UserSearchIndex"
	userStatusIndex = "UserStatusIndex"
	requestIDIndex  = "RequestIdIndex"
	searchPageSize  = 25
)

type dynamodbAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// DynamoDBStore persists mirrored relationships and queries user summaries.
type DynamoDBStore struct {
	client           dynamodbAPI
	usersTable       string
	friendshipsTable string
}

// NewDynamoDBStore creates a friendship store over the existing tables.
func NewDynamoDBStore(client dynamodbAPI, usersTable, friendshipsTable string) *DynamoDBStore {
	return &DynamoDBStore{client: client, usersTable: usersTable, friendshipsTable: friendshipsTable}
}

type userItem struct {
	UserID             string `dynamodbav:"userId"`
	Username           string `dynamodbav:"username"`
	NormalizedUsername string `dynamodbav:"normalizedUsername"`
	DisplayName        string `dynamodbav:"displayName"`
}

type relationshipItem struct {
	UserID              string `dynamodbav:"userId"`
	RelatedUserID       string `dynamodbav:"relatedUserId"`
	RequestID           string `dynamodbav:"requestId"`
	Status              string `dynamodbav:"status"`
	StatusRelatedUserID string `dynamodbav:"statusRelatedUserId"`
	RequestedBy         string `dynamodbav:"requestedBy"`
	RequestedAt         string `dynamodbav:"requestedAt"`
	AcceptedAt          string `dynamodbav:"acceptedAt,omitempty"`
}

type pageToken struct {
	Username string `json:"u"`
	UserID   string `json:"i"`
}

func (s *DynamoDBStore) SearchUsers(ctx context.Context, query, token, excludeUserID string) ([]UserSummary, string, error) {
	if s.usersTable == "" {
		return nil, "", errors.New("USERS_TABLE_NAME is not configured")
	}
	input := &dynamodb.QueryInput{
		TableName:              &s.usersTable,
		IndexName:              strptr(userSearchIndex),
		KeyConditionExpression: strptr("searchPartition = :partition AND begins_with(normalizedUsername, :query)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":partition": &types.AttributeValueMemberS{Value: "USER"},
			":query":     &types.AttributeValueMemberS{Value: strings.ToLower(strings.TrimSpace(query))},
		},
		Limit: int32ptr(searchPageSize),
	}
	if token != "" {
		decoded, err := decodePageToken(token)
		if err != nil {
			return nil, "", &DomainError{Status: http.StatusBadRequest, Code: "invalid_request", Message: "nextToken is invalid"}
		}
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"searchPartition":    &types.AttributeValueMemberS{Value: "USER"},
			"normalizedUsername": &types.AttributeValueMemberS{Value: decoded.Username},
			"userId":             &types.AttributeValueMemberS{Value: decoded.UserID},
		}
	}
	output, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("query user search index: %w", err)
	}
	items := make([]UserSummary, 0, len(output.Items))
	for _, encoded := range output.Items {
		var item userItem
		if err := attributevalue.UnmarshalMap(encoded, &item); err != nil {
			return nil, "", fmt.Errorf("decode user search result: %w", err)
		}
		if item.UserID != excludeUserID {
			items = append(items, item.summary())
		}
	}
	nextToken := ""
	if len(output.LastEvaluatedKey) > 0 {
		username, usernameOK := stringAttribute(output.LastEvaluatedKey["normalizedUsername"])
		userID, userIDOK := stringAttribute(output.LastEvaluatedKey["userId"])
		if !usernameOK || !userIDOK {
			return nil, "", errors.New("user search returned an invalid pagination key")
		}
		nextToken, err = encodePageToken(pageToken{Username: username, UserID: userID})
		if err != nil {
			return nil, "", fmt.Errorf("encode user search token: %w", err)
		}
	}
	return items, nextToken, nil
}

func (s *DynamoDBStore) SendRequest(ctx context.Context, fromUserID, toUserID, requestID, requestedAt string) (FriendRequest, error) {
	target, err := s.getUser(ctx, toUserID)
	if err != nil {
		return FriendRequest{}, err
	}
	if target == nil {
		return FriendRequest{}, &DomainError{Status: http.StatusNotFound, Code: "not_found", Message: "user not found"}
	}
	outgoing := relationshipItem{UserID: fromUserID, RelatedUserID: toUserID, RequestID: requestID, Status: StatusPendingOutgoing, StatusRelatedUserID: StatusPendingOutgoing + "#" + toUserID, RequestedBy: fromUserID, RequestedAt: requestedAt}
	incoming := relationshipItem{UserID: toUserID, RelatedUserID: fromUserID, RequestID: requestID, Status: StatusPendingIncoming, StatusRelatedUserID: StatusPendingIncoming + "#" + fromUserID, RequestedBy: fromUserID, RequestedAt: requestedAt}
	first, err := attributevalue.MarshalMap(outgoing)
	if err != nil {
		return FriendRequest{}, fmt.Errorf("marshal outgoing request: %w", err)
	}
	second, err := attributevalue.MarshalMap(incoming)
	if err != nil {
		return FriendRequest{}, fmt.Errorf("marshal incoming request: %w", err)
	}
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Put: &types.Put{TableName: &s.friendshipsTable, Item: first, ConditionExpression: strptr("attribute_not_exists(userId) AND attribute_not_exists(relatedUserId)")}},
		{Put: &types.Put{TableName: &s.friendshipsTable, Item: second, ConditionExpression: strptr("attribute_not_exists(userId) AND attribute_not_exists(relatedUserId)")}},
	}})
	if err != nil {
		if transactionConditionFailed(err) {
			return FriendRequest{}, &DomainError{Status: http.StatusConflict, Code: "conflict", Message: "a friendship or pending request already exists"}
		}
		return FriendRequest{}, fmt.Errorf("write mirrored friend request: %w", err)
	}
	return FriendRequest{RequestID: requestID, User: target.summary(), Status: StatusPendingOutgoing, RequestedAt: requestedAt}, nil
}

func (s *DynamoDBStore) ListRequests(ctx context.Context, userID string) ([]FriendRequest, []FriendRequest, error) {
	incomingItems, err := s.queryRelationships(ctx, userID, StatusPendingIncoming)
	if err != nil {
		return nil, nil, err
	}
	outgoingItems, err := s.queryRelationships(ctx, userID, StatusPendingOutgoing)
	if err != nil {
		return nil, nil, err
	}
	incoming, err := s.hydrateRequests(ctx, incomingItems)
	if err != nil {
		return nil, nil, err
	}
	outgoing, err := s.hydrateRequests(ctx, outgoingItems)
	if err != nil {
		return nil, nil, err
	}
	return incoming, outgoing, nil
}

func (s *DynamoDBStore) AcceptRequest(ctx context.Context, userID, requestID, acceptedAt string) (Friendship, error) {
	relation, err := s.resolveIncoming(ctx, userID, requestID)
	if err != nil {
		return Friendship{}, err
	}
	friend, err := s.getUser(ctx, relation.RelatedUserID)
	if err != nil {
		return Friendship{}, err
	}
	if friend == nil {
		return Friendship{}, errors.New("friend request references a missing user")
	}
	names := map[string]string{"#status": "status", "#statusRelatedUserId": "statusRelatedUserId", "#acceptedAt": "acceptedAt"}
	valuesFor := func(expected, related string) map[string]types.AttributeValue {
		return map[string]types.AttributeValue{
			":requestId": &types.AttributeValueMemberS{Value: requestID}, ":expected": &types.AttributeValueMemberS{Value: expected},
			":accepted": &types.AttributeValueMemberS{Value: StatusAccepted}, ":statusRelated": &types.AttributeValueMemberS{Value: StatusAccepted + "#" + related},
			":acceptedAt": &types.AttributeValueMemberS{Value: acceptedAt},
		}
	}
	update := "SET #status = :accepted, #statusRelatedUserId = :statusRelated, #acceptedAt = :acceptedAt"
	condition := "requestId = :requestId AND #status = :expected"
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Update: &types.Update{TableName: &s.friendshipsTable, Key: relationshipKey(userID, relation.RelatedUserID), UpdateExpression: &update, ConditionExpression: &condition, ExpressionAttributeNames: names, ExpressionAttributeValues: valuesFor(StatusPendingIncoming, relation.RelatedUserID)}},
		{Update: &types.Update{TableName: &s.friendshipsTable, Key: relationshipKey(relation.RelatedUserID, userID), UpdateExpression: &update, ConditionExpression: &condition, ExpressionAttributeNames: names, ExpressionAttributeValues: valuesFor(StatusPendingOutgoing, userID)}},
	}})
	if err != nil {
		if transactionConditionFailed(err) {
			return Friendship{}, &DomainError{Status: http.StatusConflict, Code: "conflict", Message: "friend request is no longer pending"}
		}
		return Friendship{}, fmt.Errorf("accept mirrored friend request: %w", err)
	}
	return Friendship{Friend: friend.summary(), Status: StatusAccepted, AcceptedAt: acceptedAt}, nil
}

func (s *DynamoDBStore) DeclineRequest(ctx context.Context, userID, requestID string) error {
	relation, err := s.resolveIncoming(ctx, userID, requestID)
	if err != nil {
		return err
	}
	return s.deletePair(ctx, userID, relation.RelatedUserID, requestID, StatusPendingIncoming, StatusPendingOutgoing, "friend request is no longer pending")
}

func (s *DynamoDBStore) RemoveFriend(ctx context.Context, userID, friendID string) error {
	relation, err := s.getRelationship(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if relation == nil || relation.Status != StatusAccepted {
		return &DomainError{Status: http.StatusNotFound, Code: "not_found", Message: "friendship not found"}
	}
	return s.deletePair(ctx, userID, friendID, relation.RequestID, StatusAccepted, StatusAccepted, "friendship is no longer accepted")
}

func (s *DynamoDBStore) deletePair(ctx context.Context, userID, relatedUserID, requestID, firstStatus, secondStatus, conflictMessage string) error {
	names := map[string]string{"#status": "status"}
	condition := "requestId = :requestId AND #status = :expected"
	values := func(status string) map[string]types.AttributeValue {
		return map[string]types.AttributeValue{":requestId": &types.AttributeValueMemberS{Value: requestID}, ":expected": &types.AttributeValueMemberS{Value: status}}
	}
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Delete: &types.Delete{TableName: &s.friendshipsTable, Key: relationshipKey(userID, relatedUserID), ConditionExpression: &condition, ExpressionAttributeNames: names, ExpressionAttributeValues: values(firstStatus)}},
		{Delete: &types.Delete{TableName: &s.friendshipsTable, Key: relationshipKey(relatedUserID, userID), ConditionExpression: &condition, ExpressionAttributeNames: names, ExpressionAttributeValues: values(secondStatus)}},
	}})
	if err != nil {
		if transactionConditionFailed(err) {
			return &DomainError{Status: http.StatusConflict, Code: "conflict", Message: conflictMessage}
		}
		return fmt.Errorf("delete mirrored relationship: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) resolveIncoming(ctx context.Context, userID, requestID string) (*relationshipItem, error) {
	output, err := s.client.Query(ctx, &dynamodb.QueryInput{TableName: &s.friendshipsTable, IndexName: strptr(requestIDIndex), KeyConditionExpression: strptr("requestId = :requestId"), ExpressionAttributeValues: map[string]types.AttributeValue{":requestId": &types.AttributeValueMemberS{Value: requestID}}})
	if err != nil {
		return nil, fmt.Errorf("resolve friend request: %w", err)
	}
	if len(output.Items) == 0 {
		return nil, &DomainError{Status: http.StatusNotFound, Code: "not_found", Message: "friend request not found"}
	}
	for _, encoded := range output.Items {
		var item relationshipItem
		if err := attributevalue.UnmarshalMap(encoded, &item); err != nil {
			return nil, fmt.Errorf("decode friend request: %w", err)
		}
		if item.UserID == userID && item.Status == StatusPendingIncoming {
			return &item, nil
		}
	}
	return nil, &DomainError{Status: http.StatusForbidden, Code: "forbidden", Message: "only the request recipient may perform this action"}
}

func (s *DynamoDBStore) queryRelationships(ctx context.Context, userID, status string) ([]relationshipItem, error) {
	output, err := s.client.Query(ctx, &dynamodb.QueryInput{TableName: &s.friendshipsTable, IndexName: strptr(userStatusIndex), KeyConditionExpression: strptr("userId = :userId AND begins_with(statusRelatedUserId, :status)"), ExpressionAttributeValues: map[string]types.AttributeValue{":userId": &types.AttributeValueMemberS{Value: userID}, ":status": &types.AttributeValueMemberS{Value: status + "#"}}})
	if err != nil {
		return nil, fmt.Errorf("query %s relationships: %w", status, err)
	}
	items := make([]relationshipItem, 0, len(output.Items))
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &items); err != nil {
		return nil, fmt.Errorf("decode %s relationships: %w", status, err)
	}
	return items, nil
}

func (s *DynamoDBStore) hydrateRequests(ctx context.Context, relationships []relationshipItem) ([]FriendRequest, error) {
	requests := make([]FriendRequest, 0, len(relationships))
	for _, relation := range relationships {
		user, err := s.getUser(ctx, relation.RelatedUserID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, errors.New("friend request references a missing user")
		}
		requests = append(requests, FriendRequest{RequestID: relation.RequestID, User: user.summary(), Status: relation.Status, RequestedAt: relation.RequestedAt})
	}
	return requests, nil
}

func (s *DynamoDBStore) getUser(ctx context.Context, userID string) (*userItem, error) {
	if s.usersTable == "" || s.friendshipsTable == "" {
		return nil, errors.New("friendship table configuration is incomplete")
	}
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: &s.usersTable, Key: map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}}, ConsistentRead: boolptr(true)})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if len(output.Item) == 0 {
		return nil, nil
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &item, nil
}

func (s *DynamoDBStore) getRelationship(ctx context.Context, userID, relatedUserID string) (*relationshipItem, error) {
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: &s.friendshipsTable, Key: relationshipKey(userID, relatedUserID), ConsistentRead: boolptr(true)})
	if err != nil {
		return nil, fmt.Errorf("get relationship: %w", err)
	}
	if len(output.Item) == 0 {
		return nil, nil
	}
	var item relationshipItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, fmt.Errorf("decode relationship: %w", err)
	}
	return &item, nil
}

func relationshipKey(userID, relatedUserID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}, "relatedUserId": &types.AttributeValueMemberS{Value: relatedUserID}}
}

func (u userItem) summary() UserSummary {
	return UserSummary{UserID: u.UserID, Username: u.Username, DisplayName: u.DisplayName}
}
func strptr(value string) *string { return &value }
func boolptr(value bool) *bool    { return &value }
func int32ptr(value int32) *int32 { return &value }

func stringAttribute(value types.AttributeValue) (string, bool) {
	attribute, ok := value.(*types.AttributeValueMemberS)
	return attribute.Value, ok
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
	if err := json.Unmarshal(encoded, &token); err != nil || token.Username == "" || token.UserID == "" {
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
