#!/usr/bin/env bash
set -euo pipefail

stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"
suffix="$(date +%s)-$$"
correlation_id="e2e-$suffix"
email_a="dollhouse-api-a-${suffix}@example.com"
email_b="dollhouse-api-b-${suffix}@example.com"
username_a="api-a-${suffix}"
username_b="api-b-${suffix}"
renamed_username_a="${username_a}-renamed"
concurrent_username="api-race-${suffix}"
password="DollhouseTest!42-${suffix}Aa"
work_dir="$(mktemp -d)"
response_file="$work_dir/response.json"
created_a=false
created_b=false
sub_a=""
sub_b=""
user_pool_id=""
users_table=""
friendships_table=""
mood_events_table=""
devices_table=""

stack_output() {
  aws cloudformation describe-stacks \
    --stack-name "$stack_name" \
    --region "$region" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue | [0]" \
    --output text
}

delete_partition() {
  local table_name="$1"
  local partition_name="$2"
  local sort_name="$3"
  local partition_value="$4"
  local keys

  [[ -n "$partition_value" ]] || return 0
  keys="$(aws dynamodb query \
    --region "$region" \
    --table-name "$table_name" \
    --key-condition-expression "#pk = :pk" \
    --expression-attribute-names "{\"#pk\":\"$partition_name\"}" \
    --expression-attribute-values "{\":pk\":{\"S\":\"$partition_value\"}}" \
    --projection-expression "#pk, $sort_name" \
    --query Items \
    --output json)"

  while IFS= read -r key; do
    [[ -n "$key" ]] || continue
    aws dynamodb delete-item \
      --region "$region" \
      --table-name "$table_name" \
      --key "$key" \
      --output json >/dev/null
  done < <(jq -c '.[]' <<<"$keys")
}

cleanup() {
  local exit_code=$?
  trap - EXIT
  set +e

  if [[ -n "$sub_a" || -n "$sub_b" ]]; then
    delete_partition "$friendships_table" userId relatedUserId "$sub_a"
    delete_partition "$friendships_table" userId relatedUserId "$sub_b"
    delete_partition "$mood_events_table" userId occurredAt "$sub_a"
    delete_partition "$mood_events_table" userId occurredAt "$sub_b"
    delete_partition "$devices_table" userId deviceId "$sub_a"
    delete_partition "$devices_table" userId deviceId "$sub_b"
    [[ -z "$sub_a" ]] || aws dynamodb delete-item --region "$region" --table-name "$users_table" --key "{\"userId\":{\"S\":\"$sub_a\"}}" --output json >/dev/null
    [[ -z "$sub_b" ]] || aws dynamodb delete-item --region "$region" --table-name "$users_table" --key "{\"userId\":{\"S\":\"$sub_b\"}}" --output json >/dev/null
    for username in "$username_a" "$username_b" "$renamed_username_a" "$concurrent_username"; do
      aws dynamodb delete-item --region "$region" --table-name "$users_table" --key "$(jq -cn --arg id "USERNAME#$username" '{userId:{S:$id}}')" --output json >/dev/null
    done
  fi

  [[ "$created_a" != true ]] || aws cognito-idp admin-delete-user --region "$region" --user-pool-id "$user_pool_id" --username "$email_a"
  [[ "$created_b" != true ]] || aws cognito-idp admin-delete-user --region "$region" --user-pool-id "$user_pool_id" --username "$email_b"
  [[ -z "$work_dir" || ! -d "$work_dir" ]] || rm -r -- "$work_dir"

  if (( exit_code == 0 )); then
    echo "Temporary Cognito users and DynamoDB records removed."
  else
    echo "API verification failed; attempted cleanup of all temporary resources." >&2
  fi
  exit "$exit_code"
}
trap cleanup EXIT

assert_status() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  if [[ "$actual" != "$expected" ]]; then
    echo "$label: expected HTTP $expected, received $actual" >&2
    jq . "$response_file" >&2 2>/dev/null || true
    exit 1
  fi
  echo "$label: HTTP $actual"
}

assert_json() {
  local label="$1"
  local expression="$2"
  if ! jq -e "$expression" "$response_file" >/dev/null; then
    echo "$label: response assertion failed: $expression" >&2
    jq . "$response_file" >&2 2>/dev/null || true
    exit 1
  fi
  echo "$label: response verified"
}

