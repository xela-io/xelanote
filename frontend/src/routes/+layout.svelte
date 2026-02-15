<script lang="ts">
  import '../app.css';

  import type { ComponentType } from 'svelte';
  import { onMount, untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { fade } from 'svelte/transition';
  import { _, locale } from 'svelte-i18n';

  import { browser } from '$app/environment';
  import { afterNavigate, beforeNavigate, goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { swipe } from '$lib/actions/swipe';
  import { setOnOfflineEnqueue as setApiOfflineCallback } from '$lib/api';
  import * as api from '$lib/api';
  import ConflictDialog from '$lib/components/ConflictDialog.svelte';
  import DesktopTitleBar from '$lib/components/DesktopTitleBar.svelte';
  import InstallPrompt from '$lib/components/InstallPrompt.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import MobileHeader from '$lib/components/MobileHeader.svelte';
  import OfflineBanner from '$lib/components/OfflineBanner.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import AlertDialog from '$lib/components/ui/AlertDialog.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import UnlockEncryptionModal from '$lib/components/UnlockEncryptionModal.svelte';
  import { clearPersistedKEK, initKEKDatabase } from '$lib/crypto/kek-persistence';
  import { initSodium } from '$lib/crypto/sodium';
  import { initOfflineDatabase } from '$lib/offline/offline-queue';
  import * as syncManager from '$lib/offline/sync-manager.svelte';
  import { shouldRedirectToLogin } from '$lib/routes/layout/auth-guards';
  import { handleBeforeUnload as handleBeforeUnloadHelper } from '$lib/routes/layout/beforeunload';
  import { initializeApp } from '$lib/routes/layout/initialize';
  import { createLayoutInteractions } from '$lib/routes/layout/interactions';
  import { shouldBlockNavigation } from '$lib/routes/layout/navigation-guards';
  import { registerPwaUpdates } from '$lib/routes/layout/pwa';
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
  import * as pwa from '$lib/stores/pwa.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as tokenRefresh from '$lib/stores/token-refresh.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import * as websocket from '$lib/stores/websocket.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  // ✅ Service Worker Registration (PWA) mit isDirty Gate
  // NOTE: Module-level variable, but only accessed within browser guards
  let updateCheckInterval: ReturnType<typeof setInterval> | null = null;
  let cleanupErrorHandler: (() => void) | null = null;

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
  let showInstallPrompt = $state(true);
  let showUnlockModal = $state(false);

  // Public routes that don't require authentication
  const publicRoutes = ['/login', '/register', '/about'];

  // Navigation guard: Only runs when URL changes (not when auth state changes)
  // This prevents race conditions during login/register operations
  $effect(() => {
    const currentPath = $page.url.pathname;
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
      goto('/login');
    }

    // Note: We don't redirect authenticated users away from login/register pages here
    // because that causes race conditions. Login/Register pages handle their own redirects.
  });

  let authInitialized = $state(false);
  let activityListenersRegistered = false;
  let resizeTimeout: ReturnType<typeof setTimeout> | null = null;

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
        history: {
          loadHistory: history.loadHistory,
        },
        ui: {
          initTheme: ui.initTheme,
          initPreviewTheme: ui.initPreviewTheme,
          initSidebarWidth: ui.initSidebarWidth,
          initSplitPosition: ui.initSplitPosition,
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
        registerActivityListeners: () => {
          if (!activityListenersRegistered) {
            registerActivityListeners();
          }
        },
        unregisterActivityListeners: () => {
          if (activityListenersRegistered) {
            unregisterActivityListeners();
          }
        },
      });
      unsubscribeTokenUpdate = result.unsubscribeTokenUpdate;
      cleanupErrorHandler = result.cleanupErrorHandler;

      // Global keyboard shortcuts (only when authenticated)
      document.addEventListener('keydown', handleKeydown);

      // Handle PWA shortcut actions (e.g. ?action=new-note from manifest shortcuts)
      const actionParam = new URL(window.location.href).searchParams.get('action');
      if (actionParam === 'new-note' && auth.isAuthenticated()) {
        window.history.replaceState(null, '', window.location.pathname);
        const note = await notes.createNote('');
        if (note?.id) {
          goto(`/note/${note.id}`);
        }
      }

      // Handle Web Share Target (Chromium: share text/URLs to create notes)
      if (auth.isAuthenticated()) {
        await processShareTarget();
      } else {
        // If not authenticated, stash share params for after login
        stashPendingShare();
      }

      // Check for pending share from pre-auth stash
      if (auth.isAuthenticated()) {
        await processPendingShare();
      }
    };

    function stashPendingShare(): void {
      try {
        const params = new URL(window.location.href).searchParams;
        const title = safeGetParam(params, 'title');
        const text = safeGetParam(params, 'text');
        const url = safeGetParam(params, 'url');
        if (!title && !text && !url) return;

        sessionStorage.setItem(
          'xelanote-pending-share',
          JSON.stringify({
            title: (title ?? '').slice(0, 200),
            text: (text ?? '').slice(0, 50_000),
            url: (url ?? '').slice(0, 2048),
          })
        );
        window.history.replaceState(null, '', window.location.pathname);
      } catch {
        // silent — sessionStorage unavailable
      }
    }

    async function processPendingShare(): Promise<void> {
      try {
        const raw = sessionStorage.getItem('xelanote-pending-share');
        if (!raw) return;
        sessionStorage.removeItem('xelanote-pending-share');

        const parsed = JSON.parse(raw);
        if (
          !parsed ||
          typeof parsed !== 'object' ||
          typeof parsed.title !== 'string' ||
          typeof parsed.text !== 'string' ||
          typeof parsed.url !== 'string'
        ) {
          return;
        }

        await createNoteFromShare(parsed.title, parsed.text, parsed.url);
      } catch {
        // Parse error or sessionStorage error — silently ignore
        try {
          sessionStorage.removeItem('xelanote-pending-share');
        } catch {
          /* silent */
        }
      }
    }

    function safeGetParam(params: URLSearchParams, key: string): string | null {
      try {
        const val = params.get(key);
        return val ? val.trim() : null;
      } catch {
        // decodeURIComponent error on malformed %-encoded params
        return null;
      }
    }

    async function processShareTarget(): Promise<void> {
      const params = new URL(window.location.href).searchParams;
      const title = (safeGetParam(params, 'title') ?? '').slice(0, 200);
      const text = (safeGetParam(params, 'text') ?? '').slice(0, 50_000);
      const url = (safeGetParam(params, 'url') ?? '').slice(0, 2048);

      if (!title && !text && !url) return;

      window.history.replaceState(null, '', window.location.pathname);
      await createNoteFromShare(title, text, url);
    }

    async function createNoteFromShare(
      title: string,
      text: string,
      url: string
    ): Promise<void> {
      let content = text;
      if (url) {
        content = content ? content + '\n\n' + url : url;
      }
      if (!title && !content) return;

      const note = await notes.createNote(title || '', content);
      if (note?.id) {
        goto(`/note/${note.id}`);
        pwa.notifySuccessfulAction();
      }
    }

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
    }

    document.addEventListener('focusin', viewportHandlers.handleFocusIn);
    document.addEventListener('focusout', viewportHandlers.handleFocusOut);
    document.addEventListener('touchstart', viewportHandlers.handleTouchStart, { passive: true });

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
      setShowUnlockModal: (value) => {
        showUnlockModal = value;
      },
      getCurrentNote: () => notes.getCurrentNote(),
      isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    });

    function registerActivityListeners() {
      if (activityListenersRegistered) return;
      document.addEventListener('mousemove', handleActivity);
      document.addEventListener('keydown', handleActivity);
      document.addEventListener('click', handleActivity);
      document.addEventListener('touchstart', handleActivity);
      activityListenersRegistered = true;
      console.log('[Layout] Activity listeners registered');
    }

    function unregisterActivityListeners() {
      if (!activityListenersRegistered) return;
      document.removeEventListener('mousemove', handleActivity);
      document.removeEventListener('keydown', handleActivity);
      document.removeEventListener('click', handleActivity);
      document.removeEventListener('touchstart', handleActivity);
      activityListenersRegistered = false;
      console.log('[Layout] Activity listeners unregistered');
    }

    // ✅ NEW: beforeunload handler for unsaved changes warning
    const handleBeforeUnload = (e: BeforeUnloadEvent) =>
      handleBeforeUnloadHelper(e, {
        isDirty: () => notes.getIsDirty(),
        isSyncing: () => syncManager.getIsSyncing(),
        warningMessage: get(_)('editor.unsaved_warning'),
      });

    window.addEventListener('beforeunload', handleBeforeUnload);

    // Mobile detection must run synchronously before async init to prevent
    // sidebar rendering in desktop mode on mobile devices
    viewportHandlers.handleResize();
    window.addEventListener('resize', viewportHandlers.debouncedHandleResize);

    // Start async initialization (non-blocking)
    void initializeAsync();

    return () => {
      document.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('resize', viewportHandlers.debouncedHandleResize);
      viewportHandlers.cleanup();
      websocket.disconnect();

      // Cleanup activity listeners
      unregisterActivityListeners();

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
    publicRoutes.some((route) => $page.url.pathname.startsWith(route)) || !auth.isAuthenticated()
  );

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

  // Watch for encryption locked error
  $effect(() => {
    if (notes.getError() === 'ENCRYPTION_LOCKED') {
      showUnlockModal = true;
    }
  });

  // Proactively show unlock modal when encryption gets locked while encrypted note is open
  // This catches the case where auto-lock triggers while user is actively editing
  $effect(() => {
    const currentNote = notes.getCurrentNote();
    const isLocked = !encryption.isEncryptionUnlocked();

    if (currentNote?.content_encrypted && isLocked && !showUnlockModal) {
      showUnlockModal = true;
    }
  });

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

  // Track in-app navigation for standalone PWA back button
  afterNavigate(({ to }) => {
    if (to?.url.pathname) {
      ui.pushNav(to.url.pathname);
    }
  });
