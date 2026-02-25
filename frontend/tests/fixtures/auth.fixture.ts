import { expect, type Page, test as base } from '@playwright/test';

import {
  createCredentials,
  type TestCredentials,
  loginViaApi,
  registerViaApi,
  spoofedClientIP,
} from '../e2e/helpers/auth';

interface AuthContext {
  page: Page;
  testNoteId: string;
  baseURL: string;
  credentials: TestCredentials;
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

    // 1. Register + Login via API (sets HttpOnly auth cookies in the browser context)
    //    Uses retry logic with exponential backoff for rate limiting (429)
    const credentials = createCredentials();
    await registerViaApi(page, credentials);
    await loginViaApi(page, credentials);

    // 2. Navigate to home (cookies already set, initAuth picks them up)
    await page.goto(`${baseURL}/`);
    await expect(page).toHaveURL(/\/$/, { timeout: 15000 });
    await page.waitForLoadState('load');
    // Wait for SvelteKit client-side auth initialization
    await page.waitForTimeout(1000);

    // 3. Create test note via Playwright API (avoids cross-origin browser fetch issues)
    const csrfCookies = await page.context().cookies(baseURL);
    const csrfToken = csrfCookies.find((c) => c.name === 'csrf_token')?.value;

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Forwarded-For': spoofedClientIP(),
    };
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    const noteResponse = await page.request.post(`${baseURL}/api/notes`, {
      headers,
      data: {
        title: 'E2E Test Note',
        content: [
          '# Project Notes',
          '',
          '## Architecture',
          'The application uses a **Go backend** with an **SQLite database**.',
          '',
          '### Key Components',
          '- REST API with Chi router',
          '- SvelteKit frontend with Svelte 5 Runes',
          '- End-to-end encryption',
          '',
          '## Links',
          '[[Architecture Overview]] | [[API Design]]',
          '',
          '```go',
          'func main() {',
          '    r := chi.NewRouter()',
          '    r.Get("/api/notes", handlers.ListNotes)',
          '}',
          '```',
          '',
          '> This is a test note for visual regression screenshots.',
        ].join('\n'),
        folder_path: '/',
      },
    });

    if (!noteResponse.ok()) {
      throw new Error(`Failed to create note: ${noteResponse.status()}`);
    }
    const note = await noteResponse.json();
    const testNoteId = note.id;

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
      baseURL,
      credentials,
    };

    await use(context);

    // Cleanup: Notes in :memory: DB are automatically cleaned up
    // No explicit cleanup needed for in-memory test database
  },
});

export { expect };
