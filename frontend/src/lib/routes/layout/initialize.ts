import type { User } from '$lib/api';

export interface InitializeAppDeps {
  initKEKDatabase: () => Promise<void>;
  initOfflineDatabase: () => Promise<void>;
  initSodium: () => Promise<void>;
  setApiOfflineCallback: (cb: (count: number) => void) => void;
  syncManager: {
    initSyncManager: () => Promise<void>;
    updatePendingCount: (count: number) => void;
    setOnTempIdResolved: (cb: (tempId: string, realId: string) => void) => void;
    getPendingCount: () => number;
    startSync: () => Promise<void>;
  };
  auth: {
    initAuth: () => Promise<void>;
    addTokenUpdateListener: (cb: (exp: number, iat: number) => void) => () => void;
    getTokenExpiry: () => number;
    getTokenIssuedAt: () => number;
    getCurrentUser: () => User | null;
    isAuthenticated: () => boolean;
  };
  tokenRefresh: {
    init: (exp: number, iat?: number) => void;
    stop: () => void;
  };
  encryption: {
    isEncryptionUnlocked: () => boolean;
    setSecurityLevel: (level: 'paranoid' | 'balanced' | 'convenient') => void;
    tryRestoreKEK: (userId: number) => Promise<boolean>;
  };
  autoLock: {
    initAutoLock: (minutes: number) => void;
  };
  api: {
    getPreferences: () => Promise<{ security_level?: string; auto_lock_timeout?: number }>;
    getCurrentUser: () => Promise<{ encryption_salt?: string }>;
    getConfig: () => Promise<{ error_reporting_enabled?: boolean }>;
  };
  clearPersistedKEK: (userId: number) => Promise<void>;
  notes: {
    loadNotes: () => void;
  };
  settings: {
    loadPreferences: () => void;
    loadVirtualTreePreference: () => void;
  };
  websocket: {
    connect: () => void;
  };
  features: {
    detectGraphFeature: () => void;
  };
  errorReporter: {
    initErrorHandler: () => () => void;
    setServiceAvailable: (available: boolean) => void;
  };
  perfMetrics: {
    initPerfMetrics: () => void;
  };
  history: {
    loadHistory: () => void;
  };
  ui: {
    initTheme: () => void;
    initPreviewTheme: () => void;
    initSidebarWidth: () => void;
    initSplitPosition: () => void;
    initEditorPanelsCollapsed: () => void;
  };
  autosave: {
    initAutoSaveSettings: () => void;
  };
  goto: (path: string, options?: { replaceState?: boolean }) => void;
  publicRoutes: string[];
  setAuthInitialized: (value: boolean) => void;
  setShowUnlockModal: (value: boolean) => void;
  setCleanupErrorHandler: (cleanup: (() => void) | null) => void;
  registerActivityListeners: () => void;
  unregisterActivityListeners: () => void;
}

export interface InitializeAppResult {
  unsubscribeTokenUpdate: (() => void) | null;
}

/**
 * Schedule callback after the browser has painted the next frame.
 * rAF fires before paint, the nested setTimeout(0) defers to after paint.
 */
function afterNextPaint(cb: () => void): void {
  requestAnimationFrame(() => {
    setTimeout(cb, 0);
  });
}

/**
 * Schedule callback during browser idle time.
 * Falls back to setTimeout(50) where requestIdleCallback is unavailable.
 */
function whenIdle(cb: () => void): void {
  if ('requestIdleCallback' in window) {
    requestIdleCallback(cb, { timeout: 2000 });
  } else {
    setTimeout(cb, 50);
  }
}

/**
 * Phased app initialization for optimal startup performance.
 *
 * Phase 1 (Critical): Auth, crypto init, UI restore — blocks first meaningful paint.
 * Phase 2 (After First Paint): Data loading, encryption, WebSocket, web-vitals.
 * Phase 3 (Idle): Error reporting, feature detection, background sync.
 */
