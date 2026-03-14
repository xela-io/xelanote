<script lang="ts">
  import '../app.css';

  import type { ComponentType } from 'svelte';
  import { onMount, untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { _, locale } from 'svelte-i18n';

  import { browser } from '$app/environment';
  import { beforeNavigate, goto, onNavigate } from '$app/navigation';
  import { page } from '$app/state';
  import { swipe } from '$lib/actions/swipe';
  import { setOnOfflineEnqueue as setApiOfflineCallback } from '$lib/api';
  import * as api from '$lib/api';
  import DesktopTitleBar from '$lib/components/DesktopTitleBar.svelte';
  import LayoutOverlays from '$lib/components/LayoutOverlays.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import MobileBottomNav from '$lib/components/MobileBottomNav.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import { isDesktop } from '$lib/config';
  import { clearPersistedKEK, initKEKDatabase } from '$lib/crypto/kek-persistence';
  import { initSodium } from '$lib/crypto/sodium';
  import type { DialogLoaderState } from '$lib/editor/dialog-loaders';
  import { loadConflictDialog, maybeLoadDialog } from '$lib/editor/dialog-loaders';
  import { initOfflineDatabase } from '$lib/offline/offline-queue';
  import * as syncManager from '$lib/offline/sync-manager.svelte';
  import { createActivityListeners } from '$lib/routes/layout/activity-listeners';
  import { shouldRedirectToLogin } from '$lib/routes/layout/auth-guards';
  import { handleBeforeUnload as handleBeforeUnloadHelper } from '$lib/routes/layout/beforeunload';
  import { initializeApp } from '$lib/routes/layout/initialize';
  import { createLayoutInteractions } from '$lib/routes/layout/interactions';
  import { shouldBlockNavigation } from '$lib/routes/layout/navigation-guards';
  import { registerPwaUpdates } from '$lib/routes/layout/pwa';
  import {
    processPendingShare,
    processShareTarget,
    stashPendingShare,
  } from '$lib/routes/layout/share-target';
  import { createViewportHandlers } from '$lib/routes/layout/viewport';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import * as autosave from '$lib/stores/autosave.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as history from '$lib/stores/history.svelte';
  import * as network from '$lib/stores/network.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as perfMetrics from '$lib/stores/perf-metrics.svelte';
  import * as pwa from '$lib/stores/pwa.svelte';
  import * as sessionRestore from '$lib/stores/session-restore.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as tabs from '$lib/stores/tabs.svelte';
  import * as tokenRefresh from '$lib/stores/token-refresh.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import * as websocket from '$lib/stores/websocket.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  // ✅ Service Worker Registration (PWA) mit isDirty Gate
  // NOTE: Module-level variable, but only accessed within browser guards
  let updateCheckInterval: ReturnType<typeof setInterval> | null = null;
  let cleanupErrorHandler: (() => void) | null = null;

  // Expose encryption helper for E2E tests (DEV-mode only, tree-shaken in production)
  if (import.meta.env.DEV && browser) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__testSetupEncryption = encryption.setupEncryption;
  }

  // Early mobile detection: runs before onMount to prevent sidebar flash on mobile
  const MOBILE_BREAKPOINT = 768;

  if (browser) {
    const earlyMobile = window.innerWidth < MOBILE_BREAKPOINT;
    ui.setIsMobile(earlyMobile);
    if (earlyMobile) {
      ui.setSidebarOpen(false);
      if (ui.getEditorMode() === 'split') {
        ui.setEditorMode('edit');
      }
    }
    ui.initStandaloneDetection();
    ui.initIOSDetection();
    pwa.initPwaDetection();
  }

  if (browser) {
    import('virtual:pwa-register').then(({ registerSW }) => {
      registerPwaUpdates({
        registerSW,
        isDirty: () => notes.getIsDirty(),
        getPendingCount: () => syncManager.getPendingCount(),
        confirm: (message) => confirm(message),
        updateMessage: () => get(_)('pwa.update_available'),
        getIntervalHandle: () => updateCheckInterval,
        setIntervalHandle: (handle) => {
          updateCheckInterval = handle;
        },
      });
    });
  }

  const { children } = $props();
  let QuickSwitcherComponent = $state<ComponentType | null>(null);
  let TabBarComponent = $state<ComponentType | null>(null);
  let showInstallPrompt = $state(true);
  let showUnlockModal = $state(false);
  let isAttemptingSilentRestore = $state(false);
  // Lazy-loaded conflict dialog
  let lazyLayoutDialogs = $state<DialogLoaderState>({});
  const setLazyLayoutDialogs = (s: DialogLoaderState) => {
    lazyLayoutDialogs = s;
  };

  // Lazy-load ConflictDialog when conflicts exist
  $effect(() => {
    const hasConflicts = syncManager.getConflicts().length > 0;
    maybeLoadDialog(hasConflicts, lazyLayoutDialogs, loadConflictDialog, setLazyLayoutDialogs);
  });

  // Public routes that don't require authentication
  const publicRoutes = ['/about'];

  // TEMPORARY: Landing page only mode — disable all routes except /about.
  // Set to false to re-enable the full app.
  const landingPageOnly = true;

  // Navigation guard: Only runs when URL changes (not when auth state changes)
  // This prevents race conditions during login/register operations
  $effect(() => {
    const currentPath = page.url.pathname;

    // Landing page only: redirect everything to /about immediately
    if (landingPageOnly) {
      if (!currentPath.startsWith('/about')) {
        goto('/about', { replaceState: true });
      }
      return;
    }

    const isPublicRoute = publicRoutes.some((route) => currentPath.startsWith(route));

    // Don't redirect before auth is initialized - initAuth() restores session from cookies
    // Without this check, isAuthenticated() is always false on first render, causing
    // a premature redirect to /login that loses the original URL (e.g. /note/[id])
    if (!authInitialized) return;

    // Use untrack() to read auth state WITHOUT making it a dependency
    // This prevents the effect from running when auth state changes
    const isAuth = untrack(() => auth.isAuthenticated());

    // Only redirect to login if trying to access protected routes while not authenticated
    if (
      shouldRedirectToLogin({
        authInitialized,
        isAuthenticated: isAuth,
        isPublicRoute,
      })
    ) {
      goto('/about', { replaceState: true });
    }

    // Note: We don't redirect authenticated users away from login/register pages here
    // because that causes race conditions. Login/Register pages handle their own redirects.
  });

  let authInitialized = $state(false);
  let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
  let viewportResyncTimeouts: ReturnType<typeof setTimeout>[] = [];
  let viewportDebugInfo = $state({
    inner: '-',
    visual: '-',
    appHeight: '-',
    safeBottom: '-',
    standalone: '-',
    keyboard: '-',
    urlFlag: '-',
  });
  const showViewportDebug = $derived(page.url.searchParams.get('debugViewport') === '1');

  function updateViewportDebugInfo() {
    if (!browser) return;
    const vv = window.visualViewport;
    const root = document.documentElement;
    const styles = window.getComputedStyle(root);
    const appHeight = styles.getPropertyValue('--app-viewport-height').trim() || '-';
    const safeBottom = styles.getPropertyValue('--safe-area-inset-bottom').trim() || '-';
    const navigatorStandalone = (window.navigator as { standalone?: boolean }).standalone === true;
    const mediaStandalone = window.matchMedia('(display-mode: standalone)').matches;

    viewportDebugInfo = {
      inner: `${window.innerWidth} x ${window.innerHeight}`,
      visual: vv
        ? `${Math.round(vv.width)} x ${Math.round(vv.height)} / top ${Math.round(vv.offsetTop)}`
        : 'n/a',
      appHeight,
      safeBottom,
      standalone: `class=${root.classList.contains('is-standalone')} nav=${navigatorStandalone} media=${mediaStandalone}`,
      keyboard: String(ui.getIsKeyboardOpen()),
      urlFlag: page.url.searchParams.get('debugViewport') ?? '-',
    };
  }

  onMount(() => {
    // Keep onMount sync so cleanup runs; move async work into inner function.
    let unsubscribeTokenUpdate: (() => void) | null = null;

    const initializeAsync = async () => {
      const result = await initializeApp({
        initKEKDatabase,
        initOfflineDatabase,
        initSodium,
        setApiOfflineCallback,
        syncManager: {
          initSyncManager: syncManager.initSyncManager,
          updatePendingCount: syncManager.updatePendingCount,
          setOnTempIdResolved: syncManager.setOnTempIdResolved,
          getPendingCount: syncManager.getPendingCount,
          startSync: syncManager.startSync,
        },
        auth: {
          initAuth: auth.initAuth,
          addTokenUpdateListener: auth.addTokenUpdateListener,
          getTokenExpiry: auth.getTokenExpiry,
          getTokenIssuedAt: auth.getTokenIssuedAt,
          getCurrentUser: auth.getCurrentUser,
          isAuthenticated: auth.isAuthenticated,
        },
        tokenRefresh: {
          init: tokenRefresh.init,
          stop: tokenRefresh.stop,
        },
        encryption: {
          isEncryptionUnlocked: encryption.isEncryptionUnlocked,
          setSecurityLevel: encryption.setSecurityLevel,
          tryRestoreKEK: encryption.tryRestoreKEK,
        },
        autoLock: {
          initAutoLock: autoLock.initAutoLock,
        },
        api: {
          getPreferences: api.getPreferences,
          getCurrentUser: api.getCurrentUser,
          getConfig: api.getConfig,
        },
        clearPersistedKEK,
        notes: {
          loadNotes: notes.loadNotes,
        },
        tabs: {
          initTabs: tabs.initTabs,
        },
        settings: {
          loadPreferences: settings.loadPreferences,
          loadVirtualTreePreference: settings.loadVirtualTreePreference,
        },
        websocket: {
          connect: websocket.connect,
        },
        features: {
          detectGraphFeature: features.detectGraphFeature,
        },
        errorReporter: {
          initErrorHandler: errorReporter.initErrorHandler,
          setServiceAvailable: errorReporter.setServiceAvailable,
        },
        perfMetrics: {
          initPerfMetrics: perfMetrics.initPerfMetrics,
        },
        history: {
          loadHistory: history.loadHistory,
        },
        ui: {
          initTheme: ui.initTheme,
          initPreviewTheme: ui.initPreviewTheme,
          initSidebarWidth: ui.initSidebarWidth,
          initSplitPosition: ui.initSplitPosition,
          initEditorPanelsCollapsed: ui.initEditorPanelsCollapsed,
        },
        autosave: {
          initAutoSaveSettings: autosave.initAutoSaveSettings,
        },
        goto,
        publicRoutes,
        setAuthInitialized: (value) => {
          authInitialized = value;
        },
        setShowUnlockModal: (value) => {
          showUnlockModal = value;
        },
        setCleanupErrorHandler: (cleanup) => {
          cleanupErrorHandler = cleanup;
        },
        registerActivityListeners: () => activityListeners.register(),
        unregisterActivityListeners: () => activityListeners.unregister(),
      });
      unsubscribeTokenUpdate = result.unsubscribeTokenUpdate;

      // Global keyboard shortcuts (only when authenticated)
      document.addEventListener('keydown', handleKeydown);

      // Handle PWA shortcut actions (e.g. ?action=new-note from manifest shortcuts)
      const actionParam = new URL(window.location.href).searchParams.get('action');
      if (actionParam === 'new-note' && auth.isAuthenticated()) {
        try {
          window.history.replaceState(null, '', window.location.pathname);
        } catch {
          // replaceState may throw SecurityError on iOS Safari PWA
        }
        const note = await notes.createNote('');
        if (note?.id) {
          goto(`/note/${note.id}`);
        }
      }

      // Handle Web Share Target (Chromium: share text/URLs to create notes)
      const shareDeps = {
        isAuthenticated: auth.isAuthenticated,
        createNote: notes.createNote,
        goto,
        notifySuccessfulAction: pwa.notifySuccessfulAction,
      };
      if (auth.isAuthenticated()) {
        await processShareTarget(shareDeps);
      } else {
        stashPendingShare();
      }

      // Check for pending share from pre-auth stash
      if (auth.isAuthenticated()) {
        await processPendingShare(shareDeps);
      }
    };

    function isInputElement(el: Element | null): boolean {
      if (!el) return false;
      const tagName = el.tagName;
      if (tagName === 'INPUT' || tagName === 'TEXTAREA') return true;
      if (el.getAttribute('contenteditable') === 'true') return true;
      if (el.closest('.cm-editor')) return true;
      return false;
    }

    const viewportHandlers = createViewportHandlers(
      {
        getIsMobile: () => ui.getIsMobile(),
        setIsMobile: (value) => ui.setIsMobile(value),
        getEditorMode: () => ui.getEditorMode(),
        setEditorMode: (mode) => ui.setEditorMode(mode),
        getSidebarOpen: () => ui.getSidebarOpen(),
        setSidebarOpen: (open) => ui.setSidebarOpen(open),
        setIsKeyboardOpen: (open) => ui.setIsKeyboardOpen(open),
      },
      {
        getResizeTimeout: () => resizeTimeout,
        setResizeTimeout: (handle) => {
          resizeTimeout = handle;
        },
        isInputElement,
        windowObj: window,
        documentObj: document,
      }
    );

    // Initialize Visual Viewport listener for keyboard detection
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', viewportHandlers.handleVisualViewportResize);
      window.visualViewport.addEventListener('scroll', viewportHandlers.handleVisualViewportScroll);
    }

    const syncViewportHeight = () => {
      viewportHandlers.handleResize();
      viewportHandlers.handleVisualViewportResize();
      updateViewportDebugInfo();
    };
    const clearViewportResyncTimeouts = () => {
      for (const timeout of viewportResyncTimeouts) clearTimeout(timeout);
      viewportResyncTimeouts = [];
    };
    const scheduleViewportResync = () => {
      clearViewportResyncTimeouts();

      // iOS PWA often restores stale viewport measurements for a short time
      // after app switching or orientation changes. Re-sample over a few frames.
      syncViewportHeight();
      requestAnimationFrame(() => syncViewportHeight());

      for (const delay of [50, 150, 350, 700]) {
        viewportResyncTimeouts.push(setTimeout(syncViewportHeight, delay));
      }
    };
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        scheduleViewportResync();
      } else if (document.visibilityState === 'hidden') {
        // Flush pending tab persistence when app goes to background (mobile)
        tabs.flushPendingPersist();
      }
    };

    document.addEventListener('focusin', viewportHandlers.handleFocusIn);
    document.addEventListener('focusout', viewportHandlers.handleFocusOut);
    document.addEventListener('touchstart', viewportHandlers.handleTouchStart, { passive: true });
    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('pageshow', scheduleViewportResync);
    window.addEventListener('orientationchange', scheduleViewportResync);
    window.addEventListener('focus', scheduleViewportResync);

    const { handleKeydown, handleActivity } = createLayoutInteractions({
      isAuthenticated: () => auth.isAuthenticated(),
      toggleQuickSwitcher: () => ui.toggleQuickSwitcher(),
      toggleMarkdownGuideDropdown: () => ui.toggleMarkdownGuideDropdown(),
      canUndo: () => history.history.canUndo,
      canRedo: () => history.history.canRedo,
      undo: () => history.undo(),
      redo: () => history.redo(),
      saveNote: () => notes.saveNote(),
      goto,
      graphEnabled: () => features.getGraphFeatureEnabled(),
      recordActivity: () => {
        autoLock.recordActivity();
        tokenRefresh.recordActivity();
      },
      closeCurrentTab: () => {
        const activeTab = tabs.getActiveTab();
        if (activeTab) {
          const saveFn = autosave.getAutoSaveEnabled()
            ? async () => {
                await notes.saveNote();
              }
            : undefined;
          tabs.closeTabAndNavigate(activeTab.id, saveFn).then(({ nextNoteId }) => {
            if (nextNoteId) {
              goto(`/note/${nextNoteId}`);
            } else {
              goto('/');
            }
          });
        }
      },
      nextTab: () => tabs.nextTab(),
      prevTab: () => tabs.prevTab(),
      isDesktop,
      getPathname: () => window.location.pathname,
    });

    const activityListeners = createActivityListeners({ handleActivity });

    // ✅ NEW: beforeunload handler for unsaved changes warning
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      // Flush pending tab persistence before unload (uses keepalive fetch)
      tabs.flushPendingPersist();

      handleBeforeUnloadHelper(e, {
        isDirty: () => notes.getIsDirty(),
        isSyncing: () => syncManager.getIsSyncing(),
        warningMessage: get(_)('editor.unsaved_warning'),
      });
    };

    window.addEventListener('beforeunload', handleBeforeUnload);

    // Mobile detection must run synchronously before async init to prevent
    // sidebar rendering in desktop mode on mobile devices
    viewportHandlers.handleResize();
    updateViewportDebugInfo();
    window.addEventListener('resize', viewportHandlers.debouncedHandleResize);
    window.addEventListener('resize', updateViewportDebugInfo);
    window.addEventListener('focus', updateViewportDebugInfo);
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', updateViewportDebugInfo);
      window.visualViewport.addEventListener('scroll', updateViewportDebugInfo);
    }

    // Start async initialization (non-blocking)
    void initializeAsync();

    return () => {
      document.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('resize', viewportHandlers.debouncedHandleResize);
      window.removeEventListener('resize', updateViewportDebugInfo);
      window.removeEventListener('focus', updateViewportDebugInfo);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('pageshow', scheduleViewportResync);
      window.removeEventListener('orientationchange', scheduleViewportResync);
      window.removeEventListener('focus', scheduleViewportResync);
      if (window.visualViewport) {
        window.visualViewport.removeEventListener('resize', updateViewportDebugInfo);
        window.visualViewport.removeEventListener('scroll', updateViewportDebugInfo);
      }
      clearViewportResyncTimeouts();
      viewportHandlers.cleanup();
      websocket.disconnect();

      // Cleanup activity listeners
      activityListeners.unregister();

      // Error reporter cleanup
      cleanupErrorHandler?.();

      // NEW: beforeunload cleanup
      window.removeEventListener('beforeunload', handleBeforeUnload);

      // NEW: SW update interval cleanup
      if (updateCheckInterval !== null) {
        clearInterval(updateCheckInterval);
        updateCheckInterval = null;
      }

      // Token refresh cleanup
      if (unsubscribeTokenUpdate) {
        unsubscribeTokenUpdate();
      }
      tokenRefresh.stop();
    };
  });

  // Determine if we should show the app layout (sidebar, etc.)
  // Show public layout if: on public route OR not authenticated
  const isPublic = $derived(
    publicRoutes.some((route) => page.url.pathname.startsWith(route)) || !auth.isAuthenticated()
  );

  // TEMPORARY: Check if we're on the about page (for landing page only mode)
  const isAboutPage = $derived(page.url.pathname.startsWith('/about'));

  // iOS Safari (non-PWA): use body-level scrolling so Safari toolbar auto-hides
  const useBodyScroll = $derived(ui.getIsMobile() && ui.getIsIOS() && !ui.getIsStandalone());

  async function loadQuickSwitcher() {
    if (QuickSwitcherComponent) return;
    const module = await import('$lib/components/QuickSwitcher.svelte');
    QuickSwitcherComponent = loadSvelteComponentFromModule(module, 'QuickSwitcher');
  }

  $effect(() => {
    if (ui.getQuickSwitcherOpen()) {
      loadQuickSwitcher();
    }
  });

  // Tab bar: lazy-load when tabs are visible and feature is enabled
  const showTabBar = $derived(
    features.getTabsFeatureEnabled() &&
      page.url.pathname.startsWith('/note/') &&
      tabs.getTabs().length > 0 &&
      tabs.isInitialized()
  );

  async function loadTabBar() {
    if (TabBarComponent) return;
    const module = await import('$lib/components/TabBar.svelte');
    TabBarComponent = loadSvelteComponentFromModule(module, 'TabBar');
  }

  $effect(() => {
    if (showTabBar) {
      loadTabBar();
    }
  });

  // Tab sync: URL -> Tab (activate/open tab when navigating to /note/X)
  // IMPORTANT: Only depend on page.url.pathname and tabs.isInitialized().
  // Do NOT read notes store reactively here — any notes change (dirty, title,
  // save status) would re-trigger this effect and re-open just-closed tabs.
  // Title resolution happens in the separate title-sync effect below.
  $effect(() => {
    if (!features.getTabsFeatureEnabled()) return;
    if (!tabs.isInitialized()) return;
    const pathname = page.url.pathname;
    if (!pathname.startsWith('/note/')) return;
    const noteId = pathname.split('/note/')[1];
    if (!noteId) return;
    // Read title AND sync call outside reactive tracking to avoid dependency
    // on notes store and tab state. Without untrack, syncTabWithRoute → findTabByNoteId
    // reads group.tabs, making this effect re-trigger on every tab change — which
    // reopens just-closed tabs because the URL still points to the old note.
    untrack(() => {
      const title = notes.getNoteById(noteId)?.title ?? '';
      tabs.syncTabWithRoute(noteId, title);
    });
  });

  // Tab sync: dirty state -> tab
  $effect(() => {
    if (!features.getTabsFeatureEnabled()) return;
    const cn = notes.getCurrentNote();
    if (cn) tabs.syncDirtyState(cn.id, notes.getIsDirty());
  });

  // Tab sync: title changes -> tab
  $effect(() => {
    if (!features.getTabsFeatureEnabled()) return;
    const cn = notes.getCurrentNote();
    if (cn) {
      const existing = tabs.findTabByNoteId(cn.id);
      if (existing && existing.tab.title !== cn.title) {
        tabs.updateTabTitle(existing.tab.id, cn.title);
      }
    }
  });

  // Tab resolve: once notes are loaded AND tab preferences are restored, resolve titles
  $effect(() => {
    if (!features.getTabsFeatureEnabled()) return;
    if (tabs.isInitialized()) return; // only once
    if (!tabs.isPreferencesLoaded()) return; // wait for initTabs (prevents race condition)
    if (notes.getNotesLoading()) return; // wait until loaded
    console.log(
      '[layout] Tab resolve effect: calling resolveTabTitles. Notes count:',
      notes.getNotes().length
    );
    tabs.resolveTabTitles((id: string) => notes.getNoteById(id));
  });

  $effect(() => {
    if (browser && showViewportDebug) {
      updateViewportDebugInfo();
    }
  });

  // Watch for encryption locked state and try silent KEK restore before showing modal.
  // For balanced/convenient security levels, KEK is persisted in IndexedDB and can be
  // restored without user interaction. Only show the modal if restore fails or paranoid mode.
  $effect(() => {
    const encryptionError = notes.getError() === 'ENCRYPTION_LOCKED';
    const currentNote = notes.getCurrentNote();
    const isLocked = !encryption.isEncryptionUnlocked();
    const needsUnlock = encryptionError || (currentNote?.content_encrypted && isLocked);

    if (needsUnlock && !showUnlockModal && !isAttemptingSilentRestore) {
      void attemptSilentRestoreOrShowModal();
    }
  });

  async function attemptSilentRestoreOrShowModal() {
    const secLevel = encryption.getSecurityLevel();
    // Use auth user ID as fallback: encryption.getUserID() is null until
    // tryRestoreKEK/setupEncryption completes, but auth user is available immediately.
    const userId = encryption.getUserID() ?? auth.getCurrentUser()?.id ?? null;

    // For balanced/convenient: try silent KEK restore from IndexedDB
    if (secLevel !== 'paranoid' && userId !== null) {
      isAttemptingSilentRestore = true;
      try {
        const restored = await encryption.tryRestoreKEK(userId);
        if (restored) {
          notes.clearError();
          showUnlockModal = false;
          // Re-init auto-lock timer
          try {
            const prefs = await api.getPreferences();
            const timeout = prefs.auto_lock_timeout ?? 15;
            autoLock.initAutoLock(timeout);
          } catch {
            autoLock.initAutoLock(15);
          }
          return;
        }
      } catch {
        // Fall through to show modal
      } finally {
        isAttemptingSilentRestore = false;
      }
    }

    // Paranoid mode or restore failed: show modal
    showUnlockModal = true;
  }

  // PWA iOS Install Coach: trigger after first successful user action
  $effect(() => {
    if (authInitialized && auth.isAuthenticated()) {
      pwa.startFallbackTimer();
    }
    return () => {
      pwa.cleanupTimers();
    };
  });

  // Watch autoSaveStatus: when 'saved', notify successful action
  $effect(() => {
    if (notes.getAutoSaveStatus() === 'saved') {
      pwa.notifySuccessfulAction();
    }
  });

  // Standalone permanently disables install coach
  $effect(() => {
    if (ui.getIsStandalone()) {
      pwa.markInstalled();
    }
  });

  // Dynamic HTML lang attribute
  $effect(() => {
    if (browser) {
      document.documentElement.lang = $locale || 'de';
    }
  });

  // Warn about unsaved changes when navigating (only if auto-save is disabled)
  beforeNavigate(({ cancel }) => {
    if (
      shouldBlockNavigation({
        autosaveEnabled: () => autosave.getAutoSaveEnabled(),
        isDirty: () => notes.getIsDirty(),
        confirm: (message) => confirm(message),
        getUnsavedMessage: () => $_('editor.unsaved_warning'),
      })
    ) {
      cancel();
    }
  });

  // View Transitions API: smooth crossfade between pages (native app feel)
  // Skip transitions TO /note/ paths — CodeMirror needs stable DOM measurements on init.
  onNavigate((navigation) => {
    if (
      !document.startViewTransition ||
      navigation.to?.url.pathname.startsWith('/note/') ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      return;
    }

    return new Promise((resolve) => {
      document.startViewTransition(async () => {
        resolve();
        await navigation.complete;
      });
    });
  });
