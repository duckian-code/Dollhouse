package profiledoll

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDB struct {
	getOutputs    []*dynamodb.GetItemOutput
	getInputs     []*dynamodb.GetItemInput
	putInput      *dynamodb.PutItemInput
	updateInput   *dynamodb.UpdateItemInput
	updateOutput  *dynamodb.UpdateItemOutput
	transactInput *dynamodb.TransactWriteItemsInput
	transactErr   error
}

func TestNormalizeUsernameTrimsAndFoldsCase(t *testing.T) {
	if got := NormalizeUsername("  ALIce  "); got != "alice" {
		t.Fatalf("normalized username = %q", got)
	}
}

func TestUsernameClaimCreatesAtomicReservationAndCompletesOnboarding(t *testing.T) {
	incomplete, complete := false, true
	current, _ := attributevalue.MarshalMap(userItem{UserID: "user-1", OnboardingComplete: &incomplete})
	updated, _ := attributevalue.MarshalMap(userItem{UserID: "user-1", Username: "Alice", NormalizedUsername: "alice", DisplayName: "Alice A", SearchPartition: "USER", OnboardingComplete: &complete})
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: current}, {Item: updated}}}
	username, displayName := "Alice", "Alice A"
	profile, err := NewDynamoDBStore(client, "users").UpdateProfile(context.Background(), "user-1", ProfileChanges{
		Username: OptionalString{Set: true, Value: &username}, DisplayName: OptionalString{Set: true, Value: &displayName},
	}, "2026-08-31T22:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.OnboardingComplete || client.transactInput == nil || len(client.transactInput.TransactItems) != 2 {
		t.Fatalf("profile = %#v, transaction = %#v", profile, client.transactInput)
	}
	put := client.transactInput.TransactItems[0].Put
	reservationKey := put.Item["userId"].(*types.AttributeValueMemberS).Value
	if reservationKey != "USERNAME#alice" || put.ConditionExpression == nil || !strings.Contains(*put.ConditionExpression, "attribute_not_exists") {
		t.Fatalf("reservation put = %#v", put)
	}
	update := client.transactInput.TransactItems[1].Update
	if update == nil || !strings.Contains(*update.UpdateExpression, "#searchPartition = :searchPartition") {
		t.Fatalf("profile update = %#v", update)
	}
}

func TestUsernameRenameReleasesPreviousReservationAtomically(t *testing.T) {
	complete := true
	current, _ := attributevalue.MarshalMap(userItem{UserID: "user-1", Username: "Alice", NormalizedUsername: "alice", DisplayName: "Alice", OnboardingComplete: &complete})
	updated, _ := attributevalue.MarshalMap(userItem{UserID: "user-1", Username: "Alex", NormalizedUsername: "alex", DisplayName: "Alice", OnboardingComplete: &complete})
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: current}, {Item: updated}}}
	username := "Alex"
	_, err := NewDynamoDBStore(client, "users").UpdateProfile(context.Background(), "user-1", ProfileChanges{Username: OptionalString{Set: true, Value: &username}}, "now")
	if err != nil {
		t.Fatal(err)
	}
	if len(client.transactInput.TransactItems) != 3 {
		t.Fatalf("transaction items = %d", len(client.transactInput.TransactItems))
	}
	deleted := client.transactInput.TransactItems[2].Delete.Key["userId"].(*types.AttributeValueMemberS).Value
	if deleted != "USERNAME#alice" {
		t.Fatalf("released reservation = %q", deleted)
	}
}

func TestConcurrentOrCaseCollidingClaimReturnsConflict(t *testing.T) {
	incomplete := false
	current, _ := attributevalue.MarshalMap(userItem{UserID: "user-2", OnboardingComplete: &incomplete})
	reason := "ConditionalCheckFailed"
	client := &fakeDynamoDB{
		getOutputs:  []*dynamodb.GetItemOutput{{Item: current}},
		transactErr: &types.TransactionCanceledException{CancellationReasons: []types.CancellationReason{{Code: &reason}}},
	}
	username := "aLiCe"
	_, err := NewDynamoDBStore(client, "users").UpdateProfile(context.Background(), "user-2", ProfileChanges{Username: OptionalString{Set: true, Value: &username}}, "now")
	var domain *DomainError
	if !errors.As(err, &domain) || domain.Status != http.StatusConflict || domain.Code != "conflict" {
		t.Fatalf("error = %#v", err)
	}
}

