let activeRefreshCount = $state(0);
let isVisible = $state(false);
let showTimer: ReturnType<typeof setTimeout> | null = null;
const SHOW_DELAY_MS = 300;

export function startSessionRestore(): void {
  activeRefreshCount += 1;
  if (activeRefreshCount !== 1) return;

  if (showTimer) {
    clearTimeout(showTimer);
  }

  showTimer = setTimeout(() => {
    showTimer = null;
    if (activeRefreshCount > 0) {
      isVisible = true;
    }
  }, SHOW_DELAY_MS);
}

export function stopSessionRestore(): void {
  activeRefreshCount = Math.max(0, activeRefreshCount - 1);
  if (activeRefreshCount > 0) return;

  if (showTimer) {
    clearTimeout(showTimer);
    showTimer = null;
  }

  isVisible = false;
}

export function isSessionRestoreActive(): boolean {
  return isVisible;
}
