import type {
  CacheDatabase,
  StoredCacheRow,
} from '@/services/cache/database.types';

const profiles = new Map<string, StoredCacheRow>();
const catalogs = new Map<string, StoredCacheRow>();
const friendSummaries = new Map<string, Map<string, StoredCacheRow>>();
const recentStatuses = new Map<string, Map<string, StoredCacheRow>>();

function values(
  store: Map<string, Map<string, StoredCacheRow>>,
  accountId: string,
) {
  return [...(store.get(accountId)?.values() ?? [])];
}

function replace(
  store: Map<string, Map<string, StoredCacheRow>>,
  accountId: string,
  rows: (StoredCacheRow & { friendId: string })[],
) {
  store.set(
    accountId,
    new Map(rows.map(({ friendId, ...row }) => [friendId, row])),
  );
}

export const webCacheDatabase: CacheDatabase = {
  initialize: async () => undefined,
  getProfile: async (accountId) => profiles.get(accountId) ?? null,
  setProfile: async (accountId, row) => void profiles.set(accountId, row),
  getAssetCatalog: async (key) => catalogs.get(key) ?? null,
  setAssetCatalog: async (key, row) => void catalogs.set(key, row),
  getFriendSummaries: async (accountId) => values(friendSummaries, accountId),
  replaceFriendSummaries: async (accountId, rows) =>
    replace(friendSummaries, accountId, rows),
  getRecentStatuses: async (accountId) => values(recentStatuses, accountId),
  replaceRecentStatuses: async (accountId, rows) =>
    replace(recentStatuses, accountId, rows),
  clearFriendData: async (accountId) => {
    friendSummaries.delete(accountId);
    recentStatuses.delete(accountId);
  },
  clearAccount: async (accountId) => {
    profiles.delete(accountId);
    friendSummaries.delete(accountId);
    recentStatuses.delete(accountId);
  },
  pruneExpired: async (nowMs) => {
    for (const store of [profiles, catalogs]) {
      for (const [key, row] of store) {
        if (row.expiresAtMs <= nowMs) store.delete(key);
      }
    }
    for (const store of [friendSummaries, recentStatuses]) {
      for (const rows of store.values()) {
        for (const [key, row] of rows) {
          if (row.expiresAtMs <= nowMs) rows.delete(key);
        }
      }
    }
  },
};