func TestUsernameAvailabilityIsCaseInsensitiveAndOwnClaimIsAvailable(t *testing.T) {
	reservation, _ := attributevalue.MarshalMap(usernameReservation{UserID: "USERNAME#alice", OwnerUserID: "user-1", NormalizedUsername: "alice"})
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: reservation}}}
	available, err := NewDynamoDBStore(client, "users").UsernameAvailable(context.Background(), NormalizeUsername("ALICE"), "user-1")
	if err != nil || !available {
		t.Fatalf("available = %v, err = %v", available, err)
	}
	key := client.getInputs[0].Key["userId"].(*types.AttributeValueMemberS).Value
	if key != "USERNAME#alice" || client.getInputs[0].ConsistentRead == nil || !*client.getInputs[0].ConsistentRead {
		t.Fatalf("availability read = %#v", client.getInputs[0])
	}
}

func TestLegacyProfileMigrationRemovesGeneratedIdentityAndEmailFields(t *testing.T) {
	legacy, _ := attributevalue.MarshalMap(map[string]any{
		"userId": "user-1", "cognitoSub": "user-1", "username": "login@example.test",
		"normalizedUsername": "login@example.test", "displayName": "login@example.test",
		"searchPartition": "USER", "email": "login@example.test", "normalizedEmail": "login@example.test",
	})
	incomplete := false
	migrated, _ := attributevalue.MarshalMap(userItem{UserID: "user-1", CognitoSub: "user-1", OnboardingComplete: &incomplete})
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: legacy}}, updateOutput: &dynamodb.UpdateItemOutput{Attributes: migrated}}
	profile, err := NewDynamoDBStore(client, "users").EnsureUser(context.Background(), Identity{UserID: "user-1"}, "now")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Username != "" || profile.DisplayName != "" || profile.OnboardingComplete || strings.Contains(strings.ToLower(profile.Username+profile.DisplayName), "example.test") {
		t.Fatalf("migrated profile = %#v", profile)
	}
	for _, field := range []string{"#username", "#normalizedUsername", "#searchPartition", "#displayName", "#email", "#normalizedEmail"} {
		if !strings.Contains(*client.updateInput.UpdateExpression, field) {
			t.Fatalf("migration did not remove %s: %s", field, *client.updateInput.UpdateExpression)
		}
	}
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

func (f *fakeDynamoDB) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.transactInput = input
	return &dynamodb.TransactWriteItemsOutput{}, f.transactErr
}

func TestEnsureUserCreatesRecordFromVerifiedIdentity(t *testing.T) {
	client := &fakeDynamoDB{}
	store := NewDynamoDBStore(client, "users")
	profile, err := store.EnsureUser(context.Background(), Identity{UserID: "sub-123"}, "2026-08-13T18:30:45Z")
	if err != nil {
		t.Fatal(err)
	}
	if profile.UserID != "sub-123" || profile.Username != "" || profile.DisplayName != "" || profile.OnboardingComplete {
		t.Fatalf("profile = %#v", profile)
	}
	if client.putInput == nil {
		t.Fatal("PutItem was not called")
	}
	var item userItem
	if err := attributevalue.UnmarshalMap(client.putInput.Item, &item); err != nil {
		t.Fatal(err)
	}
	if item.UserID != "sub-123" || item.CognitoSub != "sub-123" || item.Username != "" || item.NormalizedUsername != "" || item.Role != "USER" || item.OnboardingComplete == nil || *item.OnboardingComplete {
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
	complete := true
	currentItem, err := attributevalue.MarshalMap(userItem{UserID: "sub-123", Username: "alice", NormalizedUsername: "alice", DisplayName: "Alice", OnboardingComplete: &complete, CreatedAt: "created", UpdatedAt: "old"})
	if err != nil {
		t.Fatal(err)
	}
	updatedItem, err := attributevalue.MarshalMap(userItem{UserID: "sub-123", Username: "alice", DisplayName: displayName, CreatedAt: "created", UpdatedAt: "updated"})
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeDynamoDB{getOutputs: []*dynamodb.GetItemOutput{{Item: currentItem}}, updateOutput: &dynamodb.UpdateItemOutput{Attributes: updatedItem}}
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
