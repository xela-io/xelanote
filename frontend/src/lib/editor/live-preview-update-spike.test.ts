import { EditorSelection, EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { createLivePreviewExtension, setLivePreviewProfilerSink } from './live-preview';

type ProfileSample = {
  phase: 'build' | 'tree' | 'structured';
  reason: string;
  ms: number;
};

function createLargeDoc(): string {
  const lines: string[] = ['# Live Preview Profiling', 'plain text baseline'];

  for (let i = 0; i < 320; i++) {
    lines.push(`## Section ${i}`);
    lines.push(`- [ ] Open task ${i}`);
    lines.push(`- [x] Done task ${i}`);
    lines.push(
      `Some paragraph ${i} with [Link](https://example.com/${i}) and [[Wiki${i}|W${i}]] @due(2027-02-10)`
    );

    if (i % 32 === 0) {
      lines.push('| A | B |');
      lines.push('| --- | --- |');
      lines.push(`| ${i} | ${i + 1} |`);
      lines.push('```ts');
      lines.push(`const block${i} = ${i};`);
      lines.push('```');
    }
  }

  return lines.join('\n');
}

function summarize(samples: ProfileSample[]): Record<string, { avg: number; count: number }> {
  const buckets = new Map<string, { total: number; count: number }>();
  for (const sample of samples) {
    const key = `${sample.reason}:${sample.phase}`;
    const bucket = buckets.get(key) ?? { total: 0, count: 0 };
    bucket.total += sample.ms;
    bucket.count += 1;
    buckets.set(key, bucket);
  }
  const result: Record<string, { avg: number; count: number }> = {};
  for (const [key, bucket] of buckets) {
    result[key] = { avg: bucket.total / bucket.count, count: bucket.count };
  }
  return result;
}

describe('live-preview update spike', () => {
  beforeEach(() => {
    vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    setLivePreviewProfilerSink(null);
    document.body.innerHTML = '';
  });

  it('profiles selection-only and doc-change hot paths', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const state = EditorState.create({
      doc: createLargeDoc(),
      selection: EditorSelection.cursor(0),
      extensions: [createLivePreviewExtension()],
    });
    const view = new EditorView({ state, parent });
    view.focus();

    const allSamples: ProfileSample[] = [];
    const runScenario = (run: () => void) => {
      const start = allSamples.length;
      run();
      return allSamples.slice(start);
    };

    setLivePreviewProfilerSink((sample) => {
      allSamples.push(sample);
    });

    const selectionSamples = runScenario(() => {
      const length = view.state.doc.length;
      for (let i = 0; i < 180; i++) {
        const pos = (i * 113) % Math.max(1, length);
        view.dispatch({
          selection: EditorSelection.cursor(pos),
        });
      }
    });

    const sameLineSelectionSamples = runScenario(() => {
      const anchor = view.state.doc.toString().indexOf('plain text baseline');
      for (let i = 0; i < 120; i++) {
        const pos = anchor + (i % 5);
        view.dispatch({
          selection: EditorSelection.cursor(pos),
        });
      }
    });

    const plainLineFrom = view.state.doc.toString().indexOf('plain text baseline') + 6;
    const nonStructuredSamples = runScenario(() => {
      for (let i = 0; i < 80; i++) {
        view.dispatch({ changes: { from: plainLineFrom, insert: 'z' } });
        view.dispatch({ changes: { from: plainLineFrom, to: plainLineFrom + 1 } });
      }
    });

    const structuredSamples = runScenario(() => {
      for (let i = 0; i < 60; i++) {
        view.dispatch({ changes: { from: plainLineFrom, insert: '`' } });
        view.dispatch({ changes: { from: plainLineFrom, to: plainLineFrom + 1 } });
      }
    });

    const selectionSummary = summarize(selectionSamples);
    const sameLineSelectionSummary = summarize(sameLineSelectionSamples);
    const nonStructuredSummary = summarize(nonStructuredSamples);
    const structuredSummary = summarize(structuredSamples);

    console.log('[UPDATE-SPIKE] selection', selectionSummary);
    console.log('[UPDATE-SPIKE] same-line selection', sameLineSelectionSummary);
    console.log('[UPDATE-SPIKE] non-structured docChanged', nonStructuredSummary);
    console.log('[UPDATE-SPIKE] structured docChanged', structuredSummary);

    expect(selectionSummary['selectionSet:build']?.count ?? 0).toBeGreaterThan(0);
    expect(selectionSummary['selectionSet:tree']?.count ?? 0).toBe(0);
    expect(selectionSummary['selectionSet:structured']?.count ?? 0).toBe(0);
    expect(sameLineSelectionSummary['selectionSet:build']?.count ?? 0).toBeLessThanOrEqual(1);

    expect(nonStructuredSummary['docChanged:build']?.count ?? 0).toBeGreaterThan(0);
    expect(nonStructuredSummary['docChanged:tree']?.count ?? 0).toBeGreaterThan(0);
    expect(nonStructuredSummary['docChanged:structured']?.count ?? 0).toBeGreaterThan(0);

    expect(structuredSummary['docChanged:build']?.count ?? 0).toBeGreaterThan(0);
    expect(structuredSummary['docChanged:tree']?.count ?? 0).toBeGreaterThan(0);
    expect(structuredSummary['docChanged:structured']?.count ?? 0).toBeGreaterThan(0);

    // Guardrails: intentionally relaxed thresholds to avoid flaky CI while still
    // catching major regressions in hot update paths.
    expect(selectionSummary['selectionSet:build']?.avg ?? Infinity).toBeLessThan(2.0);
    expect(nonStructuredSummary['docChanged:build']?.avg ?? Infinity).toBeLessThan(1.5);
    expect(nonStructuredSummary['docChanged:tree']?.avg ?? Infinity).toBeLessThan(0.5);
    expect(nonStructuredSummary['docChanged:structured']?.avg ?? Infinity).toBeLessThan(4.0);
    expect(structuredSummary['docChanged:build']?.avg ?? Infinity).toBeLessThan(1.5);
    expect(structuredSummary['docChanged:structured']?.avg ?? Infinity).toBeLessThan(2.0);

    view.destroy();
  });
});
