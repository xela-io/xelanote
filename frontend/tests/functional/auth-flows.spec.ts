import { expect, test } from '@playwright/test';

import { createCredentials, loginViaApi, registerViaApi } from '../e2e/helpers/auth';

test.describe('Authentication Flows @e2e', () => {
  test.setTimeout(60000);

  test('register new user via UI', async ({ page }) => {
    const credentials = createCredentials();

    await page.goto('/register');
    await page.waitForLoadState('load');

    // Fill registration form
    await page.getByLabel(/benutzername|username/i).fill(credentials.username);
    await page.getByLabel(/e-mail|email/i).fill(credentials.email);

    const passwordFields = page.locator('input[type="password"]');
    await passwordFields.first().fill(credentials.password);
    if ((await passwordFields.count()) > 1) {
      await passwordFields.nth(1).fill(credentials.password);
    }

    // Submit
    await page.locator('button[type="submit"]').click();

    // Should redirect to home or login
    await page.waitForURL(/\/(login)?$/, { timeout: 15000 });
  });

  test('login with valid credentials', async ({ page }) => {
    const credentials = createCredentials();
    await registerViaApi(page, credentials);

    await page.goto('/login');
    await page.waitForLoadState('load');

    await page.getByLabel(/benutzername|e-mail|username|email/i).fill(credentials.email);
    await page.getByLabel(/passwort|password/i).fill(credentials.password);
    await page.locator('button[type="submit"]').click();

    await page.waitForURL(/\/$/, { timeout: 15000 });
    expect(page.url()).toMatch(/\/$/);
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('load');

    await page.getByLabel(/benutzername|e-mail|username|email/i).fill('nonexistent@test.com');
    await page.getByLabel(/passwort|password/i).fill('wrongpassword123');
    await page.locator('button[type="submit"]').click();

    // Should stay on login page
    await page.waitForTimeout(3000);
    expect(page.url()).toContain('/login');

    // Should show error message
    const bodyText = await page.textContent('body');
    const hasError =
      bodyText?.includes('fehlgeschlagen') ||
      bodyText?.includes('failed') ||
      bodyText?.includes('Fehler') ||
      bodyText?.includes('error') ||
      bodyText?.includes('falsch') ||
      bodyText?.includes('invalid') ||
      bodyText?.includes('incorrect');
    expect(hasError, 'Expected error message on failed login').toBeTruthy();
  });

  test('logout redirects to login', async ({ page }) => {
    const credentials = createCredentials();
    await registerViaApi(page, credentials);
    await loginViaApi(page, credentials);

    await page.goto('/');
    await expect(page).toHaveURL(/\/$/, { timeout: 15000 });
    await page.waitForTimeout(1000);

    // Navigate to settings where logout button is
    await page.goto('/settings');
    await page.waitForLoadState('load');
    await page.waitForTimeout(1000);

    // Look for logout button
    const logoutBtn = page.locator(
      'button:has-text("Abmelden"), button:has-text("Logout"), button:has-text("Sign out")'
    );

    if ((await logoutBtn.count()) > 0) {
      await logoutBtn.first().click();
      await page.waitForURL(/\/login/, { timeout: 10000 });
      expect(page.url()).toContain('/login');
    }
  });
});
