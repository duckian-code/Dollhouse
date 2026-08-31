package profiledoll

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const usernameReservationPrefix = "USERNAME#"

type dynamodbAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// DynamoDBStore persists profiles, dolls, and username reservations.
type DynamoDBStore struct {
	client    dynamodbAPI
	tableName string
}

func NewDynamoDBStore(client dynamodbAPI, tableName string) *DynamoDBStore {
	return &DynamoDBStore{client: client, tableName: tableName}
}

type userItem struct {
	UserID             string             `dynamodbav:"userId"`
	CognitoSub         string             `dynamodbav:"cognitoSub"`
	Username           string             `dynamodbav:"username,omitempty"`
	NormalizedUsername string             `dynamodbav:"normalizedUsername,omitempty"`
	SearchPartition    string             `dynamodbav:"searchPartition,omitempty"`
	DisplayName        string             `dynamodbav:"displayName,omitempty"`
	Bio                *string            `dynamodbav:"bio,omitempty"`
	Role               string             `dynamodbav:"role"`
	OnboardingComplete *bool              `dynamodbav:"onboardingComplete,omitempty"`
	DollConfiguration  *DollConfiguration `dynamodbav:"dollConfiguration,omitempty"`
	CreatedAt          string             `dynamodbav:"createdAt"`
	UpdatedAt          string             `dynamodbav:"updatedAt"`
}

type usernameReservation struct {
	UserID             string `dynamodbav:"userId"`
	EntityType         string `dynamodbav:"entityType"`
	OwnerUserID        string `dynamodbav:"ownerUserId"`
	NormalizedUsername string `dynamodbav:"normalizedUsername"`
	CreatedAt          string `dynamodbav:"createdAt"`
	UpdatedAt          string `dynamodbav:"updatedAt"`
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
		if item.OnboardingComplete == nil {
			item, err = s.migrateLegacyProfile(ctx, identity.UserID, now)
			if err != nil {
				return Profile{}, err
			}
		}
		return item.profile(), nil
	}

	incomplete := false
	created := userItem{
		UserID: identity.UserID, CognitoSub: identity.UserID, Role: "USER",
		OnboardingComplete: &incomplete, CreatedAt: now, UpdatedAt: now,
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

func (s *DynamoDBStore) migrateLegacyProfile(ctx context.Context, userID, now string) (*userItem, error) {
	names := map[string]string{
		"#onboardingComplete": "onboardingComplete", "#updatedAt": "updatedAt",
		"#username": "username", "#normalizedUsername": "normalizedUsername",
		"#searchPartition": "searchPartition", "#displayName": "displayName",
		"#email": "email", "#normalizedEmail": "normalizedEmail",
	}
	output, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                &s.tableName,
		Key:                      map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
		UpdateExpression:         strptr("SET #onboardingComplete = :incomplete, #updatedAt = :updatedAt REMOVE #username, #normalizedUsername, #searchPartition, #displayName, #email, #normalizedEmail"),
		ConditionExpression:      strptr("attribute_exists(userId) AND attribute_not_exists(#onboardingComplete)"),
		ExpressionAttributeNames: names,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":incomplete": &types.AttributeValueMemberBOOL{Value: false},
			":updatedAt":  &types.AttributeValueMemberS{Value: now},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		var conditional *types.ConditionalCheckFailedException
		if !errors.As(err, &conditional) {
			return nil, fmt.Errorf("migrate legacy profile: %w", err)
		}
		return s.getUser(ctx, userID)
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(output.Attributes, &item); err != nil {
		return nil, fmt.Errorf("decode migrated profile: %w", err)
	}
	return &item, nil
}

func (s *DynamoDBStore) UpdateProfile(ctx context.Context, userID string, changes ProfileChanges, now string) (Profile, error) {
	current, err := s.getUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if current == nil {
		return Profile{}, errors.New("profile not found")
	}
	if current.OnboardingComplete == nil {
		current, err = s.migrateLegacyProfile(ctx, userID, now)
		if err != nil {
			return Profile{}, err
		}
	}

	username := current.Username
	if changes.Username.Set {
		username = *changes.Username.Value
	}
	displayName := current.DisplayName
	if changes.DisplayName.Set {
		displayName = *changes.DisplayName.Value
	}
	complete := strings.TrimSpace(username) != "" && strings.TrimSpace(displayName) != ""

	if changes.Username.Set {
		return s.updateProfileWithUsername(ctx, *current, changes, complete, now)
	}
	return s.updateProfileFields(ctx, userID, changes, complete, now)
}

func (s *DynamoDBStore) updateProfileWithUsername(ctx context.Context, current userItem, changes ProfileChanges, complete bool, now string) (Profile, error) {
	username := *changes.Username.Value
	normalized := NormalizeUsername(username)
	reservation, err := attributevalue.MarshalMap(usernameReservation{
		UserID: usernameReservationKey(normalized), EntityType: "USERNAME_RESERVATION",
		OwnerUserID: current.UserID, NormalizedUsername: normalized, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return Profile{}, fmt.Errorf("marshal username reservation: %w", err)
	}

	update, names, values := profileUpdateExpression(changes, complete, now)
	names["#normalizedUsername"] = "normalizedUsername"
	values[":normalizedUsername"] = &types.AttributeValueMemberS{Value: normalized}
	update = strings.Replace(update, "SET ", "SET #normalizedUsername = :normalizedUsername, ", 1)
	condition := "attribute_exists(userId)"
	if current.NormalizedUsername == "" {
		condition += " AND attribute_not_exists(#normalizedUsername)"
	} else {
		values[":currentNormalizedUsername"] = &types.AttributeValueMemberS{Value: current.NormalizedUsername}
		condition += " AND #normalizedUsername = :currentNormalizedUsername"
	}

	transaction := []types.TransactWriteItem{
		{Put: &types.Put{
			TableName: &s.tableName, Item: reservation,
			ConditionExpression:       strptr("attribute_not_exists(userId) OR ownerUserId = :ownerUserId"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":ownerUserId": &types.AttributeValueMemberS{Value: current.UserID}},
		}},
		{Update: &types.Update{
			TableName:        &s.tableName,
			Key:              map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: current.UserID}},
			UpdateExpression: &update, ConditionExpression: &condition,
			ExpressionAttributeNames: names, ExpressionAttributeValues: values,
		}},
	}
	if current.NormalizedUsername != "" && current.NormalizedUsername != normalized {
		transaction = append(transaction, types.TransactWriteItem{Delete: &types.Delete{
			TableName:                 &s.tableName,
			Key:                       map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: usernameReservationKey(current.NormalizedUsername)}},
			ConditionExpression:       strptr("attribute_not_exists(userId) OR ownerUserId = :ownerUserId"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":ownerUserId": &types.AttributeValueMemberS{Value: current.UserID}},
		}})
	}
	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: transaction})
	if err != nil {
		if transactionConditionFailed(err) {
			return Profile{}, &DomainError{Status: http.StatusConflict, Code: "conflict", Message: "username is already in use"}
		}
		return Profile{}, fmt.Errorf("update profile and username reservation: %w", err)
	}
	updated, err := s.getUser(ctx, current.UserID)
	if err != nil {
		return Profile{}, err
	}
	if updated == nil {
		return Profile{}, errors.New("updated profile could not be read")
	}
	return updated.profile(), nil
}

