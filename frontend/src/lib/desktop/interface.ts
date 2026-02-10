/**
 * Desktop Bridge Interface
 *
 * Abstraction layer for desktop-specific functionality.
 * Implementations: Tauri, Electron, Web (noop)
 */

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user_id: number | null;
}

export interface DesktopBridge {
  // Platform identification
  readonly platform: 'tauri' | 'electron' | 'web';
  readonly isDesktop: boolean;

  // Token Management (OS keyring / secure storage)
  storeAuthTokens(serverUrl: string, tokens: AuthTokens): Promise<void>;
  loadAuthTokens(serverUrl: string): Promise<AuthTokens | null>;
  deleteAuthTokens(serverUrl: string): Promise<void>;

  // KEK Management (in-memory secure storage)
  storeKek(kek: Uint8Array): Promise<void>;
  getKek(): Promise<Uint8Array | null>;
  lockKek(): Promise<void>;
  isKekLocked(): Promise<boolean>;

  // Window Controls
  minimize(): Promise<void>;
  maximize(): Promise<void>;
  toggleMaximize(): Promise<void>;
  close(): Promise<void>;
  isMaximized(): Promise<boolean>;
  onMaximizeChange(callback: (maximized: boolean) => void): () => void;

  // Window Drag Region (for frameless windows)
  startDrag(): Promise<void>;
}

// Singleton instance - lazy initialized
let bridgeInstance: DesktopBridge | null = null;

/**
 * Get the desktop bridge for the current platform.
 * Returns the appropriate implementation based on runtime detection.
 */
export async function getDesktopBridge(): Promise<DesktopBridge> {
  if (bridgeInstance) return bridgeInstance;

  // Detect platform at runtime
  if (typeof window !== 'undefined') {
    // Check for Tauri
    if ('__TAURI__' in window) {
      const { tauriBridge } = await import('./tauri-bridge');
      bridgeInstance = tauriBridge;
      return bridgeInstance;
    }

    // Check for Electron (preload exposes electronAPI)
    if ('electronAPI' in window) {
      const { electronBridge } = await import('./electron-bridge');
      bridgeInstance = electronBridge;
      return bridgeInstance;
    }
  }

  // Fallback to web (noop)
  const { webBridge } = await import('./web-bridge');
  bridgeInstance = webBridge;
  return bridgeInstance;
}

/**
 * Get the desktop bridge synchronously (returns null if not initialized).
 * Use getDesktopBridge() for async initialization.
 */
export function getDesktopBridgeSync(): DesktopBridge | null {
  return bridgeInstance;
}

/**
 * Reset the bridge instance (for testing).
 */
export function resetBridge(): void {
  bridgeInstance = null;
}
