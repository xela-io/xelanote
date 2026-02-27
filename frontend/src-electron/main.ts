/**
 * Electron Main Process
 *
 * Entry point for the Electron desktop app.
 * Handles window creation, IPC, and security.
 */

import { app, BrowserWindow, net, protocol, session } from 'electron';
import { mkdirSync } from 'fs';
import { join } from 'path';
import { pathToFileURL } from 'url';

import { registerIpcHandlers } from './modules/ipc-handlers';
import { createMainWindow } from './windows/main-window';

// Linux compatibility - handle GPU/Wayland issues and shared memory problems
// Electron 33+ has known issues with GPU process on some systems
if (process.platform === 'linux') {
  // Cloudflare Turnstile may attempt Private Access Token flows that can
  // crash the Electron renderer on some Linux setups (bad IPC message).
  // Disable these browser features so Turnstile falls back to regular challenge mode.
  app.commandLine.appendSwitch('disable-features', 'PrivateStateTokens,TrustTokens');

  // Completely disable GPU acceleration
  app.disableHardwareAcceleration();

  // Disable GPU-related features
  app.commandLine.appendSwitch('disable-gpu');
  app.commandLine.appendSwitch('disable-gpu-compositing');
  app.commandLine.appendSwitch('disable-software-rasterizer');

  // Use runtime tmp from environment (set by npm script) when available.
  // This avoids putting Chromium shared-memory files onto synced/network
  // folders (e.g. Nextcloud), which can break Turnstile on Linux.
  const userDataPath = app.getPath('userData');
  const runtimeTmpPath = process.env.TMPDIR || `${userDataPath}/tmp`;
  try {
    mkdirSync(runtimeTmpPath, { recursive: true });
    process.env.TMPDIR = runtimeTmpPath;
    process.env.TMP = runtimeTmpPath;
    process.env.TEMP = runtimeTmpPath;
    app.setPath('temp', runtimeTmpPath);
  } catch (err) {
    console.warn('[Main] Failed to prepare runtime tmp dir:', err);
  }
  app.commandLine.appendSwitch('disk-cache-dir', `${userDataPath}/cache`);
  // Fallback for systems with broken /dev/shm permissions.
  app.commandLine.appendSwitch('disable-dev-shm-usage');
}

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (process.platform === 'win32') {
  app.setAppUserModelId(app.getName());
}

// Prevent multiple instances
const gotTheLock = app.requestSingleInstanceLock();
if (!gotTheLock) {
  app.quit();
  process.exit(0);
}

let mainWindow: BrowserWindow | null = null;

// Determine if we're in development mode
const isDev = process.env.NODE_ENV === 'development';

// Register custom protocol for serving static files in production
// This allows absolute paths like /_app/... to resolve correctly from the build directory
if (!isDev) {
  protocol.registerSchemesAsPrivileged([
    {
      scheme: 'app',
      privileges: {
        standard: true,
        secure: true,
        supportFetchAPI: true,
        corsEnabled: true,
      },
    },
  ]);
}

// Create window when Electron has finished initialization
app.whenReady().then(async () => {
  console.log('[Main] Electron app is ready');

  // Dev CORS bridge for production API testing:
  // keep renderer webSecurity enabled and strip Origin only for xelanote hosts.
  if (isDev) {
    session.defaultSession.webRequest.onBeforeSendHeaders(
      { urls: ['https://xelanote.com/*', 'https://www.xelanote.com/*'] },
      (details, callback) => {
        delete details.requestHeaders.Origin;
        callback({ requestHeaders: details.requestHeaders });
      }
    );
  }

  // In production, register the protocol handler that serves files from the build directory
  if (!isDev) {
    const buildPath = join(app.getAppPath(), 'build');
    console.log('[Main] Registering app:// protocol for:', buildPath);

    protocol.handle('app', (request) => {
      // Parse the URL and get the path
      const url = new URL(request.url);
      let filePath = url.pathname;

      // Default to index.html for root path
      if (filePath === '/' || filePath === '') {
        filePath = '/index.html';
      }

      // Remove leading slash and construct full path
      const fullPath = join(buildPath, filePath);
      console.log(`[Protocol] ${request.url} -> ${fullPath}`);

      // Return the file using net.fetch with file:// URL
      return net.fetch(pathToFileURL(fullPath).toString());
    });
  }

  // Set Content Security Policy
  // In development, do not override response CSP headers.
  // This is required for remote iframe content such as /captcha, which
  // needs its own CSP to load Cloudflare Turnstile scripts.
  session.defaultSession.webRequest.onHeadersReceived((details, callback) => {
    if (isDev) {
      callback({ responseHeaders: details.responseHeaders });
      return;
    }

    // Production CSP - allow app:// protocol and WebAssembly
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': [
          [
            "default-src 'self' app:",
            "script-src 'self' app: 'unsafe-inline' 'wasm-unsafe-eval'",
            "style-src 'self' app: 'unsafe-inline'",
            "connect-src 'self' app: https: wss:",
            "img-src 'self' app: data: blob: https:",
            "font-src 'self' app: data:",
            // CAPTCHA iframe and Turnstile challenge are loaded over HTTPS only in production.
            'frame-src https:',
          ].join('; '),
        ],
      },
    });
  });

  // Register IPC handlers
  registerIpcHandlers();

  // Create the main window
  console.log('[Main] Creating main window...');
  mainWindow = createMainWindow();
  console.log('[Main] Main window created');

  // Handle second instance
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });
});

// Quit when all windows are closed, except on macOS
app.on('window-all-closed', () => {
  console.log('[Main] All windows closed');
  if (process.platform !== 'darwin') {
    console.log('[Main] Quitting app...');
    app.quit();
  }
});

// On macOS, re-create a window when dock icon is clicked
app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    mainWindow = createMainWindow();
  }
});

// Log when app is about to quit
app.on('before-quit', () => {
  console.log('[Main] App is about to quit');
});

// Security: Disable navigation to unknown origins
app.on('web-contents-created', (_, contents) => {
  contents.on('will-navigate', (event, navigationUrl) => {
    const parsedUrl = new URL(navigationUrl);

    // Allow app:// protocol (our custom protocol)
    if (parsedUrl.protocol === 'app:') {
      return;
    }

    const allowedOrigins = ['localhost', '127.0.0.1'];

    // In production, allow the app's own origin
    if (process.env.NODE_ENV === 'production') {
      allowedOrigins.push('xelanote.com');
    }

    if (!allowedOrigins.includes(parsedUrl.hostname)) {
      event.preventDefault();
      console.warn(`Blocked navigation to: ${navigationUrl}`);
    }
  });

  // Disable new window creation
  contents.setWindowOpenHandler(() => {
    return { action: 'deny' };
  });
});

// Export for type checking
export { mainWindow };
