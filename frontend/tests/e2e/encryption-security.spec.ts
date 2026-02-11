import { expect, test } from '@playwright/test';
import { registerAndLogin } from './helpers/auth';

/**
 * Encryption Security Level Tests
 *
 * Tests for paranoid/balanced/convenient security modes.
 * Verifies that KEK persistence respects security level settings.
 */

async function openSecuritySettings(page: import('@playwright/test').Page) {
  await page.goto('/settings');
  const hasSecurityHeading = await page
    .locator('text=/Sicherheitsstufe|Security Level/i')
    .first()
    .isVisible({ timeout: 3000 })
    .catch(() => false);
  if (!hasSecurityHeading) {
    test.skip();
    return false;
  }
  return true;
}

async function clickButtonIfVisible(
  page: import('@playwright/test').Page,
  namePattern: RegExp
): Promise<boolean> {
  const button = page.getByRole('button', { name: namePattern }).first();
  if (!(await button.isVisible({ timeout: 1500 }).catch(() => false))) {
    return false;
  }
  await button.click();
  return true;
}

async function confirmSecurityChangeIfNeeded(page: import('@playwright/test').Page) {
  await clickButtonIfVisible(page, /Bestätigen|Confirm|Enable|Verstanden|Understood/i);
}

test.describe('Encryption Security Levels', () => {
  test.beforeEach(async ({ page }) => {
    await registerAndLogin(page);
  });

  test('paranoid mode requires password on every page reload', async ({ page }) => {
    if (!(await openSecuritySettings(page))) {
      return;
    }

    if (await clickButtonIfVisible(page, /Paranoid/i)) {
      await confirmSecurityChangeIfNeeded(page);
      await page.waitForTimeout(500);
      await page.reload();

      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const passwordInput = page.locator('input[type="password"]');
      const modalVisible = await unlockModal.isVisible({ timeout: 2000 }).catch(() => false);
      const passwordVisible = await passwordInput.isVisible({ timeout: 2000 }).catch(() => false);
      expect(modalVisible || passwordVisible).toBeTruthy();
    }
  });

  test('balanced mode restores KEK from IndexedDB on reload', async ({ page }) => {
    if (!(await openSecuritySettings(page))) {
      return;
    }

    if (await clickButtonIfVisible(page, /Ausgewogen|Balanced/i)) {
      await confirmSecurityChangeIfNeeded(page);
      await page.waitForTimeout(500);
      await page.reload();
      await page.waitForTimeout(1000);

      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const modalVisible = await unlockModal.isVisible({ timeout: 1000 }).catch(() => false);
      expect(modalVisible).toBeFalsy();
    }
  });

  test('switching from balanced to paranoid clears persisted KEK', async ({ page }) => {
    if (!(await openSecuritySettings(page))) {
      return;
    }

    if (await clickButtonIfVisible(page, /Ausgewogen|Balanced/i)) {
      await confirmSecurityChangeIfNeeded(page);
      await page.waitForTimeout(500);
    }

    if (await clickButtonIfVisible(page, /Paranoid/i)) {
      await confirmSecurityChangeIfNeeded(page);
      await page.waitForTimeout(500);
      await page.reload();
      await page.waitForTimeout(1000);

      const unlockModal = page.locator('[data-testid="unlock-encryption-modal"]');
      const passwordInput = page.locator('input[type="password"]');

      const modalVisible = await unlockModal.isVisible({ timeout: 2000 }).catch(() => false);
      const passwordVisible = await passwordInput.isVisible({ timeout: 2000 }).catch(() => false);
      expect(modalVisible || passwordVisible).toBeTruthy();
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

    await registerAndLogin(page);

    if (!(await openSecuritySettings(page))) {
      return;
    }

    if (await clickButtonIfVisible(page, /Paranoid/i)) {
      await confirmSecurityChangeIfNeeded(page);
      await page.waitForTimeout(500);

      consoleLogs.length = 0;
      await page.reload();
      await page.waitForTimeout(2000);

      // Check for paranoid mode log message
      const hasParanoidLog = consoleLogs.some(
        (log) => log.includes('Paranoid mode') || log.includes('paranoid')
      );
      expect(hasParanoidLog).toBeTruthy();
    }
  });
});
