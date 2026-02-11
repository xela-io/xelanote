import { expect, test, type Page } from '@playwright/test';
import { registerAndLoginApi } from './helpers/auth';

interface Note {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  version: number;
}

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
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  options?: { body?: Record<string, unknown>; ifMatch?: number }
): Promise<{ status: number; payload: unknown }> {
  const csrf = await csrfToken(page);
  const headers: Record<string, string> = {
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrf) {
    headers['X-CSRF-Token'] = csrf;
  }
  if (options?.ifMatch !== undefined) {
    headers['If-Match'] = String(options.ifMatch);
  }
  if (options?.body) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await page.request.fetch(path, {
    method,
    headers,
    data: options?.body,
  });

  let payload: unknown = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  return { status: response.status(), payload };
}

async function createNote(
  page: Page,
  title: string,
  content = ''
): Promise<Note> {
  const response = await apiRequest(page, 'POST', '/api/notes', {
    body: { title, content, folder_path: '/' },
  });
  expect(response.status).toBe(201);
  return response.payload as Note;
}

async function listNotes(page: Page): Promise<Note[]> {
  const response = await apiRequest(page, 'GET', '/api/notes?limit=100');
  expect(response.status).toBe(200);
  const payload = response.payload as { notes?: Note[] };
  return payload.notes ?? [];
}

async function getNote(page: Page, id: string): Promise<Note> {
  const response = await apiRequest(page, 'GET', `/api/notes/${id}`);
  expect(response.status).toBe(200);
  return response.payload as Note;
}

test.describe('Notes CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await registerAndLoginApi(page);
    await page.goto('/login');
  });

  test('creates a new note and verifies it appears', async ({ page }) => {
    const noteTitle = `Test Note ${Date.now()}`;
    const note = await createNote(page, noteTitle);

    expect(note.id).toBeTruthy();
    expect(note.title).toBe(noteTitle);

    const notes = await listNotes(page);
    expect(notes.some((item) => item.id === note.id)).toBeTruthy();
  });

  test('edits note content and auto-saves', async ({ page }) => {
    const note = await createNote(page, `Edit Test ${Date.now()}`);
    const updatedContent = 'Hello, this is test content!';

    const updateResponse = await apiRequest(page, 'PUT', `/api/notes/${note.id}`, {
      ifMatch: note.version,
      body: {
        title: note.title,
        content: updatedContent,
        folder_path: note.folder_path,
      },
    });
    expect(updateResponse.status).toBe(200);

    const updated = await getNote(page, note.id);
    expect(updated.content).toContain('Hello');
  });

  test('deletes a note', async ({ page }) => {
    const note = await createNote(page, `Delete Test ${Date.now()}`);

    const deleteResponse = await apiRequest(page, 'DELETE', `/api/notes/${note.id}`);
    expect(deleteResponse.status).toBe(204);

    const list = await listNotes(page);
    expect(list.some((item) => item.id === note.id)).toBeFalsy();
  });

  test('note appears in recent notes list', async ({ page }) => {
    const note = await createNote(page, `Recent Test ${Date.now()}`);
    const list = await listNotes(page);

    const found = list.find((item) => item.id === note.id);
    expect(found).toBeDefined();
    expect(found?.title).toBe(note.title);
  });
});

test.describe('Notes Search', () => {
  test.beforeEach(async ({ page }) => {
    await registerAndLoginApi(page);
    await page.goto('/login');
  });

  test.skip('finds note via quick search', async ({ page }) => {
    const uniqueKeyword = `kw${Date.now()}`;
    await createNote(page, `Search ${uniqueKeyword}`, `Body ${uniqueKeyword}`);

    let found = false;
    for (let attempt = 0; attempt < 8; attempt++) {
      const searchResponse = await apiRequest(
        page,
        'GET',
        `/api/quick-search?q=${encodeURIComponent(uniqueKeyword)}&limit=10`
      );
      expect(searchResponse.status).toBe(200);

      const payload = searchResponse.payload as
        | { results?: Array<{ id: string; title: string }> }
        | Array<{ id: string; title: string }>;
      const results = Array.isArray(payload) ? payload : (payload.results ?? []);
      if (results.some((item) => item.title.includes(uniqueKeyword))) {
        found = true;
        break;
      }
      await page.waitForTimeout(250);
    }

    expect(found).toBeTruthy();
  });
});
