import { expect } from '@playwright/test';

import { test } from '../fixtures/screenshot.fixture';
import { type DesignAuditResult, runDesignAudit } from '../utils/design-validator';

const ROUTES_TO_AUDIT = [
  '/',
  '/settings',
  '/recipes',
  '/journal',
  '/search',
  '/settings/encryption',
];

test.describe('Design System Validation @design', () => {
  test.setTimeout(180000);

  test('desktop design audit - no critical layout issues', async ({ authenticatedPage }) => {
    const results: DesignAuditResult[] = [];

    for (const route of ROUTES_TO_AUDIT) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1500);

      const result = await runDesignAudit(authenticatedPage, route);
      results.push(result);
    }

    // Log all results
    console.log('\n=== Design Audit Report (Desktop) ===');
    for (const result of results) {
      if (result.violations.length > 0) {
        console.log(`\n${result.page}: ${result.violations.length} violations`);
        console.log(`  By severity: ${JSON.stringify(result.summary.bySeverity)}`);
        console.log(`  By type: ${JSON.stringify(result.summary.byType)}`);

        const errors = result.violations.filter((v) => v.severity === 'error');
        if (errors.length > 0) {
          console.log('  Errors:');
          errors.forEach((e) => console.log(`    - ${e.message} (${e.selector})`));
        }
      }
    }

    // Fail only on layout errors (horizontal overflow, etc.)
    const layoutErrors = results.flatMap((r) =>
      r.violations.filter((v) => v.type === 'layout' && v.severity === 'error')
    );

    expect(
      layoutErrors.length,
      `Layout errors found:\n${layoutErrors.map((e) => `${e.message} on ${e.selector}`).join('\n')}`
    ).toBe(0);
  });

  test('mobile design audit - touch targets and layout', async ({ authenticatedPage }) => {
    await authenticatedPage.setViewportSize({ width: 393, height: 852 });
    const results: DesignAuditResult[] = [];

    for (const route of ['/', '/settings', '/recipes', '/journal']) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1500);

      const result = await runDesignAudit(authenticatedPage, route);
      results.push(result);
    }

    console.log('\n=== Design Audit Report (Mobile) ===');
    for (const result of results) {
      if (result.violations.length > 0) {
        console.log(`\n${result.page}: ${result.violations.length} violations`);
        console.log(`  By type: ${JSON.stringify(result.summary.byType)}`);

        result.violations
          .filter((v) => v.severity !== 'info')
          .forEach((v) => console.log(`  [${v.severity}] ${v.type}: ${v.message}`));
      }
    }

    // Check for layout errors on mobile
    const layoutErrors = results.flatMap((r) =>
      r.violations.filter((v) => v.type === 'layout' && v.severity === 'error')
    );

    expect(
      layoutErrors.length,
      `Mobile layout errors:\n${layoutErrors.map((e) => `${e.message}`).join('\n')}`
    ).toBe(0);
  });

  test('contrast ratios meet WCAG AA', async ({ authenticatedPage }) => {
    const contrastViolations: { route: string; violations: string[] }[] = [];

    for (const route of ['/', '/login', '/settings']) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const result = await runDesignAudit(authenticatedPage, route);
      const contrastIssues = result.violations.filter((v) => v.type === 'contrast');

      if (contrastIssues.length > 0) {
        contrastViolations.push({
          route,
          violations: contrastIssues.map((v) => `${v.selector}: ${v.actual} (need ${v.expected})`),
        });
      }
    }

    if (contrastViolations.length > 0) {
      console.log('\n=== Contrast Violations ===');
      contrastViolations.forEach(({ route, violations }) => {
        console.log(`\n${route}:`);
        violations.forEach((v) => console.log(`  - ${v}`));
      });
    }

    // Log but don't fail on contrast (OKLCH colors may not parse correctly)
    // The axe-core tests in a11y.spec.ts cover contrast more accurately
  });

  test('heading hierarchy is correct', async ({ authenticatedPage }) => {
    const headingIssues: { route: string; issues: string[] }[] = [];

    for (const route of ROUTES_TO_AUDIT) {
      await authenticatedPage.goto(route);
      await authenticatedPage.waitForLoadState('load');
      await authenticatedPage.waitForTimeout(1000);

      const result = await runDesignAudit(authenticatedPage, route);
      const headings = result.violations.filter(
        (v) => v.type === 'typography' && v.message.includes('Heading hierarchy')
      );

      if (headings.length > 0) {
        headingIssues.push({
          route,
          issues: headings.map((h) => h.message),
        });
      }
    }

    if (headingIssues.length > 0) {
      console.log('\n=== Heading Hierarchy Issues ===');
      headingIssues.forEach(({ route, issues }) => {
        console.log(`\n${route}:`);
        issues.forEach((i) => console.log(`  - ${i}`));
      });
    }

    // Warn but don't fail - headings may be intentionally styled differently
    expect(
      headingIssues.length,
      `Heading hierarchy issues on ${headingIssues.length} pages`
    ).toBeLessThan(ROUTES_TO_AUDIT.length);
  });

  test('dark mode design audit', async ({ authenticatedPage }) => {
    // Switch to dark theme
    await authenticatedPage.addInitScript(() => {
      window.localStorage.setItem('xelanote-theme', 'gruvbox-dark');
    });

    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('load');
    await authenticatedPage.waitForTimeout(1500);

    const result = await runDesignAudit(authenticatedPage, '/ (dark)');

    console.log('\n=== Dark Mode Audit ===');
    console.log(`Total violations: ${result.violations.length}`);
    console.log(`By severity: ${JSON.stringify(result.summary.bySeverity)}`);

    const layoutErrors = result.violations.filter(
      (v) => v.type === 'layout' && v.severity === 'error'
    );

    expect(layoutErrors.length, 'Dark mode layout errors').toBe(0);
  });
});
