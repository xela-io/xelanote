export function handleSidebarEscape(
  e: KeyboardEvent,
  deps: {
    isMobile: () => boolean;
    isOpen: () => boolean;
    isQuickSwitcherOpen: () => boolean;
    close: () => void;
  }
) {
  if (
    e.key === 'Escape' &&
    deps.isMobile() &&
    deps.isOpen() &&
    !deps.isQuickSwitcherOpen()
  ) {
    deps.close();
  }
}
