export interface SplitResizeHandlers {
  getContainerRect: () => DOMRect | null;
  setSplitPosition: (pos: number) => void;
  setActive: (active: boolean) => void;
}

export function handleSplitResizeStart(e: PointerEvent, handlers: SplitResizeHandlers) {
  e.preventDefault();
  handlers.setActive(true);
  (e.target as HTMLElement).setPointerCapture(e.pointerId);
  document.body.style.userSelect = 'none';
  document.body.style.cursor = 'col-resize';
}

export function handleSplitResizeMove(e: PointerEvent, handlers: SplitResizeHandlers) {
  const rect = handlers.getContainerRect();
  if (!rect) return;
  const pos = ((e.clientX - rect.left) / rect.width) * 100;
  handlers.setSplitPosition(pos);
}

export function handleSplitResizeEnd(handlers: SplitResizeHandlers) {
  handlers.setActive(false);
  document.body.style.userSelect = '';
  document.body.style.cursor = '';
}

export function handleSplitResizeDblClick(handlers: SplitResizeHandlers) {
  handlers.setSplitPosition(50);
}
