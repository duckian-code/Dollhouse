#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || -z "$1" ]]; then
  echo "usage: $0 <event-id>" >&2
  exit 2
fi

event_id="$1"
stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"
timeout_seconds="${REDRIVE_TIMEOUT_SECONDS:-900}"
poll_seconds="${REDRIVE_POLL_SECONDS:-15}"
started_at="$(date +%s)"

stack_output() {
  aws cloudformation describe-stacks \
    --stack-name "$stack_name" \
    --region "$region" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue | [0]" \
    --output text
}

source_queue_url="$(stack_output NotificationQueueUrl)"
dlq_url="$(stack_output NotificationDeadLetterQueueUrl)"
source_attributes="$(aws sqs get-queue-attributes \
  --region "$region" \
  --queue-url "$source_queue_url" \
  --attribute-names RedrivePolicy VisibilityTimeout \
  --query Attributes \
  --output json)"

if [[ "$(jq -r '.RedrivePolicy | fromjson | .maxReceiveCount' <<<"$source_attributes")" != "5" ]]; then
  echo "notification queue maxReceiveCount is not 5" >&2
  exit 1
fi
if [[ "$(jq -r '.VisibilityTimeout' <<<"$source_attributes")" != "120" ]]; then
  echo "notification queue visibility timeout is not 120 seconds" >&2
  exit 1
fi
echo "notification queue redrive policy: verified"

while (( $(date +%s) - started_at < timeout_seconds )); do
  messages="$(aws sqs receive-message \
    --region "$region" \
    --queue-url "$dlq_url" \
    --max-number-of-messages 10 \
    --visibility-timeout 5 \
    --wait-time-seconds 10 \
    --attribute-names ApproximateReceiveCount \
    --message-attribute-names All \
    --query 'Messages || `[]`' \
    --output json)"

  while IFS= read -r message; do
    [[ -n "$message" ]] || continue
    receipt_handle="$(jq -r '.ReceiptHandle' <<<"$message")"
    body="$(jq -r '.Body' <<<"$message")"
    if jq -e --arg event_id "$event_id" '.eventId == $event_id' <<<"$body" >/dev/null 2>&1; then
      receive_count="$(jq -r '.Attributes.ApproximateReceiveCount // "unknown"' <<<"$message")"
      aws sqs delete-message \
        --region "$region" \
        --queue-url "$dlq_url" \
        --receipt-handle "$receipt_handle"
      echo "notification event $event_id reached the DLQ (DLQ receive count: $receive_count)"
      echo "verified test message removed from the DLQ"
      exit 0
    fi

    # Restore unrelated messages immediately rather than holding them invisible.
    aws sqs change-message-visibility \
      --region "$region" \
      --queue-url "$dlq_url" \
      --receipt-handle "$receipt_handle" \
      --visibility-timeout 0
  done < <(jq -c '.[]' <<<"$messages")

  sleep "$poll_seconds"
done

echo "notification event $event_id did not reach the DLQ within $timeout_seconds seconds" >&2
exit 1
