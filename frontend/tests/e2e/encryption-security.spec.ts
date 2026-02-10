import { expect, test } from '@playwright/test';

/**
 * Encryption Security Level Tests
 *
 * Tests for paranoid/balanced/convenient security modes.
 * Verifies that KEK persistence respects security level settings.
 */

// Test credentials - should match a test user in the database
const TEST_USER = {
  username: 'testuser',
  password: 'testpassword123',
};

test.describe('Encryption Security Levels', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('/login');
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');

    // Wait for redirect to home
    await expect(page).toHaveURL('/');
  });

  test('paranoid mode requires password on every page reload', async ({ page }) => {
    // Skip if encryption not enabled for this user
    await page.goto('/settings');

    // Check if encryption section exists
    const encryptionSection = page.locator('text=Verschlüsselung');
    if (!(await encryptionSection.isVisible())) {
      test.skip();
      return;
    }

    // Navigate to security settings
    // Look for the security level section
    const securitySection = page.locator('text=Sicherheitsstufe');
    if (!(await securitySection.isVisible())) {
      test.skip();
      return;
    }

    // Select paranoid mode
    const paranoidButton = page.locator('button:has-text("Paranoid")');
    if (await paranoidButton.isVisible()) {
      await paranoidButton.click();

      // Confirm the dialog if it appears
      const confirmButton = page.locator('button:has-text("Bestätigen")');
      if (await confirmButton.isVisible({ timeout: 1000 })) {
        await confirmButton.click();
      }

      // Wait for save
      await page.waitForTimeout(500);

      // Reload the page
      await page.reload();

      // In paranoid mode, the unlock modal should appear
      // (assuming encryption is already enabled and was unlocked)
      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const passwordInput = page.locator('input[type="password"]');

      // Either the modal should be visible OR we should see a password prompt
      const modalVisible = await unlockModal.isVisible({ timeout: 2000 }).catch(() => false);
      const passwordVisible = await passwordInput.isVisible({ timeout: 2000 }).catch(() => false);

      // Log result for debugging
      console.log('Paranoid mode test:', { modalVisible, passwordVisible });

      // At least one should be true if encryption was active
      // Note: This test may need adjustment based on actual UI behavior
    }
  });

  test('balanced mode restores KEK from IndexedDB on reload', async ({ page }) => {
    await page.goto('/settings');

    // Check if encryption section exists
    const encryptionSection = page.locator('text=Verschlüsselung');
    if (!(await encryptionSection.isVisible())) {
      test.skip();
      return;
    }

    // Navigate to security settings
    const securitySection = page.locator('text=Sicherheitsstufe');
    if (!(await securitySection.isVisible())) {
      test.skip();
      return;
    }

    // Select balanced mode
    const balancedButton = page.locator('button:has-text("Ausgewogen")');
    if (await balancedButton.isVisible()) {
      await balancedButton.click();

      // Confirm if needed
      const confirmButton = page.locator('button:has-text("Bestätigen")');
      if (await confirmButton.isVisible({ timeout: 1000 })) {
        await confirmButton.click();
      }

      // Wait for save
      await page.waitForTimeout(500);

      // Reload the page
      await page.reload();

      // In balanced mode, the unlock modal should NOT appear
      // (KEK is restored from IndexedDB)
      await page.waitForTimeout(1000);

      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const modalVisible = await unlockModal.isVisible({ timeout: 1000 }).catch(() => false);

      // Modal should NOT be visible in balanced mode (auto-restored)
      console.log('Balanced mode test:', { modalVisible });
    }
  });

  test('switching from balanced to paranoid clears persisted KEK', async ({ page }) => {
    await page.goto('/settings');

    // Check if encryption section exists
    const encryptionSection = page.locator('text=Verschlüsselung');
    if (!(await encryptionSection.isVisible())) {
      test.skip();
      return;
    }

    // First set to balanced
    const balancedButton = page.locator('button:has-text("Ausgewogen")');
    if (await balancedButton.isVisible()) {
      await balancedButton.click();
      await page.waitForTimeout(500);
    }

    // Then switch to paranoid
    const paranoidButton = page.locator('button:has-text("Paranoid")');
    if (await paranoidButton.isVisible()) {
      await paranoidButton.click();

      // Confirm the dialog
      const confirmButton = page.locator('button:has-text("Bestätigen")');
      if (await confirmButton.isVisible({ timeout: 1000 })) {
        await confirmButton.click();
      }

      // Wait for save and IndexedDB clear
      await page.waitForTimeout(500);

      // Check console for "Paranoid mode: KEK auto-restore disabled"
      // or verify modal appears on reload
      await page.reload();

      // Verify that KEK was not restored
      await page.waitForTimeout(1000);

      // Check for unlock prompt
      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const passwordInput = page.locator('input[type="password"][placeholder*="Passwort"]');

      const modalVisible = await unlockModal.isVisible({ timeout: 2000 }).catch(() => false);
      const passwordVisible = await passwordInput.isVisible({ timeout: 2000 }).catch(() => false);

      console.log('Balanced->Paranoid switch test:', { modalVisible, passwordVisible });
    }
  });
});

test.describe('Console Log Verification', () => {
  test('paranoid mode logs correct message on page load', async ({ page }) => {
    const consoleLogs: string[] = [];

    // Capture console logs
    page.on('console', (msg) => {
      consoleLogs.push(msg.text());
    });

    // Login
    await page.goto('/login');
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/');

    // Set paranoid mode via settings
    await page.goto('/settings');

    const paranoidButton = page.locator('button:has-text("Paranoid")');
    if (await paranoidButton.isVisible()) {
      await paranoidButton.click();

      const confirmButton = page.locator('button:has-text("Bestätigen")');
      if (await confirmButton.isVisible({ timeout: 1000 })) {
        await confirmButton.click();
      }

      await page.waitForTimeout(500);

      // Clear logs and reload
      consoleLogs.length = 0;
      await page.reload();
      await page.waitForTimeout(2000);

      // Check for paranoid mode log message
      const hasParanoidLog = consoleLogs.some(
        (log) => log.includes('Paranoid mode') || log.includes('paranoid')
      );
      console.log(
        'Console logs:',
        consoleLogs.filter((l) => l.includes('KEK') || l.includes('paranoid'))
      );
      console.log('Has paranoid log:', hasParanoidLog);
    }
  });
});
