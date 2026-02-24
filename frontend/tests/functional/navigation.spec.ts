import { expect } from '@playwright/test';

import { test } from '../fixtures/screenshot.fixture';
import { ErrorCollector } from '../utils/error-collector';

const PROTECTED_ROUTES = [
  '/',
  '/recipes',
  '/journal',
  '/graph',
  '/search',
  '/due-dates',
  '/trash',
  '/settings',
  '/settings/encryption',
  '/settings/migration',
];

const PUBLIC_ROUTES = ['/login', '/register', '/about'];

test.describe('Navigation & Routing @e2e', () => {
  test.setTimeout(120000);

  test('all protected routes are accessible when authenticated', async ({ authenticatedPage }) => {
    const collector = new ErrorCollector();
    collector.attach(authenticatedPage);

    for (const route of PROTECTED_ROUTES) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');

      // Should not redirect to login
      const url = authenticatedPage.url();
      expect(url).not.toContain('/login');

      // Page should have meaningful content (not blank)
      const bodyText = await authenticatedPage.textContent('body');
      expect(bodyText?.trim().length).toBeGreaterThan(0);
    }

    const summary = collector.getSummary();
    expect(
      summary.bySeverity.critical,
      `Critical errors on protected routes: ${JSON.stringify(summary.errors.filter((e) => e.severity === 'critical'))}`
    ).toBe(0);
  });

  test('public routes are accessible without auth', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    const collector = new ErrorCollector();
    collector.attach(page);

    try {
      for (const route of PUBLIC_ROUTES) {
        await page.goto(`http://localhost:4173${route}`);
        await page.waitForLoadState('load');

        const bodyText = await page.textContent('body');
        expect(bodyText?.trim().length).toBeGreaterThan(0);
      }

      expect(collector.hasErrors('critical')).toBe(false);
    } finally {
      await context.close();
    }
  });

  test('unauthenticated users are redirected to login', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    try {
      for (const route of PROTECTED_ROUTES) {
        await page.goto(`http://localhost:4173${route}`, {
          waitUntil: 'load',
        });
        await page.waitForTimeout(2000);

        const url = page.url();
        expect(url, `Route ${route} should redirect to /login`).toContain('/login');
      }
    } finally {
      await context.close();
    }
  });

  test('404 page for invalid routes', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    try {
      await page.goto('http://localhost:4173/nonexistent-page-xyz', {
        waitUntil: 'load',
      });
      await page.waitForTimeout(1000);

      // SvelteKit should show error page or redirect
      const bodyText = await page.textContent('body');
      expect(bodyText).toBeTruthy();
    } finally {
      await context.close();
    }
  });

  test('browser back/forward navigation works', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');

    await authenticatedPage.goto('/settings');
    await authenticatedPage.waitForLoadState('load');
    expect(authenticatedPage.url()).toContain('/settings');

    await authenticatedPage.goBack();
    await authenticatedPage.waitForLoadState('load');
    expect(authenticatedPage.url()).toMatch(/\/$/);

    await authenticatedPage.goForward();
    await authenticatedPage.waitForLoadState('load');
    expect(authenticatedPage.url()).toContain('/settings');
  });
});

test.describe('Error Monitoring @e2e', () => {
  test.setTimeout(180000);

  test('no critical JS errors on any page', async ({ authenticatedPage }) => {
    const collector = new ErrorCollector();
    collector.attach(authenticatedPage);

    for (const route of PROTECTED_ROUTES) {
      collector.clear();
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(2000);

      const jsExceptions = collector.getErrors().filter((e) => e.type === 'js-exception');
      expect(
        jsExceptions,
        `JS exceptions on ${route}: ${JSON.stringify(jsExceptions)}`
      ).toHaveLength(0);
    }
  });

  test('no broken internal links', async ({ authenticatedPage }) => {
    const brokenLinks: { page: string; href: string; status: number }[] = [];

    for (const route of ['/', '/settings', '/recipes']) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const links = await authenticatedPage.evaluate(() => {
        return Array.from(document.querySelectorAll('a[href]'))
          .map((a) => (a as HTMLAnchorElement).href)
          .filter(
            (href) =>
              href.startsWith(window.location.origin) &&
              !href.includes('#') &&
              !href.includes('mailto:') &&
              !href.includes('javascript:')
          );
      });

      const uniqueLinks = [...new Set(links)];

      for (const link of uniqueLinks.slice(0, 20)) {
        try {
          const response = await authenticatedPage.request.get(link);
          if (response.status() >= 400 && response.status() !== 401) {
            brokenLinks.push({
              page: route,
              href: link,
              status: response.status(),
            });
          }
        } catch {
          // Network error - might be expected for some links
        }
      }
    }

    expect(brokenLinks, `Broken links found: ${JSON.stringify(brokenLinks, null, 2)}`).toHaveLength(
      0
    );
  });
});

test.describe('Layout Integrity @e2e', () => {
  test.setTimeout(120000);

  test('no horizontal overflow on any page', async ({ authenticatedPage }) => {
    for (const route of PROTECTED_ROUTES) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const hasOverflow = await authenticatedPage.evaluate(() => {
        return document.documentElement.scrollWidth > window.innerWidth;
      });

      expect(hasOverflow, `Horizontal overflow detected on ${route}`).toBe(false);
    }
  });

  test('no horizontal overflow on mobile viewport', async ({ authenticatedPage }) => {
    await authenticatedPage.setViewportSize({ width: 393, height: 852 });

    for (const route of ['/', '/settings', '/journal', '/recipes']) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const hasOverflow = await authenticatedPage.evaluate(() => {
        return document.documentElement.scrollWidth > window.innerWidth;
      });

      expect(hasOverflow, `Mobile horizontal overflow on ${route}`).toBe(false);
    }
  });
});
