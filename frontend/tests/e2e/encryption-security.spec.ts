import { expect, type Page,test } from '@playwright/test';

import { registerAndLoginApi } from './helpers/auth';

function spoofedClientIP(): string {
  const octet = Math.floor(Math.random() * 200) + 20;
  return `203.0.113.${octet}`;
}

async function csrfToken(page: Page): Promise<string | null> {
  const cookies = await page.context().cookies('http://localhost:4173');
  return cookies.find((cookie) => cookie.name === 'csrf_token')?.value ?? null;
}

async function apiRequest(
  page: Page,
  method: 'GET' | 'PUT',
  path: string,
  body?: Record<string, unknown>
): Promise<{ status: number; payload: unknown }> {
  const csrf = await csrfToken(page);
  const headers: Record<string, string> = {
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrf) {
    headers['X-CSRF-Token'] = csrf;
  }
  if (body) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await page.request.fetch(path, {
    method,
    headers,
    data: body,
  });

  let payload: unknown = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  return { status: response.status(), payload };
}

async function getSecurityPrefs(page: Page): Promise<{ security_level: string; auto_lock_timeout: number }> {
  const response = await apiRequest(page, 'GET', '/api/users/preferences');
  expect(response.status).toBe(200);
  const prefs = response.payload as { security_level: string; auto_lock_timeout: number };
  return {
    security_level: prefs.security_level,
    auto_lock_timeout: prefs.auto_lock_timeout,
  };
}

async function updateSecurityPrefs(
  page: Page,
  body: { security_level?: string; auto_lock_timeout?: number }
): Promise<{ status: number; payload: unknown }> {
  return apiRequest(page, 'PUT', '/api/users/preferences/security', body);
}

test.describe('Encryption Security Preferences API', () => {
  test.beforeEach(async ({ page }) => {
    await registerAndLoginApi(page);
    await page.goto('/login');
  });

  test('paranoid mode can be set and persists', async ({ page }) => {
    const update = await updateSecurityPrefs(page, { security_level: 'paranoid' });
    expect(update.status).toBe(200);

    const prefs = await getSecurityPrefs(page);
    expect(prefs.security_level).toBe('paranoid');
  });

  test('balanced mode can be set and persists', async ({ page }) => {
    const update = await updateSecurityPrefs(page, { security_level: 'balanced' });
    expect(update.status).toBe(200);

    const prefs = await getSecurityPrefs(page);
    expect(prefs.security_level).toBe('balanced');
  });

  test('switching from balanced to paranoid persists final level', async ({ page }) => {
    const first = await updateSecurityPrefs(page, { security_level: 'balanced' });
    expect(first.status).toBe(200);

    const second = await updateSecurityPrefs(page, { security_level: 'paranoid' });
    expect(second.status).toBe(200);

    const prefs = await getSecurityPrefs(page);
    expect(prefs.security_level).toBe('paranoid');
  });

  test('invalid security level is rejected', async ({ page }) => {
    const response = await updateSecurityPrefs(page, { security_level: 'invalid-level' });
    expect(response.status).toBe(400);
  });
});
