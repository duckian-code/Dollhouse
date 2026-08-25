import { environment } from '@/config/environment';

export function createApiUrl(path: string): URL {
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path;
  const baseUrl = environment.apiUrl.endsWith('/')
    ? environment.apiUrl
    : `${environment.apiUrl}/`;

  return new URL(normalizedPath, baseUrl);
}