export async function initializeApp(deps: InitializeAppDeps): Promise<InitializeAppResult> {
  // ── Phase 1: Critical (blocks first meaningful paint) ──────────────
  // Run independent init tasks in parallel for faster startup.

  // Group A: IndexedDB + Sync Manager (sequential within, parallel with others)
  const offlineReady = (async () => {
    try {
      await deps.initKEKDatabase();
    } catch (_err) {
      console.warn('IndexedDB unavailable, using paranoid mode');
    }

    try {
      await deps.initOfflineDatabase();
      await deps.syncManager.initSyncManager();
      deps.setApiOfflineCallback(deps.syncManager.updatePendingCount);
      deps.syncManager.setOnTempIdResolved((tempId, realId) => {
        if (window.location.pathname === `/note/${tempId}`) {
          deps.goto(`/note/${realId}`, { replaceState: true });
        }
      });
    } catch (err) {
      console.warn('Offline queue initialization failed:', err);
    }
  })();

  // Group B: Crypto lib init (parallel with offline + auth)
  // Group C: Auth restore from cookies/keyring (parallel with offline + sodium)
  await Promise.all([offlineReady, deps.initSodium(), deps.auth.initAuth()]);

  // Auth is ready — unblock first paint
  deps.setAuthInitialized(true);

  // Token update listener (synchronous setup)
  const unsubscribeTokenUpdate = deps.auth.addTokenUpdateListener((exp, iat) => {
    if (exp === 0) {
      console.log('[Layout] Token cleared (logout), stopping token-refresh');
      deps.tokenRefresh.stop();
      deps.unregisterActivityListeners();
      return;
    }

    deps.registerActivityListeners();
    console.log('[Layout] Token updated, re-initializing token-refresh');
    deps.tokenRefresh.init(exp, iat > 0 ? iat : undefined);
  });

  // Initialize token refresh from current token
  const tokenExpiry = deps.auth.getTokenExpiry();
  const tokenIssued = deps.auth.getTokenIssuedAt();
  if (tokenExpiry > 0) {
    deps.tokenRefresh.init(tokenExpiry, tokenIssued > 0 ? tokenIssued : undefined);
  } else {
    console.log('[Layout] Tokens not yet available, waiting for update');
  }

  // Auth-based redirects
  const currentPath = window.location.pathname;
  const isPublicRoute = deps.publicRoutes.some((route) => currentPath.startsWith(route));
  if (deps.auth.isAuthenticated() && isPublicRoute) {
    deps.goto('/');
  } else if (!deps.auth.isAuthenticated() && !isPublicRoute) {
    deps.goto('/login');
  }

  // UI restore: synchronous localStorage reads, no network
  deps.ui.initTheme();
  deps.ui.initPreviewTheme();
  deps.ui.initSidebarWidth();
  deps.ui.initSplitPosition();
  deps.ui.initEditorPanelsCollapsed();

  // Activity listeners right after confirmed auth (security: auto-lock + token refresh)
  if (deps.auth.isAuthenticated()) {
    deps.registerActivityListeners();
  }

  // ── Phase 2: After First Paint ─────────────────────────────────────
  // Yield to browser for first paint, then load data + heavier init.
  afterNextPaint(() => {
    // Encryption setup (needs sodium from phase 1, already loaded)
    const user = deps.auth.getCurrentUser();
    if (user && !deps.encryption.isEncryptionUnlocked()) {
      void (async () => {
        let securityLevel = 'balanced';
        try {
          const prefs = await deps.api.getPreferences();
          securityLevel = prefs.security_level || 'balanced';
          deps.encryption.setSecurityLevel(securityLevel as 'paranoid' | 'balanced' | 'convenient');
        } catch (_err) {
          console.warn('Could not load security preferences, using default');
        }

        if (securityLevel === 'paranoid') {
          await deps.clearPersistedKEK(user.id);
          const currentUser = await deps.api.getCurrentUser();
          if (currentUser.encryption_salt) {
            deps.setShowUnlockModal(true);
          }
        } else {
          const restored = await deps.encryption.tryRestoreKEK(user.id);
          if (restored) {
            try {
              const prefs = await deps.api.getPreferences();
              const autoLockTimeout = prefs.auto_lock_timeout ?? 15;
              deps.autoLock.initAutoLock(autoLockTimeout);
            } catch {
              deps.autoLock.initAutoLock(15);
            }
          }
        }
      })();
    }

    // Settings from localStorage (instant)
    deps.autosave.initAutoSaveSettings();
    deps.history.loadHistory();
    deps.settings.loadVirtualTreePreference();

    // Data loading + WebSocket (authenticated users)
    if (deps.auth.isAuthenticated()) {
      deps.notes.loadNotes();
      deps.settings.loadPreferences();
      deps.websocket.connect();
    }

    // Web Vitals: must be in after-first-paint to capture early metrics (FCP, LCP)
    deps.perfMetrics.initPerfMetrics();
  });

  // ── Phase 3: Idle ──────────────────────────────────────────────────
  // Non-urgent work that can wait for browser idle time.
  whenIdle(() => {
    const cleanup = deps.errorReporter.initErrorHandler();
    deps.setCleanupErrorHandler(cleanup);

    deps.api
      .getConfig()
      .then((config) => {
        deps.errorReporter.setServiceAvailable(config.error_reporting_enabled ?? false);
      })
      .catch(() => {
        /* no-op */
      });

    deps.features.detectGraphFeature();

    if (deps.auth.isAuthenticated() && navigator.onLine && deps.syncManager.getPendingCount() > 0) {
      console.log('[Layout] Pending offline ops found, starting sync');
      deps.syncManager.startSync().catch((err) => {
        console.error('[Layout] Initial sync failed:', err);
      });
    }
  });

  return { unsubscribeTokenUpdate };
}
