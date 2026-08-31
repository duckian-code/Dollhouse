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

assert_equals() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [[ "$actual" != "$expected" ]]; then
    echo "$label: expected '$expected', received '$actual'" >&2
    exit 1
  fi
  echo "$label: verified"
}

assert_decision() {
  local label="$1"
  local role_name="$2"
  local action="$3"
  local resource="$4"
  local expected="$5"
  local actual
  actual="$(aws iam simulate-principal-policy \
    --policy-source-arn "arn:aws:iam::${account_id}:role/$role_name" \
    --action-names "$action" \
    --resource-arns "$resource" \
    --query 'EvaluationResults[0].EvalDecision' \
    --output text)"
  assert_equals "$label" "$expected" "$actual"
}

stack="$(aws cloudformation describe-stacks --stack-name "$stack_name" --region "$region" --query 'Stacks[0]' --output json)"
assert_equals "stack status" UPDATE_COMPLETE "$(jq -r '.StackStatus' <<<"$stack")"
environment="$(jq -r '.Parameters[] | select(.ParameterKey == "Environment") | .ParameterValue' <<<"$stack")"

api_url="$(stack_output ApiUrl)"
users_table="$(stack_output UsersTableName)"
friendships_table="$(stack_output FriendshipsTableName)"
mood_events_table="$(stack_output MoodEventsTableName)"
devices_table="$(stack_output DevicesTableName)"
assets_bucket="$(stack_output AssetsBucketName)"
notification_queue_url="$(stack_output NotificationQueueUrl)"
notification_dlq_url="$(stack_output NotificationDeadLetterQueueUrl)"

resources="$(aws cloudformation list-stack-resources --stack-name "$stack_name" --region "$region" --query StackResourceSummaries --output json)"
functions="$(jq -r '.[] | select(.ResourceType == "AWS::Lambda::Function") | .PhysicalResourceId' <<<"$resources")"
function_count="$(wc -w <<<"$functions" | tr -d ' ')"
assert_equals "Lambda function count" 14 "$function_count"

while IFS= read -r function_name; do
  [[ -n "$function_name" ]] || continue
  configuration="$(aws lambda get-function-configuration --function-name "$function_name" --region "$region" --output json)"
  assert_equals "$function_name state" Active "$(jq -r '.State' <<<"$configuration")"
  assert_equals "$function_name update" Successful "$(jq -r '.LastUpdateStatus' <<<"$configuration")"
  assert_equals "$function_name runtime" provided.al2023 "$(jq -r '.Runtime' <<<"$configuration")"
  assert_equals "$function_name architecture" arm64 "$(jq -r '.Architectures[0]' <<<"$configuration")"
  assert_equals "$function_name tracing" Active "$(jq -r '.TracingConfig.Mode' <<<"$configuration")"
  assert_equals "$function_name log format" JSON "$(jq -r '.LoggingConfig.LogFormat' <<<"$configuration")"
  assert_equals "$function_name APP_ENV" "$environment" "$(jq -r '.Environment.Variables.APP_ENV' <<<"$configuration")"
  assert_equals "$function_name users table" "$users_table" "$(jq -r '.Environment.Variables.USERS_TABLE_NAME' <<<"$configuration")"
  assert_equals "$function_name friendships table" "$friendships_table" "$(jq -r '.Environment.Variables.FRIENDSHIPS_TABLE_NAME' <<<"$configuration")"
  assert_equals "$function_name mood table" "$mood_events_table" "$(jq -r '.Environment.Variables.MOOD_EVENTS_TABLE_NAME' <<<"$configuration")"
  assert_equals "$function_name devices table" "$devices_table" "$(jq -r '.Environment.Variables.DEVICES_TABLE_NAME' <<<"$configuration")"
  assert_equals "$function_name asset bucket" "$assets_bucket" "$(jq -r '.Environment.Variables.ASSETS_BUCKET_NAME' <<<"$configuration")"
  assert_equals "$function_name catalog key" catalog/v1.json "$(jq -r '.Environment.Variables.ASSET_CATALOG_KEY' <<<"$configuration")"
  assert_equals "$function_name notification queue" "$notification_queue_url" "$(jq -r '.Environment.Variables.NOTIFICATION_QUEUE_URL' <<<"$configuration")"

  log_group="$(jq -r '.LoggingConfig.LogGroup' <<<"$configuration")"
  log_group_count="$(aws logs describe-log-groups --region "$region" --log-group-name-prefix "$log_group" --query "length(logGroups[?logGroupName=='$log_group'])" --output text)"
  assert_equals "$function_name log group" 1 "$log_group_count"
done <<<"$functions"

api_log_retention="$(aws logs describe-log-groups \
  --region "$region" \
  --log-group-name-prefix "/aws/http-api/${stack_name}" \
  --query "logGroups[?logGroupName=='/aws/http-api/${stack_name}'].retentionInDays | [0]" \
  --output text)"
assert_equals "API access-log retention" 30 "$api_log_retention"