</script>

<!-- Root container: full viewport height with flex column -->
<div class="h-screen-safe flex flex-col overflow-hidden">
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
  {:else if isPublic}
    <!-- Public pages (login/register) - no sidebar -->
    <div class="flex-1 overflow-auto">
      {@render children()}
    </div>
  {:else}
    <!-- Protected app pages - with sidebar -->
    <div
      class="flex flex-1 overflow-hidden"
      use:swipe={{
        direction: 'right',
        edge: 'left',
        onSwipe: () => ui.setSidebarOpen(true),
        enabled: () => ui.getIsMobile() && !ui.getSidebarOpen(),
      }}
    >
      <Sidebar />

      <main
        id="main-content"
        tabindex="-1"
        class="flex-1 overflow-hidden flex flex-col focus:outline-none relative z-0"
      >
        {#if ui.getIsMobile() && !ui.getIsKeyboardOpen() && !$page.url.pathname.startsWith('/note/')}
          <MobileHeader />
        {/if}
        <div class="flex-1 overflow-hidden">
          {@render children()}
        </div>
      </main>

      <!-- Mobile backdrop - after main for correct stacking -->
      {#if ui.getIsMobile() && ui.getSidebarOpen()}
        <div
          class="fixed inset-0 bg-black/40 backdrop-blur-md motion-reduce:backdrop-blur-none motion-reduce:bg-black/50 z-40"
          transition:fade={{ duration: 200 }}
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

<!-- Global Toast Notifications -->
<Toast />

<!-- Offline Banner (handles its own visibility based on offline state + sync state) -->
{#if network.getShowOfflineBanner() || syncManager.getIsSyncing()}
  <OfflineBanner />
{/if}

<!-- Conflict Dialog (shown when sync conflicts need resolution) -->
{#if syncManager.getConflicts().length > 0}
  <ConflictDialog />{/if}

<!-- Install Prompt -->
{#if showInstallPrompt && !isPublic}
  <InstallPrompt onClose={() => (showInstallPrompt = false)} />
{/if}

<!-- Global encryption unlock modal -->
<UnlockEncryptionModal
  bind:isOpen={showUnlockModal}
  onSuccess={() => {
    // Just clear the error, user can retry their action
    notes.clearError();
  }}
  onCancel={() => {
    // Clear error and navigate home if user cancels
    notes.clearError();
    goto('/');
  }}
/>

<!-- Global accessible dialogs -->
<ConfirmDialog />
<AlertDialog />