request() {
  local method="$1"
  local path="$2"
  local token="${3:-}"
  local body="${4:-}"
  local args=(
    --silent --show-error
    --output "$response_file"
    --write-out '%{http_code}'
    --request "$method"
    --header 'Content-Type: application/json'
    --header "X-Correlation-Id: $correlation_id"
  )
  [[ -z "$token" ]] || args+=(--header "Authorization: Bearer $token")
  [[ -z "$body" ]] || args+=(--data "$body")
  curl "${args[@]}" "${api_url}${path}"
}

create_user() {
  local email="$1"
  aws cognito-idp admin-create-user \
    --region "$region" \
    --user-pool-id "$user_pool_id" \
    --username "$email" \
    --user-attributes "Name=email,Value=$email" Name=email_verified,Value=true \
    --message-action SUPPRESS \
    --output json >/dev/null
  aws cognito-idp admin-set-user-password \
    --region "$region" \
    --user-pool-id "$user_pool_id" \
    --username "$email" \
    --password "$password" \
    --permanent
  aws cognito-idp admin-add-user-to-group \
    --region "$region" \
    --user-pool-id "$user_pool_id" \
    --username "$email" \
    --group-name User
}

user_sub() {
  aws cognito-idp admin-get-user \
    --region "$region" \
    --user-pool-id "$user_pool_id" \
    --username "$1" \
    --query "UserAttributes[?Name=='sub'].Value | [0]" \
    --output text
}

id_token() {
  aws cognito-idp initiate-auth \
    --region "$region" \
    --client-id "$user_pool_client_id" \
    --auth-flow USER_PASSWORD_AUTH \
    --auth-parameters "USERNAME=$1,PASSWORD=$password" \
    --query AuthenticationResult.IdToken \
    --output text
}

api_url="$(stack_output ApiUrl)"
user_pool_id="$(stack_output UserPoolId)"
user_pool_client_id="$(stack_output UserPoolClientId)"
users_table="$(stack_output UsersTableName)"
friendships_table="$(stack_output FriendshipsTableName)"
mood_events_table="$(stack_output MoodEventsTableName)"
devices_table="$(stack_output DevicesTableName)"

assert_status "missing-token authorization" 401 "$(request GET /profile)"
assert_status "invalid-token authorization" 401 "$(request GET /profile invalid-token)"

create_user "$email_a"
created_a=true
create_user "$email_b"
created_b=true
sub_a="$(user_sub "$email_a")"
sub_b="$(user_sub "$email_b")"
token_a="$(id_token "$email_a")"
token_b="$(id_token "$email_b")"

assert_status "create/read profile A" 200 "$(request GET /profile "$token_a")"
assert_json "profile A requires safe onboarding" ".data.profile.userId == \"$sub_a\" and .data.profile.username == \"\" and .data.profile.displayName == \"\" and .data.profile.onboardingComplete == false and ((tostring | contains(\"@example.com\")) | not)"
assert_status "create/read profile B" 200 "$(request GET /profile "$token_b")"
assert_json "profile B requires safe onboarding" ".data.profile.userId == \"$sub_b\" and .data.profile.onboardingComplete == false and ((tostring | contains(\"@example.com\")) | not)"

assert_status "available username" 200 "$(request GET "/users/username-availability?username=$username_a" "$token_a")"
assert_json "available username response" ".data.username == \"$username_a\" and .data.available == true"
assert_status "invalid username availability" 400 "$(request GET '/users/username-availability?username=%20%20%20' "$token_a")"
assert_json "invalid availability error" '.error.code == "validation_failed"'

profile_a="$(jq -cn --arg username "$username_a" '{username:$username,displayName:"API Test A",bio:"temporary verification user"}')"
profile_b="$(jq -cn --arg username "$username_b" '{username:$username,displayName:"API Test B",bio:null}')"
assert_status "update profile A" 200 "$(request PUT /profile "$token_a" "$profile_a")"
assert_json "updated profile A" ".data.profile.username == \"$username_a\" and .data.profile.bio == \"temporary verification user\" and .data.profile.onboardingComplete == true"
assert_status "case-insensitive username collision" 409 "$(request PUT /profile "$token_b" "$(jq -cn --arg username "${username_a^^}" '{username:$username}')")"
assert_json "username collision error" '.error.code == "conflict"'
assert_status "claimed username unavailable" 200 "$(request GET "/users/username-availability?username=${username_a^^}" "$token_b")"
assert_json "claimed availability response" ".data.available == false"
assert_status "update profile B" 200 "$(request PUT /profile "$token_b" "$profile_b")"
assert_json "updated profile B" ".data.profile.username == \"$username_b\" and .data.profile.bio == null"

