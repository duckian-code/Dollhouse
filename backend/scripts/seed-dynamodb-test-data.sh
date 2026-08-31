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
  '{"userId":{"S":"TEST#USER#ALICE"},"cognitoSub":{"S":"TEST#SUB#ALICE"},"searchPartition":{"S":"USER"},"username":{"S":"test-alice"},"normalizedUsername":{"S":"test-alice"},"displayName":{"S":"Test Alice"},"onboardingComplete":{"BOOL":true},"role":{"S":"USER"},"bio":{"S":"Fixture account"},"dollConfiguration":{"M":{"bodyAssetId":{"S":"TEST#ASSET#BODY"},"hairAssetId":{"S":"TEST#ASSET#HAIR"},"eyesAssetId":{"S":"TEST#ASSET#EYES"},"noseAssetId":{"S":"TEST#ASSET#NOSE"},"mouthAssetId":{"S":"TEST#ASSET#MOUTH"},"clothingAssetIds":{"L":[{"S":"TEST#ASSET#SHIRT"}]},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}},"createdAt":{"S":"2026-08-08T16:00:00Z"},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"cognitoSub":{"S":"TEST#SUB#BOB"},"searchPartition":{"S":"USER"},"username":{"S":"test-bob"},"normalizedUsername":{"S":"test-bob"},"displayName":{"S":"Test Bob"},"onboardingComplete":{"BOOL":true},"role":{"S":"USER"},"currentStatus":{"M":{"status":{"S":"Doing well"},"stress":{"N":"2"},"fatigue":{"NULL":true},"discomfort":{"N":"0"},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}},"createdAt":{"S":"2026-08-08T16:00:00Z"},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
  '{"userId":{"S":"TEST#USER#CAROL"},"cognitoSub":{"S":"TEST#SUB#CAROL"},"searchPartition":{"S":"USER"},"username":{"S":"test-carol"},"normalizedUsername":{"S":"test-carol"},"displayName":{"S":"Test Carol"},"onboardingComplete":{"BOOL":true},"role":{"S":"USER"},"createdAt":{"S":"2026-08-08T16:00:00Z"},"updatedAt":{"S":"2026-08-08T16:00:00Z"}}'

for reservation in test-alice test-bob test-carol; do
  owner="TEST#USER#${reservation#test-}"
  owner="${owner^^}"
  aws dynamodb put-item --region "$region" --table-name "$users_table" --item \
    "$(jq -cn --arg key "USERNAME#$reservation" --arg owner "$owner" --arg username "$reservation" '{userId:{S:$key},entityType:{S:"USERNAME_RESERVATION"},ownerUserId:{S:$owner},normalizedUsername:{S:$username},createdAt:{S:"2026-08-08T16:00:00Z"},updatedAt:{S:"2026-08-08T16:00:00Z"}}')"
done

aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#ALICE"},"relatedUserId":{"S":"TEST#USER#BOB"},"requestId":{"S":"TEST#REQUEST#001"},"status":{"S":"ACCEPTED"},"statusRelatedUserId":{"S":"ACCEPTED#TEST#USER#BOB"},"requestedBy":{"S":"TEST#USER#ALICE"},"requestedAt":{"S":"2026-08-08T16:30:00Z"},"acceptedAt":{"S":"2026-08-08T16:45:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"relatedUserId":{"S":"TEST#USER#ALICE"},"requestId":{"S":"TEST#REQUEST#001"},"status":{"S":"ACCEPTED"},"statusRelatedUserId":{"S":"ACCEPTED#TEST#USER#ALICE"},"requestedBy":{"S":"TEST#USER#ALICE"},"requestedAt":{"S":"2026-08-08T16:30:00Z"},"acceptedAt":{"S":"2026-08-08T16:45:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#ALICE"},"relatedUserId":{"S":"TEST#USER#CAROL"},"requestId":{"S":"TEST#REQUEST#002"},"status":{"S":"PENDING_OUTGOING"},"statusRelatedUserId":{"S":"PENDING_OUTGOING#TEST#USER#CAROL"},"requestedBy":{"S":"TEST#USER#ALICE"},"requestedAt":{"S":"2026-08-08T17:00:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$friendships_table" --item \
  '{"userId":{"S":"TEST#USER#CAROL"},"relatedUserId":{"S":"TEST#USER#ALICE"},"requestId":{"S":"TEST#REQUEST#002"},"status":{"S":"PENDING_INCOMING"},"statusRelatedUserId":{"S":"PENDING_INCOMING#TEST#USER#ALICE"},"requestedBy":{"S":"TEST#USER#ALICE"},"requestedAt":{"S":"2026-08-08T17:00:00Z"}}'

# Remove the pre-contract-change fixture whose sort key and fields used the
# retired moodId model. This key is deterministic and reserved for test data.
aws dynamodb delete-item --region "$region" --table-name "$mood_events_table" --key \
  '{"userId":{"S":"TEST#USER#BOB"},"occurredAt":{"S":"2026-08-08T17:00:00.000Z#TEST#EVENT#001"}}'
aws dynamodb put-item --region "$region" --table-name "$mood_events_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"occurredAt":{"S":"2026-08-08T17:00:00Z#TEST#EVENT#001"},"eventId":{"S":"TEST#EVENT#001"},"status":{"S":"Doing well"},"stress":{"N":"2"},"fatigue":{"NULL":true},"discomfort":{"N":"0"},"visibility":{"S":"FRIENDS"},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}'
aws dynamodb put-item --region "$region" --table-name "$devices_table" --item \
  '{"userId":{"S":"TEST#USER#BOB"},"deviceId":{"S":"TEST#DEVICE#001"},"platform":{"S":"ios"},"pushToken":{"S":"TEST#PUSH#TOKEN"},"enabled":{"BOOL":true},"updatedAt":{"S":"2026-08-08T17:00:00Z"}}'

echo "Seeded deterministic TEST# records into $stack_name ($region)."
