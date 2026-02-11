import { getApiBaseUrl } from '../config';
import { refreshWithMutex, request } from './client';
import type {
  AuthResponse,
  FIDO2CredentialInfo,
  RefreshResponse,
  RefreshResult,
  TwoFactorSetup,
  TwoFactorStatus,
  User,
} from './types';

export async function register(
  username: string,
  email: string,
  password: string,
  captchaToken?: string
): Promise<AuthResponse> {
  return request('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password, captcha_token: captchaToken }),
  });
}

export async function login(
  usernameOrEmail: string,
  password: string,
  captchaToken?: string,
  totpCode?: string,
  backupCode?: string
): Promise<AuthResponse> {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({
      username_or_email: usernameOrEmail,
      password,
      captcha_token: captchaToken,
      totp_code: totpCode,
      backup_code: backupCode,
    }),
  });
}

export async function refreshToken(refreshToken: string): Promise<RefreshResponse> {
  return request('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

/**
 * SEC-006: Refresh token using HttpOnly cookie (no body needed).
 * Used by proactive token refresh after page reload when token is not in memory.
 * credentials: 'include' sends the refresh_token cookie automatically.
 * @returns RefreshResult with differentiated error reasons
 */
export async function refreshTokenViaCookie(): Promise<RefreshResult> {
  return refreshWithMutex();
}

export async function logoutApi(refreshToken: string): Promise<void> {
  return request('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export async function getCurrentUser(): Promise<User> {
  return request('/auth/me');
}

// Two-Factor Authentication
export async function setup2FA(): Promise<TwoFactorSetup> {
  return request('/2fa/setup', {
    method: 'POST',
  });
}

export async function verify2FA(code: string): Promise<{ message: string }> {
  return request('/2fa/verify', {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
}

export async function disable2FA(
  password: string,
  totpCode?: string,
  backupCode?: string
): Promise<{ message: string }> {
  return request('/2fa', {
    method: 'DELETE',
    body: JSON.stringify({
      password,
      totp_code: totpCode,
      backup_code: backupCode,
    }),
  });
}

export async function get2FAStatus(): Promise<TwoFactorStatus> {
  return request('/2fa/status');
}

// SEC-009: Requires password re-authentication
export async function regenerateBackupCodes(password: string): Promise<{ backup_codes: string[] }> {
  return request('/2fa/backup-codes/regenerate', {
    method: 'POST',
    body: JSON.stringify({ password }),
  });
}

// FIDO2/WebAuthn 2FA API

export async function beginFIDO2Registration(): Promise<unknown> {
  return request('/2fa/fido2/register/begin', { method: 'POST' });
}

export async function finishFIDO2Registration(
  deviceName: string,
  credential: Record<string, unknown>
): Promise<{ credential_id: number; backup_codes?: string[] }> {
  const response = await fetch(
    `${getApiBaseUrl()}/2fa/fido2/register/finish?device_name=${encodeURIComponent(deviceName)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(credential),
    }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Registration failed' }));
    throw new Error(error.error || 'Registration failed');
  }
  return response.json();
}

export async function listFIDO2Credentials(): Promise<FIDO2CredentialInfo[]> {
  return request('/2fa/fido2/credentials');
}

export async function deleteFIDO2Credential(id: number): Promise<void> {
  return request(`/2fa/fido2/credentials/${id}`, { method: 'DELETE' });
}

export async function beginFIDO2Auth(
  pendingLoginToken: string
): Promise<unknown> {
  const response = await fetch(`${getApiBaseUrl()}/auth/fido2/begin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ pending_login_token: pendingLoginToken }),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Authentication failed' }));
    throw new Error(error.error || 'Authentication failed');
  }
  return response.json();
}

export async function finishFIDO2Auth(
  pendingLoginToken: string,
  credential: Record<string, unknown>
): Promise<AuthResponse> {
  const response = await fetch(
    `${getApiBaseUrl()}/auth/fido2/finish?pending_login_token=${encodeURIComponent(pendingLoginToken)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(credential),
    }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Authentication failed' }));
    throw new Error(error.error || 'Authentication failed');
  }
  return response.json();
}
