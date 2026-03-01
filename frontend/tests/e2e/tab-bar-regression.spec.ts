import type { Page } from '@playwright/test';

import { expect, test } from '../fixtures/auth.fixture';
import { spoofedClientIP } from './helpers/auth';

async function csrfToken(page: Page, baseURL: string): Promise<string | null> {
  const cookies = await page.context().cookies(baseURL);
  return cookies.find((cookie) => cookie.name === 'csrf_token')?.value ?? null;
}

async function apiHeaders(page: Page, baseURL: string): Promise<Record<string, string>> {
  const csrf = await csrfToken(page, baseURL);
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrf) headers['X-CSRF-Token'] = csrf;
  return headers;
}

async function enableTabsFeature(page: Page, baseURL: string): Promise<void> {
  const headers = await apiHeaders(page, baseURL);
  const response = await page.request.put(`${baseURL}/api/features/tabs`, {
    headers,
    data: { enabled: true },
  });
  expect(response.ok(), `tabs feature enable failed: ${response.status()}`).toBeTruthy();
}

async function createNote(page: Page, baseURL: string, title: string): Promise<string> {
  const headers = await apiHeaders(page, baseURL);
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

/** Wait for tabs feature to be loaded and the tab bar to appear. */
async function waitForTabBar(page: Page, minTabs = 1): Promise<void> {
  await page.waitForFunction(
    (min) => document.querySelectorAll('.tab-bar [role="tab"]').length >= min,
    minTabs,
    { timeout: 20000 }
  );
}

test.describe('Tab bar regression', () => {
  test.describe.configure({ mode: 'serial' });

  test('supports close via X, close via middle click and reorder via drag handle', async ({
    authenticatedContext,
  }) => {
    const { page, baseURL } = authenticatedContext;

    // Enable tabs feature for this user
    await enableTabsFeature(page, baseURL);

    const titleA = `tab-a-${Date.now()}`;
    const titleB = `tab-b-${Date.now()}`;
    const titleC = `tab-c-${Date.now()}`;
    const idA = await createNote(page, baseURL, titleA);
    const idB = await createNote(page, baseURL, titleB);
    const idC = await createNote(page, baseURL, titleC);

    // Reload to pick up the tabs feature flag
    await page.reload();
    await page.waitForLoadState('load');

    await navigateToNoteClientSide(page, idA);
    await navigateToNoteClientSide(page, idB);
    await navigateToNoteClientSide(page, idC);

    await waitForTabBar(page, 3);
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

  test('closing all tabs persists: tab bar gone after refresh', async ({
    authenticatedContext,
  }) => {
    const { page, baseURL } = authenticatedContext;

    await enableTabsFeature(page, baseURL);

    const titleA = `persist-a-${Date.now()}`;
    const titleB = `persist-b-${Date.now()}`;
    const idA = await createNote(page, baseURL, titleA);
    const idB = await createNote(page, baseURL, titleB);

    // Reload to pick up tabs feature
    await page.reload();
    await page.waitForLoadState('load');

    // Open two notes as tabs
    await navigateToNoteClientSide(page, idA);
    await navigateToNoteClientSide(page, idB);
    await waitForTabBar(page, 2);

    // Close tab B (active) — should navigate to A
    const tabB = tabByTitle(page, titleB);
    await tabB.hover();
    await tabB.locator('.close-btn').click();
    await expect(tabByTitle(page, titleB)).toHaveCount(0);
    await expect(page).toHaveURL(new RegExp(`/note/${idA}$`));

    // Close tab A (last remaining) — should navigate to /
    const tabA = tabByTitle(page, titleA);
    await tabA.hover();
    await tabA.locator('.close-btn').click();
    await expect(tabByTitle(page, titleA)).toHaveCount(0);
    await expect(page).toHaveURL(/\/$/);

    // Wait for persistence to complete (debounce + keepalive)
    await page.waitForTimeout(3000);

    // Refresh — tabs should NOT reappear
    await page.reload();
    await page.waitForLoadState('load');
    await page.waitForTimeout(2000);

    // Navigate to a note — should only show 1 tab (the new one), not the old ones
    await navigateToNoteClientSide(page, idA);

    // Wait for potential tab bar to appear
    await page.waitForTimeout(2000);

    const tabCount = await page.locator('.tab-bar [role="tab"]').count();
    // Should be 0 or 1 (only the freshly opened note), NOT 2 (the old tabs)
    expect(tabCount).toBeLessThanOrEqual(1);
  });

  test('tabs persist across page refresh', async ({ authenticatedContext }) => {
    const { page, baseURL } = authenticatedContext;

    await enableTabsFeature(page, baseURL);

    const titleA = `refresh-a-${Date.now()}`;
    const titleB = `refresh-b-${Date.now()}`;
    const idA = await createNote(page, baseURL, titleA);
    const idB = await createNote(page, baseURL, titleB);

    // Reload to pick up tabs feature
    await page.reload();
    await page.waitForLoadState('load');

    // Open two notes as tabs
    await navigateToNoteClientSide(page, idA);
    await navigateToNoteClientSide(page, idB);
    await waitForTabBar(page, 2);

    // Wait for persistence debounce to complete
    await page.waitForTimeout(3000);

    // Refresh the page
    await page.reload();
    await page.waitForLoadState('load');

    // Navigate to one of the notes to trigger tab bar display
    await navigateToNoteClientSide(page, idA);

    // Wait for tab bar to restore
    await waitForTabBar(page, 2);

    // Both tabs should be restored
    await expect(tabByTitle(page, titleA)).toBeVisible();
    await expect(tabByTitle(page, titleB)).toBeVisible();
  });
});
