// Feature flags for xelanote
export const FEATURE_FLAGS = {
  colorSyntax: true,
  taskLists: true,
  imageResize: true,
  dueDateSyntax: true,
  // LLM-based features (requires Ollama backend)
  tagSuggestions: true,
  linkSuggestions: true,
  spellCheck: true,
};

// ===== Desktop App Configuration =====

/**
 * Check if running in Tauri desktop app.
 * Uses a function to handle timing issues where config.ts may be evaluated
 * before the Tauri/Electron preload scripts expose their APIs.
 */
export function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI__' in window;
}

/**
 * Check if running in Electron desktop app.
 * Uses a function to handle timing issues where config.ts may be evaluated
 * before the Electron preload script exposes window.electronAPI.
 */
export function isElectron(): boolean {
  return typeof window !== 'undefined' && 'electronAPI' in window;
}

/**
 * Check if running in any desktop app (Tauri or Electron).
 */
export function isDesktop(): boolean {
  return isTauri() || isElectron();
}

// Legacy constants - kept for backward compatibility but may have timing issues
// Prefer using the function versions above for reliable detection
export const IS_TAURI = typeof window !== 'undefined' && '__TAURI__' in window;
export const IS_ELECTRON = typeof window !== 'undefined' && 'electronAPI' in window;
export const IS_DESKTOP = IS_TAURI || IS_ELECTRON;

// Server URL storage key (used in desktop apps)
const SERVER_URL_KEY = 'xelanote_server_url';
const DEFAULT_SERVER = 'https://xelanote.com';

/**
 * Get the current server URL.
 * In Desktop (Tauri/Electron): returns user-configured server URL (defaults to xelanote.com)
 * In web: returns empty string (use relative paths)
 */
export function getServerUrl(): string {
  if (isDesktop() && typeof localStorage !== 'undefined') {
    return localStorage.getItem(SERVER_URL_KEY) || DEFAULT_SERVER;
  }
  // Web version: always same-origin
  return '';
}

/**
 * Set the server URL (Desktop only).
 * @param url - Server URL (e.g., "https://xelanote.com" or "https://my-server.com")
 */
export function setServerUrl(url: string): void {
  if (typeof localStorage !== 'undefined') {
    // Normalize URL: remove trailing slash
    const normalized = url.replace(/\/+$/, '');
    localStorage.setItem(SERVER_URL_KEY, normalized);
  }
}

/**
 * Get the API base URL for making requests.
 * Called per-request to support dynamic server URL changes.
 *
 * In Desktop (Tauri/Electron): returns "{serverUrl}/api"
 * In web (dev): returns "http://localhost:8080/api"
 * In web (prod): returns "/api" (relative path)
 */
export function getApiBaseUrl(): string {
  const server = getServerUrl();
  if (!server) {
    // Web version: relative path (or localhost in dev)
    return import.meta.env.DEV ? 'http://localhost:8080/api' : '/api';
  }
  return `${server}/api`;
}

/**
 * Get the WebSocket base URL for real-time updates.
 *
 * In Desktop (Tauri/Electron): returns "{serverUrl}/ws" with ws:// or wss:// protocol
 * In web: returns relative WebSocket URL
 */
export function getWsBaseUrl(): string {
  const server = getServerUrl();
  if (!server) {
    // Web version: derive from current location
    if (typeof window === 'undefined') return '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = import.meta.env.DEV ? 'localhost:8080' : window.location.host;
    return `${protocol}//${host}/ws`;
  }
  // Desktop: convert https:// to wss:// or http:// to ws://
  const wsServer = server.replace(/^https:/, 'wss:').replace(/^http:/, 'ws:');
  return `${wsServer}/ws`;
}

/**
 * Get the uploads base URL for images.
 *
 * In Desktop (Tauri/Electron): returns "{serverUrl}/api/uploads"
 * In web: returns "/api/uploads" (relative)
 */
export function getUploadsBaseUrl(): string {
  const server = getServerUrl();
  if (!server) {
    return import.meta.env.DEV ? 'http://localhost:8080/api/uploads' : '/api/uploads';
  }
  return `${server}/api/uploads`;
}

/**
 * Get the default server URL.
 */
export function getDefaultServerUrl(): string {
  return DEFAULT_SERVER;
}

/**
 * Check if current server URL is the default (xelanote.com).
 */
export function isDefaultServer(): boolean {
  return getServerUrl() === DEFAULT_SERVER || getServerUrl() === '';
}
