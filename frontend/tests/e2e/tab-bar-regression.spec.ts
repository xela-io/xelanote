import type { Page } from '@playwright/test';

import { expect, test } from '../fixtures/auth.fixture';
import { spoofedClientIP } from './helpers/auth';

async function csrfToken(page: Page, baseURL: string): Promise<string | null> {
  const cookies = await page.context().cookies(baseURL);
  return cookies.find((cookie) => cookie.name === 'csrf_token')?.value ?? null;
}

async function createNote(page: Page, baseURL: string, title: string): Promise<string> {
  const csrf = await csrfToken(page, baseURL);
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrf) headers['X-CSRF-Token'] = csrf;

  const response = await page.request.post(`${baseURL}/api/notes`, {
    headers,
    data: { title, content: `content:${title}`, folder_path: '/' },
  });
  expect(response.ok(), `note creation failed: ${response.status()}`).toBeTruthy();
  const note = (await response.json()) as { id: string };
  return note.id;
}

async function navigateToNoteClientSide(page: Page, noteId: string): Promise<void> {
  await page.evaluate((id) => {
    const anchor = document.createElement('a');
    anchor.href = `/note/${id}`;
    anchor.style.display = 'none';
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  }, noteId);
  await page.waitForURL(new RegExp(`/note/${noteId}$`), { timeout: 15000 });
  await page.waitForSelector('.cm-editor', { timeout: 15000 });
}

function tabByTitle(page: Page, title: string) {
  return page.locator(`.tab-bar [role="tab"][title="${title}"]`);
}

test.describe('Tab bar regression', () => {
  test.describe.configure({ mode: 'serial' });

  test('supports close via X, close via middle click and reorder via drag handle', async ({
    authenticatedContext,
  }) => {
    const { page, baseURL } = authenticatedContext;

    const titleA = `tab-a-${Date.now()}`;
    const titleB = `tab-b-${Date.now()}`;
    const titleC = `tab-c-${Date.now()}`;
    const idA = await createNote(page, baseURL, titleA);
    const idB = await createNote(page, baseURL, titleB);
    const idC = await createNote(page, baseURL, titleC);

    await navigateToNoteClientSide(page, idA);
    await navigateToNoteClientSide(page, idB);
    await navigateToNoteClientSide(page, idC);

    await page.waitForFunction(
      () => document.querySelectorAll('.tab-bar [role="tab"]').length >= 3,
      {
        timeout: 20000,
      }
    );
    await expect(tabByTitle(page, titleA)).toBeVisible();
    await expect(tabByTitle(page, titleC)).toHaveClass(/active/);

    // Case 1: Close inactive tab via X (must keep current active note).
    const currentUrl = page.url();
    const tabA = tabByTitle(page, titleA);
    await tabA.hover();
    await tabA.locator('.close-btn').click();

    await expect(tabByTitle(page, titleA)).toHaveCount(0);
    await expect(page).toHaveURL(currentUrl);
    await expect(tabByTitle(page, titleC)).toHaveClass(/active/);

    // Case 2: Close inactive tab via middle click (must keep current active note).
    await tabByTitle(page, titleB).click({ button: 'middle' });
    await expect(tabByTitle(page, titleB)).toHaveCount(0);
    await expect(page).toHaveURL(currentUrl);
    await expect(tabByTitle(page, titleC)).toHaveClass(/active/);

    // Case 3: Reorder via drag handle.
    const sourceHandle = tabByTitle(page, titleC).locator('.drag-handle');
    const targetTab = page.locator('.tab-bar [role="tab"]').first();

    await sourceHandle.dragTo(targetTab);

    const orderedTitles = await page
      .locator('.tab-bar [role="tab"]')
      .evaluateAll((nodes) => nodes.map((n) => n.getAttribute('title') ?? ''));

    expect(orderedTitles[0]).toBe(titleC);
  });
});
