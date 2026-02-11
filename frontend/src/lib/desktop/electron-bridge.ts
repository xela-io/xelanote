/**
 * Electron Bridge Implementation
 *
 * Uses Electron's contextBridge API exposed via preload script.
 */

import type { AuthTokens, DesktopBridge } from './interface';

// Type definition for the exposed Electron API
interface ElectronAPI {
  // Token Management
  storeAuthTokens(serverUrl: string, tokens: AuthTokens): Promise<void>;
  loadAuthTokens(serverUrl: string): Promise<AuthTokens | null>;
  deleteAuthTokens(serverUrl: string): Promise<void>;

  // KEK Management
  storeKek(kek: number[]): Promise<void>;
  getKek(): Promise<number[] | null>;
  lockKek(): Promise<void>;
  isKekLocked(): Promise<boolean>;

  // Window Controls
  minimize(): Promise<void>;
  maximize(): Promise<void>;
  toggleMaximize(): Promise<void>;
  close(): Promise<void>;
  isMaximized(): Promise<boolean>;
  onMaximizeChange(callback: (maximized: boolean) => void): () => void;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}

// Get the exposed API from window
function getElectronAPI(): ElectronAPI {
  const api = window.electronAPI;
  if (!api) {
    throw new Error('electronAPI not available - not running in Electron');
  }
  return api;
}

class ElectronBridge implements DesktopBridge {
  readonly platform = 'electron' as const;
  readonly isDesktop = true;

  // Token Management
  async storeAuthTokens(serverUrl: string, tokens: AuthTokens): Promise<void> {
    const api = getElectronAPI();
    await api.storeAuthTokens(serverUrl, tokens);
  }

  async loadAuthTokens(serverUrl: string): Promise<AuthTokens | null> {
    const api = getElectronAPI();
    return api.loadAuthTokens(serverUrl);
  }

  async deleteAuthTokens(serverUrl: string): Promise<void> {
    const api = getElectronAPI();
    await api.deleteAuthTokens(serverUrl);
  }

  // KEK Management
  async storeKek(kek: Uint8Array): Promise<void> {
    const api = getElectronAPI();
    await api.storeKek(Array.from(kek));
  }

  async getKek(): Promise<Uint8Array | null> {
    const api = getElectronAPI();
    const kek = await api.getKek();
    return kek ? new Uint8Array(kek) : null;
  }

  async lockKek(): Promise<void> {
    const api = getElectronAPI();
    await api.lockKek();
  }

  async isKekLocked(): Promise<boolean> {
    const api = getElectronAPI();
    return api.isKekLocked();
  }

  // Window Controls
  async minimize(): Promise<void> {
    const api = getElectronAPI();
    await api.minimize();
  }

  async maximize(): Promise<void> {
    const api = getElectronAPI();
    await api.maximize();
  }

  async toggleMaximize(): Promise<void> {
    const api = getElectronAPI();
    await api.toggleMaximize();
  }

  async close(): Promise<void> {
    const api = getElectronAPI();
    await api.close();
  }

  async isMaximized(): Promise<boolean> {
    const api = getElectronAPI();
    return api.isMaximized();
  }

  onMaximizeChange(callback: (maximized: boolean) => void): () => void {
    const api = getElectronAPI();
    return api.onMaximizeChange(callback);
  }

  async startDrag(): Promise<void> {
    // Electron handles drag via CSS: -webkit-app-region: drag
    // No API call needed
  }
}

export const electronBridge = new ElectronBridge();
