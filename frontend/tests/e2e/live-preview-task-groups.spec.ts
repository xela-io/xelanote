import type { Page } from '@playwright/test';

import { expect, test } from '../fixtures/auth.fixture';

function spoofedClientIP(): string {
  const octet = Math.floor(Math.random() * 200) + 20;
  return `203.0.113.${octet}`;
}

async function csrfToken(page: Page, baseURL: string): Promise<string | null> {
  const cookies = await page.context().cookies(baseURL);
  return cookies.find((cookie) => cookie.name === 'csrf_token')?.value ?? null;
}

async function createNote(
  page: Page,
  baseURL: string,
  title: string,
  content: string
): Promise<string> {
  const csrf = await csrfToken(page, baseURL);
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrf) headers['X-CSRF-Token'] = csrf;

  const response = await page.request.post(`${baseURL}/api/notes`, {
    headers,
    data: { title, content, folder_path: '/' },
  });
  expect(response.ok(), `note creation failed: ${response.status()}`).toBeTruthy();
  const note = (await response.json()) as { id: string };
  return note.id;
}

function buildLongViewportDoc(): string {
  const lines: string[] = ['# Viewport Regression', ''];
  for (let i = 1; i <= 120; i++) lines.push(`- [ ] Open task ${i}`);
  lines.push('- [x] DoneViewportA');
  lines.push('- [x] DoneViewportB');
  lines.push('- [x] DoneViewportC');
  for (let i = 121; i <= 260; i++) lines.push(`- [ ] Open task ${i}`);
  return lines.join('\n');
}

test.describe('Live Preview Task Groups', () => {
  test('keeps collapse and indentation stable across viewport scroll cycles', async ({
    authenticatedContext,
  }) => {
    const { page, baseURL } = authenticatedContext;

    const noteId = await createNote(
      page,
      baseURL,
      `Live Viewport ${Date.now()}`,
      buildLongViewportDoc()
    );

    // Use client-side navigation from the already-initialized app state.
    await page.evaluate((id) => {
      const anchor = document.createElement('a');
      anchor.href = `/note/${id}`;
      anchor.style.display = 'none';
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
    }, noteId);
    await page.waitForURL(new RegExp(`/note/${noteId}$`), { timeout: 15000 });
    await page.waitForSelector('.cm-editor', { timeout: 20000 });

    const editorShell = page.locator('.editor-shell');
    const liveModeButton = page.getByRole('radio', { name: /live preview/i });
    await expect(liveModeButton).toBeVisible();
    if ((await liveModeButton.getAttribute('aria-checked')) !== 'true') {
      await liveModeButton.click();
    }
    await expect(editorShell).toHaveAttribute('data-editor-mode', 'live');

    const scroller = page.locator('.cm-editor .cm-scroller').first();
    await expect(scroller).toBeVisible();

    // Scroll in chunks until the target completed group is visible.
    const targetDoneLine = page.locator('.cm-line', { hasText: 'DoneViewportA' }).first();
    let foundTarget = false;
    for (const ratio of [0.2, 0.35, 0.5, 0.65, 0.8, 0.92]) {
      await scroller.evaluate((element, nextRatio) => {
        element.scrollTop = element.scrollHeight * nextRatio;
      }, ratio);
      await page.waitForTimeout(120);
      if ((await targetDoneLine.count()) > 0) {
        foundTarget = true;
        break;
      }
    }
    expect(foundTarget).toBe(true);
    await expect(targetDoneLine).toBeVisible({ timeout: 5000 });

    const toggle = page.locator('.cm-live-task-group-toggle').first();
    await expect(toggle).toBeVisible();
    await toggle.click();

    const summary = page.locator('.cm-live-task-group-summary').first();
    await expect(summary).toBeVisible();
    const collapseScrollTop = await scroller.evaluate((element) => element.scrollTop);

    // Leave viewport and return: collapsed state must still be rendered correctly.
    await scroller.evaluate((element) => {
      element.scrollTop = 0;
    });
    await page.waitForTimeout(120);
    await scroller.evaluate((element, top) => {
      element.scrollTop = top;
    }, collapseScrollTop);

    await expect(summary).toBeVisible({ timeout: 5000 });
    await summary.click();
    await expect(page.locator('.cm-live-task-group-summary')).toHaveCount(0);

    const firstDoneLine = targetDoneLine;
    await expect(firstDoneLine).toBeVisible({ timeout: 5000 });
    await expect(firstDoneLine).toHaveClass(/cm-live-task-group-first/);

    const marginLeftPx = await firstDoneLine.evaluate((element) =>
      Number.parseFloat(getComputedStyle(element).marginLeft)
    );
    expect(marginLeftPx).toBeGreaterThan(0);
  });
});
