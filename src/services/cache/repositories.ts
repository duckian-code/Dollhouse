import { cacheDatabase } from '@/services/cache/database';
import { expiresAt } from '@/services/cache/policy';
import type { CachedResult, CacheRecord } from '@/services/cache/types';
import type {
  AvatarCatalog,
  FriendStatus,
  Profile,
  UserSummary,
} from '@/types/api';

const ASSET_CATALOG_KEY = 'default';

function serialize<T>(domain: Parameters<typeof expiresAt>[0], value: T) {
  const cachedAtMs = Date.now();
  return {
    payloadJson: JSON.stringify(value),
    cachedAtMs,
    expiresAtMs: expiresAt(domain, cachedAtMs),
  };
}

function deserialize<T>(
  row: { payloadJson: string; cachedAtMs: number; expiresAtMs: number } | null,
): CacheRecord<T> | null {
  if (!row) return null;
  try {
    return {
      value: JSON.parse(row.payloadJson) as T,
      cachedAtMs: row.cachedAtMs,
      expiresAtMs: row.expiresAtMs,
    };
  } catch {
    return null;
  }
}

function result<T>(record: CacheRecord<T>): CachedResult<T> {
  return {
    data: record.value,
    source: 'cache',
    isStale: record.expiresAtMs <= Date.now(),
    cachedAt: new Date(record.cachedAtMs).toISOString(),
  };
}

export async function initializeCache() {
  await cacheDatabase.initialize();
}

export async function readCachedProfile(accountId: string) {
  const record = deserialize<Profile>(
    await cacheDatabase.getProfile(accountId),
  );
  return record ? result(record) : null;
}

export async function cacheProfile(accountId: string, profile: Profile) {
  await cacheDatabase.setProfile(accountId, serialize('profile', profile));
}

export async function readCachedAssetCatalog() {
  const record = deserialize<AvatarCatalog>(
    await cacheDatabase.getAssetCatalog(ASSET_CATALOG_KEY),
  );
  return record && isAvatarCatalog(record.value) ? result(record) : null;
}

function isAvatarCatalog(value: AvatarCatalog) {
  return (
    value !== null &&
    typeof value === 'object' &&
    typeof value.catalogVersion === 'string' &&
    typeof value.expiresAt === 'string' &&
    Array.isArray(value.assets)
  );
}

export async function cacheAssetCatalog(catalog: AvatarCatalog) {
  const cachedAtMs = Date.now();
  const serverExpiryMs = Date.parse(catalog.expiresAt);
  await cacheDatabase.setAssetCatalog(ASSET_CATALOG_KEY, {
    payloadJson: JSON.stringify(catalog),
    cachedAtMs,
    expiresAtMs: Number.isFinite(serverExpiryMs)
      ? serverExpiryMs
      : expiresAt('assetCatalog', cachedAtMs),
  });
}

export async function readCachedFriendSummaries(accountId: string) {
  const records = await cacheDatabase.getFriendSummaries(accountId);
  const values = records
    .map((row) => deserialize<UserSummary>(row))
    .filter((row): row is CacheRecord<UserSummary> => row !== null);
  if (!values.length) return null;
  const oldest = values.reduce((left, right) =>
    left.cachedAtMs < right.cachedAtMs ? left : right,
  );
  return result<UserSummary[]>({
    value: values.map(({ value }) => value),
    cachedAtMs: oldest.cachedAtMs,
    expiresAtMs: Math.min(...values.map(({ expiresAtMs }) => expiresAtMs)),
  });
}

export async function cacheFriendSummaries(
  accountId: string,
  friends: UserSummary[],
) {
  await cacheDatabase.replaceFriendSummaries(
    accountId,
    friends.map((friend) => ({
      friendId: friend.userId,
      ...serialize('friendSummary', friend),
    })),
  );
}

export async function readCachedRecentStatuses(accountId: string) {
  const records = await cacheDatabase.getRecentStatuses(accountId);
  const values = records
    .map((row) => deserialize<FriendStatus>(row))
    .filter((row): row is CacheRecord<FriendStatus> => row !== null);
  if (!values.length) return null;
  const oldest = values.reduce((left, right) =>
    left.cachedAtMs < right.cachedAtMs ? left : right,
  );
  return result<FriendStatus[]>({
    value: values.map(({ value }) => value),
    cachedAtMs: oldest.cachedAtMs,
    expiresAtMs: Math.min(...values.map(({ expiresAtMs }) => expiresAtMs)),
  });
}

export async function cacheRecentStatuses(
  accountId: string,
  statuses: FriendStatus[],
) {
  await cacheDatabase.replaceRecentStatuses(
    accountId,
    statuses.map((status) => ({
      friendId: status.friend.userId,
      ...serialize('recentStatus', status),
    })),
  );
}

export async function clearAccountCache(accountId: string) {
  await cacheDatabase.clearAccount(accountId);
}

export async function clearFriendCache(accountId: string) {
  await cacheDatabase.clearFriendData(accountId);
}

export async function pruneExpiredCache(nowMs = Date.now()) {
  await cacheDatabase.pruneExpired(nowMs);
}