assert_status "rename profile A" 200 "$(request PUT /profile "$token_a" "$(jq -cn --arg username "$renamed_username_a" '{username:$username}')")"
assert_status "released username available" 200 "$(request GET "/users/username-availability?username=$username_a" "$token_b")"
assert_json "released availability response" '.data.available == true'
assert_status "own renamed username available" 200 "$(request GET "/users/username-availability?username=$renamed_username_a" "$token_a")"
assert_json "own availability response" '.data.available == true'
assert_status "restore profile A username" 200 "$(request PUT /profile "$token_a" "$(jq -cn --arg username "$username_a" '{username:$username}')")"

race_body="$(jq -cn --arg username "$concurrent_username" '{username:$username}')"
race_status_a="$work_dir/race-a.status"
race_status_b="$work_dir/race-b.status"
curl --silent --show-error --output "$work_dir/race-a.json" --write-out '%{http_code}\n' --request PUT --header 'Content-Type: application/json' --header "Authorization: Bearer $token_a" --data "$race_body" "${api_url}/profile" >"$race_status_a" &
race_pid_a=$!
curl --silent --show-error --output "$work_dir/race-b.json" --write-out '%{http_code}\n' --request PUT --header 'Content-Type: application/json' --header "Authorization: Bearer $token_b" --data "$race_body" "${api_url}/profile" >"$race_status_b" &
race_pid_b=$!
wait "$race_pid_a"
wait "$race_pid_b"
race_statuses="$(sort "$race_status_a" "$race_status_b" | tr '\n' ' ')"
[[ "$race_statuses" == "200 409 " ]] || { echo "concurrent claim: expected one 200 and one 409, received $race_statuses" >&2; exit 1; }
echo "concurrent username claim: one winner and one conflict"
assert_status "restore profile A after race" 200 "$(request PUT /profile "$token_a" "$(jq -cn --arg username "$username_a" '{username:$username}')")"
assert_status "restore profile B after race" 200 "$(request PUT /profile "$token_b" "$(jq -cn --arg username "$username_b" '{username:$username}')")"

assert_status "asset catalog" 200 "$(request GET /assets/catalog "$token_a")"
assert_json "asset catalog contract" '.data.catalogVersion != null and (.data.assets | length) > 0'
body_asset_id="$(jq -er '[.data.assets[] | select(.category == "body")][0].assetId' "$response_file")"
hair_asset_id="$(jq -er '[.data.assets[] | select(.category == "hair")][0].assetId' "$response_file")"
eyes_asset_id="$(jq -er '[.data.assets[] | select(.category == "eyes")][0].assetId' "$response_file")"
nose_asset_id="$(jq -er '[.data.assets[] | select(.category == "nose")][0].assetId' "$response_file")"
mouth_asset_id="$(jq -er '[.data.assets[] | select(.category == "mouth")][0].assetId' "$response_file")"
clothing_asset_ids="$(jq -c '[.data.assets[] | select(.category == "clothing")][0:1] | map(.assetId)' "$response_file")"
doll_payload="$(jq -cn \
  --arg body "$body_asset_id" \
  --arg hair "$hair_asset_id" \
  --arg eyes "$eyes_asset_id" \
  --arg nose "$nose_asset_id" \
  --arg mouth "$mouth_asset_id" \
  --argjson clothing "$clothing_asset_ids" \
  '{bodyAssetId:$body,hairAssetId:$hair,eyesAssetId:$eyes,noseAssetId:$nose,mouthAssetId:$mouth,clothingAssetIds:$clothing}')"

