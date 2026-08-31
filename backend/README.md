# Dollhouse backend

The backend is a Go module containing one deployable command per AWS Lambda. API
handlers are intentionally thin entry points; reusable behavior belongs in
`internal` or `pkg` packages.

## Lambda entry points

| Domain | Lambda | Source |
| --- | --- | --- |
| Profile | Get current profile | `cmd/get-current-profile` |
| Profile | Update profile | `cmd/update-profile` |
| Doll | Get doll configuration | `cmd/get-doll-configuration` |
| Doll | Update doll configuration | `cmd/update-doll-configuration` |
| Assets | Get approved asset catalog | `cmd/get-asset-catalog` |
| Friendship | Search users | `cmd/search-users` |
| Friendship | Send friend request | `cmd/send-friend-request` |
| Friendship | List friend requests | `cmd/list-friend-requests` |
| Friendship | Accept friend request | `cmd/accept-friend-request` |
| Friendship | Decline friend request | `cmd/decline-friend-request` |
| Friendship | Remove friend | `cmd/remove-friend` |
| Mood | Publish mood | `cmd/publish-mood` |
| Status | Get friend statuses | `cmd/get-friend-statuses` |
| Notifications | Consume queued notifications | `cmd/notification-consumer` |

## Local development

Go 1.24 or newer is required. From this directory:

```sh
go mod download
make test
make build
make validate-infra
```

`make build` cross-compiles every entry point for the AWS Lambda `provided.al2023`
ARM64 runtime and writes artifacts under `.build/<lambda>/bootstrap`.

Copy `.env.example` to `.env` when running handlers against local AWS-compatible
services. Environment variables are loaded by `internal/config`; deployed values
should be supplied by infrastructure rather than committed to source control.

## Infrastructure

AWS resources are defined in [`infrastructure/template.yaml`](infrastructure/template.yaml)
using AWS SAM. The checked-in [`samconfig.toml`](samconfig.toml) provides repeatable
development defaults for `us-east-2`.

Validate and build the complete application from this directory:

```sh
sam validate --lint
sam build --parallel
```

Deploy the development stack after reviewing the generated change set:

```sh
sam deploy
```

To enable email notifications, override the empty default on deployment:

```sh
sam deploy --parameter-overrides AlertEmail=team@example.com
```

An SNS confirmation email must be accepted before alarm notifications are
delivered. Use a distinct `Environment` parameter for additional stacks.

### Mood notification jobs

After `POST /moods` saves the user's current status and history event, the
publisher resolves all accepted friend IDs and sends one versioned job to the
notification FIFO queue:

```json
{
  "schemaVersion": 1,
  "eventId": "opaque-backend-event-id",
  "senderUserId": "cognito-sub",
  "recipientUserIds": ["friend-cognito-sub"],
  "correlationId": "api-request-or-client-correlation-id",
  "createdAt": "2026-08-18T16:00:00Z"
}
```

Jobs contain identifiers only. Free-form mood text, disclosed slider values,
and push tokens are deliberately excluded. The event ID is the FIFO
deduplication ID, and jobs from one sender share a message group so their order
is preserved. Failed deliveries are retried up to five receives before SQS
moves them to the encrypted dead-letter queue.

### Cognito authorization verification

The API uses the Cognito mobile client's audience and the user pool issuer for
JWT authorization. Lambda handlers derive the caller's object identity only from
API Gateway's verified `sub` claim; request bodies and path parameters cannot
override it. Cognito's `cognito:groups` claim supplies roles, and administrator
routes must use the shared `authorization.RequireAdmin` guard. The current asset
catalog API is read-only, and its Lambda IAM policy has no S3 write permission.

After deployment, verify an issued token and rejection paths without printing
token values:

```sh
VALID_ID_TOKEN="$VALID_ID_TOKEN" EXPIRED_ID_TOKEN="$EXPIRED_ID_TOKEN" \
  bash scripts/verify-cognito-auth.sh \
  "https://api-id.execute-api.us-east-2.amazonaws.com/dev"
```

The script treats any response other than `401` or `403` as proof that API
Gateway accepted the valid token; feature handlers may still return `501` until
their implementation tickets are complete.

### DynamoDB access patterns

Table keys, item shapes, consistency rules, and every MVP query are documented in
[`infrastructure/dynamodb-access-patterns.md`](infrastructure/dynamodb-access-patterns.md).
Seed and verify the non-production fixtures against the deployed development stack:

```sh
bash scripts/seed-dynamodb-test-data.sh
bash scripts/verify-dynamodb-access-patterns.sh
```

Fixture primary keys begin with `TEST#` and the seed operation is idempotent.

### Secure asset storage

The asset bucket is private, encrypted with S3-managed keys, versioned, and uses
bucket-owner-enforced object ownership. Its bucket policy denies every request
that does not use TLS. Incomplete multipart uploads are removed after seven days
to avoid paying for abandoned upload parts.

Approved launch assets are described by `catalog/v1.json` in the bucket. The
authenticated `GET /assets/catalog` route reads that private manifest and
returns 15-minute, version-specific presigned URLs. Clients use the opaque
`assetId` values for saved doll configurations and should refresh the catalog
when its URLs expire. Do not use public-read ACLs or a public bucket policy. The
bucket's CORS configuration only permits `GET` and `HEAD` and does not grant
access by itself.

After deploying the stack, verify the security controls without reading any
objects:

```sh
make verify-s3-security
```

Pass a different stack name as the script's first argument when needed:

```sh
bash scripts/verify-s3-security.sh dollhouse-staging
```
