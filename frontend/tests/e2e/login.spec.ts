import { expect, test } from '@playwright/test';

test('redirects unauthenticated users to login', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: 'xelanote' })).toBeVisible();
  await expect(page.getByText('Melde dich an')).toBeVisible();
});