assert_status "missing doll" 404 "$(request GET /doll "$token_a")"
assert_status "save doll A" 200 "$(request PUT /doll "$token_a" "$doll_payload")"
assert_json "saved doll A" ".data.configuration.bodyAssetId == \"$body_asset_id\""
assert_status "save doll B" 200 "$(request PUT /doll "$token_b" "$doll_payload")"
assert_status "read doll A" 200 "$(request GET /doll "$token_a")"
assert_json "read doll A" ".data.configuration.bodyAssetId == \"$body_asset_id\""

found_b=false
for _ in {1..15}; do
  assert_status "search users" 200 "$(request GET "/users/search?q=$username_b" "$token_a")"
  if jq -e --arg id "$sub_b" '.data.items[] | select(.userId == $id)' "$response_file" >/dev/null; then
    found_b=true
    break
  fi
  sleep 2
done
[[ "$found_b" == true ]] || { echo "search users: test user B did not appear after GSI propagation wait" >&2; exit 1; }
echo "search users: user B found"

assert_status "reject self friend request" 400 "$(request POST /friend-requests "$token_a" "{\"userId\":\"$sub_a\"}")"
assert_json "self friend error" '.error.code == "validation_failed"'
assert_status "send friend request A to B" 201 "$(request POST /friend-requests "$token_a" "{\"userId\":\"$sub_b\"}")"
request_id="$(jq -er '.data.friendRequest.requestId' "$response_file")"
assert_status "list outgoing friend requests" 200 "$(request GET /friend-requests "$token_a")"
assert_json "outgoing friend request" ".data.outgoing[] | select(.requestId == \"$request_id\")"
assert_status "list incoming friend requests" 200 "$(request GET /friend-requests "$token_b")"
assert_json "incoming friend request" ".data.incoming[] | select(.requestId == \"$request_id\")"
assert_status "reject outgoing-user acceptance" 403 "$(request POST "/friend-requests/$request_id/accept" "$token_a")"
assert_json "accept authorization error" '.error.code == "forbidden"'
assert_status "accept friend request as recipient" 200 "$(request POST "/friend-requests/$request_id/accept" "$token_b")"
assert_json "accepted friendship" ".data.friendship.friend.userId == \"$sub_a\" and .data.friendship.status == \"ACCEPTED\""

mood_payload='{"status":"API verification mood","stress":4,"fatigue":null,"discomfort":0}'
assert_status "publish mood" 201 "$(request POST /moods "$token_a" "$mood_payload")"
mood_event_id="$(jq -er '.data.eventId' "$response_file")"
assert_json "published mood contract" '.data.status.status == "API verification mood" and .data.status.stress == 4 and .data.status.fatigue == null and .data.status.discomfort == 0'
assert_status "get own mood entries" 200 "$(request GET /moods "$token_a")"
assert_json "own mood history contract" ".data.items[] | select(.eventId == \"$mood_event_id\" and .status == \"API verification mood\" and .stress == 4 and .fatigue == null and .discomfort == 0)"

found_status=false
for _ in {1..15}; do
  assert_status "get friend statuses" 200 "$(request GET /friend-statuses "$token_b")"
  if jq -e --arg id "$sub_a" '.data.items[] | select(.friend.userId == $id and .status.status == "API verification mood")' "$response_file" >/dev/null; then
    found_status=true
    break
  fi
  sleep 2
done
[[ "$found_status" == true ]] || { echo "friend statuses: accepted friend mood did not appear after GSI propagation wait" >&2; exit 1; }
echo "friend statuses: accepted friend mood found"

assert_status "remove accepted friend" 204 "$(request DELETE "/friends/$sub_b" "$token_a")"
assert_status "send friend request B to A" 201 "$(request POST /friend-requests "$token_b" "{\"userId\":\"$sub_a\"}")"
decline_request_id="$(jq -er '.data.friendRequest.requestId' "$response_file")"
assert_status "reject outgoing-user decline" 403 "$(request POST "/friend-requests/$decline_request_id/decline" "$token_b")"
assert_json "decline authorization error" '.error.code == "forbidden"'
assert_status "decline friend request as recipient" 204 "$(request POST "/friend-requests/$decline_request_id/decline" "$token_a")"

echo "All documented HTTP routes and object-level authorization checks passed against $stack_name ($region)."
echo "Notification event queued for retry/DLQ verification: $mood_event_id"
echo "Monitoring correlation ID: $correlation_id"
