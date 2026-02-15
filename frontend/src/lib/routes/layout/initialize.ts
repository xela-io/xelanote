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
  };
  autosave: {
    initAutoSaveSettings: () => void;
  };
  goto: (path: string, options?: { replaceState?: boolean }) => void;
  publicRoutes: string[];
  setAuthInitialized: (value: boolean) => void;
  setShowUnlockModal: (value: boolean) => void;
  registerActivityListeners: () => void;
  unregisterActivityListeners: () => void;
}

export interface InitializeAppResult {
  unsubscribeTokenUpdate: (() => void) | null;
  cleanupErrorHandler: (() => void) | null;
}

export async function initializeApp(deps: InitializeAppDeps): Promise<InitializeAppResult> {
  let unsubscribeTokenUpdate: (() => void) | null = null;
  let cleanupErrorHandler: (() => void) | null = null;

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

  await deps.initSodium();

  await deps.auth.initAuth();
  deps.setAuthInitialized(true);

  unsubscribeTokenUpdate = deps.auth.addTokenUpdateListener((exp, iat) => {
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

  const tokenExpiry = deps.auth.getTokenExpiry();
  const tokenIssued = deps.auth.getTokenIssuedAt();

  if (tokenExpiry > 0) {
    deps.tokenRefresh.init(tokenExpiry, tokenIssued > 0 ? tokenIssued : undefined);
  } else {
    console.log('[Layout] Tokens not yet available, waiting for update');
  }

  const user = deps.auth.getCurrentUser();
  if (user && !deps.encryption.isEncryptionUnlocked()) {
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
          const autoLockTimeout = prefs.auto_lock_timeout || 15;
          deps.autoLock.initAutoLock(autoLockTimeout);
        } catch {
          deps.autoLock.initAutoLock(15);
        }
      }
    }
  }

  const currentPath = window.location.pathname;
  const isPublicRoute = deps.publicRoutes.some((route) => currentPath.startsWith(route));

  if (deps.auth.isAuthenticated() && isPublicRoute) {
    deps.goto('/');
  } else if (!deps.auth.isAuthenticated() && !isPublicRoute) {
    deps.goto('/login');
  }

  deps.ui.initTheme();
  deps.ui.initPreviewTheme();
  deps.ui.initSidebarWidth();
  deps.ui.initSplitPosition();

  deps.autosave.initAutoSaveSettings();
  deps.history.loadHistory();
  deps.settings.loadVirtualTreePreference();

  if (deps.auth.isAuthenticated()) {
    deps.notes.loadNotes();
    deps.settings.loadPreferences();
    deps.websocket.connect();

    if (navigator.onLine && deps.syncManager.getPendingCount() > 0) {
      console.log('[Layout] Pending offline ops found, starting sync');
      deps.syncManager.startSync().catch((err) => {
        console.error('[Layout] Initial sync failed:', err);
      });
    }
  }

  cleanupErrorHandler = deps.errorReporter.initErrorHandler();
  try {
    const config = await deps.api.getConfig();
    deps.errorReporter.setServiceAvailable(config.error_reporting_enabled ?? false);
  } catch {
    // no-op
  }

  // Initialize Web Vitals performance metrics reporting
  deps.perfMetrics.initPerfMetrics();

  deps.features.detectGraphFeature();

  if (deps.auth.isAuthenticated()) {
    deps.registerActivityListeners();
  }

  return { unsubscribeTokenUpdate, cleanupErrorHandler };
}
