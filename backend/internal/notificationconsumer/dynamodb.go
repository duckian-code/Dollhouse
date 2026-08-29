package notificationconsumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamodbAPI interface {
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// DynamoDBStore reads and updates the Devices table.
type DynamoDBStore struct {
	client    dynamodbAPI
	tableName string
	now       func() time.Time
}

// NewDynamoDBStore creates a Devices table store.
func NewDynamoDBStore(client dynamodbAPI, tableName string) *DynamoDBStore {
	return &DynamoDBStore{client: client, tableName: tableName, now: time.Now}
}

// ListEnabledDevices returns enabled devices for all supplied recipients.
func (s *DynamoDBStore) ListEnabledDevices(ctx context.Context, userIDs []string) ([]Device, error) {
	if s.client == nil || s.tableName == "" {
		return nil, errors.New("devices table configuration is incomplete")
	}
	devices := make([]Device, 0)
	for _, userID := range userIDs {
		input := &dynamodb.QueryInput{
			TableName:              &s.tableName,
			KeyConditionExpression: strptr("userId = :userId"),
			FilterExpression:       strptr("enabled = :enabled"),
			ProjectionExpression:   strptr("userId, deviceId, pushToken"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":userId":  &types.AttributeValueMemberS{Value: userID},
				":enabled": &types.AttributeValueMemberBOOL{Value: true},
			},
		}
		for {
			output, err := s.client.Query(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("query devices for recipient: %w", err)
			}
			var page []Device
			if err := attributevalue.UnmarshalListOfMaps(output.Items, &page); err != nil {
				return nil, fmt.Errorf("decode devices: %w", err)
			}
			for _, device := range page {
				if device.UserID == "" || device.DeviceID == "" || device.PushToken == "" {
					return nil, errors.New("enabled device record is incomplete")
				}
				devices = append(devices, device)
			}
			if len(output.LastEvaluatedKey) == 0 {
				break
			}
			input.ExclusiveStartKey = output.LastEvaluatedKey
		}
	}
	return devices, nil
}

// DisableDevice disables an invalid token only if the record still contains
// the token Expo rejected, preserving a concurrently refreshed registration.
func (s *DynamoDBStore) DisableDevice(ctx context.Context, device Device) error {
	if s.client == nil || s.tableName == "" {
		return errors.New("devices table configuration is incomplete")
	}
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"userId":   &types.AttributeValueMemberS{Value: device.UserID},
			"deviceId": &types.AttributeValueMemberS{Value: device.DeviceID},
		},
		UpdateExpression:    strptr("SET enabled = :disabled, updatedAt = :updatedAt"),
		ConditionExpression: strptr("pushToken = :pushToken"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":disabled":  &types.AttributeValueMemberBOOL{Value: false},
			":updatedAt": &types.AttributeValueMemberS{Value: s.now().UTC().Format(time.RFC3339)},
			":pushToken": &types.AttributeValueMemberS{Value: device.PushToken},
		},
	})
	if err != nil {
		var conditionFailure *types.ConditionalCheckFailedException
		if errors.As(err, &conditionFailure) {
			return nil
		}
		return fmt.Errorf("update invalid device: %w", err)
	}
	return nil
}

func strptr(value string) *string { return &value }
