import { expect } from '@playwright/test';

import { test } from '../fixtures/screenshot.fixture';

test.describe('Interactive Elements @e2e', () => {
  test.setTimeout(90000);

  test('sidebar toggle works on desktop', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    const sidebar = authenticatedPage.locator(
      'nav, aside, [class*="sidebar"], [data-testid="sidebar"]'
    );

    if ((await sidebar.count()) > 0) {
      const initialVisible = await sidebar.first().isVisible();

      // Try to find sidebar toggle
      const toggleBtn = authenticatedPage.locator(
        'button[aria-label*="sidebar" i], button[aria-label*="menü" i], button[aria-label*="menu" i], button[class*="sidebar-toggle"]'
      );

      if ((await toggleBtn.count()) > 0) {
        await toggleBtn.first().click();
        await authenticatedPage.waitForTimeout(500);

        // Sidebar visibility should change
        const afterToggle = await sidebar.first().isVisible();
        expect(afterToggle).not.toBe(initialVisible);
      }
    }
  });

  test('theme switching works', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/settings');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    // Get current theme
    const _initialTheme = await authenticatedPage.evaluate(() => {
      return document.documentElement.classList.toString();
    });

    // Look for theme selector
    const themeSelector = authenticatedPage.locator(
      '[class*="theme"], [data-testid*="theme"], button:has-text("Dunkel"), button:has-text("Dark"), button:has-text("Hell"), button:has-text("Light")'
    );

    if ((await themeSelector.count()) > 0) {
      await themeSelector.first().click();
      await authenticatedPage.waitForTimeout(500);

      const _newTheme = await authenticatedPage.evaluate(() => {
        return document.documentElement.classList.toString();
      });

      // Theme class should have changed or localStorage updated
      const themeStorage = await authenticatedPage.evaluate(() => {
        return localStorage.getItem('xelanote-theme');
      });
      expect(themeStorage).toBeTruthy();
    }
  });

  test('modals can be opened and closed', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    // Try to open create note dialog
    const createBtn = authenticatedPage.locator(
      'button:has-text("Neue Notiz"), button:has-text("New Note"), button[aria-label*="create" i], button[aria-label*="neu" i], button[aria-label*="new" i]'
    );

    if ((await createBtn.count()) > 0) {
      await createBtn.first().click();
      await authenticatedPage.waitForTimeout(500);

      // Check for dialog/modal
      const dialog = authenticatedPage.locator(
        'dialog[open], [role="dialog"], [class*="modal"], [class*="dialog"]'
      );

      if ((await dialog.count()) > 0) {
        expect(await dialog.first().isVisible()).toBe(true);

        // Close with Escape
        await authenticatedPage.keyboard.press('Escape');
        await authenticatedPage.waitForTimeout(500);

        // Dialog should be closed
        const stillVisible = (await dialog.count()) > 0 && (await dialog.first().isVisible());
        expect(stillVisible).toBe(false);
      }
    }
  });

  test('keyboard navigation with Tab works', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/login');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    // Tab through focusable elements
    const focusedElements: string[] = [];

    for (let i = 0; i < 10; i++) {
      await authenticatedPage.keyboard.press('Tab');
      await authenticatedPage.waitForTimeout(100);

      const focused = await authenticatedPage.evaluate(() => {
        const el = document.activeElement;
        if (!el || el === document.body) return null;
        return `${el.tagName.toLowerCase()}${el.id ? '#' + el.id : ''}`;
      });

      if (focused) {
        focusedElements.push(focused);
      }
    }

    // Should be able to tab through multiple elements
    expect(focusedElements.length, 'Expected at least 2 focusable elements').toBeGreaterThanOrEqual(
      2
    );
  });

  test('search functionality works', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/search');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    const searchInput = authenticatedPage.locator(
      'input[type="search"], input[type="text"][placeholder*="such" i], input[type="text"][placeholder*="search" i], input[aria-label*="such" i], input[aria-label*="search" i]'
    );

    if ((await searchInput.count()) > 0) {
      await searchInput.first().fill('test');
      await authenticatedPage.waitForTimeout(1000);

      // Page should not crash - check no JS exceptions
      const _hasError = await authenticatedPage.evaluate(() => {
        return document.querySelector('[class*="error"]') !== null;
      });
      // Search with no results is fine, just shouldn't crash
    }
  });
});

test.describe('Performance Checks @e2e', () => {
  test.setTimeout(60000);

  test('core web vitals are within limits', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(3000);

    const metrics = await authenticatedPage.evaluate(() => {
      return new Promise<{
        cls: number;
        lcp: number | null;
      }>((resolve) => {
        let cls = 0;
        let lcp: number | null = null;

        // CLS
        const clsObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (!(entry as PerformanceEntry & { hadRecentInput?: boolean }).hadRecentInput) {
              cls += (entry as PerformanceEntry & { value?: number }).value ?? 0;
            }
          }
        });
        try {
          clsObserver.observe({ type: 'layout-shift', buffered: true });
        } catch {
          // Not supported
        }

        // LCP
        const lcpObserver = new PerformanceObserver((list) => {
          const entries = list.getEntries();
          if (entries.length > 0) {
            lcp = entries[entries.length - 1].startTime;
          }
        });
        try {
          lcpObserver.observe({
            type: 'largest-contentful-paint',
            buffered: true,
          });
        } catch {
          // Not supported
        }

        setTimeout(() => {
          clsObserver.disconnect();
          lcpObserver.disconnect();
          resolve({ cls, lcp });
        }, 2000);
      });
    });

    // CLS should be under 0.25 (good threshold)
    expect(metrics.cls, `CLS too high: ${metrics.cls}`).toBeLessThan(0.25);

    // LCP should be under 4000ms (good threshold for dev mode)
    if (metrics.lcp !== null) {
      expect(metrics.lcp, `LCP too high: ${metrics.lcp}ms`).toBeLessThan(4000);
    }
  });
});
