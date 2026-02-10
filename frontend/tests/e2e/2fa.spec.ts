import { expect, test } from '@playwright/test';
import * as OTPAuth from 'otpauth';

test('allows a user to set up and log in with 2FA', async ({ page }) => {
  // 1. Create a new user and log in
  const username = `testuser${Date.now()}`;
  const email = `testuser-${Date.now()}@example.com`;
  const password = 'password123';

  await page.goto('/register', { waitUntil: 'networkidle' });
  await page.waitForSelector('input[name="username"]', { state: 'visible' });

  await page.fill('input[name="username"]', username);
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.fill('input[name="confirmPassword"]', password);

  // Wait for form to be interactive and submit
  await page.waitForTimeout(500);
  await page.press('input[name="confirmPassword"]', 'Enter');
  await page.waitForURL(/\/login$/, { timeout: 10000 });
  await expect(page).toHaveURL(/\/login$/);

  // Login with the new user
  await page.goto('/login');
  await page.fill('input[name="username_or_email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/$/);

  // Wait for encryption to initialize (libsodium + KEK derivation)
  await page.waitForTimeout(2000);

  // 2. Navigate to settings via sidebar button
  // Note: i18n might not be loaded, so title could be "sidebar.settings" literally
  const settingsButton = page
    .locator(
      'button[title="Settings"], button[title="Einstellungen"], ' +
        'button[title="sidebar.settings"], button:has-text("sidebar.settings")'
    )
    .first();
  await settingsButton.click();
  await page.waitForURL(/\/settings/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');

  // Wait for 2FA status API call to complete
  await page.waitForTimeout(1000);

  // Click Account tab (i18n key fallback: "settings.tabs.account")
  await page.waitForSelector(
    'button:has-text("Account"), button:has-text("settings.tabs.account")',
    { state: 'visible', timeout: 10000 }
  );
  await page.click('button:has-text("Account"), button:has-text("settings.tabs.account")');

  // Wait for loading to finish and click the setup button in settings
  // Button text: "2FA einrichten" or i18n key "settings.account.setup_twofa"
  await page.waitForSelector(
    'button:has-text("2FA einrichten"), button:has-text("Set up 2FA"), button:has-text("settings.account.setup_twofa")',
    { state: 'visible', timeout: 10000 }
  );
  await page.click(
    'button:has-text("2FA einrichten"), button:has-text("Set up 2FA"), button:has-text("settings.account.setup_twofa")'
  );

  // 3. Get the TOTP secret from the API response
  const setupResponsePromise = page.waitForResponse('**/api/2fa/setup');

  // Wait for the dialog intro step and click the setup button to start the process
  // The dialog button also says "2FA einrichten"
  await page.waitForTimeout(500); // Wait for dialog to appear
  const dialogSetupButton = page.locator('button:has-text("2FA einrichten")').last();
  await dialogSetupButton.click();

  const setupResponse = await setupResponsePromise;
  const { secret } = await setupResponse.json();

  // 4. Verify the TOTP code using otpauth library
  const totp = new OTPAuth.TOTP({
    secret: OTPAuth.Secret.fromBase32(secret),
    algorithm: 'SHA1',
    digits: 6,
    period: 30,
  });
  const code = totp.generate();

  // QR code step - click "Weiter" (Next) to proceed to verification
  await page.waitForSelector('button:has-text("Weiter")', { state: 'visible', timeout: 5000 });
  await page.click('button:has-text("Weiter")');

  // Wait for verify step - input has placeholder "000000" and maxlength="6"
  await page.waitForSelector('input[maxlength="6"]', { state: 'visible', timeout: 10000 });
  await page.fill('input[maxlength="6"]', code);

  // Click "Bestätigen" button
  await page.click('button:has-text("Bestätigen")');

  // 5. Save backup codes and complete setup
  // Wait for backup codes to appear - button says "Alle kopieren"
  await page.waitForSelector('button:has-text("Alle kopieren")', {
    state: 'visible',
    timeout: 5000,
  });
  await page.click('button:has-text("Alle kopieren")');

  // Check the confirmation checkbox
  await page.click('input[type="checkbox"]');

  // Complete setup - button says "Fertig"
  await page.click('button:has-text("Fertig")');

  // Wait for dialog to close and settings to reload
  await page.waitForTimeout(1000);

  // Verify 2FA is now enabled - check for the disable button
  // Button text: "2FA deaktivieren" or i18n key "settings.account.disable_twofa"
  await expect(
    page.locator(
      'button:has-text("2FA deaktivieren"), button:has-text("settings.account.disable_twofa")'
    )
  ).toBeVisible({ timeout: 5000 });

  // 6. Log out by clearing cookies and local storage
  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });
  await page.context().clearCookies();

  // Navigate to login page
  await page.goto('/login');
  await expect(page).toHaveURL(/\/login$/);

  // 7. Log back in with 2FA
  await page.fill('input[name="username_or_email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');

  // Wait for 2FA prompt (should show "requires_two_factor" response)
  // The login page should show the TOTP input field
  await page.waitForSelector('input[name="totp_code"]', { state: 'visible', timeout: 10000 });

  // Generate a new TOTP code (time might have changed)
  const code2 = totp.generate();
  await page.fill('input[name="totp_code"]', code2);
  await page.click('button[type="submit"]');

  // Should be logged in and redirected to home
  await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
});
