import {
  acceptFriendRequest,
  declineFriendRequest,
  getAvatarCatalog,
  getFriendStatuses,
  getProfile,
  removeFriend,
  sendFriendRequest,
  updateProfile,
} from '@/services/api/dollhouse';
import { getCurrentAccountId } from '@/services/auth/cognito';
import {
  cacheFriendSummaries,
  cacheAssetCatalog,
  cacheProfile,
  cacheRecentStatuses,
  clearFriendCache,
  readCachedFriendSummaries,
  readCachedAssetCatalog,
  readCachedProfile,
  readCachedRecentStatuses,
} from '@/services/cache/repositories';
import type { CachedResult } from '@/services/cache/types';
import type {
  FriendStatus,
  AvatarCatalog,
  Profile,
  SendFriendRequestRequest,
  UpdateProfileRequest,
  UserSummary,
} from '@/types/api';

export interface CacheThenNetwork<T> {
  cached: CachedResult<T> | null;
  refresh: Promise<CachedResult<T>>;
}

function networkResult<T>(data: T): CachedResult<T> {
  return {
    data,
    source: 'network',
    isStale: false,
    cachedAt: new Date().toISOString(),
  };
}

export async function loadProfile(): Promise<CacheThenNetwork<Profile>> {
  const accountId = await getCurrentAccountId();
  const cached = await readCachedProfile(accountId);
  const refresh = getProfile().then(async ({ data }) => {
    await cacheProfile(accountId, data.profile);
    return networkResult(data.profile);
  });
  return { cached, refresh };
}

export async function loadAvatarCatalog(): Promise<
  CacheThenNetwork<AvatarCatalog>
> {
  const cached = await readCachedAssetCatalog();
  const refresh = getAvatarCatalog().then(async ({ data }) => {
    await cacheAssetCatalog(data);
    return networkResult(data);
  });
  return { cached, refresh };
}

export async function saveProfile(request: UpdateProfileRequest) {
  const accountId = await getCurrentAccountId();
  const response = await updateProfile(request);
  await cacheProfile(accountId, response.data.profile);
  return response;
}

export async function loadFriendSummaries(): Promise<
  CacheThenNetwork<UserSummary[]>
> {
  const accountId = await getCurrentAccountId();
  const cached = await readCachedFriendSummaries(accountId);
  const refresh = getFriendStatuses().then(async ({ data }) => {
    const friends = data.items.map(({ friend }) => friend);
    await cacheFriendSummaries(accountId, friends);
    return networkResult(friends);
  });
  return { cached, refresh };
}

export async function loadRecentStatuses(): Promise<
  CacheThenNetwork<FriendStatus[]>
> {
  const accountId = await getCurrentAccountId();
  const cached = await readCachedRecentStatuses(accountId);
  const refresh = getFriendStatuses().then(async ({ data }) => {
    await Promise.all([
      cacheRecentStatuses(accountId, data.items),
      cacheFriendSummaries(
        accountId,
        data.items.map(({ friend }) => friend),
      ),
    ]);
    return networkResult(data.items);
  });
  return { cached, refresh };
}

async function refreshFriendDataAfter<T>(request: Promise<T>) {
  const accountId = await getCurrentAccountId();
  const response = await request;
  await clearFriendCache(accountId);
  return response;
}

export function sendFriendRequestAndRefresh(request: SendFriendRequestRequest) {
  return refreshFriendDataAfter(sendFriendRequest(request));
}

export function acceptFriendRequestAndRefresh(requestId: string) {
  return refreshFriendDataAfter(acceptFriendRequest(requestId));
}

export function declineFriendRequestAndRefresh(requestId: string) {
  return refreshFriendDataAfter(declineFriendRequest(requestId));
}

export function removeFriendAndRefresh(friendId: string) {
  return refreshFriendDataAfter(removeFriend(friendId));
}