</script>

<!-- Root container: fixed to viewport edges, all scrolling in inner containers -->
<div
  class="{useBodyScroll
    ? 'relative min-h-dvh'
    : 'fixed inset-0 overflow-hidden'} pt-safe flex flex-col bg-background"
>
  {#if showViewportDebug}
    <div
      class="fixed top-2 right-2 z-[9999] max-w-[calc(100vw-1rem)] rounded-md border border-border bg-background/95 px-3 py-2 text-[11px] leading-tight shadow-lg backdrop-blur-sm"
      style="font-family: ui-monospace, SFMono-Regular, Menlo, monospace;"
    >
      <div>inner: {viewportDebugInfo.inner}</div>
      <div>visual: {viewportDebugInfo.visual}</div>
      <div>--app-viewport-height: {viewportDebugInfo.appHeight}</div>
      <div>safe-bottom: {viewportDebugInfo.safeBottom}</div>
      <div>standalone: {viewportDebugInfo.standalone}</div>
      <div>keyboard: {viewportDebugInfo.keyboard}</div>
      <div>?debugViewport: {viewportDebugInfo.urlFlag}</div>
    </div>
  {/if}

  <!-- Skip Link for keyboard navigation -->
  <a
    href="#main-content"
    class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:bg-primary focus:text-primary-foreground focus:rounded-md focus:shadow-lg focus:outline-none focus:ring-2 focus:ring-ring"
  >
    {$_('accessibility.skip_to_main')}
  </a>

  <!-- Desktop Title Bar (Electron/Tauri) -->
  <DesktopTitleBar />

  {#if !authInitialized}
    <!-- Loading state while checking auth - uses CSS variables for instant theming -->
    <div
      class="flex-1 flex items-center justify-center"
      style="background-color: var(--color-background, oklch(97% 0.03 85)); color-scheme: light dark;"
    >
      <div class="text-center animate-pulse">
        <div class="mx-auto mb-4">
          <Logo size="xl" />
        </div>
        <p style="color: var(--color-muted-foreground, #888);">{$_('common.loading')}</p>
      </div>
    </div>
  {:else if landingPageOnly && !isAboutPage}
    <!-- TEMPORARY: Landing page only mode — don't render anything, redirect is in progress -->
    <div class="flex-1"></div>
  {:else if isPublic}
    <!-- Public pages (login/register) - no sidebar -->
    <div class="flex-1 overflow-auto">
      {@render children()}
    </div>
  {:else}
    <!-- Protected app pages - with sidebar -->
    <div
      class="flex flex-1 {useBodyScroll ? '' : 'overflow-hidden'}"
      use:swipe={{
        direction: 'right',
        edge: 'left',
        onSwipe: () => ui.setSidebarOpen(true),
        enabled: () => ui.getIsMobile() && !ui.getSidebarOpen(),
      }}
    >
      <svelte:boundary>
        <Sidebar />
        {#snippet failed(_error, reset)}
          <div
            class="w-64 shrink-0 border-r border-border bg-sidebar flex items-center justify-center p-4"
          >
            <div class="text-center text-sm text-muted-foreground">
              <p class="mb-2">{$_('error_page.component_crashed')}</p>
              <button
                onclick={reset}
                class="px-3 py-1.5 bg-primary text-primary-foreground rounded text-xs hover:bg-primary/90"
                >{$_('error_page.retry')}</button
              >
            </div>
          </div>
        {/snippet}
      </svelte:boundary>

      <!-- Mobile: content stays fixed; sidebar overlays on top -->
      <div
        class="flex-1 flex flex-col min-w-0 {ui.getIsMobile()
          ? 'transition-opacity duration-200'
          : ''}"
      >
        {#if showTabBar && TabBarComponent}
          <TabBarComponent />
        {/if}
        <main
          id="main-content"
          tabindex="-1"
          class="flex-1 {useBodyScroll
            ? ''
            : 'overflow-hidden'} flex flex-col focus:outline-none relative z-0"
          style:padding-bottom={ui.getIsMobile() && !ui.getIsKeyboardOpen() && !useBodyScroll
            ? 'calc(2.5rem + var(--safe-area-inset-bottom))'
            : '0'}
        >
          <div class="flex-1 {useBodyScroll ? '' : 'overflow-hidden'}">
            {@render children()}
          </div>
        </main>

        {#if ui.getIsMobile() && !ui.getIsKeyboardOpen()}
          <MobileBottomNav />
        {/if}
      </div>

      <!-- Mobile: backdrop overlay to close sidebar on tap -->
      {#if ui.getIsMobile() && ui.getSidebarOpen()}
        <div
          class="fixed inset-0 z-40 bg-black/30 backdrop-blur-[1px]"
          onclick={() => ui.setSidebarOpen(false)}
          onkeydown={(e) => {
            if (e.key === 'Escape') ui.setSidebarOpen(false);
          }}
          tabindex="-1"
          role="presentation"
          use:swipe={{
            direction: 'left',
            edge: 'none',
            onSwipe: () => ui.setSidebarOpen(false),
            enabled: () => ui.getIsMobile(),
          }}
        ></div>
      {/if}
    </div>

    {#if ui.getQuickSwitcherOpen() && QuickSwitcherComponent}
      <QuickSwitcherComponent />
    {/if}
  {/if}
</div>

<!-- Global overlays: Toast, OfflineBanner, ConflictDialog, InstallPrompt, Encryption, Dialogs -->
<LayoutOverlays
  showOfflineBanner={network.getShowOfflineBanner()}
  isSyncing={syncManager.getIsSyncing()}
  hasConflicts={syncManager.getConflicts().length > 0}
  conflictDialog={lazyLayoutDialogs.conflictDialog}
  {showInstallPrompt}
  {isPublic}
  showSessionRestoreBanner={sessionRestore.isSessionRestoreActive()}
  bind:showUnlockModal
  onCloseInstallPrompt={() => (showInstallPrompt = false)}
/>
