package notificationconsumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamoDB struct {
	queryOutputs []*dynamodb.QueryOutput
	queryInputs  []*dynamodb.QueryInput
	updateInput  *dynamodb.UpdateItemInput
	updateErr    error
}

func (f *fakeDynamoDB) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	f.queryInputs = append(f.queryInputs, input)
	output := f.queryOutputs[0]
	f.queryOutputs = f.queryOutputs[1:]
	return output, nil
}

func (f *fakeDynamoDB) UpdateItem(_ context.Context, input *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updateInput = input
	return &dynamodb.UpdateItemOutput{}, f.updateErr
}

func TestDynamoDBStoreListsEnabledDevicesAcrossPagesAndRecipients(t *testing.T) {
	device1, _ := attributevalue.MarshalMap(Device{UserID: "user-1", DeviceID: "device-1", PushToken: "token-1"})
	device2, _ := attributevalue.MarshalMap(Device{UserID: "user-2", DeviceID: "device-2", PushToken: "token-2"})
	client := &fakeDynamoDB{queryOutputs: []*dynamodb.QueryOutput{
		{Items: []map[string]types.AttributeValue{device1}, LastEvaluatedKey: map[string]types.AttributeValue{"deviceId": &types.AttributeValueMemberS{Value: "device-1"}}},
		{},
		{Items: []map[string]types.AttributeValue{device2}},
	}}
	devices, err := NewDynamoDBStore(client, "devices").ListEnabledDevices(context.Background(), []string{"user-1", "user-2"})
	if err != nil || len(devices) != 2 {
		t.Fatalf("devices=%#v err=%v", devices, err)
	}
	if len(client.queryInputs) != 3 || client.queryInputs[1].ExclusiveStartKey == nil {
		t.Fatalf("query inputs=%#v", client.queryInputs)
	}
	for _, input := range client.queryInputs {
		if input.FilterExpression == nil || *input.FilterExpression != "enabled = :enabled" {
			t.Fatalf("input does not filter enabled devices: %#v", input)
		}
	}
}

func TestDynamoDBStoreDisablesOnlyMatchingToken(t *testing.T) {
	client := &fakeDynamoDB{}
	store := NewDynamoDBStore(client, "devices")
	store.now = func() time.Time { return time.Date(2026, 8, 29, 20, 0, 0, 0, time.UTC) }
	device := Device{UserID: "user", DeviceID: "device", PushToken: "token"}
	if err := store.DisableDevice(context.Background(), device); err != nil {
		t.Fatal(err)
	}
	input := client.updateInput
	if input == nil || input.ConditionExpression == nil || *input.ConditionExpression != "pushToken = :pushToken" {
		t.Fatalf("input=%#v", input)
	}
	if enabled, ok := input.ExpressionAttributeValues[":disabled"].(*types.AttributeValueMemberBOOL); !ok || enabled.Value {
		t.Fatalf("disabled value=%#v", input.ExpressionAttributeValues[":disabled"])
	}
	updated := input.ExpressionAttributeValues[":updatedAt"].(*types.AttributeValueMemberS)
	if updated.Value != "2026-08-29T20:00:00Z" {
		t.Fatalf("updatedAt=%q", updated.Value)
	}
}

func TestDynamoDBStoreTreatsConcurrentTokenRefreshAsSuccess(t *testing.T) {
	client := &fakeDynamoDB{updateErr: &types.ConditionalCheckFailedException{Message: strptr("changed")}}
	if err := NewDynamoDBStore(client, "devices").DisableDevice(context.Background(), Device{UserID: "u", DeviceID: "d", PushToken: "old"}); err != nil {
		t.Fatal(err)
	}
	client.updateErr = errors.New("unavailable")
	if err := NewDynamoDBStore(client, "devices").DisableDevice(context.Background(), Device{UserID: "u", DeviceID: "d", PushToken: "old"}); err == nil {
		t.Fatal("expected update failure")
	}
}
