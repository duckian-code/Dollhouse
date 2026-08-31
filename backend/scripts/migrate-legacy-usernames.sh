#!/usr/bin/env bash
set -euo pipefail

stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"
timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

users_table="$(aws cloudformation describe-stacks \
  --stack-name "$stack_name" \
  --region "$region" \
  --query "Stacks[0].Outputs[?OutputKey=='UsersTableName'].OutputValue | [0]" \
  --output text)"

profiles="$(aws dynamodb scan \
  --region "$region" \
  --table-name "$users_table" \
  --filter-expression 'attribute_exists(cognitoSub)' \
  --projection-expression 'userId, onboardingComplete' \
  --output json)"

migrated=0
cleaned=0
while IFS= read -r profile; do
  user_id="$(jq -r '.userId.S' <<<"$profile")"
  if jq -e 'has("onboardingComplete")' <<<"$profile" >/dev/null; then
    aws dynamodb update-item \
      --region "$region" \
      --table-name "$users_table" \
      --key "$(jq -cn --arg id "$user_id" '{userId:{S:$id}}')" \
      --update-expression 'REMOVE #email, #normalizedEmail' \
      --condition-expression 'attribute_exists(cognitoSub)' \
      --expression-attribute-names '{"#email":"email","#normalizedEmail":"normalizedEmail"}' \
      --output json >/dev/null
    cleaned=$((cleaned + 1))
    continue
  fi

  aws dynamodb update-item \
    --region "$region" \
    --table-name "$users_table" \
    --key "$(jq -cn --arg id "$user_id" '{userId:{S:$id}}')" \
    --update-expression 'SET #onboardingComplete = :incomplete, #updatedAt = :updatedAt REMOVE #username, #normalizedUsername, #searchPartition, #displayName, #email, #normalizedEmail' \
    --condition-expression 'attribute_exists(cognitoSub) AND attribute_not_exists(#onboardingComplete)' \
    --expression-attribute-names '{"#onboardingComplete":"onboardingComplete","#updatedAt":"updatedAt","#username":"username","#normalizedUsername":"normalizedUsername","#searchPartition":"searchPartition","#displayName":"displayName","#email":"email","#normalizedEmail":"normalizedEmail"}' \
    --expression-attribute-values "$(jq -cn --arg now "$timestamp" '{":incomplete":{BOOL:false},":updatedAt":{S:$now}}')" \
    --output json >/dev/null
  migrated=$((migrated + 1))
done < <(jq -c '.Items[]' <<<"$profiles")

echo "Migrated $migrated legacy profile(s) to username onboarding and removed private email fields from $cleaned current profile(s) in $users_table."
