import { test as base, expect, type Page } from '@playwright/test';

interface AuthContext {
  page: Page;
  testNoteId: string;
}

interface AuthFixture {
  authenticatedContext: AuthContext;
}

export const test = base.extend<AuthFixture>({
  authenticatedContext: async ({ page, baseURL }, use) => {
    // IMPORTANT: baseURL must be set in playwright.config.ts (e.g., http://127.0.0.1:4173)
    // All API requests go through Vite dev server proxy (/api -> localhost:8080)
    if (!baseURL) {
      throw new Error('baseURL must be set in playwright.config.ts');
    }

    // 1. Register test user via API (through frontend proxy)
    const testUsername = `e2e-test-${Date.now()}`;
    const testPassword = 'Test123!@#';

    try {
      await page.request.post(`${baseURL}/api/auth/register`, {
        data: {
          username: testUsername,
          email: `${testUsername}@test.local`,
          password: testPassword,
        },
      });
    } catch (_error) {
      // User might already exist or registration failed - ignore, login will handle it
    }

    // 2. Login via UI (establishes browser session)
    // Use baseURL explicitly to avoid failures if baseURL not set
    await page.goto(`${baseURL}/login`);
    await page.fill('input[name="username_or_email"]', testUsername);
    await page.fill('input[name="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForURL(/^(?!.*\/login)/, { timeout: 10000 });

    // Wait for home page to fully load (ensures cookies are set)
    await page.waitForLoadState('networkidle');

    // 3. Create test note via page.evaluate (uses browser's fetch with cookies)
    const testNoteId = await page.evaluate(async () => {
      const response = await fetch('/api/notes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          title: 'E2E Test Note',
          content: '# Test Content\n\n[[Test Wikilink]]',
          folder_path: '/',
        }),
      });
      if (!response.ok) {
        throw new Error(`Failed to create note: ${response.status}`);
      }
      const note = await response.json();
      return note.id;
    });

    // 4. Navigate to note page via anchor click (SvelteKit intercepts for client-side routing)
    await page.evaluate((noteId) => {
      const anchor = document.createElement('a');
      anchor.href = `/note/${noteId}`;
      anchor.style.display = 'none';
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
    }, testNoteId);
    await page.waitForURL(/\/note\//, { timeout: 10000 });

    // Wait for editor to be ready
    await page.waitForSelector('.cm-editor', { timeout: 15000 });

    // Pass both page and testNoteId directly (no global state)
    const context: AuthContext = {
      page,
      testNoteId,
    };

    await use(context);

    // Cleanup: Notes in :memory: DB are automatically cleaned up
    // No explicit cleanup needed for in-memory test database

    // Note: User deletion requires admin privileges (/admin/users/{id})
    // Test users will accumulate. See cleanup strategy below.
  },
});

export { expect };
