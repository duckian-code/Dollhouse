import { ApiError } from '@/services/api/client';

const apiMessages: Record<string, string> = {
  network_unavailable:
    'You appear to be offline. Check your connection and try again.',
  unauthenticated: 'Your session expired. Please sign in again.',
  forbidden: 'You do not have permission to do that.',
  not_found: 'That information is no longer available.',
  conflict: 'That change conflicts with a newer update. Refresh and try again.',
  rate_limited: 'Too many attempts. Wait a moment and try again.',
  unexpected_response: 'The service is temporarily unavailable. Try again.',
};

export function getUserFacingError(
  caught: unknown,
  fallback = 'Something went wrong. Please try again.',
) {
  if (caught instanceof ApiError) {
    if (apiMessages[caught.code]) return apiMessages[caught.code];
    if (caught.status === 401) return apiMessages.unauthenticated;
    if (caught.status === 403) return apiMessages.forbidden;
    if (caught.status === 404) return apiMessages.not_found;
    if (caught.status === 409) return apiMessages.conflict;
    if (caught.status === 429) return apiMessages.rate_limited;
    if (caught.status >= 500) return apiMessages.unexpected_response;
    return fallback;
  }
  if (caught instanceof TypeError) return apiMessages.network_unavailable;
  return fallback;
}
