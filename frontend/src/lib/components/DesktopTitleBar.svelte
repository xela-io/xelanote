<script lang="ts">
  import { Maximize2, Minimize2, Minus, X } from 'lucide-svelte';
  import { onMount } from 'svelte';

  import { type DesktopBridge, getDesktopBridge } from '$lib/desktop';

  import Logo from './Logo.svelte';

  let isMaximized = $state(false);
  let bridge: DesktopBridge | null = $state(null);
  let cleanup: (() => void) | null = null;

  // Use reactive state for desktop detection to handle timing issues
  // The preload script may not have finished exposing electronAPI when config.ts was first evaluated
  let isDesktopApp = $state(false);
  let isTauriApp = $state(false);
  let isElectronApp = $state(false);

  onMount(() => {
    let isActive = true;

    void (async () => {
      // Check directly for electronAPI - more reliable than the config functions
      // The preload script exposes window.electronAPI via contextBridge
      const hasElectronAPI =
        typeof window !== 'undefined' &&
        'electronAPI' in window &&
        window.electronAPI !== undefined;
      const hasTauriAPI = typeof window !== 'undefined' && '__TAURI__' in window;

      isElectronApp = hasElectronAPI;
      isTauriApp = hasTauriAPI;
      isDesktopApp = hasElectronAPI || hasTauriAPI;

      if (!isDesktopApp) return;

      try {
        const resolvedBridge = await getDesktopBridge();
        if (!isActive) return;
        bridge = resolvedBridge;
        isMaximized = await resolvedBridge.isMaximized();

        // Track maximize state changes
        cleanup = resolvedBridge.onMaximizeChange((maximized) => {
          isMaximized = maximized;
        });
      } catch (err) {
        console.error('DesktopTitleBar error:', err);
      }
    })();

    return () => {
      isActive = false;
      cleanup?.();
    };
  });

  async function minimize() {
    await bridge?.minimize();
  }

  async function toggleMaximize() {
    await bridge?.toggleMaximize();
  }

  async function close() {
    await bridge?.close();
  }

  // Handle drag for both Tauri and Electron
  function handleMouseDown(event: MouseEvent) {
    // Only handle drag on the titlebar itself, not on buttons
    const target = event.target as HTMLElement;
    if (target.closest('.titlebar-buttons')) return;

    // For Tauri, the data-tauri-drag-region handles this
    // For Electron, we use -webkit-app-region: drag in CSS
  }
</script>

{#if isDesktopApp}
  <!-- Double-click to maximize -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    data-tauri-drag-region={isTauriApp ? true : undefined}
    class="titlebar"
    class:electron={isElectronApp}
    ondblclick={toggleMaximize}
    onmousedown={handleMouseDown}
  >
    <span class="titlebar-title"><Logo size="sm" /></span>

    <div class="titlebar-buttons">
      <button onclick={minimize} aria-label="Minimize" title="Minimize">
        <Minus size={16} />
      </button>
      <button
        onclick={toggleMaximize}
        aria-label={isMaximized ? 'Restore' : 'Maximize'}
        title={isMaximized ? 'Restore' : 'Maximize'}
      >
        {#if isMaximized}
          <Minimize2 size={16} />
        {:else}
          <Maximize2 size={16} />
        {/if}
      </button>
      <button onclick={close} class="close" aria-label="Close" title="Close">
        <X size={16} />
      </button>
    </div>
  </div>
{/if}

<style>
  .titlebar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 32px;
    background: var(--color-sidebar-background);
    color: var(--color-sidebar-foreground);
    border-bottom: 1px solid var(--color-sidebar-border);
    user-select: none;
    -webkit-user-select: none;
    padding: 0 8px;
    font-size: 12px;
    transition: background-color 150ms cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* Electron: use CSS for drag region */
  .titlebar.electron {
    -webkit-app-region: drag;
  }

  .titlebar-title {
    font-weight: 500;
    letter-spacing: 0.02em;
    opacity: 0.9;
  }

  .titlebar-buttons {
    display: flex;
    gap: 2px;
    /* Buttons should not be draggable */
    -webkit-app-region: no-drag;
  }

  .titlebar-buttons button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 28px;
    background: transparent;
    border: none;
    color: var(--color-sidebar-foreground);
    cursor: pointer;
    border-radius: 4px;
    transition:
      background-color 150ms cubic-bezier(0.4, 0, 0.2, 1),
      color 150ms cubic-bezier(0.4, 0, 0.2, 1),
      transform 100ms cubic-bezier(0.4, 0, 0.2, 1);
  }

  .titlebar-buttons button:hover {
    background: var(--color-sidebar-accent);
    color: var(--color-sidebar-primary);
  }

  .titlebar-buttons button:active {
    transform: scale(0.98);
  }

  .titlebar-buttons button.close:hover {
    background: var(--color-destructive);
    color: var(--color-destructive-foreground);
  }

  .titlebar-buttons button:focus-visible {
    outline: 2px solid var(--color-sidebar-ring);
    outline-offset: 2px;
  }

  @media (prefers-reduced-motion: reduce) {
    .titlebar,
    .titlebar-buttons button {
      transition: none;
    }
    .titlebar-buttons button:active {
      transform: none;
      /* Alternatives Feedback ohne Animation */
      background: var(--color-sidebar-accent);
      opacity: 0.9;
    }
  }
</style>
