import fs from 'node:fs/promises';
import path from 'node:path';

import type { BrowserContext, Page } from '@playwright/test';

import { test } from '../fixtures/auth.fixture';

async function ensureDir(dir: string) {
  await fs.mkdir(dir, { recursive: true });
}

type UiVariant = {
  locale: 'de' | 'en';
  theme: 'gruvbox-light' | 'gruvbox-dark';
  suffix: string;
};

const DEFAULT_VARIANT: UiVariant = {
  locale: 'en',
  theme: 'gruvbox-light',
  suffix: 'en-gruvbox-light',
};

const REVIEW_VARIANTS: UiVariant[] = [
  DEFAULT_VARIANT,
  { locale: 'de', theme: 'gruvbox-light', suffix: 'de-gruvbox-light' },
  { locale: 'de', theme: 'gruvbox-dark', suffix: 'de-gruvbox-dark' },
];

async function waitForAppReady(page: Page) {
  await page.waitForLoadState('load');
  await page.waitForTimeout(800);

  // Wait until the splash/loading screen is gone (best-effort)
  await page
    .waitForFunction(
      () => {
        const text = document.body?.innerText ?? '';
        return !text.includes('Laden...');
      },
      { timeout: 10000 }
    )
    .catch(() => {});
}

async function applyUiVariant(page: Page, variant: UiVariant) {
  await page.addInitScript(({ locale, theme }) => {
    window.localStorage.setItem('locale', locale);
    window.localStorage.setItem('xelanote-theme', theme);
  }, variant);
}

async function captureOnNewPage(
  context: BrowserContext,
  route: string,
  filePath: string,
  viewport: { width: number; height: number },
  variant: UiVariant = DEFAULT_VARIANT
): Promise<string | null> {
  const page = await context.newPage();
  try {
    await page.setViewportSize(viewport);
    await applyUiVariant(page, variant);
    await page.goto(route, { waitUntil: 'domcontentloaded' });
    await waitForAppReady(page);
    await page.evaluate(() => window.scrollTo(0, 0));

    try {
      await page.screenshot({ path: filePath, animations: 'disabled' });
    } catch {
      await page.locator('body').screenshot({ path: filePath, animations: 'disabled' });
    }
    return null;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return `${route}: ${message}`;
  } finally {
    await page.close().catch(() => {});
  }
}

test.describe('UI consistency screenshots', () => {
  test('capture desktop core pages', async ({ authenticatedContext }, testInfo) => {
    const { page, testNoteId } = authenticatedContext;
    const context = page.context();
    const outDir = path.join(testInfo.config.rootDir, 'test-results', 'ui-review', 'desktop');
    await ensureDir(outDir);
    const failures: string[] = [];
    const viewport = { width: 1440, height: 900 };

    for (const [route, name] of [
      ['/', `home-dashboard--${DEFAULT_VARIANT.suffix}.png`],
      [`/note/${testNoteId}`, `note-editor--${DEFAULT_VARIANT.suffix}.png`],
      ['/recipes', `recipes--${DEFAULT_VARIANT.suffix}.png`],
      ['/journal', `journal--${DEFAULT_VARIANT.suffix}.png`],
      ['/graph', `graph--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings', `settings--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings/encryption', `settings-encryption--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings/migration', `settings-migration--${DEFAULT_VARIANT.suffix}.png`],
    ] as const) {
      const error = await captureOnNewPage(
        context,
        route,
        path.join(outDir, name),
        viewport,
        DEFAULT_VARIANT
      );
      if (error) failures.push(error);
    }

    // Theme/locale matrix on stable pages for design review
    for (const variant of REVIEW_VARIANTS) {
      for (const [route, prefix] of [
        ['/', 'home-dashboard'],
        ['/settings', 'settings'],
        ['/recipes', 'recipes'],
      ] as const) {
        const name = `${prefix}--${variant.suffix}.png`;
        const error = await captureOnNewPage(
          context,
          route,
          path.join(outDir, name),
          viewport,
          variant
        );
        if (error) failures.push(error);
      }
    }

    if (failures.length) {
      await fs.writeFile(
        path.join(outDir, '_capture-failures.txt'),
        failures.join('\n') + '\n',
        'utf8'
      );
    } else {
      await fs.unlink(path.join(outDir, '_capture-failures.txt')).catch(() => {});
    }
  });

  test('capture mobile core pages', async ({ authenticatedContext }, testInfo) => {
    const { page, testNoteId } = authenticatedContext;
    const context = page.context();
    const outDir = path.join(testInfo.config.rootDir, 'test-results', 'ui-review', 'mobile');
    await ensureDir(outDir);
    const failures: string[] = [];
    const viewport = { width: 393, height: 852 };

    for (const [route, name] of [
      ['/', `home-dashboard-mobile--${DEFAULT_VARIANT.suffix}.png`],
      [`/note/${testNoteId}`, `note-editor-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/recipes', `recipes-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/journal', `journal-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings', `settings-mobile--${DEFAULT_VARIANT.suffix}.png`],
    ] as const) {
      const error = await captureOnNewPage(
        context,
        route,
        path.join(outDir, name),
        viewport,
        DEFAULT_VARIANT
      );
      if (error) failures.push(error);
    }

    for (const variant of REVIEW_VARIANTS) {
      for (const [route, prefix] of [
        ['/', 'home-dashboard-mobile'],
        ['/settings', 'settings-mobile'],
      ] as const) {
        const name = `${prefix}--${variant.suffix}.png`;
        const error = await captureOnNewPage(
          context,
          route,
          path.join(outDir, name),
          viewport,
          variant
        );
        if (error) failures.push(error);
      }
    }

    if (failures.length) {
      await fs.writeFile(
        path.join(outDir, '_capture-failures.txt'),
        failures.join('\n') + '\n',
        'utf8'
      );
    } else {
      await fs.unlink(path.join(outDir, '_capture-failures.txt')).catch(() => {});
    }
  });
});
