package profiledoll

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamodbAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// DynamoDBStore persists profile and doll fields in the existing Users table.
type DynamoDBStore struct {
	client    dynamodbAPI
	tableName string
}

// NewDynamoDBStore creates a Users-table store.
func NewDynamoDBStore(client dynamodbAPI, tableName string) *DynamoDBStore {
	return &DynamoDBStore{client: client, tableName: tableName}
}

type userItem struct {
	UserID             string             `dynamodbav:"userId"`
	CognitoSub         string             `dynamodbav:"cognitoSub"`
	Username           string             `dynamodbav:"username"`
	NormalizedUsername string             `dynamodbav:"normalizedUsername"`
	SearchPartition    string             `dynamodbav:"searchPartition"`
	DisplayName        string             `dynamodbav:"displayName"`
	Bio                *string            `dynamodbav:"bio,omitempty"`
	Role               string             `dynamodbav:"role"`
	DollConfiguration  *DollConfiguration `dynamodbav:"dollConfiguration,omitempty"`
	CreatedAt          string             `dynamodbav:"createdAt"`
	UpdatedAt          string             `dynamodbav:"updatedAt"`
}

func (s *DynamoDBStore) EnsureUser(ctx context.Context, identity Identity, now string) (Profile, error) {
	if s.tableName == "" {
		return Profile{}, errors.New("USERS_TABLE_NAME is not configured")
	}
	item, err := s.getUser(ctx, identity.UserID)
	if err != nil {
		return Profile{}, err
	}
	if item != nil {
		return item.profile(), nil
	}

	created := userItem{
		UserID: identity.UserID, CognitoSub: identity.UserID,
		Username: identity.Username, NormalizedUsername: normalize(identity.Username),
		SearchPartition: "USER", DisplayName: identity.DisplayName, Role: "USER",
		CreatedAt: now, UpdatedAt: now,
	}
	attributes, err := attributevalue.MarshalMap(created)
	if err != nil {
		return Profile{}, fmt.Errorf("marshal initial user: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName, Item: attributes,
		ConditionExpression: strptr("attribute_not_exists(userId)"),
	})
	if err == nil {
		return created.profile(), nil
	}
	var conditional *types.ConditionalCheckFailedException
	if !errors.As(err, &conditional) {
		return Profile{}, fmt.Errorf("create initial user: %w", err)
	}
	item, err = s.getUser(ctx, identity.UserID)
	if err != nil {
		return Profile{}, err
	}
	if item == nil {
		return Profile{}, errors.New("user was concurrently created but could not be read")
	}
	return item.profile(), nil
}

func (s *DynamoDBStore) UpdateProfile(ctx context.Context, userID string, changes ProfileChanges, now string) (Profile, error) {
	setParts := []string{"#updatedAt = :updatedAt"}
	removeParts := make([]string, 0, 1)
	names := map[string]string{"#updatedAt": "updatedAt"}
	values := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: now},
	}
	if changes.Username.Set {
		setParts = append(setParts, "#username = :username", "#normalizedUsername = :normalizedUsername")
		names["#username"] = "username"
		names["#normalizedUsername"] = "normalizedUsername"
		values[":username"] = &types.AttributeValueMemberS{Value: *changes.Username.Value}
		values[":normalizedUsername"] = &types.AttributeValueMemberS{Value: normalize(*changes.Username.Value)}
	}
	if changes.DisplayName.Set {
		setParts = append(setParts, "#displayName = :displayName")
		names["#displayName"] = "displayName"
		values[":displayName"] = &types.AttributeValueMemberS{Value: *changes.DisplayName.Value}
	}
	if changes.Bio.Set {
		names["#bio"] = "bio"
		if changes.Bio.Value == nil {
			removeParts = append(removeParts, "#bio")
		} else {
			setParts = append(setParts, "#bio = :bio")
			values[":bio"] = &types.AttributeValueMemberS{Value: *changes.Bio.Value}
		}
	}
	expression := "SET " + strings.Join(setParts, ", ")
	if len(removeParts) > 0 {
		expression += " REMOVE " + strings.Join(removeParts, ", ")
	}
	output, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &s.tableName,
		Key:              map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
		UpdateExpression: &expression, ExpressionAttributeNames: names,
		ExpressionAttributeValues: values, ReturnValues: types.ReturnValueAllNew,
		ConditionExpression: strptr("attribute_exists(userId)"),
	})
	if err != nil {
		return Profile{}, fmt.Errorf("update profile: %w", err)
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(output.Attributes, &item); err != nil {
		return Profile{}, fmt.Errorf("decode updated profile: %w", err)
	}
	return item.profile(), nil
}

func (s *DynamoDBStore) GetDoll(ctx context.Context, userID string) (*DollConfiguration, error) {
	item, err := s.getUser(ctx, userID)
	if err != nil || item == nil || item.DollConfiguration == nil {
		return nil, err
	}
	configuration := *item.DollConfiguration
	return &configuration, nil
}

func (s *DynamoDBStore) UpdateDoll(ctx context.Context, userID string, configuration DollConfiguration, now string) (DollConfiguration, error) {
	configuration.UpdatedAt = now
	encoded, err := attributevalue.Marshal(configuration)
	if err != nil {
		return DollConfiguration{}, fmt.Errorf("marshal doll configuration: %w", err)
	}
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &s.tableName,
		Key:              map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
		UpdateExpression: strptr("SET dollConfiguration = :configuration, updatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":configuration": encoded,
			":updatedAt":     &types.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: strptr("attribute_exists(userId)"),
	})
	if err != nil {
		return DollConfiguration{}, fmt.Errorf("update doll configuration: %w", err)
	}
	return configuration, nil
}

func (s *DynamoDBStore) getUser(ctx context.Context, userID string) (*userItem, error) {
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &s.tableName,
		Key:            map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
		ConsistentRead: boolptr(true),
	})
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

func (u userItem) profile() Profile {
	return Profile{UserID: u.UserID, Username: u.Username, DisplayName: u.DisplayName, Bio: u.Bio, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func strptr(value string) *string   { return &value }
func boolptr(value bool) *bool      { return &value }
