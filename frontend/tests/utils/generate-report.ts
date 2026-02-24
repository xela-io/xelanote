/**
 * Post-test report generator.
 * Run after tests complete: npx tsx tests/utils/generate-report.ts
 *
 * Reads the Playwright JSON report and generates a standalone HTML report
 * with visual regression, design audit, and accessibility summaries.
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import {
  generateHtmlReport,
  type ReportData,
  type TestResult,
  type VisionReviewSummary,
} from './report-generator';

interface PlaywrightResult {
  suites: PlaywrightSuite[];
  stats: {
    startTime: string;
    duration: number;
    expected: number;
    unexpected: number;
    flaky: number;
    skipped: number;
  };
}

interface PlaywrightSuite {
  title: string;
  suites?: PlaywrightSuite[];
  specs?: PlaywrightSpec[];
}

interface PlaywrightSpec {
  title: string;
  ok: boolean;
  tests: Array<{
    status: string;
    duration: number;
    results: Array<{
      status: string;
      duration: number;
      error?: { message?: string };
    }>;
  }>;
}

function flattenSpecs(suites: PlaywrightSuite[], prefix = ''): TestResult[] {
  const results: TestResult[] = [];

  for (const suite of suites) {
    const suiteName = prefix ? `${prefix} > ${suite.title}` : suite.title;

    if (suite.specs) {
      for (const spec of suite.specs) {
        const test = spec.tests[0];
        if (!test) continue;

        const lastResult = test.results[test.results.length - 1];
        const status: TestResult['status'] =
          test.status === 'expected' ? 'passed' : test.status === 'skipped' ? 'skipped' : 'failed';

        results.push({
          name: `${suiteName} > ${spec.title}`,
          status,
          duration: lastResult?.duration ?? test.duration,
          error: lastResult?.error?.message,
        });
      }
    }

    if (suite.suites) {
      results.push(...flattenSpecs(suite.suites, suiteName));
    }
  }

  return results;
}

async function main() {
  const jsonPath = path.resolve(__dirname, '../../tests/reports/results.json');

  let rawData: PlaywrightResult;

  try {
    const content = await fs.readFile(jsonPath, 'utf8');
    rawData = JSON.parse(content);
  } catch {
    console.log('No Playwright JSON report found at', jsonPath);
    console.log('Run tests first: npx playwright test --reporter=json');
    process.exit(1);
  }

  const tests = flattenSpecs(rawData.suites);

  // Try to load vision review data if available
  let visionReview: VisionReviewSummary | undefined;
  const visionJsonPath = path.resolve(__dirname, '../../tests/reports/vision/vision-review.json');
  try {
    const visionContent = await fs.readFile(visionJsonPath, 'utf8');
    const visionData = JSON.parse(visionContent);
    visionReview = {
      averageScore: visionData.averageScore,
      criticalIssueCount: visionData.criticalIssueCount,
      pageCount: visionData.pageReviews?.length ?? 0,
      reportPath: 'vision/vision-design-review.html',
    };
    console.log(`Vision review data loaded: ${visionReview.averageScore}/10 avg score`);
  } catch {
    // Vision review not available — that's fine
  }

  const reportData: ReportData = {
    timestamp: rawData.stats.startTime || new Date().toISOString(),
    duration: rawData.stats.duration,
    tests,
    visualDiffs: [], // Populated by visual tests if run
    designIssues: [], // Populated by design tests if run
    a11yIssues: [], // Populated by a11y tests if run
    errors: [],
    visionReview,
  };

  const outputPath = path.resolve(__dirname, '../../tests/reports/test-report.html');
  await generateHtmlReport(reportData, outputPath);

  console.log(`Report generated: ${outputPath}`);
  console.log(
    `Tests: ${tests.filter((t) => t.status === 'passed').length} passed, ${tests.filter((t) => t.status === 'failed').length} failed, ${tests.filter((t) => t.status === 'skipped').length} skipped`
  );
}

main().catch(console.error);
