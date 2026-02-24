// Element-level scroll synchronization between editor and preview panes.
// Uses data-source-line attributes on rendered HTML elements as anchors
// to interpolate scroll positions accurately, replacing ratio-based sync.

interface Anchor {
  /** 1-based source line number */
  line: number;
  /** Offset from top of the preview container */
  offsetTop: number;
}

/**
 * Collect elements with data-source-line from the preview container.
 * Returns sorted anchors by line number.
 */
function collectAnchors(previewContainer: HTMLElement): Anchor[] {
  const elements = previewContainer.querySelectorAll<HTMLElement>('[data-source-line]');
  const anchors: Anchor[] = [];

  for (const el of elements) {
    const line = parseInt(el.dataset.sourceLine || '', 10);
    if (Number.isNaN(line) || line <= 0) continue;
    anchors.push({ line, offsetTop: el.offsetTop });
  }

  // Sort by line number (should already be in order, but be safe)
  anchors.sort((a, b) => a.line - b.line);
  return anchors;
}

/**
 * Given an editor top line, compute the preview scroll position
 * by interpolating between the two nearest anchors.
 */
function computePreviewScroll(editorTopLine: number, anchors: Anchor[]): number | null {
  if (anchors.length === 0) return null;

  // Before first anchor — scroll to top
  if (editorTopLine <= anchors[0].line) {
    return anchors[0].offsetTop * (editorTopLine / anchors[0].line);
  }

  // After last anchor — extrapolate from last two anchors or scroll to end
  const last = anchors[anchors.length - 1];
  if (editorTopLine >= last.line) {
    if (anchors.length >= 2) {
      const prev = anchors[anchors.length - 2];
      const pxPerLine = (last.offsetTop - prev.offsetTop) / (last.line - prev.line);
      return last.offsetTop + pxPerLine * (editorTopLine - last.line);
    }
    return last.offsetTop;
  }

  // Find the two anchors that bracket the editor line
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

/**
 * Given a preview scroll position, compute the editor line
 * by reverse-interpolating from anchor offsets.
 */
function computeEditorLine(previewScrollTop: number, anchors: Anchor[]): number | null {
  if (anchors.length === 0) return null;

  // Before first anchor
  if (previewScrollTop <= anchors[0].offsetTop) {
    if (anchors[0].offsetTop === 0) return anchors[0].line;
    return Math.max(1, anchors[0].line * (previewScrollTop / anchors[0].offsetTop));
  }

  // After last anchor
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

  // Interpolate between bracketing anchors
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

/**
 * Get the top visible line number from the editor's scroll position.
 */
function getEditorTopLine(editorScroller: HTMLElement, editorView: EditorViewLike): number {
  const block = editorView.lineBlockAtHeight(editorScroller.scrollTop);
  const line = editorView.state.doc.lineAt(block.from);
  return line.number;
}

/** Minimal EditorView interface for scroll sync (avoids full import). */
interface EditorViewLike {
  lineBlockAtHeight(height: number): { from: number };
  state: {
    doc: {
      lineAt(pos: number): { number: number; from: number };
      line?(lineNumber: number): { from: number };
      lines: number;
    };
  };
  scrollDOM: HTMLElement;
}

/**
 * Set up bidirectional element-level scroll synchronization.
 * Returns a cleanup function.
 */
export function setupElementScrollSync(
  editorView: EditorViewLike,
  previewScroller: HTMLElement
): () => void {
  let syncingFromEditor = false;
  let syncingFromPreview = false;
  let anchorCache: Anchor[] | null = null;
  const editorScroller = editorView.scrollDOM;

  function getAnchors(): Anchor[] {
    if (!anchorCache) {
      // The preview content is inside the scroller (first child or the scroller itself)
      const previewContent =
        previewScroller.querySelector<HTMLElement>('.markdown-preview') ?? previewScroller;
      anchorCache = collectAnchors(previewContent);
    }
    return anchorCache;
  }

  /** Invalidate anchor cache (call after re-render). */
  function invalidateAnchors() {
    anchorCache = null;
  }

  const syncPreviewFromEditor = () => {
    if (syncingFromPreview) return;
    syncingFromEditor = true;

    const topLine = getEditorTopLine(editorScroller, editorView);
    const anchors = getAnchors();
    const scrollTop = computePreviewScroll(topLine, anchors);
    if (scrollTop !== null) {
      previewScroller.scrollTop = Math.max(0, scrollTop);
    }

    requestAnimationFrame(() => {
      syncingFromEditor = false;
    });
  };

  const syncEditorFromPreview = () => {
    if (syncingFromEditor) return;
    syncingFromPreview = true;

    const anchors = getAnchors();
    const editorLine = computeEditorLine(previewScroller.scrollTop, anchors);
    if (editorLine !== null) {
      // Scroll editor to the computed line
      const lineNumber = Math.max(1, Math.min(Math.round(editorLine), editorView.state.doc.lines));
      const doc = editorView.state.doc;
      const lineStart = doc.line ? doc.line(lineNumber).from : 0;
      const lineInfo = doc.lineAt(lineStart);
      const block = (
        editorView as unknown as { lineBlockAt(pos: number): { top: number } }
      ).lineBlockAt(lineInfo.from);
      editorScroller.scrollTop = block.top;
    }

    requestAnimationFrame(() => {
      syncingFromPreview = false;
    });
  };

  // Observe DOM changes in preview to invalidate anchor cache
  const observer = new MutationObserver(invalidateAnchors);
  observer.observe(previewScroller, { childList: true, subtree: true });

  // Initial sync
  syncPreviewFromEditor();

  editorScroller.addEventListener('scroll', syncPreviewFromEditor, { passive: true });
  previewScroller.addEventListener('scroll', syncEditorFromPreview, { passive: true });

  return () => {
    observer.disconnect();
    editorScroller.removeEventListener('scroll', syncPreviewFromEditor);
    previewScroller.removeEventListener('scroll', syncEditorFromPreview);
  };
}

/**
 * Sync preview scroll position to match editor after content re-render.
 * Call inside a requestAnimationFrame.
 */
export function syncPreviewToEditorElements(
  previewScroller: HTMLElement,
  editorView: EditorViewLike
): void {
  const previewContent =
    previewScroller.querySelector<HTMLElement>('.markdown-preview') ?? previewScroller;
  const anchors = collectAnchors(previewContent);
  const topLine = getEditorTopLine(editorView.scrollDOM, editorView);
  const scrollTop = computePreviewScroll(topLine, anchors);
  if (scrollTop !== null) {
    previewScroller.scrollTop = Math.max(0, scrollTop);
  }
}
