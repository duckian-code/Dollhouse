package friendship

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
	getOutputs    []*dynamodb.GetItemOutput
	queryOutputs  []*dynamodb.QueryOutput
	getInputs     []*dynamodb.GetItemInput
	queryInputs   []*dynamodb.QueryInput
	transactInput *dynamodb.TransactWriteItemsInput
	transactErr   error
}

func (f *fakeDynamoDB) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.getInputs = append(f.getInputs, input)
	if len(f.getOutputs) == 0 {
		return &dynamodb.GetItemOutput{}, nil
	}
	output := f.getOutputs[0]
	f.getOutputs = f.getOutputs[1:]
	return output, nil
}
func (f *fakeDynamoDB) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	f.queryInputs = append(f.queryInputs, input)
	if len(f.queryOutputs) == 0 {
		return &dynamodb.QueryOutput{}, nil
	}
	output := f.queryOutputs[0]
	f.queryOutputs = f.queryOutputs[1:]
	return output, nil
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

func TestSearchUsersQueriesPrefixExcludesSelfAndReturnsOpaqueToken(t *testing.T) {
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{{
		Items: []map[string]types.AttributeValue{
			marshalItem(t, userItem{UserID: "self", Username: "alice", NormalizedUsername: "alice", DisplayName: "Alice"}),
			marshalItem(t, userItem{UserID: "other", Username: "alex", NormalizedUsername: "alex", DisplayName: "Alex"}),
		},
		LastEvaluatedKey: map[string]types.AttributeValue{"normalizedUsername": &types.AttributeValueMemberS{Value: "alex"}, "userId": &types.AttributeValueMemberS{Value: "other"}},
	}}}
	store := NewDynamoDBStore(client, "users", "friendships")
	items, token, err := store.SearchUsers(context.Background(), " Al ", "", "self")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UserID != "other" || token == "" {
		t.Fatalf("items=%#v token=%q", items, token)
	}
	input := client.queryInputs[0]
	if input.IndexName == nil || *input.IndexName != userSearchIndex {
		t.Fatalf("input=%#v", input)
	}
	query := input.ExpressionAttributeValues[":query"].(*types.AttributeValueMemberS).Value
	if query != "al" {
		t.Fatalf("query=%q", query)
	}
	decoded, err := decodePageToken(token)
	if err != nil || decoded.UserID != "other" || decoded.Username != "alex" {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
}

func TestSearchUsersRejectsInvalidTokenBeforeQuery(t *testing.T) {
	client := &fakeDynamoDB{}
	_, _, err := NewDynamoDBStore(client, "users", "friendships").SearchUsers(context.Background(), "a", "not-token", "self")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusBadRequest || len(client.queryInputs) != 0 {
		t.Fatalf("err=%v inputs=%d", err, len(client.queryInputs))
	}
}

func TestSendRequestWritesMirroredPendingRecordsAtomically(t *testing.T) {
	target := marshalItem(t, userItem{UserID: "bob", Username: "bob", DisplayName: "Bob"})
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: target}}}
	got, err := NewDynamoDBStore(client, "users", "friendships").SendRequest(context.Background(), "alice", "bob", "request-1", "2026-08-16T18:30:45Z")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusPendingOutgoing || got.User.UserID != "bob" {
		t.Fatalf("request=%#v", got)
	}
	if client.transactInput == nil || len(client.transactInput.TransactItems) != 2 {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
	var outgoing, incoming relationshipItem
	if err := attributevalue.UnmarshalMap(client.transactInput.TransactItems[0].Put.Item, &outgoing); err != nil {
		t.Fatal(err)
	}
	if err := attributevalue.UnmarshalMap(client.transactInput.TransactItems[1].Put.Item, &incoming); err != nil {
		t.Fatal(err)
	}
	if outgoing.UserID != "alice" || outgoing.Status != StatusPendingOutgoing || incoming.UserID != "bob" || incoming.Status != StatusPendingIncoming || outgoing.RequestID != incoming.RequestID {
		t.Fatalf("out=%#v in=%#v", outgoing, incoming)
	}
	if client.transactInput.TransactItems[0].Put.ConditionExpression == nil {
		t.Fatal("missing duplicate condition")
	}
}

func TestSendRequestMapsConditionalCancellationToConflict(t *testing.T) {
	code := "ConditionalCheckFailed"
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: marshalItem(t, userItem{UserID: "bob"})}}, transactErr: &types.TransactionCanceledException{CancellationReasons: []types.CancellationReason{{Code: &code}}}}
	_, err := NewDynamoDBStore(client, "users", "friendships").SendRequest(context.Background(), "alice", "bob", "request-1", "now")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusConflict {
		t.Fatalf("err=%v", err)
	}
}

