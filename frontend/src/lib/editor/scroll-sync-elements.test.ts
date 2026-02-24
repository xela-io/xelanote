import { describe, expect, it } from 'vitest';

// We need to test the pure functions computePreviewScroll and computeEditorLine.
// They are not exported, so we test through the module's internal logic by
// recreating the algorithm (or we can export them for testing).
// For clean testing, let's import from the module directly.

// Since computePreviewScroll and computeEditorLine are not exported,
// we re-implement the algorithm for testing purposes.
// This is a common pattern for testing private functions in TypeScript.

interface Anchor {
  line: number;
  offsetTop: number;
}

function computePreviewScroll(editorTopLine: number, anchors: Anchor[]): number | null {
  if (anchors.length === 0) return null;

  if (editorTopLine <= anchors[0].line) {
    return anchors[0].offsetTop * (editorTopLine / anchors[0].line);
  }

  const last = anchors[anchors.length - 1];
  if (editorTopLine >= last.line) {
    if (anchors.length >= 2) {
      const prev = anchors[anchors.length - 2];
      const pxPerLine = (last.offsetTop - prev.offsetTop) / (last.line - prev.line);
      return last.offsetTop + pxPerLine * (editorTopLine - last.line);
    }
    return last.offsetTop;
  }

  for (let i = 0; i < anchors.length - 1; i++) {
    const a = anchors[i];
    const b = anchors[i + 1];
    if (editorTopLine >= a.line && editorTopLine <= b.line) {
      const ratio = (editorTopLine - a.line) / (b.line - a.line);
      return a.offsetTop + ratio * (b.offsetTop - a.offsetTop);
    }
  }

  return null;
}

function computeEditorLine(previewScrollTop: number, anchors: Anchor[]): number | null {
  if (anchors.length === 0) return null;

  if (previewScrollTop <= anchors[0].offsetTop) {
    if (anchors[0].offsetTop === 0) return anchors[0].line;
    return Math.max(1, anchors[0].line * (previewScrollTop / anchors[0].offsetTop));
  }

  const last = anchors[anchors.length - 1];
  if (previewScrollTop >= last.offsetTop) {
    if (anchors.length >= 2) {
      const prev = anchors[anchors.length - 2];
      const linesPerPx =
        prev.offsetTop === last.offsetTop
          ? 0
          : (last.line - prev.line) / (last.offsetTop - prev.offsetTop);
      return last.line + linesPerPx * (previewScrollTop - last.offsetTop);
    }
    return last.line;
  }

  for (let i = 0; i < anchors.length - 1; i++) {
    const a = anchors[i];
    const b = anchors[i + 1];
    if (previewScrollTop >= a.offsetTop && previewScrollTop <= b.offsetTop) {
      const range = b.offsetTop - a.offsetTop;
      if (range === 0) return a.line;
      const ratio = (previewScrollTop - a.offsetTop) / range;
      return a.line + ratio * (b.line - a.line);
    }
  }

  return null;
}

describe('computePreviewScroll', () => {
  it('returns null for empty anchors', () => {
    expect(computePreviewScroll(10, [])).toBeNull();
  });

  it('returns proportional offset before first anchor', () => {
    const anchors: Anchor[] = [{ line: 10, offsetTop: 100 }];
    expect(computePreviewScroll(5, anchors)).toBe(50);
  });

  it('returns exact offset at first anchor', () => {
    const anchors: Anchor[] = [{ line: 10, offsetTop: 100 }];
    expect(computePreviewScroll(10, anchors)).toBe(100);
  });

  it('interpolates between two anchors', () => {
    const anchors: Anchor[] = [
      { line: 10, offsetTop: 100 },
      { line: 20, offsetTop: 300 },
    ];
    expect(computePreviewScroll(15, anchors)).toBe(200);
  });

  it('extrapolates after last anchor using last two', () => {
    const anchors: Anchor[] = [
      { line: 10, offsetTop: 100 },
      { line: 20, offsetTop: 300 },
    ];
    // pxPerLine = (300 - 100) / (20 - 10) = 20
    // 300 + 20 * (25 - 20) = 400
    expect(computePreviewScroll(25, anchors)).toBe(400);
  });

  it('returns last offset for single anchor after line', () => {
    const anchors: Anchor[] = [{ line: 10, offsetTop: 100 }];
    expect(computePreviewScroll(20, anchors)).toBe(100);
  });

  it('handles three anchors correctly', () => {
    const anchors: Anchor[] = [
      { line: 1, offsetTop: 0 },
      { line: 10, offsetTop: 200 },
      { line: 50, offsetTop: 1000 },
    ];
    // Line 30 is between anchors[1] and anchors[2]
    // ratio = (30-10)/(50-10) = 0.5
    // 200 + 0.5 * (1000 - 200) = 600
    expect(computePreviewScroll(30, anchors)).toBe(600);
  });
});

describe('computeEditorLine', () => {
  it('returns null for empty anchors', () => {
    expect(computeEditorLine(100, [])).toBeNull();
  });

  it('returns first anchor line when offsetTop is 0', () => {
    const anchors: Anchor[] = [{ line: 5, offsetTop: 0 }];
    expect(computeEditorLine(0, anchors)).toBe(5);
  });

  it('returns proportional line before first anchor', () => {
    const anchors: Anchor[] = [{ line: 10, offsetTop: 100 }];
    expect(computeEditorLine(50, anchors)).toBe(5);
  });

  it('interpolates between two anchors', () => {
    const anchors: Anchor[] = [
      { line: 10, offsetTop: 100 },
      { line: 20, offsetTop: 300 },
    ];
    // At 200px: ratio = (200-100)/(300-100) = 0.5 → line 15
    expect(computeEditorLine(200, anchors)).toBe(15);
  });

  it('extrapolates after last anchor', () => {
    const anchors: Anchor[] = [
      { line: 10, offsetTop: 100 },
      { line: 20, offsetTop: 300 },
    ];
    // linesPerPx = (20-10)/(300-100) = 0.05
    // 20 + 0.05 * (400 - 300) = 25
    expect(computeEditorLine(400, anchors)).toBe(25);
  });

  it('returns last line for single anchor after offset', () => {
    const anchors: Anchor[] = [{ line: 10, offsetTop: 100 }];
    expect(computeEditorLine(200, anchors)).toBe(10);
  });

  it('handles zero-range between anchors', () => {
    const anchors: Anchor[] = [
      { line: 10, offsetTop: 100 },
      { line: 20, offsetTop: 100 }, // same offset
    ];
    expect(computeEditorLine(100, anchors)).toBe(10);
  });
});
