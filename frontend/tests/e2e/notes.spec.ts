import { expect, test } from '@playwright/test';

// Helper function to register and login a new user (based on working 2fa.spec.ts pattern)
async function registerAndLogin(page: import('@playwright/test').Page) {
  const username = `testuser${Date.now()}`;
  const email = `testuser-${Date.now()}@example.com`;
  const password = 'password123';

  // Register - use exact same pattern as 2fa.spec.ts
  await page.goto('/register', { waitUntil: 'networkidle' });
  await page.waitForSelector('input[name="username"]', { state: 'visible' });

  await page.fill('input[name="username"]', username);
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.fill('input[name="confirmPassword"]', password);

  // Wait for form to be interactive and submit
  await page.waitForTimeout(500);

  // Click the submit button directly
  await page.click('button[type="submit"]');

  // Wait for redirect to login page
  await page.waitForURL(/\/login$/, { timeout: 15000 });
  await expect(page).toHaveURL(/\/login$/);

  // Login
  await page.goto('/login');
  await page.fill('input[name="username_or_email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

  // Wait for encryption to initialize (libsodium + KEK derivation)
  await page.waitForTimeout(2000);

  return { username, email, password };
}

// Helper to unlock encryption if modal appears
async function unlockEncryptionIfNeeded(page: import('@playwright/test').Page, password: string) {
  // Wait a bit for modal to appear
  await page.waitForTimeout(500);

  // Check if encryption unlock modal is visible (contains "Encryption Locked" text)
  const modal = page.locator('text=Encryption Locked');
  const isModalVisible = await modal.isVisible({ timeout: 2000 }).catch(() => false);

  if (isModalVisible) {
    console.log('Encryption modal detected, unlocking...');
    // Enter password in the modal's password field
    const passwordInput = page.locator('input[type="password"]');
    await passwordInput.fill(password);
    // Click the Unlock button
    await page.locator('button:has-text("Unlock")').click();
    // Wait for modal to close
    await page.waitForTimeout(1000);
  }
}

// Helper to create a note via the main page button (handles native prompt dialog)
async function createNote(page: import('@playwright/test').Page, title: string, password: string) {
  // Set up dialog handler BEFORE clicking the button
  // The handler auto-accepts the dialog with the provided title
  page.once('dialog', (dialog) => {
    dialog.accept(title);
  });

  // Click the main "Create new note" button on home page
  // Select the primary button (bg-primary class) which is the create note button
  const createButton = page.locator('button.bg-primary').first();
  await createButton.click();

  // Wait a moment for either redirect or encryption modal
  await page.waitForTimeout(500);

  // If encryption modal appears, unlock it
  await unlockEncryptionIfNeeded(page, password);

  // If we're still on home page, the dialog may have failed - click button again
  if (page.url().endsWith('/')) {
    // Set up dialog handler again
    page.once('dialog', (dialog) => {
      dialog.accept(title);
    });
    await createButton.click();
    await page.waitForTimeout(500);
  }

  // Should redirect to the note editor
  await expect(page).toHaveURL(/\/note\//, { timeout: 10000 });
}

test.describe('Notes CRUD', () => {
  test('creates a new note and verifies it appears', async ({ page }) => {
    const { password } = await registerAndLogin(page);

    // Wait for main page to fully load
    await page.waitForLoadState('networkidle');

    const noteTitle = `Test Note ${Date.now()}`;
    await createNote(page, noteTitle, password);

    // Note title should be visible somewhere (in editor header or page)
    // Use .first() to avoid strict mode violation when title appears multiple times
    await expect(page.getByText(noteTitle).first()).toBeVisible({ timeout: 5000 });
  });

  test('edits note content and auto-saves', async ({ page }) => {
    const { password } = await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const noteTitle = `Edit Test ${Date.now()}`;
    await createNote(page, noteTitle, password);

    // Wait for editor to be ready (CodeMirror)
    await page.waitForSelector('.cm-editor', { state: 'visible', timeout: 5000 });

    // Type some content in the editor
    const testContent = 'Hello, this is test content!';
    await page.locator('.cm-content').click();
    await page.keyboard.type(testContent);

    // Wait for auto-save indicator
    await page.waitForTimeout(3000); // Give time for auto-save

    // Navigate back to the note (instead of reload which may lose session)
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Handle encryption modal if needed
    await unlockEncryptionIfNeeded(page, password);

    // Click on the note in the recent list to reopen it
    await page.locator(`text=${noteTitle}`).first().click();
    await page.waitForURL(/\/note\//, { timeout: 5000 });

    await page.waitForSelector('.cm-editor', { state: 'visible', timeout: 10000 });

    // Verify content is still there
    await expect(page.locator('.cm-content')).toContainText('Hello');
  });

  test('deletes a note', async ({ page }) => {
    const { password } = await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const noteTitle = `Delete Test ${Date.now()}`;
    await createNote(page, noteTitle, password);

    // Wait for editor to load
    await page.waitForSelector('.cm-editor', { state: 'visible', timeout: 5000 });

    // Set up dialog handler BEFORE clicking delete (native confirm dialog)
    page.once('dialog', (dialog) => {
      dialog.accept(); // Confirm the deletion
    });

    // Click the delete button (title="Löschen" or "Delete")
    const deleteButton = page.locator('button[title="Löschen"], button[title="Delete"]');
    await deleteButton.click();

    // Should redirect to home page after deletion
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
  });

  test('note appears in recent notes list', async ({ page }) => {
    const { password } = await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const noteTitle = `Recent Test ${Date.now()}`;
    await createNote(page, noteTitle, password);

    // Wait for editor
    await page.waitForSelector('.cm-editor', { state: 'visible', timeout: 5000 });

    // Go back to home
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // The note should appear somewhere on the page (sidebar or recent notes)
    // Use .first() to avoid strict mode violation when title appears multiple times
    await expect(page.getByText(noteTitle).first()).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Notes Search', () => {
  test('finds note via quick search', async ({ page }) => {
    const { password } = await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    // Create a note with unique content
    const uniqueId = Date.now();
    const noteTitle = `Searchable ${uniqueId}`;
    await createNote(page, noteTitle, password);

    // Wait for editor and add content
    await page.waitForSelector('.cm-editor', { state: 'visible', timeout: 5000 });
    await page.locator('.cm-content').click();
    await page.keyboard.type(`This note contains unique keyword ${uniqueId}`);

    // Wait for save
    await page.waitForTimeout(2000);

    // Go home
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Open quick search (Ctrl+P)
    await page.keyboard.press('Control+p');

    // Wait for quick search input
    await page.waitForSelector('input[type="text"]', { state: 'visible', timeout: 5000 });

    // Type search query
    await page.keyboard.type(noteTitle.substring(0, 10));

    // Wait for results
    await page.waitForTimeout(500);

    // Note should appear in results
    // Use .first() to avoid strict mode violation when title appears multiple times
    await expect(page.getByText(noteTitle).first()).toBeVisible({ timeout: 5000 });
  });
});