func TestListRequestsHydratesIncomingAndOutgoingUsers(t *testing.T) {
	incoming := relationshipItem{UserID: "alice", RelatedUserID: "bob", RequestID: "in", Status: StatusPendingIncoming, RequestedAt: "one"}
	outgoing := relationshipItem{UserID: "alice", RelatedUserID: "carol", RequestID: "out", Status: StatusPendingOutgoing, RequestedAt: "two"}
	client := &fakeDynamoDB{
		queryOutputs: []*dynamodb.QueryOutput{{Items: []map[string]types.AttributeValue{marshalItem(t, incoming)}}, {Items: []map[string]types.AttributeValue{marshalItem(t, outgoing)}}},
		getOutputs:   []*dynamodb.GetItemOutput{{Item: marshalItem(t, userItem{UserID: "bob", Username: "bob"})}, {Item: marshalItem(t, userItem{UserID: "carol", Username: "carol"})}},
	}
	in, out, err := NewDynamoDBStore(client, "users", "friendships").ListRequests(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 1 || in[0].User.UserID != "bob" || len(out) != 1 || out[0].User.UserID != "carol" {
		t.Fatalf("in=%#v out=%#v", in, out)
	}
	if *client.queryInputs[0].IndexName != userStatusIndex || *client.queryInputs[1].IndexName != userStatusIndex {
		t.Fatalf("queries=%#v", client.queryInputs)
	}
}

func TestAcceptRequestAuthorizesRecipientAndUpdatesBothSides(t *testing.T) {
	incoming := relationshipItem{UserID: "bob", RelatedUserID: "alice", RequestID: "request-1", Status: StatusPendingIncoming}
	outgoing := relationshipItem{UserID: "alice", RelatedUserID: "bob", RequestID: "request-1", Status: StatusPendingOutgoing}
	client := &fakeDynamoDB{
		queryOutputs: []*dynamodb.QueryOutput{{Items: []map[string]types.AttributeValue{marshalItem(t, incoming), marshalItem(t, outgoing)}}},
		getOutputs:   []*dynamodb.GetItemOutput{{Item: marshalItem(t, userItem{UserID: "alice", Username: "alice", DisplayName: "Alice"})}},
	}
	got, err := NewDynamoDBStore(client, "users", "friendships").AcceptRequest(context.Background(), "bob", "request-1", "accepted")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusAccepted || got.Friend.UserID != "alice" || got.AcceptedAt != "accepted" {
		t.Fatalf("friendship=%#v", got)
	}
	if len(client.transactInput.TransactItems) != 2 || client.transactInput.TransactItems[0].Update == nil || client.transactInput.TransactItems[1].Update == nil {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
	firstValues := client.transactInput.TransactItems[0].Update.ExpressionAttributeValues
	secondValues := client.transactInput.TransactItems[1].Update.ExpressionAttributeValues
	if firstValues[":expected"].(*types.AttributeValueMemberS).Value != StatusPendingIncoming || secondValues[":expected"].(*types.AttributeValueMemberS).Value != StatusPendingOutgoing {
		t.Fatalf("values=%#v %#v", firstValues, secondValues)
	}
}

func TestAcceptRequestRejectsSender(t *testing.T) {
	outgoing := relationshipItem{UserID: "alice", RelatedUserID: "bob", RequestID: "request-1", Status: StatusPendingOutgoing}
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{{Items: []map[string]types.AttributeValue{marshalItem(t, outgoing)}}}}
	_, err := NewDynamoDBStore(client, "users", "friendships").AcceptRequest(context.Background(), "alice", "request-1", "accepted")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusForbidden || client.transactInput != nil {
		t.Fatalf("err=%v transaction=%#v", err, client.transactInput)
	}
}

func TestDeclineRequestRejectsNonRecipient(t *testing.T) {
	outgoing := relationshipItem{UserID: "alice", RelatedUserID: "bob", RequestID: "request-1", Status: StatusPendingOutgoing}
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{{Items: []map[string]types.AttributeValue{marshalItem(t, outgoing)}}}}
	err := NewDynamoDBStore(client, "users", "friendships").DeclineRequest(context.Background(), "alice", "request-1")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusForbidden || domain.Code != "forbidden" || client.transactInput != nil {
		t.Fatalf("err=%v transaction=%#v", err, client.transactInput)
	}
}

func TestRemoveFriendCannotDeleteAnotherUsersRelationship(t *testing.T) {
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{}}}
	err := NewDynamoDBStore(client, "users", "friendships").RemoveFriend(context.Background(), "alice", "stranger")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusNotFound || domain.Code != "not_found" || client.transactInput != nil {
		t.Fatalf("err=%v transaction=%#v", err, client.transactInput)
	}
	key := client.getInputs[0].Key
	if key["userId"].(*types.AttributeValueMemberS).Value != "alice" || key["relatedUserId"].(*types.AttributeValueMemberS).Value != "stranger" {
		t.Fatalf("key=%#v", key)
	}
}

func TestDeclineAndRemoveDeleteMirroredRecordsAtomically(t *testing.T) {
	incoming := relationshipItem{UserID: "bob", RelatedUserID: "alice", RequestID: "request-1", Status: StatusPendingIncoming}
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{{Items: []map[string]types.AttributeValue{marshalItem(t, incoming)}}}}
	store := NewDynamoDBStore(client, "users", "friendships")
	if err := store.DeclineRequest(context.Background(), "bob", "request-1"); err != nil {
		t.Fatal(err)
	}
	if len(client.transactInput.TransactItems) != 2 || client.transactInput.TransactItems[0].Delete == nil {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
	accepted := relationshipItem{UserID: "alice", RelatedUserID: "bob", RequestID: "request-2", Status: StatusAccepted}
	client.getOutputs = []*dynamodb.GetItemOutput{{Item: marshalItem(t, accepted)}}
	if err := store.RemoveFriend(context.Background(), "alice", "bob"); err != nil {
		t.Fatal(err)
	}
	if len(client.transactInput.TransactItems) != 2 || client.transactInput.TransactItems[1].Delete == nil {
		t.Fatalf("transaction=%#v", client.transactInput)
	}
}
