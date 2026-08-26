#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: VALID_ID_TOKEN=<token> EXPIRED_ID_TOKEN=<token> $0 <api-url>" >&2
  exit 2
fi

: "${VALID_ID_TOKEN:?VALID_ID_TOKEN is required}"
: "${EXPIRED_ID_TOKEN:?EXPIRED_ID_TOKEN is required}"

api_url="${1%/}/profile"
valid_token="$VALID_ID_TOKEN"
expired_token="$EXPIRED_ID_TOKEN"

status() {
  curl --silent --show-error --output /dev/null --write-out '%{http_code}' "$@" "$api_url"
}

assert_rejected() {
  local label="$1"
  local actual="$2"
  if [[ "$actual" != "401" && "$actual" != "403" ]]; then
    echo "$label: expected 401 or 403, received $actual" >&2
    exit 1
  fi
  echo "$label: rejected ($actual)"
}

missing_status="$(status)"
invalid_status="$(status --header 'Authorization: Bearer invalid-token')"
valid_status="$(status --header "Authorization: Bearer $valid_token")"
expired_status="$(status --header "Authorization: Bearer $expired_token")"

assert_rejected "missing token" "$missing_status"
assert_rejected "invalid token" "$invalid_status"

if [[ "$valid_status" == "401" || "$valid_status" == "403" ]]; then
  echo "valid token: unexpectedly rejected ($valid_status)" >&2
  exit 1
fi
echo "valid token: accepted by authorizer ($valid_status from handler)"

assert_rejected "expired token" "$expired_status"
