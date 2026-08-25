package profiledoll

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDB struct {
	getOutputs   []*dynamodb.GetItemOutput
	getInputs    []*dynamodb.GetItemInput
	putInput     *dynamodb.PutItemInput
	updateInput  *dynamodb.UpdateItemInput
	updateOutput *dynamodb.UpdateItemOutput
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

func (f *fakeDynamoDB) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.putInput = input
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDynamoDB) UpdateItem(_ context.Context, input *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updateInput = input
	if f.updateOutput != nil {
		return f.updateOutput, nil
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

func TestEnsureUserCreatesRecordFromVerifiedIdentity(t *testing.T) {
	client := &fakeDynamoDB{}
	store := NewDynamoDBStore(client, "users")
	profile, err := store.EnsureUser(context.Background(), Identity{UserID: "sub-123", Username: "Alice", DisplayName: "Alice A"}, "2026-08-13T18:30:45Z")
	if err != nil {
		t.Fatal(err)
	}
	if profile.UserID != "sub-123" || profile.Username != "Alice" {
		t.Fatalf("profile = %#v", profile)
	}
	if client.putInput == nil {
		t.Fatal("PutItem was not called")
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(client.putInput.Item, &item); err != nil {
		t.Fatal(err)
	}
	if item.UserID != "sub-123" || item.CognitoSub != "sub-123" || item.NormalizedUsername != "alice" || item.Role != "USER" {
		t.Fatalf("stored item = %#v", item)
	}
}

func TestGetDollReadsOnlyRequestedUserKey(t *testing.T) {
	configuration := DollConfiguration{BodyAssetID: "body-1", ClothingAssetIDs: []string{}}
	item, err := attributevalue.MarshalMap(userItem{UserID: "sub-456", DollConfiguration: &configuration})
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: item}}}
	got, err := NewDynamoDBStore(client, "users").GetDoll(context.Background(), "sub-456")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.BodyAssetID != "body-1" {
		t.Fatalf("configuration = %#v", got)
	}
	key, ok := client.getInputs[0].Key["userId"].(*types.AttributeValueMemberS)
	if !ok || key.Value != "sub-456" {
		t.Fatalf("read key = %#v", client.getInputs[0].Key)
	}
	if client.getInputs[0].ConsistentRead == nil || !*client.getInputs[0].ConsistentRead {
		t.Fatal("user read was not strongly consistent")
	}
}

func TestUpdateProfileTargetsAuthenticatedUserAndPreservesOmittedFields(t *testing.T) {
	displayName := "Alice Updated"
	updatedItem, err := attributevalue.MarshalMap(userItem{UserID: "sub-123", Username: "alice", DisplayName: displayName, CreatedAt: "created", UpdatedAt: "updated"})
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeDynamoDB{updateOutput: &dynamodb.UpdateItemOutput{Attributes: updatedItem}}
	changes := ProfileChanges{DisplayName: OptionalString{Set: true, Value: &displayName}}
	_, err = NewDynamoDBStore(client, "users").UpdateProfile(context.Background(), "sub-123", changes, "updated")
	if err != nil {
		t.Fatal(err)
	}
	key := client.updateInput.Key["userId"].(*types.AttributeValueMemberS)
	if key.Value != "sub-123" {
		t.Fatalf("update key = %q", key.Value)
	}
	if _, exists := client.updateInput.ExpressionAttributeNames["#username"]; exists {
		t.Fatal("omitted username was included in update")
	}
	if client.updateInput.ConditionExpression == nil || *client.updateInput.ConditionExpression != "attribute_exists(userId)" {
		t.Fatalf("condition = %#v", client.updateInput.ConditionExpression)
	}
}

func TestUpdateDollTargetsUserAndSetsServerTimestamp(t *testing.T) {
	client := &fakeDynamoDB{}
	configuration := DollConfiguration{BodyAssetID: "body-1", HairAssetID: "hair-1", EyesAssetID: "eyes-1", NoseAssetID: "nose-1", MouthAssetID: "mouth-1", ClothingAssetIDs: []string{}}
	got, err := NewDynamoDBStore(client, "users").UpdateDoll(context.Background(), "sub-789", configuration, "2026-08-13T18:30:45Z")
	if err != nil {
		t.Fatal(err)
	}
	if got.UpdatedAt != "2026-08-13T18:30:45Z" {
		t.Fatalf("updatedAt = %q", got.UpdatedAt)
	}
	key := client.updateInput.Key["userId"].(*types.AttributeValueMemberS)
	if key.Value != "sub-789" {
		t.Fatalf("update key = %q", key.Value)
	}
	var stored DollConfiguration
	if err := attributevalue.Unmarshal(client.updateInput.ExpressionAttributeValues[":configuration"], &stored); err != nil {
		t.Fatal(err)
	}
	if stored.UpdatedAt != got.UpdatedAt {
		t.Fatalf("stored configuration = %#v", stored)
	}
}
