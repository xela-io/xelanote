import { type BrowserContext, expect, type Page } from '@playwright/test';

import { test } from '../fixtures/auth.fixture';

type UiVariant = {
  locale: 'de' | 'en';
  theme: 'gruvbox-light' | 'gruvbox-dark';
  slug: string;
};

function resolveVisualVariant(): UiVariant {
  const locale = (process.env.VISUAL_LOCALE ?? 'de') as UiVariant['locale'];
  const theme = (process.env.VISUAL_THEME ?? 'gruvbox-light') as UiVariant['theme'];
  return {
    locale,
    theme,
    slug: `${locale}-${theme}`,
  };
}

const VISUAL_VARIANT = resolveVisualVariant();

async function prepPage(page: Page, variant: UiVariant) {
  await page.addInitScript(({ locale, theme }) => {
    window.localStorage.setItem('locale', locale);
    window.localStorage.setItem('xelanote-theme', theme);
  }, variant);
}

async function gotoReady(page: Page, route: string) {
  await page.goto(route, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('load');
  await page.waitForTimeout(1000);
  await page
    .waitForFunction(() => !(document.body?.innerText ?? '').includes('Laden...'), {
      timeout: 10000,
    })
    .catch(() => {});
  await page.evaluate(() => window.scrollTo(0, 0));
}

async function captureAndAssert(
  context: BrowserContext,
  route: string,
  filename: string,
  viewport: { width: number; height: number },
  variant: UiVariant
) {
  const page = await context.newPage();
  try {
    await page.setViewportSize(viewport);
    await prepPage(page, variant);
    await gotoReady(page, route);
    await expect(page).toHaveScreenshot(filename, {
      animations: 'disabled',
      maxDiffPixels: 500,
    });
  } finally {
    await page.close().catch(() => {});
  }
}

test.describe('UI visual regression', () => {
  test.setTimeout(180000);

  test('desktop stable pages', async ({ authenticatedContext }) => {
    const { page } = authenticatedContext;
    const context = page.context();
    const viewport = { width: 1440, height: 900 };

    for (const [route, filename] of [
      ['/', `desktop-home-${VISUAL_VARIANT.slug}.png`],
      ['/recipes', `desktop-recipes-${VISUAL_VARIANT.slug}.png`],
      ['/journal', `desktop-journal-${VISUAL_VARIANT.slug}.png`],
      ['/graph', `desktop-graph-${VISUAL_VARIANT.slug}.png`],
      ['/settings', `desktop-settings-${VISUAL_VARIANT.slug}.png`],
      ['/settings/encryption', `desktop-settings-encryption-${VISUAL_VARIANT.slug}.png`],
      ['/settings/migration', `desktop-settings-migration-${VISUAL_VARIANT.slug}.png`],
    ] as const) {
      await captureAndAssert(context, route, filename, viewport, VISUAL_VARIANT);
    }
  });

  test('mobile stable pages', async ({ authenticatedContext }) => {
    const { page } = authenticatedContext;
    const context = page.context();
    const viewport = { width: 393, height: 852 };

    for (const [route, filename] of [
      ['/', `mobile-home-${VISUAL_VARIANT.slug}.png`],
      ['/recipes', `mobile-recipes-${VISUAL_VARIANT.slug}.png`],
      ['/journal', `mobile-journal-${VISUAL_VARIANT.slug}.png`],
      ['/settings', `mobile-settings-${VISUAL_VARIANT.slug}.png`],
    ] as const) {
      await captureAndAssert(context, route, filename, viewport, VISUAL_VARIANT);
    }
  });
});
