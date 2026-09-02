import { apiRequest } from '@/services/api/client';
import type {
  AvatarCatalogResponse,
  DollConfigurationResponse,
  FriendRequestResponse,
  FriendRequestsResponse,
  FriendStatusesResponse,
  FriendshipResponse,
  MoodEntriesResponse,
  PaginatedResponse,
  ProfileResponse,
  PublishMoodRequest,
  PublishMoodResponse,
  SendFriendRequestRequest,
  UpdateDollConfigurationRequest,
  UpdateProfileRequest,
  UserSummary,
  UsernameAvailabilityResponse,
} from '@/types/api';

function jsonRequest(method: 'POST' | 'PUT', body: unknown): RequestInit {
  return { method, body: JSON.stringify(body) };
}

function routeId(value: string) {
  return encodeURIComponent(value);
}

export function getProfile() {
  return apiRequest<ProfileResponse>('/profile');
}

export function updateProfile(request: UpdateProfileRequest) {
  return apiRequest<ProfileResponse>('/profile', jsonRequest('PUT', request));
}

export function getUsernameAvailability(username: string) {
  const params = new URLSearchParams({ username });
  return apiRequest<UsernameAvailabilityResponse>(
    `/users/username-availability?${params.toString()}`,
  );
}

export function getDollConfiguration() {
  return apiRequest<DollConfigurationResponse>('/doll');
}

export function getAvatarCatalog() {
  return apiRequest<AvatarCatalogResponse>('/assets/catalog');
}

export function updateDollConfiguration(
  request: UpdateDollConfigurationRequest,
) {
  return apiRequest<DollConfigurationResponse>(
    '/doll',
    jsonRequest('PUT', request),
  );
}

export function searchUsers(query: string, nextToken?: string) {
  const params = new URLSearchParams({ q: query });
  if (nextToken) params.set('nextToken', nextToken);
  return apiRequest<PaginatedResponse<UserSummary>>(
    `/users/search?${params.toString()}`,
  );
}

export function sendFriendRequest(request: SendFriendRequestRequest) {
  return apiRequest<FriendRequestResponse>(
    '/friend-requests',
    jsonRequest('POST', request),
  );
}

export function listFriendRequests() {
  return apiRequest<FriendRequestsResponse>('/friend-requests');
}

export function acceptFriendRequest(requestId: string) {
  return apiRequest<FriendshipResponse>(
    `/friend-requests/${routeId(requestId)}/accept`,
    { method: 'POST' },
  );
}

export function declineFriendRequest(requestId: string) {
  return apiRequest<void>(`/friend-requests/${routeId(requestId)}/decline`, {
    method: 'POST',
  });
}

export function removeFriend(friendId: string) {
  return apiRequest<void>(`/friends/${routeId(friendId)}`, {
    method: 'DELETE',
  });
}

export function publishMood(request: PublishMoodRequest) {
  return apiRequest<PublishMoodResponse>(
    '/moods',
    jsonRequest('POST', request),
  );
}

export function getMoodEntries(nextToken?: string) {
  const params = new URLSearchParams();
  if (nextToken) params.set('nextToken', nextToken);
  const query = params.size ? `?${params.toString()}` : '';
  return apiRequest<MoodEntriesResponse>(`/moods${query}`);
}

export function getFriendStatuses(nextToken?: string) {
  const params = new URLSearchParams();
  if (nextToken) params.set('nextToken', nextToken);
  const query = params.size ? `?${params.toString()}` : '';
  return apiRequest<FriendStatusesResponse>(`/friend-statuses${query}`);
}
