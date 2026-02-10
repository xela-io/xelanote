import { expect, test } from '../fixtures/auth.fixture';

test.describe('Code Splitting', () => {
  test('Login page should NOT load CodeMirror', async ({ page, baseURL }) => {
    const requests: string[] = [];
    page.on('request', (request) => requests.push(request.url()));

    // Use baseURL explicitly (required by fixture)
    await page.goto(`${baseURL}/login`);
    await page.waitForLoadState('networkidle');

    const codemirrorRequests = requests.filter((url) => url.includes('codemirror'));
    expect(codemirrorRequests).toHaveLength(0);
  });

  test('Note page SHOULD load CodeMirror dynamically', async ({ authenticatedContext }) => {
    const { page } = authenticatedContext;

    // Fixture already navigated to note page and waited for .cm-editor
    // Verify editor is visible
    const editor = page.locator('.cm-editor');
    await expect(editor).toBeVisible({ timeout: 5000 });

    // Verify we're on the note page
    expect(page.url()).toContain('/note/');
  });

  test('Editor features work after dynamic import', async ({ authenticatedContext }) => {
    const { page } = authenticatedContext;

    // Fixture already navigated to note page with editor ready
    const editor = page.locator('.cm-editor');
    await expect(editor).toBeVisible({ timeout: 5000 });

    // Test basic typing
    await editor.click();
    await page.keyboard.type('Test content');

    // Verify wikilink syntax highlighting
    await page.keyboard.type(' [[New Link]]');

    // Wait for the wikilink element we just typed to appear
    const wikilink = page.locator('.cm-wikilink', { hasText: 'New Link' });
    await expect(wikilink).toBeVisible({ timeout: 2000 });
  });
});
