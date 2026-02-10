/**
 * Tauri Bridge Implementation
 *
 * Uses Tauri's IPC to communicate with Rust backend for secure operations.
 */

import type { AuthTokens, DesktopBridge } from './interface';

// Tauri invoke function (lazy loaded to avoid SSR issues)
async function invoke<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: tauriInvoke } = await import('@tauri-apps/api/core');
  return tauriInvoke<T>(cmd, args);
}

// Window API (lazy loaded)
async function getWindow() {
  const { getCurrentWindow } = await import('@tauri-apps/api/window');
  return getCurrentWindow();
}

class TauriBridge implements DesktopBridge {
  readonly platform = 'tauri' as const;
  readonly isDesktop = true;

  private unlistenResize: (() => void) | null = null;

  // Token Management
  async storeAuthTokens(serverUrl: string, tokens: AuthTokens): Promise<void> {
    await invoke('store_auth_tokens', {
      serverUrl,
      tokens: {
        access_token: tokens.access_token,
        refresh_token: tokens.refresh_token,
        user_id: tokens.user_id,
      },
    });
  }

  async loadAuthTokens(serverUrl: string): Promise<AuthTokens | null> {
    return invoke<AuthTokens | null>('load_auth_tokens', { serverUrl });
  }

  async deleteAuthTokens(serverUrl: string): Promise<void> {
    await invoke('delete_auth_tokens', { serverUrl });
  }

  // KEK Management
  async storeKek(kek: Uint8Array): Promise<void> {
    await invoke('store_kek', { kek: Array.from(kek) });
  }

  async getKek(): Promise<Uint8Array | null> {
    const kek = await invoke<number[] | null>('get_kek');
    return kek ? new Uint8Array(kek) : null;
  }

  async lockKek(): Promise<void> {
    await invoke('lock_kek');
  }

  async isKekLocked(): Promise<boolean> {
    return invoke<boolean>('is_kek_locked');
  }

  // Window Controls
  async minimize(): Promise<void> {
    const win = await getWindow();
    await win.minimize();
  }

  async maximize(): Promise<void> {
    const win = await getWindow();
    await win.maximize();
  }

  async toggleMaximize(): Promise<void> {
    const win = await getWindow();
    await win.toggleMaximize();
  }

  async close(): Promise<void> {
    const win = await getWindow();
    await win.close();
  }

  async isMaximized(): Promise<boolean> {
    const win = await getWindow();
    return win.isMaximized();
  }

  onMaximizeChange(callback: (maximized: boolean) => void): () => void {
    // Set up listener asynchronously
    (async () => {
      const win = await getWindow();

      // Clean up existing listener
      if (this.unlistenResize) {
        this.unlistenResize();
      }

      // Listen for resize events and check maximize state
      this.unlistenResize = await win.onResized(async () => {
        const maximized = await win.isMaximized();
        callback(maximized);
      });
    })();

    // Return cleanup function
    return () => {
      if (this.unlistenResize) {
        this.unlistenResize();
        this.unlistenResize = null;
      }
    };
  }

  async startDrag(): Promise<void> {
    const win = await getWindow();
    await win.startDragging();
  }
}

export const tauriBridge = new TauriBridge();
