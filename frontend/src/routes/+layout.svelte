<script lang="ts">
  import '../app.css';

  import type { ComponentType } from 'svelte';
  import { onMount, untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { _, locale } from 'svelte-i18n';

  import { browser } from '$app/environment';
  import { beforeNavigate,goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { swipe } from '$lib/actions/swipe';
  import { setOnOfflineEnqueue as setApiOfflineCallback } from '$lib/api';
  import * as api from '$lib/api';
  import ConflictDialog from '$lib/components/ConflictDialog.svelte';
  import DesktopTitleBar from '$lib/components/DesktopTitleBar.svelte';
  import InstallPrompt from '$lib/components/InstallPrompt.svelte';
  import MobileHeader from '$lib/components/MobileHeader.svelte';
  import OfflineBanner from '$lib/components/OfflineBanner.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import AlertDialog from '$lib/components/ui/AlertDialog.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import UnlockEncryptionModal from '$lib/components/UnlockEncryptionModal.svelte';
  import { clearPersistedKEK,initKEKDatabase } from '$lib/crypto/kek-persistence';
  import { initSodium } from '$lib/crypto/sodium';
  import { initOfflineDatabase } from '$lib/offline/offline-queue';
  import * as syncManager from '$lib/offline/sync-manager.svelte';
  import { initializeApp } from '$lib/routes/layout/initialize';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import * as autosave from '$lib/stores/autosave.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as history from '$lib/stores/history.svelte';
  import * as network from '$lib/stores/network.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as tokenRefresh from '$lib/stores/token-refresh.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import * as websocket from '$lib/stores/websocket.svelte';
  import { registerPwaUpdates } from '$lib/routes/layout/pwa';

  // ✅ Service Worker Registration (PWA) mit isDirty Gate
  // NOTE: Module-level variable, but only accessed within browser guards
  let updateCheckInterval: ReturnType<typeof setInterval> | null = null;
  let cleanupErrorHandler: (() => void) | null = null;

  // Early mobile detection: runs before onMount to prevent sidebar flash on mobile
  if (browser) {
    const earlyMobile = window.innerWidth < 768;
    ui.setIsMobile(earlyMobile);
    if (earlyMobile) {
      ui.setSidebarOpen(false);
      if (ui.getEditorMode() === 'split') {
        ui.setEditorMode('edit');
      }
    }
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
    if (!isAuth && !isPublicRoute) {
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
    };

    function handleResize() {
      const mobile = window.innerWidth < 768;
      const wasMobile = ui.getIsMobile();
      ui.setIsMobile(mobile);

      // When switching to mobile: close drawer, exit split mode
      // When switching to desktop: open sidebar
      if (mobile && !wasMobile) {
        ui.setSidebarOpen(false);
        if (ui.getEditorMode() === 'split') {
          ui.setEditorMode('edit');
        }
      }
      if (!mobile && wasMobile) {
        ui.setSidebarOpen(true);
      }
    }

    // Debounced resize handler to prevent rapid updates during iOS keyboard animation
    function debouncedHandleResize() {
      if (resizeTimeout) clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(handleResize, 150);
    }

    // Keyboard detection via Visual Viewport API + focus fallback (for Firefox iOS)
    let inputFocused = false;

    function isInputElement(el: Element | null): boolean {
      if (!el) return false;
      const tagName = el.tagName;
      // Standard form inputs
      if (tagName === 'INPUT' || tagName === 'TEXTAREA') return true;
      // Contenteditable elements (including CodeMirror)
      if (el.getAttribute('contenteditable') === 'true') return true;
      // Check for CodeMirror editor (element or ancestor)
      if (el.closest('.cm-editor')) return true;
      return false;
    }

    function updateKeyboardState() {
      // Visual Viewport check (works well in Safari iOS)
      let viewportKeyboard = false;
      if (window.visualViewport) {
        const viewportHeight = window.visualViewport.height;
        const windowHeight = window.innerHeight;
        viewportKeyboard = windowHeight - viewportHeight > 150;
      }

      // On mobile: keyboard is open if either viewport shrinks OR input is focused
      // This handles Firefox iOS which doesn't properly report viewport changes
      const keyboardOpen = viewportKeyboard || (ui.getIsMobile() && inputFocused);
      ui.setIsKeyboardOpen(keyboardOpen);
    }

    function handleVisualViewportResize() {
      updateKeyboardState();
    }

    function handleFocusIn(e: FocusEvent) {
      const target = e.target as Element | null;
      if (target && isInputElement(target)) {
        inputFocused = true;
        updateKeyboardState();
      }
    }

    function handleFocusOut() {
      // Delay to handle focus moving between inputs
      setTimeout(() => {
        if (!isInputElement(document.activeElement)) {
          inputFocused = false;
          updateKeyboardState();
        }
      }, 100);
    }

    // Touchstart fallback: Firefox iOS may not fire focusin reliably for CodeMirror
    function handleTouchStart(e: TouchEvent) {
      if (!ui.getIsMobile()) return;
      const target = e.target as Element | null;
      if (target && isInputElement(target)) {
        // Delay slightly to let focus settle
        setTimeout(() => {
          inputFocused = true;
          updateKeyboardState();
        }, 50);
      }
    }

    // Initialize Visual Viewport listener for keyboard detection
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', handleVisualViewportResize);
    }

    // Focus-based fallback for Firefox iOS (and other browsers with incomplete Visual Viewport support)
    document.addEventListener('focusin', handleFocusIn);
    document.addEventListener('focusout', handleFocusOut);
    // Touch fallback for browsers where focusin doesn't fire reliably
    document.addEventListener('touchstart', handleTouchStart, { passive: true });

    function handleKeydown(e: KeyboardEvent) {
      if (!auth.isAuthenticated()) return;

      // Ctrl+P: Quick Switcher
      if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
        e.preventDefault();
        ui.toggleQuickSwitcher();
      }

      // Ctrl+G: Graph (only if feature enabled)
      if ((e.ctrlKey || e.metaKey) && e.key === 'g' && features.getGraphFeatureEnabled()) {
        e.preventDefault();
        goto('/graph');
      }

      // Ctrl+/: Markdown Guide Dropdown
      if ((e.ctrlKey || e.metaKey) && e.key === '/') {
        e.preventDefault();
        ui.toggleMarkdownGuideDropdown();
      }

      // Ctrl+S: Save
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        notes.saveNote();
      }

      // Ctrl+Z: Undo
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
        if (history.history.canUndo) {
          e.preventDefault();
          history.undo();
        }
      }

      // Ctrl+Shift+Z or Ctrl+Y: Redo
      if (
        ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'z') ||
        ((e.ctrlKey || e.metaKey) && e.key === 'y')
      ) {
        if (history.history.canRedo) {
          e.preventDefault();
          history.redo();
        }
      }
    }

    function handleActivity() {
      autoLock.recordActivity();
      tokenRefresh.recordActivity();

      // FIX: Show unlock modal immediately if encryption was locked while user was away
      // This prevents data loss when user returns and starts typing before realizing
      // encryption is locked (the changes would be lost after unlock + reload)
      const currentNote = notes.getCurrentNote();
      if (currentNote?.content_encrypted && !encryption.isEncryptionUnlocked()) {
        showUnlockModal = true;
      }
    }

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
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (notes.getIsDirty() || syncManager.getIsSyncing()) {
        e.preventDefault();
        e.returnValue = 'Sie haben ungespeicherte Änderungen';
        return e.returnValue;
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);

    // Mobile detection must run synchronously before async init to prevent
    // sidebar rendering in desktop mode on mobile devices
    handleResize();
    window.addEventListener('resize', debouncedHandleResize);

    // Start async initialization (non-blocking)
    void initializeAsync();

    return () => {
      document.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('resize', debouncedHandleResize);
      if (window.visualViewport) {
        window.visualViewport.removeEventListener('resize', handleVisualViewportResize);
      }
      document.removeEventListener('focusin', handleFocusIn);
      document.removeEventListener('focusout', handleFocusOut);
      document.removeEventListener('touchstart', handleTouchStart);
      if (resizeTimeout) clearTimeout(resizeTimeout);
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
    QuickSwitcherComponent = module.default as unknown as ComponentType;
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

  // Dynamic HTML lang attribute
  $effect(() => {
    if (browser) {
      document.documentElement.lang = $locale || 'de';
    }
  });

  // Warn about unsaved changes when navigating (only if auto-save is disabled)
  beforeNavigate(({ cancel }) => {
    if (!autosave.getAutoSaveEnabled() && notes.getIsDirty()) {
      if (!confirm($_('editor.unsaved_warning'))) {
        cancel();
      }
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
      style="background-color: var(--color-background, #111);"
    >
      <div class="text-center">
        <div
          class="animate-spin rounded-full h-12 w-12 border-b-2 mx-auto mb-4"
          style="border-color: var(--color-muted-foreground, #666);"
        ></div>
        <p style="color: var(--color-muted-foreground, #888);">Laden...</p>
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
          class="fixed inset-0 bg-black/50 z-40"
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
