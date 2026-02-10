/**
 * Window Manager
 *
 * Handles multi-window management for the desktop app.
 * Used for split panes and multiple note views.
 */

import { BrowserWindow } from 'electron';

import { setupWindowListeners } from './ipc-handlers';

// Track all open windows
const windows = new Map<string, BrowserWindow>();

/**
 * Register a window with the manager.
 *
 * @param id - Unique window ID
 * @param win - BrowserWindow instance
 */
export function registerWindow(id: string, win: BrowserWindow): void {
  windows.set(id, win);

  // Set up event forwarding
  setupWindowListeners(win);

  // Remove from registry when closed
  win.on('closed', () => {
    windows.delete(id);
  });

  console.log(`[WindowManager] Window registered: ${id}`);
}

/**
 * Get a window by ID.
 *
 * @param id - Window ID
 * @returns BrowserWindow or undefined
 */
export function getWindow(id: string): BrowserWindow | undefined {
  return windows.get(id);
}

/**
 * Get all registered windows.
 *
 * @returns Map of window IDs to BrowserWindow instances
 */
export function getAllWindows(): Map<string, BrowserWindow> {
  return new Map(windows);
}

/**
 * Close a window by ID.
 *
 * @param id - Window ID
 */
export function closeWindow(id: string): void {
  const win = windows.get(id);
  if (win && !win.isDestroyed()) {
    win.close();
  }
}

/**
 * Close all windows.
 */
export function closeAllWindows(): void {
  for (const [_id, win] of windows) {
    if (!win.isDestroyed()) {
      win.close();
    }
  }
  windows.clear();
}

/**
 * Focus a window by ID.
 *
 * @param id - Window ID
 */
export function focusWindow(id: string): void {
  const win = windows.get(id);
  if (win) {
    if (win.isMinimized()) win.restore();
    win.focus();
  }
}
