/**
 * Web Bridge Implementation
 *
 * Noop fallback for browser environment.
 * All operations are no-ops or throw errors as appropriate.
 */

import type { DesktopBridge, AuthTokens } from './interface';

class WebBridge implements DesktopBridge {
  readonly platform = 'web' as const;
  readonly isDesktop = false;

  // Token Management - not supported in web (uses sessionStorage directly)
  async storeAuthTokens(_serverUrl: string, _tokens: AuthTokens): Promise<void> {
    // Noop - web uses sessionStorage in auth store
    console.warn('[WebBridge] storeAuthTokens called - use sessionStorage directly in web');
  }

  async loadAuthTokens(_serverUrl: string): Promise<AuthTokens | null> {
    // Noop - web uses sessionStorage in auth store
    console.warn('[WebBridge] loadAuthTokens called - use sessionStorage directly in web');
    return null;
  }

  async deleteAuthTokens(_serverUrl: string): Promise<void> {
    // Noop - web uses sessionStorage in auth store
    console.warn('[WebBridge] deleteAuthTokens called - use sessionStorage directly in web');
  }

  // KEK Management - not supported in web (uses IndexedDB via kek-persistence)
  async storeKek(_kek: Uint8Array): Promise<void> {
    // Noop - web uses IndexedDB via kek-persistence
    console.warn('[WebBridge] storeKek called - use kek-persistence directly in web');
  }

  async getKek(): Promise<Uint8Array | null> {
    // Noop - web uses IndexedDB via kek-persistence
    console.warn('[WebBridge] getKek called - use kek-persistence directly in web');
    return null;
  }

  async lockKek(): Promise<void> {
    // Noop - web uses IndexedDB via kek-persistence
    console.warn('[WebBridge] lockKek called - use kek-persistence directly in web');
  }

  async isKekLocked(): Promise<boolean> {
    // Always "locked" in web context (not using native storage)
    return true;
  }

  // Window Controls - not available in web
  async minimize(): Promise<void> {
    console.warn('[WebBridge] minimize not available in web');
  }

  async maximize(): Promise<void> {
    console.warn('[WebBridge] maximize not available in web');
  }

  async toggleMaximize(): Promise<void> {
    console.warn('[WebBridge] toggleMaximize not available in web');
  }

  async close(): Promise<void> {
    console.warn('[WebBridge] close not available in web');
  }

  async isMaximized(): Promise<boolean> {
    return false;
  }

  onMaximizeChange(_callback: (maximized: boolean) => void): () => void {
    // Noop - return empty cleanup function
    return () => {};
  }

  async startDrag(): Promise<void> {
    // Noop - not available in web
  }
}

export const webBridge = new WebBridge();
