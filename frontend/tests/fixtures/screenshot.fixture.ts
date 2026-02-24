import { expect, type Page, test as base } from '@playwright/test';

import { createCredentials, loginViaApi, registerViaApi } from '../e2e/helpers/auth';

interface ScreenshotFixtures {
  authenticatedPage: Page;
  stabilizePage: (page: Page) => Promise<void>;
  captureScreenshot: (
    page: Page,
    name: string,
    options?: { fullPage?: boolean; mask?: string[] }
  ) => Promise<Buffer>;
}

async function stabilizePage(page: Page): Promise<void> {
  // Disable animations and transitions
  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        animation-duration: 0s !important;
        animation-delay: 0s !important;
        transition-duration: 0s !important;
        transition-delay: 0s !important;
        scroll-behavior: auto !important;
      }
      /* Hide blinking cursors */
      .cm-cursor, .cm-cursor-primary { opacity: 0 !important; }
    `,
  });

  // Scroll to top
  await page.evaluate(() => window.scrollTo(0, 0));

  // Wait for fonts
  await page.evaluate(() => document.fonts.ready);

  // Wait for images
  await page.evaluate(async () => {
    const images = Array.from(document.querySelectorAll('img'));
    await Promise.allSettled(
      images
        .filter((img) => !img.complete)
        .map(
          (img) =>
            new Promise<void>((resolve) => {
              img.addEventListener('load', () => resolve());
              img.addEventListener('error', () => resolve());
            })
        )
    );
  });

  // Wait for network idle (best-effort)
  await page.waitForLoadState('networkidle').catch(() => {});

  // Wait for any loading indicators to disappear
  await page
    .waitForFunction(
      () => {
        const text = document.body?.innerText ?? '';
        return !text.includes('Laden...') && !text.includes('Loading...');
      },
      { timeout: 10000 }
    )
    .catch(() => {});

  // Small settle time for rendering
  await page.waitForTimeout(300);
}

export const test = base.extend<ScreenshotFixtures>({
  authenticatedPage: async ({ browser }, use) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    const credentials = createCredentials();
    await registerViaApi(page, credentials);
    await loginViaApi(page, credentials);

    await page.goto('/');
    await expect(page).toHaveURL(/\/$/, { timeout: 15000 });
    await page.waitForLoadState('load');
    await page.waitForTimeout(1000);

    await use(page);

    await context.close();
  },

  // eslint-disable-next-line no-empty-pattern
  stabilizePage: async ({}, use) => {
    await use(stabilizePage);
  },

  // eslint-disable-next-line no-empty-pattern
  captureScreenshot: async ({}, use) => {
    const capture = async (
      page: Page,
      name: string,
      options?: { fullPage?: boolean; mask?: string[] }
    ): Promise<Buffer> => {
      await stabilizePage(page);

      // Mask dynamic elements
      const maskLocators = options?.mask?.map((selector) => page.locator(selector)) ?? [];

      const buffer = await page.screenshot({
        fullPage: options?.fullPage ?? true,
        animations: 'disabled',
        mask: maskLocators,
      });

      return buffer;
    };

    await use(capture);
  },
});

export { expect };
export { stabilizePage };
