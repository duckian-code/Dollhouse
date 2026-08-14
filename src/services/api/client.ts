import { environment } from '@/config/environment';
import { getAccessToken } from '@/services/auth/cognito';
import type { ErrorResponse } from '@/types/api';

export function createApiUrl(path: string): URL {
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path;
  const baseUrl = environment.apiUrl.endsWith('/')
    ? environment.apiUrl
    : `${environment.apiUrl}/`;

  return new URL(normalizedPath, baseUrl);
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

function isErrorResponse(value: unknown): value is ErrorResponse {
  if (!value || typeof value !== 'object' || !('error' in value)) return false;
  const detail = value.error;
  return (
    detail !== null &&
    typeof detail === 'object' &&
    'code' in detail &&
    typeof detail.code === 'string' &&
    'message' in detail &&
    typeof detail.message === 'string'
  );
}

export async function apiRequest<T>(path: string, init: RequestInit = {}) {
  const accessToken = await getAccessToken();
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  headers.set('Content-Type', 'application/json');
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`);

  const response = await fetch(createApiUrl(path), { ...init, headers });
  if (response.status === 204) return undefined as T;

  const text = await response.text();
  let body: unknown;
  try {
    body = text ? JSON.parse(text) : undefined;
  } catch {
    body = undefined;
  }

  if (!response.ok) {
    if (isErrorResponse(body)) {
      throw new ApiError(response.status, body.error.code, body.error.message);
    }
    // API Gateway can generate its own non-contract 401 response body.
    if (response.status === 401) {
      throw new ApiError(401, 'unauthenticated', 'Please sign in again.');
    }
    throw new ApiError(
      response.status,
      'unexpected_response',
      `The server returned HTTP ${response.status}.`,
    );
  }

  return body as T;
}
