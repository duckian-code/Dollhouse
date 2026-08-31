#!/usr/bin/env bash
set -euo pipefail

stack_name="${STACK_NAME:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"
alarm_name="${ALARM_NAME:-${stack_name}-api-latency}"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

topic_arn="$(aws cloudformation describe-stacks \
  --stack-name "$stack_name" \
  --region "$region" \
  --query "Stacks[0].Outputs[?OutputKey=='AlertTopicArn'].OutputValue | [0]" \
  --output text)"

confirmed_subscriptions="$(aws sns list-subscriptions-by-topic \
  --topic-arn "$topic_arn" \
  --region "$region" \
  --query 'Subscriptions[?Protocol==`email` && SubscriptionArn!=`PendingConfirmation`] | length(@)' \
  --output text)"
if [[ "$confirmed_subscriptions" -lt 1 ]]; then
  echo "no confirmed SNS email subscription exists for $topic_arn" >&2
  exit 1
fi

aws cloudwatch set-alarm-state \
  --alarm-name "$alarm_name" \
  --state-value OK \
  --state-reason "Preparing Dollhouse monitoring test" \
  --region "$region"

aws cloudwatch set-alarm-state \
  --alarm-name "$alarm_name" \
  --state-value ALARM \
  --state-reason "Dollhouse monitoring test alert" \
  --region "$region"
echo "test ALARM state sent through $alarm_name to $confirmed_subscriptions confirmed email subscription(s)"

action_succeeded=false
for _ in {1..12}; do
  history="$(aws cloudwatch describe-alarm-history \
    --alarm-name "$alarm_name" \
    --history-item-type Action \
    --start-date "$started_at" \
    --region "$region" \
    --query 'AlarmHistoryItems[0]' \
    --output json)"
  summary="$(jq -r '.HistorySummary // ""' <<<"$history")"
  if [[ "$summary" == Successfully* ]]; then
    action_succeeded=true
    break
  fi
  if [[ "$summary" == Failed* ]]; then
    echo "$summary" >&2
    jq -r '.HistoryData' <<<"$history" >&2
    exit 1
  fi
  sleep 5
done

aws cloudwatch set-alarm-state \
  --alarm-name "$alarm_name" \
  --state-value OK \
  --state-reason "Dollhouse monitoring test completed" \
  --region "$region"
if [[ "$action_succeeded" != true ]]; then
  echo "CloudWatch did not report a successful SNS action within 60 seconds" >&2
  exit 1
fi
echo "CloudWatch successfully invoked SNS and the alarm returned to OK"
echo "confirm receipt of the SNS email before closing the ticket"
