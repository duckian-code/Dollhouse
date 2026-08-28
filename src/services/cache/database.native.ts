import * as SQLite from 'expo-sqlite';

import type {
  CacheDatabase,
  StoredCacheRow,
} from '@/services/cache/database.types';

const DATABASE_NAME = 'dollhouse-cache.db';
const SCHEMA_VERSION = 1;

type SqlRow = {
  payload_json: string;
  cached_at_ms: number;
  expires_at_ms: number;
};

let databasePromise: Promise<SQLite.SQLiteDatabase> | undefined;

async function openDatabase() {
  databasePromise ??= SQLite.openDatabaseAsync(DATABASE_NAME);
  return databasePromise;
}

async function migrate() {
  const database = await openDatabase();
  await database.execAsync('PRAGMA journal_mode = WAL;');
  const version = await database.getFirstAsync<{ user_version: number }>(
    'PRAGMA user_version;',
  );
  if ((version?.user_version ?? 0) >= SCHEMA_VERSION) return;

  await database.withTransactionAsync(async () => {
    await database.execAsync(`
      CREATE TABLE IF NOT EXISTS profile_cache (
        account_id TEXT PRIMARY KEY NOT NULL,
        payload_json TEXT NOT NULL,
        cached_at_ms INTEGER NOT NULL,
        expires_at_ms INTEGER NOT NULL
      );
      CREATE TABLE IF NOT EXISTS asset_catalog_cache (
        catalog_key TEXT PRIMARY KEY NOT NULL,
        payload_json TEXT NOT NULL,
        cached_at_ms INTEGER NOT NULL,
        expires_at_ms INTEGER NOT NULL
      );
      CREATE TABLE IF NOT EXISTS friend_summary_cache (
        account_id TEXT NOT NULL,
        friend_id TEXT NOT NULL,
        payload_json TEXT NOT NULL,
        cached_at_ms INTEGER NOT NULL,
        expires_at_ms INTEGER NOT NULL,
        PRIMARY KEY (account_id, friend_id)
      );
      CREATE TABLE IF NOT EXISTS recent_status_cache (
        account_id TEXT NOT NULL,
        friend_id TEXT NOT NULL,
        payload_json TEXT NOT NULL,
        cached_at_ms INTEGER NOT NULL,
        expires_at_ms INTEGER NOT NULL,
        PRIMARY KEY (account_id, friend_id)
      );
      CREATE INDEX IF NOT EXISTS friend_summary_expiry
        ON friend_summary_cache(expires_at_ms);
      CREATE INDEX IF NOT EXISTS recent_status_expiry
        ON recent_status_cache(expires_at_ms);
      PRAGMA user_version = 1;
    `);
  });
}

function toStoredRow(row: SqlRow | null): StoredCacheRow | null {
  return row
    ? {
        payloadJson: row.payload_json,
        cachedAtMs: row.cached_at_ms,
        expiresAtMs: row.expires_at_ms,
      }
    : null;
}

async function getOne(table: string, keyColumn: string, key: string) {
  const database = await openDatabase();
  const row = await database.getFirstAsync<SqlRow>(
    `SELECT payload_json, cached_at_ms, expires_at_ms FROM ${table} WHERE ${keyColumn} = ?`,
    key,
  );
  return toStoredRow(row);
}

async function setOne(
  table: string,
  keyColumn: string,
  key: string,
  row: StoredCacheRow,
) {
  const database = await openDatabase();
  await database.runAsync(
    `INSERT OR REPLACE INTO ${table} (${keyColumn}, payload_json, cached_at_ms, expires_at_ms) VALUES (?, ?, ?, ?)`,
    key,
    row.payloadJson,
    row.cachedAtMs,
    row.expiresAtMs,
  );
}

async function getMany(table: string, accountId: string) {
  const database = await openDatabase();
  const rows = await database.getAllAsync<SqlRow>(
    `SELECT payload_json, cached_at_ms, expires_at_ms FROM ${table} WHERE account_id = ? ORDER BY cached_at_ms DESC`,
    accountId,
  );
  return rows.map((row) => toStoredRow(row)!);
}

async function replaceMany(
  table: string,
  accountId: string,
  rows: (StoredCacheRow & { friendId: string })[],
) {
  const database = await openDatabase();
  await database.withTransactionAsync(async () => {
    await database.runAsync(
      `DELETE FROM ${table} WHERE account_id = ?`,
      accountId,
    );
    for (const row of rows) {
      await database.runAsync(
        `INSERT INTO ${table} (account_id, friend_id, payload_json, cached_at_ms, expires_at_ms) VALUES (?, ?, ?, ?, ?)`,
        accountId,
        row.friendId,
        row.payloadJson,
        row.cachedAtMs,
        row.expiresAtMs,
      );
    }
  });
}

export const cacheDatabase: CacheDatabase = {
  initialize: migrate,
  getProfile: (accountId) => getOne('profile_cache', 'account_id', accountId),
  setProfile: (accountId, row) =>
    setOne('profile_cache', 'account_id', accountId, row),
  getAssetCatalog: (catalogKey) =>
    getOne('asset_catalog_cache', 'catalog_key', catalogKey),
  setAssetCatalog: (catalogKey, row) =>
    setOne('asset_catalog_cache', 'catalog_key', catalogKey, row),
  getFriendSummaries: (accountId) => getMany('friend_summary_cache', accountId),
  replaceFriendSummaries: (accountId, rows) =>
    replaceMany('friend_summary_cache', accountId, rows),
  getRecentStatuses: (accountId) => getMany('recent_status_cache', accountId),
  replaceRecentStatuses: (accountId, rows) =>
    replaceMany('recent_status_cache', accountId, rows),
  async clearFriendData(accountId) {
    const database = await openDatabase();
    await database.withTransactionAsync(async () => {
      await database.runAsync(
        'DELETE FROM friend_summary_cache WHERE account_id = ?',
        accountId,
      );
      await database.runAsync(
        'DELETE FROM recent_status_cache WHERE account_id = ?',
        accountId,
      );
    });
  },
  async clearAccount(accountId) {
    const database = await openDatabase();
    await database.withTransactionAsync(async () => {
      await database.runAsync(
        'DELETE FROM profile_cache WHERE account_id = ?',
        accountId,
      );
      await database.runAsync(
        'DELETE FROM friend_summary_cache WHERE account_id = ?',
        accountId,
      );
      await database.runAsync(
        'DELETE FROM recent_status_cache WHERE account_id = ?',
        accountId,
      );
    });
  },
  async pruneExpired(nowMs) {
    const database = await openDatabase();
    for (const table of [
      'profile_cache',
      'asset_catalog_cache',
      'friend_summary_cache',
      'recent_status_cache',
    ]) {
      await database.runAsync(
        `DELETE FROM ${table} WHERE expires_at_ms <= ?`,
        nowMs,
      );
    }
  },
};
