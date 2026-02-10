export interface SidebarResizeHandlers {
  getIsMobile: () => boolean;
  getSidebarWidth: () => number;
  setSidebarWidth: (value: number) => void;
  setActive: (active: boolean) => void;
  setStartX: (value: number) => void;
  setStartWidth: (value: number) => void;
}

export function handleSidebarResizeStart(e: PointerEvent, handlers: SidebarResizeHandlers) {
  if (handlers.getIsMobile()) return;
  e.preventDefault();
  handlers.setActive(true);
  handlers.setStartX(e.clientX);
  handlers.setStartWidth(handlers.getSidebarWidth());
  (e.target as HTMLElement).setPointerCapture(e.pointerId);
  document.body.style.userSelect = 'none';
  document.body.style.cursor = 'col-resize';
}

export function handleSidebarResizeMove(
  e: PointerEvent,
  handlers: SidebarResizeHandlers,
  startX: number,
  startWidth: number
) {
  const delta = e.clientX - startX;
  handlers.setSidebarWidth(startWidth + delta);
}

export function handleSidebarResizeEnd(handlers: SidebarResizeHandlers) {
  handlers.setActive(false);
  document.body.style.userSelect = '';
  document.body.style.cursor = '';
}

export function handleSidebarResizeDblClick(handlers: SidebarResizeHandlers) {
  handlers.setSidebarWidth(256);
}