roles="$(jq -r '.[] | select(.ResourceType == "AWS::IAM::Role") | .PhysicalResourceId' <<<"$resources")"
allowed_managed_policies='[
  "arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess",
  "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
  "arn:aws:iam::aws:policy/service-role/AWSLambdaSQSQueueExecutionRole"
]'
while IFS= read -r role_name; do
  [[ -n "$role_name" ]] || continue
  attached="$(aws iam list-attached-role-policies --role-name "$role_name" --query 'AttachedPolicies[].PolicyArn' --output json)"
  if ! jq -e --argjson allowed "$allowed_managed_policies" 'all(.[]; . as $policy | $allowed | index($policy) != null)' <<<"$attached" >/dev/null; then
    echo "$role_name: unexpected managed policy" >&2
    jq . <<<"$attached" >&2
    exit 1
  fi

  policy_names="$(aws iam list-role-policies --role-name "$role_name" --query PolicyNames --output json)"
  while IFS= read -r policy_name; do
    [[ -n "$policy_name" ]] || continue
    policy="$(aws iam get-role-policy --role-name "$role_name" --policy-name "$policy_name" --output json)"
    if jq -e '[.PolicyDocument.Statement[] | .Action | if type == "array" then .[] else . end] | any(. == "*")' <<<"$policy" >/dev/null; then
      echo "$role_name/$policy_name: wildcard action is not allowed" >&2
      exit 1
    fi
    if jq -e '[.PolicyDocument.Statement[] | .Resource | if type == "array" then .[] else . end] | any(. == "*")' <<<"$policy" >/dev/null; then
      echo "$role_name/$policy_name: wildcard resource is not allowed" >&2
      exit 1
    fi
  done < <(jq -r '.[]' <<<"$policy_names")
  echo "$role_name IAM policy boundaries: verified"
done <<<"$roles"

publish_role="$(jq -r '.[] | select(.LogicalResourceId == "PublishMoodFunctionRole") | .PhysicalResourceId' <<<"$resources")"
catalog_role="$(jq -r '.[] | select(.LogicalResourceId == "GetAssetCatalogFunctionRole") | .PhysicalResourceId' <<<"$resources")"
search_role="$(jq -r '.[] | select(.LogicalResourceId == "SearchUsersFunctionRole") | .PhysicalResourceId' <<<"$resources")"
account_id="$(aws sts get-caller-identity --query Account --output text)"
friendship_index_arn="arn:aws:dynamodb:${region}:${account_id}:table/${friendships_table}/index/UserStatusIndex"
user_index_arn="arn:aws:dynamodb:${region}:${account_id}:table/${users_table}/index/UserSearchIndex"
asset_arn="arn:aws:s3:::${assets_bucket}/catalog/v1.json"

assert_decision "PublishMood friendships query" "$publish_role" dynamodb:Query "$friendship_index_arn" allowed
assert_decision "PublishMood unrelated S3 write" "$publish_role" s3:PutObject "$asset_arn" implicitDeny
assert_decision "GetAssetCatalog catalog read" "$catalog_role" s3:GetObject "$asset_arn" allowed
assert_decision "GetAssetCatalog catalog write" "$catalog_role" s3:PutObject "$asset_arn" implicitDeny
assert_decision "SearchUsers index query" "$search_role" dynamodb:Query "$user_index_arn" allowed
assert_decision "SearchUsers table write" "$search_role" dynamodb:PutItem "arn:aws:dynamodb:${region}:${account_id}:table/${users_table}" implicitDeny

queue_attributes="$(aws sqs get-queue-attributes --region "$region" --queue-url "$notification_queue_url" --attribute-names RedrivePolicy VisibilityTimeout KmsMasterKeyId --query Attributes --output json)"
assert_equals "notification visibility timeout" 120 "$(jq -r '.VisibilityTimeout' <<<"$queue_attributes")"
assert_equals "notification max receive count" 5 "$(jq -r '.RedrivePolicy | fromjson | .maxReceiveCount' <<<"$queue_attributes")"
assert_equals "notification queue encryption" alias/aws/sqs "$(jq -r '.KmsMasterKeyId' <<<"$queue_attributes")"
dlq_encryption="$(aws sqs get-queue-attributes --region "$region" --queue-url "$notification_dlq_url" --attribute-names KmsMasterKeyId --query Attributes.KmsMasterKeyId --output text)"
assert_equals "notification DLQ encryption" alias/aws/sqs "$dlq_encryption"

mapping_id="$(jq -r '.[] | select(.LogicalResourceId == "NotificationConsumerFunctionNotificationJobs") | .PhysicalResourceId' <<<"$resources")"
mapping="$(aws lambda get-event-source-mapping --uuid "$mapping_id" --region "$region" --output json)"
assert_equals "notification event mapping state" Enabled "$(jq -r '.State' <<<"$mapping")"
assert_equals "partial batch failure reporting" ReportBatchItemFailures "$(jq -r '.FunctionResponseTypes[0]' <<<"$mapping")"

alarms="$(aws cloudwatch describe-alarms --region "$region" --alarm-name-prefix "${stack_name}-" --query MetricAlarms --output json)"
assert_equals "alarm count" 4 "$(jq 'length' <<<"$alarms")"
if ! jq -e 'all(.[]; .ActionsEnabled == true and (.AlarmActions | length) > 0 and .StateValue != "INSUFFICIENT_DATA")' <<<"$alarms" >/dev/null; then
  echo "alarms are not action-enabled and healthy" >&2
  jq '[.[] | {AlarmName,ActionsEnabled,AlarmActions,StateValue}]' <<<"$alarms" >&2
  exit 1
fi
echo "CloudWatch alarms and actions: verified"

if [[ ! "$api_url" =~ ^https://[a-z0-9]+\.execute-api\.${region}\.amazonaws\.com/${environment}$ ]]; then
  echo "API URL stack output has an unexpected format: $api_url" >&2
  exit 1
fi
echo "API URL stack output: verified"
echo "Deployment configuration, IAM boundaries, queues, logs, and alarms verified for $stack_name ($region)."
