/**
 * Electron Preload Script
 *
 * Exposes a safe API to the renderer process via contextBridge.
 * This is the only way the renderer can communicate with the main process.
 */

import { contextBridge, ipcRenderer } from 'electron';

// Type for auth tokens
interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user_id: number | null;
}

// Store cleanup functions for event listeners
const cleanupFunctions = new Map<string, () => void>();

// Expose safe API to renderer
contextBridge.exposeInMainWorld('electronAPI', {
  // Token Management
  storeAuthTokens: (serverUrl: string, tokens: AuthTokens): Promise<void> => {
    return ipcRenderer.invoke('store-auth-tokens', serverUrl, tokens);
  },
  loadAuthTokens: (serverUrl: string): Promise<AuthTokens | null> => {
    return ipcRenderer.invoke('load-auth-tokens', serverUrl);
  },
  deleteAuthTokens: (serverUrl: string): Promise<void> => {
    return ipcRenderer.invoke('delete-auth-tokens', serverUrl);
  },

  // KEK Management
  storeKek: (kek: number[]): Promise<void> => {
    return ipcRenderer.invoke('store-kek', kek);
  },
  getKek: (): Promise<number[] | null> => {
    return ipcRenderer.invoke('get-kek');
  },
  lockKek: (): Promise<void> => {
    return ipcRenderer.invoke('lock-kek');
  },
  isKekLocked: (): Promise<boolean> => {
    return ipcRenderer.invoke('is-kek-locked');
  },

  // Window Controls
  minimize: (): Promise<void> => {
    return ipcRenderer.invoke('window-minimize');
  },
  maximize: (): Promise<void> => {
    return ipcRenderer.invoke('window-maximize');
  },
  toggleMaximize: (): Promise<void> => {
    return ipcRenderer.invoke('window-toggle-maximize');
  },
  close: (): Promise<void> => {
    return ipcRenderer.invoke('window-close');
  },
  isMaximized: (): Promise<boolean> => {
    return ipcRenderer.invoke('window-is-maximized');
  },
  onMaximizeChange: (callback: (maximized: boolean) => void): (() => void) => {
    // Create unique ID for this listener
    const listenerId = `maximize-change-${Date.now()}-${Math.random()}`;

    // Handler for the event
    const handler = (_event: Electron.IpcRendererEvent, maximized: boolean) => {
      callback(maximized);
    };

    // Register listener
    ipcRenderer.on('window-maximize-change', handler);

    // Store cleanup function
    const cleanup = () => {
      ipcRenderer.removeListener('window-maximize-change', handler);
      cleanupFunctions.delete(listenerId);
    };
    cleanupFunctions.set(listenerId, cleanup);

    return cleanup;
  },
});

// Log preload execution for debugging
console.log('[Preload] electronAPI exposed to renderer');
