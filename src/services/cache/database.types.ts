export interface StoredCacheRow {
  payloadJson: string;
  cachedAtMs: number;
  expiresAtMs: number;
}

export interface CacheDatabase {
  initialize(): Promise<void>;
  getProfile(accountId: string): Promise<StoredCacheRow | null>;
  setProfile(accountId: string, row: StoredCacheRow): Promise<void>;
  getAssetCatalog(catalogKey: string): Promise<StoredCacheRow | null>;
  setAssetCatalog(catalogKey: string, row: StoredCacheRow): Promise<void>;
  getFriendSummaries(accountId: string): Promise<StoredCacheRow[]>;
  replaceFriendSummaries(
    accountId: string,
    rows: (StoredCacheRow & { friendId: string })[],
  ): Promise<void>;
  getRecentStatuses(accountId: string): Promise<StoredCacheRow[]>;
  replaceRecentStatuses(
    accountId: string,
    rows: (StoredCacheRow & { friendId: string })[],
  ): Promise<void>;
  clearFriendData(accountId: string): Promise<void>;
  clearAccount(accountId: string): Promise<void>;
  pruneExpired(nowMs: number): Promise<void>;
}
