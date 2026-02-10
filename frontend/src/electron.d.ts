/**
 * TypeScript declarations for Electron API exposed via preload.
 */

interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user_id: number | null;
}

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

export {};
