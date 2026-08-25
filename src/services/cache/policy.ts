import type { CacheDomain } from '@/services/cache/types';

const MINUTE = 60_000;

export const cacheTtlMs: Record<CacheDomain, number> = {
  recentStatus: 5 * MINUTE,
  friendSummary: 15 * MINUTE,
  profile: 30 * MINUTE,
  assetCatalog: 15 * MINUTE,
};

export function expiresAt(domain: CacheDomain, cachedAtMs: number) {
  return cachedAtMs + cacheTtlMs[domain];
}
