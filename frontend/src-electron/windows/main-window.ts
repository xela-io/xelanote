/**
 * Main Window Factory
 *
 * Creates the frameless main window with proper configuration.
 */

import { BrowserWindow, shell } from 'electron';
import { join } from 'path';

import { setupWindowListeners } from '../modules/ipc-handlers';
import { registerWindow } from '../modules/window-manager';

// Determine if we're in development
const isDev = process.env.NODE_ENV === 'development';
const allowedExternalProtocols = new Set(['http:', 'https:', 'mailto:']);

function isSafeExternalUrl(rawUrl: string): boolean {
  try {
    const parsed = new URL(rawUrl);
    return allowedExternalProtocols.has(parsed.protocol.toLowerCase());
  } catch {
    return false;
  }
}

/**
 * Create the main application window.
 * Uses frameless configuration for custom titlebar (Linux + Windows + macOS).
 */
export function createMainWindow(): BrowserWindow {
  console.log('[MainWindow] Creating BrowserWindow...');
  const win = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 800,
    minHeight: 600,
    frame: false, // Frameless for custom titlebar
    backgroundColor: '#1e1e1e',
    show: false, // Don't show until ready
    webPreferences: {
      preload: join(__dirname, '../preload/index.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      // Keep webSecurity enabled in dev as well.
      // Cross-origin requests to xelanote.com are handled in the main process
      // via header rewriting (see main.ts onBeforeSendHeaders).
      webSecurity: true,
      allowRunningInsecureContent: false,
    },
  });

  // Show window when ready to prevent visual flash
  win.once('ready-to-show', () => {
    console.log('[MainWindow] Window ready to show');
    win.show();

    // Open DevTools only if ELECTRON_DEBUG is set
    if (isDev && process.env.ELECTRON_DEBUG === 'true') {
      setTimeout(() => {
        console.log('[MainWindow] Opening DevTools...');
        win.webContents.openDevTools();
      }, 500);
    }
  });

  // Log when window is closed
  win.on('closed', () => {
    console.log('[MainWindow] Window closed');
  });

  // Load the app
  console.log(`[MainWindow] isDev: ${isDev}, NODE_ENV: ${process.env.NODE_ENV}`);

  if (isDev) {
    // Development: load from SvelteKit dev server on port 5173
    // We explicitly use port 5173 because that's where our SvelteKit dev server runs
    const svelteKitUrl = 'http://localhost:5173';

    // Wait for dev server to be ready, then load
    let retryCount = 0;
    const maxRetries = 30; // 30 seconds max wait

    const tryLoad = async () => {
      try {
        console.log(`[Electron] Attempting to load ${svelteKitUrl}...`);
        await win.loadURL(svelteKitUrl);
        console.log(`[Electron] Loaded from ${svelteKitUrl}`);
      } catch (err) {
        retryCount++;
        if (retryCount < maxRetries) {
          // Server not ready yet, retry after delay
          console.log(
            `[Electron] Waiting for SvelteKit dev server at ${svelteKitUrl}... (attempt ${retryCount}/${maxRetries})`
          );
          setTimeout(tryLoad, 1000);
        } else {
          console.error(`[Electron] Failed to connect to dev server after ${maxRetries} attempts`);
          console.error(`[Electron] Error:`, err);
        }
      }
    };
    tryLoad();

    // Add error logging for page load failures
    win.webContents.on('did-fail-load', (event, errorCode, errorDescription, validatedURL) => {
      console.error(
        `[Electron] Page load failed: ${errorDescription} (${errorCode}) for ${validatedURL}`
      );
    });

    // Log when page finishes loading
    win.webContents.on('did-finish-load', () => {
      console.log(`[Electron] Page finished loading`);
    });

    // Log console messages from the renderer
    win.webContents.on('console-message', (event, level, message, _line, _sourceId) => {
      const levelStr = ['verbose', 'info', 'warning', 'error'][level] || 'unknown';
      console.log(`[Renderer ${levelStr}] ${message}`);
    });
  } else {
    // Production: load from custom app:// protocol
    // This protocol handler is registered in main.ts and serves files from build/
    console.log('[MainWindow] Loading from app:// protocol');
    win.loadURL('app://./index.html');

    // Add error logging for production page load failures
    win.webContents.on('did-fail-load', (event, errorCode, errorDescription, validatedURL) => {
      console.error(
        `[Electron] Page load failed: ${errorDescription} (${errorCode}) for ${validatedURL}`
      );
    });

    // Log console messages from the renderer in production for debugging
    win.webContents.on('console-message', (event, level, message, _line, _sourceId) => {
      if (level >= 2) {
        // Only log warnings and errors
        const levelStr = ['verbose', 'info', 'warning', 'error'][level] || 'unknown';
        console.log(`[Renderer ${levelStr}] ${message}`);
      }
    });
  }

  // Open external links in default browser
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (isSafeExternalUrl(url)) {
      shell.openExternal(url);
    } else {
      console.warn(`[Security] Blocked external URL with unsupported protocol: ${url}`);
    }
    return { action: 'deny' };
  });

  // Register with window manager and set up event listeners
  registerWindow('main', win);
  setupWindowListeners(win);

  return win;
}
