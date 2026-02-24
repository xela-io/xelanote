/**
 * Vision Design Review — captures screenshots and sends them to Claude
 * for AI-powered design analysis via the `claude` CLI (Claude Max).
 *
 * Run:  npm run test:design:vision
 * Skip: automatically skipped if `claude` CLI is not available.
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import { expect, test } from '../fixtures/auth.fixture';
import {
  isClaudeCliAvailable,
  reviewConsistency,
  reviewPages,
  type ScreenshotEntry,
  type VisionReviewReport,
} from '../utils/vision-design-reviewer';
import { generateVisionReport } from '../utils/vision-report-generator';

// ---------------------------------------------------------------------------
// Page definitions
// ---------------------------------------------------------------------------

const DESKTOP_PAGES = [
  { route: '/', name: 'Home Dashboard' },
  { route: '/recipes', name: 'Recipes' },
  { route: '/journal', name: 'Journal' },
  { route: '/search', name: 'Search' },
  { route: '/graph', name: 'Graph' },
  { route: '/settings', name: 'Settings' },
];

const MOBILE_PAGES = [
  { route: '/', name: 'Home Dashboard' },
  { route: '/recipes', name: 'Recipes' },
  { route: '/journal', name: 'Journal' },
  { route: '/search', name: 'Search' },
  { route: '/settings', name: 'Settings' },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function capturePage(
  context: import('@playwright/test').BrowserContext,
  route: string,
  filePath: string,
  viewport: { width: number; height: number }
): Promise<void> {
  const page = await context.newPage();
  try {
    await page.setViewportSize(viewport);
    await page.goto(route, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('load');
    await page.waitForTimeout(1500);

    // Disable animations for deterministic screenshots
    await page.addStyleTag({
      content: `
        *, *::before, *::after {
          animation-duration: 0s !important;
          animation-delay: 0s !important;
          transition-duration: 0s !important;
          transition-delay: 0s !important;
        }
        .cm-cursor, .cm-cursor-primary { opacity: 0 !important; }
      `,
    });

    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(300);
    await page.screenshot({ path: filePath, animations: 'disabled' });
  } finally {
    await page.close().catch(() => {});
  }
}

// ---------------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------------

const cliAvailable = isClaudeCliAvailable();

test.describe('Vision Design Review @vision', () => {
  // Claude CLI calls are slow — give the whole suite 10 minutes
  test.setTimeout(600_000);

  test.skip(!cliAvailable, 'Skipping vision design review: claude CLI not available');

  test('AI design review of all core pages', async ({ authenticatedContext }, testInfo) => {
    const { page } = authenticatedContext;
    const context = page.context();

    // Output directories
    const outDir = path.join(testInfo.config.rootDir, 'tests', 'reports', 'vision');
    const screenshotDir = path.join(outDir, 'screenshots');
    await fs.mkdir(screenshotDir, { recursive: true });

    // --- Capture desktop screenshots ---
    const desktopScreenshots: ScreenshotEntry[] = [];

    for (const { route, name } of DESKTOP_PAGES) {
      const fileName = `desktop-${name.toLowerCase().replace(/\s+/g, '-')}.png`;
      const filePath = path.join(screenshotDir, fileName);
      await capturePage(context, route, filePath, {
        width: 1440,
        height: 900,
      });
      desktopScreenshots.push({
        path: filePath,
        page: name,
        viewport: 'Desktop (1440\u00d7900)',
      });
    }

    // --- Capture mobile screenshots ---
    const mobileScreenshots: ScreenshotEntry[] = [];

    for (const { route, name } of MOBILE_PAGES) {
      const fileName = `mobile-${name.toLowerCase().replace(/\s+/g, '-')}.png`;
      const filePath = path.join(screenshotDir, fileName);
      await capturePage(context, route, filePath, {
        width: 393,
        height: 852,
      });
      mobileScreenshots.push({
        path: filePath,
        page: `${name} (Mobile)`,
        viewport: 'Mobile (393\u00d7852)',
      });
    }

    const allScreenshots = [...desktopScreenshots, ...mobileScreenshots];

    // --- Send to Claude for design review ---
    console.log(`\nSending ${allScreenshots.length} screenshots to Claude for design review...`);

    console.log('Reviewing desktop pages...');
    const desktopReviews = reviewPages(desktopScreenshots);

    console.log('Reviewing mobile pages...');
    const mobileReviews = reviewPages(mobileScreenshots);

    const allReviews = [...desktopReviews, ...mobileReviews];

    // --- Cross-page consistency analysis ---
    console.log('Analyzing cross-page consistency...');
    let consistencyReport = null;
    try {
      consistencyReport = reviewConsistency(desktopScreenshots);
    } catch (err) {
      console.warn('Consistency review failed:', err instanceof Error ? err.message : err);
    }

    // --- Build report ---
    const avgScore =
      allReviews.reduce((sum, r) => sum + r.overallScore, 0) / (allReviews.length || 1);

    const criticalCount = allReviews.reduce(
      (sum, r) => sum + r.issues.filter((i) => i.severity === 'critical').length,
      0
    );

    const report: VisionReviewReport = {
      timestamp: new Date().toISOString(),
      pageReviews: allReviews,
      consistencyReport,
      averageScore: Math.round(avgScore * 10) / 10,
      criticalIssueCount: criticalCount,
    };

    // Save JSON
    await fs.writeFile(
      path.join(outDir, 'vision-review.json'),
      JSON.stringify(report, null, 2),
      'utf8'
    );

    // Generate HTML report
    const htmlPath = path.join(outDir, 'vision-design-review.html');
    await generateVisionReport(report, allScreenshots, htmlPath);

    // --- Console summary ---
    console.log('\n=== Vision Design Review Results ===');
    console.log(`Average Score: ${report.averageScore}/10`);
    console.log(`Critical Issues: ${report.criticalIssueCount}`);
    console.log(`Report: ${htmlPath}\n`);

    for (const review of allReviews) {
      const issues = review.issues.length ? ` (${review.issues.length} issues)` : '';
      console.log(`  ${review.page} [${review.viewport}]: ${review.overallScore}/10${issues}`);
    }

    if (consistencyReport) {
      console.log(`\n  Cross-page consistency: ${consistencyReport.overallConsistency}/10`);
    }

    // --- Assertions ---
    expect(
      report.averageScore,
      `Average design score ${report.averageScore}/10 is below minimum threshold of 4`
    ).toBeGreaterThanOrEqual(4);

    expect(
      report.criticalIssueCount,
      `Found ${report.criticalIssueCount} critical design issues:\n${allReviews
        .flatMap((r) =>
          r.issues
            .filter((i) => i.severity === 'critical')
            .map((i) => `  [${r.page}] ${i.area}: ${i.description}`)
        )
        .join('\n')}`
    ).toBe(0);
  });
});