func (s *DynamoDBStore) updateProfileFields(ctx context.Context, userID string, changes ProfileChanges, complete bool, now string) (Profile, error) {
	update, names, values := profileUpdateExpression(changes, complete, now)
	output, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        &s.tableName,
		Key:              map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: userID}},
		UpdateExpression: &update, ExpressionAttributeNames: names,
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

func profileUpdateExpression(changes ProfileChanges, complete bool, now string) (string, map[string]string, map[string]types.AttributeValue) {
	setParts := []string{"#updatedAt = :updatedAt", "#onboardingComplete = :onboardingComplete"}
	removeParts := make([]string, 0, 2)
	names := map[string]string{"#updatedAt": "updatedAt", "#onboardingComplete": "onboardingComplete", "#searchPartition": "searchPartition"}
	values := map[string]types.AttributeValue{
		":updatedAt":          &types.AttributeValueMemberS{Value: now},
		":onboardingComplete": &types.AttributeValueMemberBOOL{Value: complete},
	}
	if complete {
		setParts = append(setParts, "#searchPartition = :searchPartition")
		values[":searchPartition"] = &types.AttributeValueMemberS{Value: "USER"}
	} else {
		removeParts = append(removeParts, "#searchPartition")
	}
	if changes.Username.Set {
		setParts = append(setParts, "#username = :username")
		names["#username"] = "username"
		values[":username"] = &types.AttributeValueMemberS{Value: *changes.Username.Value}
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
	return expression, names, values
}

func (s *DynamoDBStore) UsernameAvailable(ctx context.Context, normalized, userID string) (bool, error) {
	if s.tableName == "" {
		return false, errors.New("USERS_TABLE_NAME is not configured")
	}
	output, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &s.tableName,
		Key:            map[string]types.AttributeValue{"userId": &types.AttributeValueMemberS{Value: usernameReservationKey(normalized)}},
		ConsistentRead: boolptr(true),
	})
	if err != nil {
		return false, fmt.Errorf("get username reservation: %w", err)
	}
	if len(output.Item) == 0 {
		return true, nil
	}
	var reservation usernameReservation
	if err := attributevalue.UnmarshalMap(output.Item, &reservation); err != nil {
		return false, fmt.Errorf("decode username reservation: %w", err)
	}
	return reservation.OwnerUserID == userID, nil
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
	complete := u.OnboardingComplete != nil && *u.OnboardingComplete
	return Profile{
		UserID: u.UserID, Username: u.Username, DisplayName: u.DisplayName, Bio: u.Bio,
		OnboardingComplete: complete, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

// NormalizeUsername is the canonical comparison key: trim surrounding
// whitespace, then apply Unicode-aware lowercase conversion.
func NormalizeUsername(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func usernameReservationKey(normalized string) string { return usernameReservationPrefix + normalized }
func strptr(value string) *string                     { return &value }
func boolptr(value bool) *bool                        { return &value }

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
