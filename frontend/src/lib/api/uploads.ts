import { getApiBaseUrl } from '../config';
import {
  ApiError,
  getAccessTokenValue,
  getCSRFToken,
  logoutAndRedirect,
  refreshWithMutex,
} from './client';
import type { UploadResponse } from './types';

export async function uploadImage(file: File): Promise<UploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const accessToken = getAccessTokenValue();
  const headers = new Headers();
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  // Add CSRF token for state-changing requests (required when cookies are sent)
  const csrfToken = getCSRFToken();
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  // SEC-006: Include credentials for cookie-based authentication
  const response = await fetch(`${getApiBaseUrl()}/uploads`, {
    method: 'POST',
    headers: headers,
    body: formData,
    credentials: 'include',
  });

  // Handle 401 with token refresh using central mutex
  if (response.status === 401 && accessToken) {
    const result = await refreshWithMutex();

    if (result.success) {
      // Retry upload with new token
      const retryHeaders = new Headers();
      retryHeaders.set('Authorization', `Bearer ${result.tokens.access_token}`);

      // CSRF token is refreshed along with access token, get the new one
      const newCsrfToken = getCSRFToken();
      if (newCsrfToken) {
        retryHeaders.set('X-CSRF-Token', newCsrfToken);
      }

      const retryResponse = await fetch(`${getApiBaseUrl()}/uploads`, {
        method: 'POST',
        headers: retryHeaders,
        body: formData,
        credentials: 'include', // SEC-006: Include credentials
      });

      if (!retryResponse.ok) {
        const error = await retryResponse.json().catch(() => ({ error: 'Upload failed' }));
        throw new ApiError(error.error || 'Upload failed', retryResponse.status);
      }

      return retryResponse.json();
    } else {
      // Refresh failed, logout
      logoutAndRedirect();
      throw new ApiError('Session expired', 401);
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Upload failed' }));
    throw new ApiError(error.error || 'Upload failed', response.status);
  }

  return response.json();
}
