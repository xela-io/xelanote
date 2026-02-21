// Scroll synchronization between editor and preview panes in split mode

function getScrollRatio(element: HTMLElement): number {
  const maxScroll = element.scrollHeight - element.clientHeight;
  if (maxScroll <= 0) return 0;
  return Math.max(0, Math.min(1, element.scrollTop / maxScroll));
}

function setScrollByRatio(element: HTMLElement, ratio: number) {
  const maxScroll = element.scrollHeight - element.clientHeight;
  if (maxScroll <= 0) {
    element.scrollTop = 0;
    return;
  }
  element.scrollTop = maxScroll * Math.max(0, Math.min(1, ratio));
}

/**
 * Set up bidirectional scroll synchronization between editor and preview.
 * Returns a cleanup function that removes event listeners.
 */
export function setupScrollSync(
  editorScroller: HTMLElement,
  previewScroller: HTMLElement
): () => void {
  let syncingFromEditor = false;
  let syncingFromPreview = false;

  const syncPreviewFromEditor = () => {
    if (syncingFromPreview) return;
    syncingFromEditor = true;
    setScrollByRatio(previewScroller, getScrollRatio(editorScroller));
    requestAnimationFrame(() => {
      syncingFromEditor = false;
    });
  };

  const syncEditorFromPreview = () => {
    if (syncingFromEditor) return;
    syncingFromPreview = true;
    setScrollByRatio(editorScroller, getScrollRatio(previewScroller));
    requestAnimationFrame(() => {
      syncingFromPreview = false;
    });
  };

  // Keep preview position aligned when entering split mode.
  syncPreviewFromEditor();

  editorScroller.addEventListener('scroll', syncPreviewFromEditor, { passive: true });
  previewScroller.addEventListener('scroll', syncEditorFromPreview, { passive: true });
  return () => {
    editorScroller.removeEventListener('scroll', syncPreviewFromEditor);
    previewScroller.removeEventListener('scroll', syncEditorFromPreview);
  };
}

/**
 * Sync preview scroll position to match the editor after content re-render.
 * Call inside a requestAnimationFrame for proper timing.
 */
export function syncPreviewToEditor(
  previewScroller: HTMLElement,
  editorScroller: HTMLElement
): void {
  setScrollByRatio(previewScroller, getScrollRatio(editorScroller));
}
