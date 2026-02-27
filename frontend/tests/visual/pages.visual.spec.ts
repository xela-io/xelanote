import { type BrowserContext, expect, type Page } from '@playwright/test';

import { test } from '../fixtures/screenshot.fixture';
import { stabilizePage } from '../fixtures/screenshot.fixture';

type UiVariant = {
  locale: 'de' | 'en';
  theme: 'gruvbox-light' | 'gruvbox-dark';
  slug: string;
};

function resolveVariant(): UiVariant {
  const locale = (process.env.VISUAL_LOCALE ?? 'de') as UiVariant['locale'];
  const theme = (process.env.VISUAL_THEME ?? 'gruvbox-light') as UiVariant['theme'];
  return { locale, theme, slug: `${locale}-${theme}` };
}

const VARIANT = resolveVariant();

const PROTECTED_ROUTES = [
  ['/', 'home'],
  ['/recipes', 'recipes'],
  ['/journal', 'journal'],
  ['/graph', 'graph'],
  ['/search', 'search'],
  ['/due-dates', 'due-dates'],
  ['/trash', 'trash'],
  ['/settings', 'settings'],
  ['/settings/encryption', 'settings-encryption'],
  ['/settings/migration', 'settings-migration'],
] as const;

const PUBLIC_ROUTES = [
  ['/login', 'login'],
  ['/register', 'register'],
  ['/about', 'about'],
] as const;

async function prepPage(page: Page, variant: UiVariant): Promise<void> {
  await page.addInitScript(
    ({ locale, theme }) => {
      window.localStorage.setItem('locale', locale);
      window.localStorage.setItem('xelanote-theme', theme);
    },
    { locale: variant.locale, theme: variant.theme }
  );
}

async function gotoAndStabilize(page: Page, route: string): Promise<void> {
  await page.goto(route, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('load');
  await stabilizePage(page);
}

async function captureAndAssert(
  context: BrowserContext,
  route: string,
  filename: string,
  variant: UiVariant
): Promise<void> {
  const page = await context.newPage();
  try {
    await prepPage(page, variant);
    await gotoAndStabilize(page, route);
    await expect(page).toHaveScreenshot(filename, {
      fullPage: true,
      animations: 'disabled',
    });
  } finally {
    await page.close().catch(() => {});
  }
}

test.describe('Visual Regression - Protected Pages @visual', () => {
  test.setTimeout(240000);

  test('all protected pages match baseline', async ({ authenticatedPage }) => {
    const context = authenticatedPage.context();

    for (const [route, name] of PROTECTED_ROUTES) {
      await captureAndAssert(context, route, `${name}-${VARIANT.slug}.png`, VARIANT);
    }
  });
});

test.describe('Visual Regression - Public Pages @visual', () => {
  test.setTimeout(300000);

  test('all public pages match baseline', async ({ browser }) => {
    const context = await browser.newContext();

    try {
      for (const [route, name] of PUBLIC_ROUTES) {
        const page = await context.newPage();
        try {
          await prepPage(page, VARIANT);
          await gotoAndStabilize(page, route);
          await expect(page).toHaveScreenshot(`${name}-${VARIANT.slug}.png`, {
            fullPage: true,
            animations: 'disabled',
          });
        } finally {
          await page.close().catch(() => {});
        }
      }
    } finally {
      await context.close();
    }
  });
});

test.describe('Visual Regression - Theme Comparison @visual', () => {
  test.setTimeout(180000);

  const THEME_TEST_ROUTES = [
    ['/', 'home'],
    ['/settings', 'settings'],
    ['/recipes', 'recipes'],
  ] as const;

  const VARIANTS: UiVariant[] = [
    { locale: 'de', theme: 'gruvbox-light', slug: 'de-gruvbox-light' },
    { locale: 'de', theme: 'gruvbox-dark', slug: 'de-gruvbox-dark' },
    { locale: 'en', theme: 'gruvbox-light', slug: 'en-gruvbox-light' },
  ];

  test('theme variants match baselines', async ({ authenticatedPage }) => {
    const context = authenticatedPage.context();

    for (const variant of VARIANTS) {
      for (const [route, name] of THEME_TEST_ROUTES) {
        await captureAndAssert(context, route, `theme-${name}-${variant.slug}.png`, variant);
      }
    }
  });
});

test.describe('Visual Regression - Responsive @visual', () => {
  test.setTimeout(180000);

  const RESPONSIVE_ROUTES = [
    ['/', 'home'],
    ['/settings', 'settings'],
    ['/journal', 'journal'],
  ] as const;

  const VIEWPORTS = [
    { name: 'tablet-portrait', width: 768, height: 1024 },
    { name: 'tablet-landscape', width: 1024, height: 768 },
  ] as const;

  test('responsive viewports match baselines', async ({ authenticatedPage }) => {
    const context = authenticatedPage.context();

    for (const vp of VIEWPORTS) {
      for (const [route, name] of RESPONSIVE_ROUTES) {
        const page = await context.newPage();
        try {
          await page.setViewportSize({
            width: vp.width,
            height: vp.height,
          });
          await prepPage(page, VARIANT);
          await gotoAndStabilize(page, route);
          await expect(page).toHaveScreenshot(`${vp.name}-${name}-${VARIANT.slug}.png`, {
            fullPage: true,
            animations: 'disabled',
          });
        } finally {
          await page.close().catch(() => {});
        }
      }
    }
  });
});
