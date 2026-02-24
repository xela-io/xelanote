import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';

import { test } from '../fixtures/screenshot.fixture';

const ROUTES_TO_TEST = [
  { route: '/', name: 'Home', requiresAuth: true },
  { route: '/journal', name: 'Journal', requiresAuth: true },
  { route: '/recipes', name: 'Recipes', requiresAuth: true },
  { route: '/search', name: 'Search', requiresAuth: true },
  { route: '/settings', name: 'Settings', requiresAuth: true },
  { route: '/settings/encryption', name: 'Settings Encryption', requiresAuth: true },
  { route: '/graph', name: 'Graph', requiresAuth: true },
  { route: '/trash', name: 'Trash', requiresAuth: true },
];

const PUBLIC_ROUTES_TO_TEST = [
  { route: '/login', name: 'Login' },
  { route: '/register', name: 'Register' },
  { route: '/about', name: 'About' },
];

test.describe('Accessibility - WCAG 2.1 AA @a11y', () => {
  test.setTimeout(240000);

  test('protected pages pass axe scan', async ({ authenticatedPage }) => {
    const violations: { route: string; count: number; issues: string[] }[] = [];

    for (const { route, name: _name } of ROUTES_TO_TEST) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1500);

      const results = await new AxeBuilder({ page: authenticatedPage })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .exclude('.cm-editor') // CodeMirror has its own a11y
        .exclude('canvas') // Graph canvas is inherently non-standard
        .analyze();

      if (results.violations.length > 0) {
        violations.push({
          route,
          count: results.violations.length,
          issues: results.violations.map(
            (v) => `[${v.impact}] ${v.id}: ${v.description} (${v.nodes.length} instances)`
          ),
        });
      }
    }

    // Report all violations but only fail on serious/critical
    const critical = violations.filter((v) =>
      v.issues.some((i) => i.includes('[critical]') || i.includes('[serious]'))
    );

    if (violations.length > 0) {
      console.log('\n=== Accessibility Audit Results ===');
      for (const v of violations) {
        console.log(`\n${v.route} (${v.count} violations):`);
        v.issues.forEach((i) => console.log(`  - ${i}`));
      }
    }

    expect(
      critical.length,
      `Critical/serious a11y violations found:\n${critical.map((v) => `${v.route}: ${v.issues.join('\n')}`).join('\n\n')}`
    ).toBe(0);
  });

  test('public pages pass axe scan', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    try {
      for (const { route, name } of PUBLIC_ROUTES_TO_TEST) {
        await page.goto(`http://localhost:4173${route}`);
        await page.waitForLoadState('load');
        await page.waitForTimeout(1000);

        const results = await new AxeBuilder({ page })
          .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
          .analyze();

        const serious = results.violations.filter(
          (v) => v.impact === 'critical' || v.impact === 'serious'
        );

        if (results.violations.length > 0) {
          console.log(`\n${name} (${route}): ${results.violations.length} violations`);
          results.violations.forEach((v) => {
            console.log(`  [${v.impact}] ${v.id}: ${v.description} (${v.nodes.length})`);
          });
        }

        expect(
          serious.length,
          `Critical a11y violations on ${route}: ${serious.map((v) => `${v.id}: ${v.description}`).join(', ')}`
        ).toBe(0);
      }
    } finally {
      await context.close();
    }
  });
});

test.describe('Accessibility - Manual Checks @a11y', () => {
  test.setTimeout(120000);

  test('all images have alt text', async ({ authenticatedPage }) => {
    const routes = ['/', '/recipes', '/settings'];

    for (const route of routes) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const missingAlt = await authenticatedPage.evaluate(() => {
        const images = document.querySelectorAll('img');
        const missing: string[] = [];

        images.forEach((img) => {
          if (
            !img.hasAttribute('alt') &&
            !img.hasAttribute('role') &&
            img.getAttribute('role') !== 'presentation'
          ) {
            missing.push(img.src.substring(img.src.lastIndexOf('/') + 1));
          }
        });

        return missing;
      });

      expect(
        missingAlt,
        `Images without alt text on ${route}: ${missingAlt.join(', ')}`
      ).toHaveLength(0);
    }
  });

  test('focus indicators are visible', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/login');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    // Tab to first focusable element
    await authenticatedPage.keyboard.press('Tab');
    await authenticatedPage.waitForTimeout(200);

    const hasFocusIndicator = await authenticatedPage.evaluate(() => {
      const focused = document.activeElement;
      if (!focused || focused === document.body) return true; // No focusable elements

      const styles = getComputedStyle(focused);
      const outlineWidth = parseFloat(styles.outlineWidth);
      const outlineStyle = styles.outlineStyle;
      const boxShadow = styles.boxShadow;

      // Check for visible focus indicator (outline or box-shadow)
      const hasOutline = outlineWidth > 0 && outlineStyle !== 'none';
      const hasBoxShadow = boxShadow !== 'none';

      return hasOutline || hasBoxShadow;
    });

    expect(hasFocusIndicator, 'Focus indicators should be visible').toBe(true);
  });

  test('ARIA labels on interactive elements', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1000);

    const unlabeledButtons = await authenticatedPage.evaluate(() => {
      const buttons = document.querySelectorAll('button');
      const unlabeled: string[] = [];

      buttons.forEach((btn) => {
        const hasText = btn.textContent?.trim();
        const hasAriaLabel = btn.getAttribute('aria-label');
        const hasAriaLabelledBy = btn.getAttribute('aria-labelledby');
        const hasTitle = btn.getAttribute('title');

        // Icon-only buttons MUST have an accessible name
        if (!hasText && !hasAriaLabel && !hasAriaLabelledBy && !hasTitle) {
          const classes = btn.className ? String(btn.className).substring(0, 50) : 'no-class';
          unlabeled.push(`button.${classes}`);
        }
      });

      return unlabeled;
    });

    if (unlabeledButtons.length > 0) {
      console.log(
        `Warning: ${unlabeledButtons.length} buttons without accessible names:\n${unlabeledButtons.join('\n')}`
      );
    }

    // Allow some unlabeled but warn
    expect(
      unlabeledButtons.length,
      `Too many buttons without accessible names: ${unlabeledButtons.join(', ')}`
    ).toBeLessThan(10);
  });
});
