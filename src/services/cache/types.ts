import type { ISODateTime } from '@/types/api';

export type CacheDomain =
  'profile' | 'assetCatalog' | 'friendSummary' | 'recentStatus';

export type CacheSource = 'cache' | 'network';

export interface CachedResult<T> {
  data: T;
  source: CacheSource;
  isStale: boolean;
  cachedAt: ISODateTime;
}

export interface CacheRecord<T> {
  value: T;
  cachedAtMs: number;
  expiresAtMs: number;
}

export interface AssetCatalogItem {
  assetId: string;
  [key: string]: unknown;
}
