#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || -z "$1" ]]; then
  echo "usage: $0 <correlation-id>" >&2
  exit 2
fi

correlation_id="$1"
stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"
timeout_seconds="${TRACE_TIMEOUT_SECONDS:-180}"
poll_seconds="${TRACE_POLL_SECONDS:-10}"
started_at="$(date +%s)"
start_time_ms="$(( (started_at - 900) * 1000 ))"

resources="$(aws cloudformation list-stack-resources \
  --stack-name "$stack_name" \
  --region "$region" \
  --query StackResourceSummaries \
  --output json)"

function_name() {
  jq -er --arg logical_id "$1" '.[] | select(.LogicalResourceId == $logical_id) | .PhysicalResourceId' <<<"$resources"
}

publish_log_group="/aws/lambda/$(function_name PublishMoodFunction)"
consumer_log_group="/aws/lambda/$(function_name NotificationConsumerFunction)"

matching_events() {
  aws logs filter-log-events \
    --region "$region" \
    --log-group-name "$1" \
    --start-time "$start_time_ms" \
    --filter-pattern "\"$correlation_id\"" \
    --query 'events[].message' \
    --output json
}

while (( $(date +%s) - started_at < timeout_seconds )); do
  publish_events="$(matching_events "$publish_log_group")"
  consumer_events="$(matching_events "$consumer_log_group")"
  if [[ "$(jq 'length' <<<"$publish_events")" -gt 0 && "$(jq 'length' <<<"$consumer_events")" -gt 0 ]]; then
    echo "correlation ID found in publish-mood logs: $(jq 'length' <<<"$publish_events") event(s)"
    echo "correlation ID found in notification-consumer logs: $(jq 'length' <<<"$consumer_events") event(s)"
    echo "request trace verified across API publication and asynchronous notification processing"
    exit 0
  fi
  sleep "$poll_seconds"
done

echo "correlation ID was not found in both producer and consumer logs within $timeout_seconds seconds" >&2
exit 1
