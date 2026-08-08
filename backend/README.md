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
```

`make build` cross-compiles every entry point for the AWS Lambda `provided.al2023`
ARM64 runtime and writes artifacts under `.build/<lambda>/bootstrap`.

Copy `.env.example` to `.env` when running handlers against local AWS-compatible
services. Environment variables are loaded by `internal/config`; deployed values
should be supplied by infrastructure rather than committed to source control.
