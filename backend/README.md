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

### Cognito authorization verification

The API uses the Cognito mobile client's audience and the user pool issuer for
JWT authorization. After deployment, verify an issued token and rejection paths
without printing token values:

```sh
VALID_ID_TOKEN="$VALID_ID_TOKEN" EXPIRED_ID_TOKEN="$EXPIRED_ID_TOKEN" \
  bash scripts/verify-cognito-auth.sh \
  "https://api-id.execute-api.us-east-2.amazonaws.com/dev"
```

The script treats any response other than `401` or `403` as proof that API
Gateway accepted the valid token; feature handlers may still return `501` until
their implementation tickets are complete.
