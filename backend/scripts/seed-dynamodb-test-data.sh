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

users_table="$(stack_output UsersTableName)"
friendships_table="$(stack_output FriendshipsTableName)"
mood_events_table="$(stack_output MoodEventsTableName)"
devices_table="$(stack_output DevicesTableName)"

aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
  '{"userId":{"S":"TEST#USER#ALICE"},"normalizedEmail":{"S":"alice@dollhouse.test"},"searchPartition":{"S":"USER"},"normalizedUsername":{"S":"test-alice"},"displayName":{"S":"Test Alice"},"dollConfiguration":{"M":{"baseAssetId":{"S":"TEST#ASSET#DOLL"}}}}'
aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"normalizedEmail":{"S":"bob@dollhouse.test"},"searchPartition":{"S":"USER"},"normalizedUsername":{"S":"test-bob"},"displayName":{"S":"Test Bob"},"currentStatus":{"M":{"moodId":{"S":"TEST#MOOD#HAPPY"},"energy":{"N":"8"},"updatedAt":{"S":"2026-08-08T17:00:00.000Z"}}}}'
aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
  '{"userId":{"S":"TEST#USER#CAROL"},"normalizedEmail":{"S":"carol@dollhouse.test"},"searchPartition":{"S":"USER"},"normalizedUsername":{"S":"test-carol"},"displayName":{"S":"Test Carol"}}'

aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#ALICE"},"relatedUserId":{"S":"TEST#USER#BOB"},"status":{"S":"ACCEPTED"},"statusRelatedUserId":{"S":"ACCEPTED#TEST#USER#BOB"},"requestedBy":{"S":"TEST#USER#ALICE"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"relatedUserId":{"S":"TEST#USER#ALICE"},"status":{"S":"ACCEPTED"},"statusRelatedUserId":{"S":"ACCEPTED#TEST#USER#ALICE"},"requestedBy":{"S":"TEST#USER#ALICE"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#ALICE"},"relatedUserId":{"S":"TEST#USER#CAROL"},"status":{"S":"PENDING_OUTGOING"},"statusRelatedUserId":{"S":"PENDING_OUTGOING#TEST#USER#CAROL"},"requestedBy":{"S":"TEST#USER#ALICE"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#CAROL"},"relatedUserId":{"S":"TEST#USER#ALICE"},"status":{"S":"PENDING_INCOMING"},"statusRelatedUserId":{"S":"PENDING_INCOMING#TEST#USER#ALICE"},"requestedBy":{"S":"TEST#USER#ALICE"}}'

aws dynamodb put-item --region "$region" --table-name "$mood_events_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"occurredAt":{"S":"2026-08-08T17:00:00.000Z#TEST#EVENT#001"},"moodId":{"S":"TEST#MOOD#HAPPY"},"energy":{"N":"8"}}'
aws dynamodb put-item --region "$region" --table-name "$devices_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"deviceId":{"S":"TEST#DEVICE#001"},"platform":{"S":"ios"},"pushToken":{"S":"TEST#PUSH#TOKEN"},"updatedAt":{"S":"2026-08-08T17:00:00.000Z"}}'

echo "Seeded deterministic TEST# records into $stack_name ($region)."
