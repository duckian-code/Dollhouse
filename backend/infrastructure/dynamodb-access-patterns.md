# DynamoDB access patterns

The development stack uses four on-demand DynamoDB tables. All tables use
AWS-owned encryption at rest, point-in-time recovery, and CloudFormation retain
policies. Attribute names below are the application-level contract for backend
feature handlers.

## Users

Primary key: `userId` (string partition key).

Indexes:

- `EmailIndex`: `normalizedEmail` partition key. Supports exact, case-insensitive
  email lookup and uniqueness checks.
- `UserSearchIndex`: `searchPartition` partition key and `normalizedUsername`
  sort key. MVP user records use `searchPartition = USER`; query with
  `begins_with(normalizedUsername, :prefix)` for username search. This single
  logical partition is acceptable at MVP scale and can be sharded later.

One item contains the profile, doll configuration, and current disclosed status.
The application model is:

```text
userId, cognitoSub, email, normalizedEmail, username, normalizedUsername,
searchPartition="USER", displayName, bio?, role, dollConfiguration?,
currentStatus?, createdAt, updatedAt
```

`dollConfiguration` contains `bodyAssetId`, `hairAssetId`, `eyesAssetId`,
`noseAssetId`, `mouthAssetId`, `clothingAssetIds`, and `updatedAt`.
`currentStatus` contains the user's open `status` string, nullable `stress`,
`fatigue`, and `discomfort` values, plus `updatedAt`. Asset IDs are references to
the approved S3 catalog, never embedded assets.

| Workflow | DynamoDB operation |
| --- | --- |
| Get current profile or doll configuration | `GetItem(userId)` |
| Update owned profile, doll, or current status | conditional `UpdateItem(userId)` |
| Find user by email | `Query EmailIndex` with `normalizedEmail = :email` |
| Search users by username prefix | `Query UserSearchIndex` with `searchPartition = USER` and `begins_with(normalizedUsername, :prefix)` |
| Read accepted friends' current status | `BatchGetItem` by IDs returned from Friendships |

User-search indexes are eventually consistent. Profile reads that immediately
follow an update may request strongly consistent reads on the base table.

## Friendships

Primary key: `userId` (partition key), `relatedUserId` (sort key).

Indexes:

- `UserStatusIndex`, keyed by `userId` and `statusRelatedUserId`. The index sort
  key is `<STATUS>#<relatedUserId>`, where status is `PENDING_INCOMING`,
  `PENDING_OUTGOING`, or `ACCEPTED`.
- `RequestIdIndex`, keyed by `requestId` and `userId`. It resolves the opaque
  request ID used by the accept and decline API routes without a table scan.

Each relationship is stored as two mirrored items. Send, accept, decline, and
remove operations must use `TransactWriteItems` so both users see one state.
Self-friendship and duplicate requests are rejected with condition expressions.
Pending mirror items share `requestId`, `requestedBy`, and `requestedAt`.
Accepted mirrors retain `requestId` for traceability and add `acceptedAt`.

| Workflow | DynamoDB operation |
| --- | --- |
| Read relationship to a specific user | `GetItem(userId, relatedUserId)` |
| List incoming/outgoing requests | `Query UserStatusIndex` with `begins_with(statusRelatedUserId, :status)` |
| List accepted friends | same index query using `ACCEPTED#` |
| Resolve an accept/decline route ID | `Query RequestIdIndex` with `requestId = :requestId` and verify the signed-in user owns the incoming item |
| Send request | transactional put of outgoing and incoming mirror items |
| Accept request | transactional update of both items to `ACCEPTED` |
| Decline request or remove friend | transactional delete of both mirror items |

## MoodEvents

Primary key: `userId` (partition key), `occurredAt` (sort key). `occurredAt` is
an ISO-8601 UTC timestamp suffixed with `#<eventId>` to prevent collisions.

| Workflow | DynamoDB operation |
| --- | --- |
| Append mood history | conditional `PutItem(userId, occurredAt)` |
| Read a user's history window | `Query userId` with `BETWEEN` bounds |
| Read most recent event | `Query userId`, descending, limit one |

Each event stores `eventId`, the open `status` string, nullable `stress`,
`fatigue`, and `discomfort` values, `visibility = FRIENDS`, and `updatedAt`.
There is no fixed mood catalogue or `moodId` in the current contract.

Publishing a status transactionally updates `currentStatus` on the Users item
and appends a MoodEvents item. Notification messages contain identifiers only;
they do not duplicate sensitive status or slider values.

## Devices

Primary key: `userId` (partition key), `deviceId` (sort key).

Each item contains `pushToken`, `platform`, `enabled`, and `updatedAt` in
addition to its keys. Disabled devices remain addressable for cleanup but are
excluded from notification targets.

| Workflow | DynamoDB operation |
| --- | --- |
| Register or refresh a device token | conditional `PutItem(userId, deviceId)` |
| List notification targets for a user | `Query userId` |
| Sign out or remove a device | `DeleteItem(userId, deviceId)` |

Push tokens are sensitive operational data. They must not appear in logs, SQS
messages, API responses, or the Users table.

## Test fixtures

`scripts/seed-dynamodb-test-data.sh` creates Alice, Bob, and Carol fixtures,
accepted and pending mirrored relationships, a mood event, and one device. All
fixture primary keys begin with `TEST#`; rerunning the script overwrites only
those deterministic fixtures. `scripts/verify-dynamodb-access-patterns.sh`
exercises base-table reads, both user indexes, the friendship status index, mood
history, device lookup, encryption state, and point-in-time recovery.
