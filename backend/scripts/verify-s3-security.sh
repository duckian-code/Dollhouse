#!/usr/bin/env bash

set -euo pipefail

stack_name="${1:-dollhouse-dev}"
region="${AWS_REGION:-us-east-2}"

bucket_name="$({
  aws cloudformation describe-stacks \
    --stack-name "$stack_name" \
    --region "$region" \
    --query "Stacks[0].Outputs[?OutputKey=='AssetsBucketName'].OutputValue | [0]" \
    --output text
})"

if [[ -z "$bucket_name" || "$bucket_name" == "None" ]]; then
  echo "AssetsBucketName output was not found for stack $stack_name" >&2
  exit 1
fi

assert_equals() {
  local label="$1"
  local expected="$2"
  local actual="$3"

  if [[ "$actual" != "$expected" ]]; then
    echo "$label: expected '$expected', got '$actual'" >&2
    exit 1
  fi

  echo "$label: verified"
}

public_access="$({
  aws s3api get-public-access-block \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'PublicAccessBlockConfiguration.[BlockPublicAcls,IgnorePublicAcls,BlockPublicPolicy,RestrictPublicBuckets]' \
    --output text
})"
assert_equals "S3 public access block" $'True\tTrue\tTrue\tTrue' "$public_access"

encryption="$({
  aws s3api get-bucket-encryption \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm' \
    --output text
})"
assert_equals "S3 default encryption" "AES256" "$encryption"

versioning="$({
  aws s3api get-bucket-versioning \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'Status' \
    --output text
})"
assert_equals "S3 versioning" "Enabled" "$versioning"

ownership="$({
  aws s3api get-bucket-ownership-controls \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'OwnershipControls.Rules[0].ObjectOwnership' \
    --output text
})"
assert_equals "S3 object ownership" "BucketOwnerEnforced" "$ownership"

lifecycle="$({
  aws s3api get-bucket-lifecycle-configuration \
    --bucket "$bucket_name" \
    --region "$region" \
    --query "Rules[?ID=='AbortIncompleteMultipartUploads'].[Status,AbortIncompleteMultipartUpload.DaysAfterInitiation] | [0]" \
    --output text
})"
assert_equals "S3 incomplete multipart upload cleanup" $'Enabled\t7' "$lifecycle"

policy_public="$({
  aws s3api get-bucket-policy-status \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'PolicyStatus.IsPublic' \
    --output text
})"
assert_equals "S3 bucket policy public status" "False" "$policy_public"

bucket_policy="$({
  aws s3api get-bucket-policy \
    --bucket "$bucket_name" \
    --region "$region" \
    --query 'Policy' \
    --output text
})"

if [[ "$bucket_policy" != *'aws:SecureTransport'* || "$bucket_policy" != *'false'* ]]; then
  echo "S3 bucket policy does not deny insecure transport" >&2
  exit 1
fi

echo "S3 TLS-only bucket policy: verified"
echo "Secure asset bucket configuration verified for $bucket_name"
