#!/usr/bin/env bash
set -euo pipefail

stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"

stack_output() {
  aws cloudformation describe-stacks \
    --stack-name "$stack_name" \
    --region "$region" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue | [0]" \
    --output text
}

assert_value() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [[ "$actual" != "$expected" ]]; then
    echo "$label: expected $expected, received $actual" >&2
    exit 1
  fi
  echo "$label: ok"
}

assert_positive() {
  local label="$1"
  local actual="$2"
  if (( actual < 1 )); then
    echo "$label: expected at least one item, received $actual" >&2
    exit 1
  fi
  echo "$label: ok ($actual item(s))"
}

verify_table_protection() {
  local table="$1"
  local status
  local encryption
  local backups
  status="$(aws dynamodb describe-table --region "$region" --table-name "$table" --query 'Table.TableStatus' --output text)"
  encryption="$(aws dynamodb describe-table --region "$region" --table-name "$table" --query 'Table.SSEDescription.Status' --output text)"
  backups="$(aws dynamodb describe-continuous-backups --region "$region" --table-name "$table" --query 'ContinuousBackupsDescription.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus' --output text)"
  assert_value "$table active" "ACTIVE" "$status"
  assert_value "$table encryption" "ENABLED" "$encryption"
  assert_value "$table point-in-time recovery" "ENABLED" "$backups"
}

users_table="$(stack_output UsersTableName)"
friendships_table="$(stack_output FriendshipsTableName)"
mood_events_table="$(stack_output MoodEventsTableName)"
devices_table="$(stack_output DevicesTableName)"

for table in "$users_table" "$friendships_table" "$mood_events_table" "$devices_table"; do
  verify_table_protection "$table"
done

profile="$(aws dynamodb get-item --region "$region" --table-name "$users_table" --key '{"userId":{"S":"TEST#USER#ALICE"}}' --query 'Item.userId.S' --output text)"
assert_value "profile lookup" "TEST#USER#ALICE" "$profile"

search_count="$(aws dynamodb query --region "$region" --table-name "$users_table" --index-name UserSearchIndex --key-condition-expression 'searchPartition = :partition AND begins_with(normalizedUsername, :prefix)' --expression-attribute-values '{":partition":{"S":"USER"},":prefix":{"S":"test-a"}}' --select COUNT --query Count --output text)"
assert_positive "username prefix search" "$search_count"

accepted_count="$(aws dynamodb query --region "$region" --table-name "$friendships_table" --index-name UserStatusIndex --key-condition-expression 'userId = :user AND begins_with(statusRelatedUserId, :status)' --expression-attribute-values '{":user":{"S":"TEST#USER#ALICE"},":status":{"S":"ACCEPTED#"}}' --select COUNT --query Count --output text)"
assert_positive "accepted friends lookup" "$accepted_count"

incoming_count="$(aws dynamodb query --region "$region" --table-name "$friendships_table" --index-name UserStatusIndex --key-condition-expression 'userId = :user AND begins_with(statusRelatedUserId, :status)' --expression-attribute-values '{":user":{"S":"TEST#USER#CAROL"},":status":{"S":"PENDING_INCOMING#"}}' --select COUNT --query Count --output text)"
assert_positive "incoming request lookup" "$incoming_count"

request_count="$(aws dynamodb query --region "$region" --table-name "$friendships_table" --index-name RequestIdIndex --key-condition-expression 'requestId = :request' --expression-attribute-values '{":request":{"S":"TEST#REQUEST#002"}}' --select COUNT --query Count --output text)"
assert_value "request ID lookup" "2" "$request_count"

mood_count="$(aws dynamodb query --region "$region" --table-name "$mood_events_table" --key-condition-expression 'userId = :user' --expression-attribute-values '{":user":{"S":"TEST#USER#BOB"}}' --select COUNT --query Count --output text)"
assert_positive "mood history lookup" "$mood_count"

status_value="$(aws dynamodb get-item --region "$region" --table-name "$mood_events_table" --key '{"userId":{"S":"TEST#USER#BOB"},"occurredAt":{"S":"2026-08-08T17:00:00Z#TEST#EVENT#001"}}' --query 'Item.status.S' --output text)"
assert_value "open status value" "Doing well" "$status_value"

device_count="$(aws dynamodb query --region "$region" --table-name "$devices_table" --key-condition-expression 'userId = :user' --expression-attribute-values '{":user":{"S":"TEST#USER#BOB"}}' --select COUNT --query Count --output text)"
assert_positive "device lookup" "$device_count"

enabled_device_count="$(aws dynamodb query --region "$region" --table-name "$devices_table" --key-condition-expression 'userId = :user' --filter-expression 'enabled = :enabled' --expression-attribute-values '{":user":{"S":"TEST#USER#BOB"},":enabled":{"BOOL":true}}' --select COUNT --query Count --output text)"
assert_positive "enabled notification target lookup" "$enabled_device_count"

echo "All DynamoDB access patterns verified against $stack_name ($region)."
