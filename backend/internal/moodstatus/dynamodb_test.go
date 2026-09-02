package moodstatus

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDB struct {
	batchOutputs  []*dynamodb.BatchGetItemOutput
	queryOutput   *dynamodb.QueryOutput
	queryOutputs  []*dynamodb.QueryOutput
	batchInputs   []*dynamodb.BatchGetItemInput
	queryInput    *dynamodb.QueryInput
	transactInput *dynamodb.TransactWriteItemsInput
	transactErr   error
}

func (f *fakeDynamoDB) BatchGetItem(_ context.Context, input *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	f.batchInputs = append(f.batchInputs, input)
	if len(f.batchOutputs) == 0 {
		return &dynamodb.BatchGetItemOutput{}, nil
	}
	output := f.batchOutputs[0]
	f.batchOutputs = f.batchOutputs[1:]
	return output, nil
}

func (f *fakeDynamoDB) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	f.queryInput = input
	if len(f.queryOutputs) > 0 {
		output := f.queryOutputs[0]
		f.queryOutputs = f.queryOutputs[1:]
		return output, nil
	}
	if f.queryOutput == nil {
		return &dynamodb.QueryOutput{}, nil
	}
	return f.queryOutput, nil
}

func TestListNotificationRecipientIDsPaginatesAndDeduplicates(t *testing.T) {
	key := map[string]types.AttributeValue{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-2"}}
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{
		{Items: []map[string]types.AttributeValue{
			{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-1"}},
			{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-2"}},
		}, LastEvaluatedKey: key},
		{Items: []map[string]types.AttributeValue{
			{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-2"}},
			{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-3"}},
		}},
	}}
	recipients, err := NewDynamoDBStore(client, "users", "friendships", "moods").ListNotificationRecipientIDs(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	if len(recipients) != 3 || recipients[0] != "friend-1" || recipients[1] != "friend-2" || recipients[2] != "friend-3" {
		t.Fatalf("recipients=%#v", recipients)
	}
	if client.queryInput.IndexName == nil || *client.queryInput.IndexName != userStatusIndex || client.queryInput.ProjectionExpression == nil || *client.queryInput.ProjectionExpression != "relatedUserId" {
		t.Fatalf("query=%#v", client.queryInput)
	}
}

func (f *fakeDynamoDB) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.transactInput = input
	return &dynamodb.TransactWriteItemsOutput{}, f.transactErr
}

func marshalItem(t *testing.T, value any) map[string]types.AttributeValue {
	t.Helper()
	item, err := attributevalue.MarshalMap(value)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestPublishMoodAtomicallyUpdatesCurrentStatusAndAppendsEvent(t *testing.T) {
	stress := 4
	state := MoodState{Status: "okay", Stress: &stress, UpdatedAt: "2026-08-16T18:30:45Z"}
	client := &fakeDynamoDB{}
	err := NewDynamoDBStore(client, "users", "friendships", "moods").PublishMood(context.Background(), "user-1", "event-1", state)
	if err != nil {
		t.Fatal(err)
	}
	if client.transactInput == nil || len(client.transactInput.TransactItems) != 2 {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
	update := client.transactInput.TransactItems[0].Update
	put := client.transactInput.TransactItems[1].Put
	if update == nil || put == nil || *update.TableName != "users" || *put.TableName != "moods" || update.ConditionExpression == nil || put.ConditionExpression == nil {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
	var event moodEvent
	if err := attributevalue.UnmarshalMap(put.Item, &event); err != nil {
		t.Fatal(err)
	}
	if event.UserID != "user-1" || event.EventID != "event-1" || event.OccurredAt != "2026-08-16T18:30:45Z#event-1" || event.Visibility != "FRIENDS" || event.Fatigue != nil {
		t.Fatalf("event=%#v", event)
	}
}

func TestPublishMoodMapsMissingProfileCondition(t *testing.T) {
	code := "ConditionalCheckFailed"
	client := &fakeDynamoDB{transactErr: &types.TransactionCanceledException{CancellationReasons: []types.CancellationReason{{Code: &code}}}}
	err := NewDynamoDBStore(client, "users", "friendships", "moods").PublishMood(context.Background(), "missing", "event", MoodState{UpdatedAt: "now"})
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusNotFound || domain.Code != "not_found" {
		t.Fatalf("err=%v", err)
	}
}

func TestListMoodsQueriesOwnHistoryNewestFirstAndPaginates(t *testing.T) {
	stress := 4
	first := moodEvent{UserID: "self", OccurredAt: "2026-08-16T18:30:45Z#event-2", EventID: "event-2", MoodState: MoodState{Status: "calm", Stress: &stress, UpdatedAt: "2026-08-16T18:30:45Z"}}
	second := moodEvent{UserID: "self", OccurredAt: "2026-08-15T18:30:45Z#event-1", EventID: "event-1", MoodState: MoodState{Status: "tired", UpdatedAt: "2026-08-15T18:30:45Z"}}
	client := &fakeDynamoDB{queryOutput: &dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{marshalItem(t, first), marshalItem(t, second)},
		LastEvaluatedKey: map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: "self"}, "occurredAt": &types.AttributeValueMemberS{Value: second.OccurredAt},
		},
	}}
	items, token, err := NewDynamoDBStore(client, "users", "friendships", "moods").ListMoods(context.Background(), "self", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].EventID != "event-2" || items[1].EventID != "event-1" || token == "" {
		t.Fatalf("items=%#v token=%q", items, token)
	}
	if client.queryInput.ScanIndexForward == nil || *client.queryInput.ScanIndexForward || client.queryInput.Limit == nil || *client.queryInput.Limit != statusPageSize || client.queryInput.IndexName != nil {
		t.Fatalf("query=%#v", client.queryInput)
	}
	queriedUser := client.queryInput.ExpressionAttributeValues[":userId"].(*types.AttributeValueMemberS).Value
	if queriedUser != "self" {
		t.Fatalf("queried user=%q", queriedUser)
	}
	decoded, err := decodeMoodPageToken(token)
	if err != nil || decoded.UserID != "self" || decoded.OccurredAt != second.OccurredAt {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
}

func TestListMoodsUsesScopedTokenAndRejectsInvalidTokens(t *testing.T) {
	valid := mustMoodToken(t, moodPageToken{UserID: "self", OccurredAt: "2026-08-15T18:30:45Z#event-1"})
	client := &fakeDynamoDB{}
	store := NewDynamoDBStore(client, "users", "friendships", "moods")
	if _, _, err := store.ListMoods(context.Background(), "self", valid); err != nil {
		t.Fatal(err)
	}
	key := client.queryInput.ExclusiveStartKey
	if key["userId"].(*types.AttributeValueMemberS).Value != "self" || key["occurredAt"].(*types.AttributeValueMemberS).Value != "2026-08-15T18:30:45Z#event-1" {
		t.Fatalf("key=%#v", key)
	}
	for _, token := range []string{"not-a-token", mustMoodToken(t, moodPageToken{UserID: "other", OccurredAt: "timestamp"})} {
		client.queryInput = nil
		_, _, err := store.ListMoods(context.Background(), "self", token)
		var domain *DomainError
		if !errors.As(err, &domain) || domain.Status != http.StatusBadRequest || client.queryInput != nil {
			t.Fatalf("token=%q err=%v query=%#v", token, err, client.queryInput)
		}
	}
}

func TestListFriendStatusesQueriesAcceptedRelationshipsAndPreservesOrder(t *testing.T) {
	firstRelation := relationshipItem{UserID: "self", RelatedUserID: "friend-2", StatusRelatedUserID: "ACCEPTED#friend-2"}
	secondRelation := relationshipItem{UserID: "self", RelatedUserID: "friend-1", StatusRelatedUserID: "ACCEPTED#friend-1"}
	firstDoll := &DollConfiguration{BodyAssetID: "body-1", ClothingAssetIDs: []string{}}
	secondDoll := &DollConfiguration{BodyAssetID: "body-2", ClothingAssetIDs: []string{}}
	client := &fakeDynamoDB{
		queryOutput: &dynamodb.QueryOutput{
			Items:            []map[string]types.AttributeValue{marshalItem(t, firstRelation), marshalItem(t, secondRelation)},
			LastEvaluatedKey: map[string]types.AttributeValue{"relatedUserId": &types.AttributeValueMemberS{Value: "friend-1"}},
		},
		batchOutputs: []*dynamodb.BatchGetItemOutput{{Responses: map[string][]map[string]types.AttributeValue{
			"users": {
				marshalItem(t, userItem{UserID: "friend-1", Username: "one", DollConfiguration: firstDoll}),
				marshalItem(t, userItem{UserID: "friend-2", Username: "two", DollConfiguration: secondDoll}),
			},
		}}},
	}
	items, token, err := NewDynamoDBStore(client, "users", "friendships", "moods").ListFriendStatuses(context.Background(), "self", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Friend.UserID != "friend-2" || items[1].Friend.UserID != "friend-1" || token == "" {
		t.Fatalf("items=%#v token=%q", items, token)
	}
	if items[0].Doll == nil || items[0].Doll.BodyAssetID != "body-2" || items[1].Doll == nil || items[1].Doll.BodyAssetID != "body-1" {
		t.Fatalf("items=%#v", items)
	}
	if client.queryInput.IndexName == nil || *client.queryInput.IndexName != userStatusIndex || client.queryInput.Limit == nil || *client.queryInput.Limit != statusPageSize {
		t.Fatalf("query=%#v", client.queryInput)
	}
	status := client.queryInput.ExpressionAttributeValues[":status"].(*types.AttributeValueMemberS).Value
	if status != "ACCEPTED#" || len(client.batchInputs) != 1 {
		t.Fatalf("status=%q batch=%#v", status, client.batchInputs)
	}
	decoded, err := decodePageToken(token)
	if err != nil || decoded.UserID != "self" || decoded.RelatedUserID != "friend-1" {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
}

func TestListFriendStatusesAllowsFriendWithoutDollConfiguration(t *testing.T) {
	relation := relationshipItem{UserID: "self", RelatedUserID: "friend-1", StatusRelatedUserID: "ACCEPTED#friend-1"}
	client := &fakeDynamoDB{
		queryOutput: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{marshalItem(t, relation)}},
		batchOutputs: []*dynamodb.BatchGetItemOutput{{Responses: map[string][]map[string]types.AttributeValue{
			"users": {marshalItem(t, userItem{UserID: "friend-1", Username: "one", DisplayName: "One"})},
		}}},
	}

	items, token, err := NewDynamoDBStore(client, "users", "friendships", "moods").ListFriendStatuses(context.Background(), "self", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Friend.UserID != "friend-1" || items[0].Doll != nil || token != "" {
		t.Fatalf("items=%#v token=%q", items, token)
	}
}

func TestListFriendStatusesRejectsInvalidOrCrossUserTokenBeforeQuery(t *testing.T) {
	client := &fakeDynamoDB{}
	store := NewDynamoDBStore(client, "users", "friendships", "moods")
	for _, token := range []string{"not-a-token", mustToken(t, pageToken{UserID: "other", RelatedUserID: "friend"})} {
		_, _, err := store.ListFriendStatuses(context.Background(), "self", token)
		var domain *DomainError
		if !errors.As(err, &domain) || domain.Status != http.StatusBadRequest || client.queryInput != nil {
			t.Fatalf("token=%q err=%v query=%#v", token, err, client.queryInput)
		}
	}
}

func mustToken(t *testing.T, token pageToken) string {
	t.Helper()
	encoded, err := encodePageToken(token)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func mustMoodToken(t *testing.T, token moodPageToken) string {
	t.Helper()
	encoded, err := encodeMoodPageToken(token)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
