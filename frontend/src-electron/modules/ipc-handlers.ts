/**
 * IPC Handlers Registration
 *
 * Registers all IPC handlers for communication between main and renderer processes.
 */

import { BrowserWindow, ipcMain } from 'electron';

import { kekManager } from './kek-manager';
import { deleteTokens, loadTokens, storeTokens } from './secure-storage';

// Type for auth tokens
interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user_id: number | null;
}

function validateServerUrl(serverUrl: string): void {
  if (typeof serverUrl !== 'string' || serverUrl.length === 0 || serverUrl.length > 2048) {
    throw new Error('invalid server URL');
  }
  const parsed = new URL(serverUrl);
  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('unsupported server URL protocol');
  }
}

function validateAuthTokens(tokens: AuthTokens): void {
  if (!tokens || typeof tokens !== 'object') {
    throw new Error('invalid tokens payload');
  }
  if (typeof tokens.access_token !== 'string' || tokens.access_token.length === 0) {
    throw new Error('invalid access token');
  }
  if (typeof tokens.refresh_token !== 'string' || tokens.refresh_token.length === 0) {
    throw new Error('invalid refresh token');
  }
  if (tokens.access_token.length > 8192 || tokens.refresh_token.length > 8192) {
    throw new Error('token payload too large');
  }
}

/**
 * Register all IPC handlers.
 * Called once during app initialization.
 */
export function registerIpcHandlers(): void {
  // Token Management
  ipcMain.handle(
    'store-auth-tokens',
    async (_event, serverUrl: string, tokens: AuthTokens): Promise<void> => {
      validateServerUrl(serverUrl);
      validateAuthTokens(tokens);
      await storeTokens(serverUrl, tokens);
    }
  );

  ipcMain.handle(
    'load-auth-tokens',
    async (_event, serverUrl: string): Promise<AuthTokens | null> => {
      validateServerUrl(serverUrl);
      return loadTokens(serverUrl);
    }
  );

  ipcMain.handle('delete-auth-tokens', async (_event, serverUrl: string): Promise<void> => {
    validateServerUrl(serverUrl);
    await deleteTokens(serverUrl);
  });

  // KEK Management
  ipcMain.handle('store-kek', async (_event, kek: number[]): Promise<void> => {
    if (!Array.isArray(kek) || kek.length === 0 || kek.length > 128) {
      throw new Error('invalid KEK');
    }
    kekManager.store(new Uint8Array(kek));
  });

  ipcMain.handle('get-kek', async (): Promise<number[] | null> => {
    const kek = kekManager.get();
    return kek ? Array.from(kek) : null;
  });

  ipcMain.handle('lock-kek', async (): Promise<void> => {
    kekManager.lock();
  });

  ipcMain.handle('is-kek-locked', async (): Promise<boolean> => {
    return kekManager.isLocked();
  });

  // Window Controls
  ipcMain.handle('window-minimize', async (event): Promise<void> => {
    const win = BrowserWindow.fromWebContents(event.sender);
    win?.minimize();
  });

  ipcMain.handle('window-maximize', async (event): Promise<void> => {
    const win = BrowserWindow.fromWebContents(event.sender);
    win?.maximize();
  });

  ipcMain.handle('window-toggle-maximize', async (event): Promise<void> => {
    const win = BrowserWindow.fromWebContents(event.sender);
    if (win) {
      if (win.isMaximized()) {
        win.unmaximize();
      } else {
        win.maximize();
      }
    }
  });

  ipcMain.handle('window-close', async (event): Promise<void> => {
    const win = BrowserWindow.fromWebContents(event.sender);
    win?.close();
  });

  ipcMain.handle('window-is-maximized', async (event): Promise<boolean> => {
    const win = BrowserWindow.fromWebContents(event.sender);
    return win?.isMaximized() ?? false;
  });

  // Set up maximize/unmaximize event forwarding
  // This is called when any window is created
  BrowserWindow.getAllWindows().forEach(setupWindowListeners);

  // Also listen for new windows
  const _originalCreateWindow = BrowserWindow.prototype.constructor;
  // Note: Window listeners are set up when windows are created via main-window.ts
}

/**
 * Set up event listeners for a window to forward maximize state changes.
 */
export function setupWindowListeners(win: BrowserWindow): void {
  const sendMaximizeState = () => {
    win.webContents.send('window-maximize-change', win.isMaximized());
  };

  win.on('maximize', sendMaximizeState);
  win.on('unmaximize', sendMaximizeState);
  win.on('resize', sendMaximizeState);
}
