import { describe, expect, it } from 'vitest';

import {
  benchmarkExtraction,
  calculateBugCount,
  extractRegexFeatures,
  extractTreeFeatures,
  generateLargeMarkdownDoc,
  spikeCases,
} from './live-preview-spike';

describe('live-preview spike (regex vs tree)', () => {
  it('reports fewer extraction bugs for tree parser on spike corpus', () => {
    const regexBugs = calculateBugCount(spikeCases, extractRegexFeatures);
    const treeBugs = calculateBugCount(spikeCases, extractTreeFeatures);

    expect(treeBugs).toBeLessThanOrEqual(regexBugs);
  });

  it('benchmarks extraction speed on large markdown corpus', () => {
    // Keep this fast enough for CI/pre-push hooks.
    const doc = generateLargeMarkdownDoc(2000);
    const result = benchmarkExtraction(doc, 40);

    // Output is part of the spike report and can be compared over time.
    console.log(
      `[SPIKE] regex avg=${result.regex.avgMs.toFixed(3)}ms tree avg=${result.tree.avgMs.toFixed(3)}ms`
    );
    console.log(
      `[SPIKE] regex matches=${result.regex.totalMatches} tree matches=${result.tree.totalMatches}`
    );

    expect(result.regex.avgMs).toBeGreaterThan(0);
    expect(result.tree.avgMs).toBeGreaterThan(0);
  });
});
