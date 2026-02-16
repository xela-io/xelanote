import { expect, type Page } from '@playwright/test';

export interface TestCredentials {
  username: string;
  email: string;
  password: string;
}

let cachedCredentials: TestCredentials | null = null;

export function spoofedClientIP(): string {
  const octet = Math.floor(Math.random() * 200) + 20;
  return `203.0.113.${octet}`;
}

export function createCredentials(): TestCredentials {
  const nonce = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  return {
    username: `testuser-${nonce}`,
    email: `testuser-${nonce}@example.com`,
    password: 'password123',
  };
}

export async function registerViaApi(page: Page, credentials: TestCredentials): Promise<void> {
  let delayMs = 1000;
  let lastStatus = 0;
  let lastBody = '';

  for (let attempt = 0; attempt < 8; attempt++) {
    const response = await page.request.post('/api/auth/register', {
      headers: {
        'X-Forwarded-For': spoofedClientIP(),
      },
      data: {
        username: credentials.username,
        email: credentials.email,
        password: credentials.password,
      },
    });
    if (response.ok()) {
      return;
    }

    lastStatus = response.status();
    lastBody = await response.text();
    if (lastStatus !== 429) {
      break;
    }

    await page.waitForTimeout(delayMs);
    delayMs = Math.min(delayMs * 2, 8000);
  }

  throw new Error(`register failed: ${lastStatus} ${lastBody}`);
}

export async function loginViaApi(page: Page, credentials: TestCredentials): Promise<void> {
  let delayMs = 1000;
  let lastStatus = 0;
  let lastBody = '';

  for (let attempt = 0; attempt < 8; attempt++) {
    const response = await page.request.post('/api/auth/login', {
      headers: {
        'X-Forwarded-For': spoofedClientIP(),
      },
      data: {
        username_or_email: credentials.email,
        password: credentials.password,
      },
    });
    if (response.ok()) {
      return;
    }

    lastStatus = response.status();
    lastBody = await response.text();
    if (lastStatus !== 429) {
      break;
    }

    await page.waitForTimeout(delayMs);
    delayMs = Math.min(delayMs * 2, 8000);
  }

  throw new Error(`login failed: ${lastStatus} ${lastBody}`);
}

async function ensureRegisteredUser(page: Page): Promise<TestCredentials> {
  if (cachedCredentials) {
    return cachedCredentials;
  }

  // Retry once in case of transient rate limiting.
  let lastError: Error | null = null;
  for (let attempt = 0; attempt < 2; attempt++) {
    try {
      const credentials = createCredentials();
      await registerViaApi(page, credentials);
      cachedCredentials = credentials;
      return credentials;
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));
      if (attempt === 0) {
        await page.waitForTimeout(1500);
      }
    }
  }

  throw lastError ?? new Error('failed to create test user');
}

export async function registerNewAndLogin(page: Page): Promise<TestCredentials> {
  const credentials = createCredentials();
  await registerViaApi(page, credentials);
  await loginViaApi(page, credentials);
  await page.goto('/');
  await expect(page).toHaveURL(/\/$/, { timeout: 15000 });
  await page.waitForTimeout(1000);
  return credentials;
}

export async function registerAndLoginApi(
  page: Page,
  options?: { forceNewUser?: boolean }
): Promise<TestCredentials> {
  const credentials = options?.forceNewUser
    ? (() => {
        // Inline to keep the same auth path without UI assumptions.
        return createCredentials();
      })()
    : await ensureRegisteredUser(page);

  if (options?.forceNewUser) {
    await registerViaApi(page, credentials);
  }
  await loginViaApi(page, credentials);
  await page.waitForTimeout(300);
  return credentials;
}

export async function registerAndLogin(
  page: Page,
  options?: { forceNewUser?: boolean }
): Promise<TestCredentials> {
  const credentials = options?.forceNewUser
    ? await registerNewAndLogin(page)
    : await ensureRegisteredUser(page);

  if (!options?.forceNewUser) {
    await loginViaApi(page, credentials);
    await page.goto('/');
    await expect(page).toHaveURL(/\/$/, { timeout: 15000 });
  }
  await page.waitForTimeout(1000);

  return credentials;
}
