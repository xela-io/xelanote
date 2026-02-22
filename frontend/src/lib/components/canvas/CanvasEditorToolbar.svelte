<script lang="ts">
  import {
    AlertCircle,
    Check,
    History,
    ImagePlus,
    Loader2,
    Lock,
    Maximize2,
    Minimize2,
    MoreVertical,
    RefreshCw,
    Save,
    WifiOff,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import { formatRelativeTime } from '$lib/utils/time';

  interface Props {
    note?: Note | null;
    isMobile?: boolean;
    saveStatus?: 'saved' | 'saving' | 'unsaved';
    uploading?: boolean;
    showMoreMenu?: boolean;
    syncing?: boolean;
    syncProgress?: { current: number; total: number };
    pendingCount?: number;
    isOnline?: boolean;
    isEncryptionUnlocked?: boolean;
    focusModeActive?: boolean;
    onTitleInput: (event: Event) => void;
    onSave: () => void;
    onUpload: () => void;
    onShowHistory: () => void;
    onToggleFocus: () => void;
    onOpenMoreMenu: (rect: DOMRect) => void;
  }

  const {
    note = null,
    isMobile = false,
    saveStatus = 'saved',
    uploading = false,
    showMoreMenu = false,
    syncing = false,
    syncProgress = { current: 0, total: 0 },
    pendingCount = 0,
    isOnline = true,
    isEncryptionUnlocked = true,
    focusModeActive = false,
    onTitleInput,
    onSave,
    onUpload,
    onShowHistory,
    onToggleFocus,
    onOpenMoreMenu,
  }: Props = $props();

  // Svelte Action: Scroll-Fade for toolbar overflow indicator
  function scrollFade(node: HTMLElement) {
    const wrapper = node.parentElement!;
    function update() {
      const hasOverflow = node.scrollWidth > node.clientWidth;
      const atEnd = node.scrollLeft + node.clientWidth >= node.scrollWidth - 2;
      wrapper.style.setProperty('--scroll-fade', hasOverflow && !atEnd ? '1' : '0');
    }
    update();
    node.addEventListener('scroll', update, { passive: true });
    const ro = new ResizeObserver(update);
    ro.observe(node);
    return {
      destroy() {
        node.removeEventListener('scroll', update);
        ro.disconnect();
      },
    };
  }

  function handleMoreMenuClick(e: MouseEvent) {
    onOpenMoreMenu((e.currentTarget as HTMLElement).getBoundingClientRect());
  }
</script>

<!-- Toolbar (fixed header, not in scroll container) -->
<div class="flex-shrink-0 z-10 border-b border-border bg-background">
  <!-- Mobile: single row | Desktop: 3-column grid for true centering -->
  <div
    class="flex items-center sm:grid sm:grid-cols-[minmax(120px,1fr)_minmax(0,auto)_1fr] sm:items-center px-2 sm:px-4 py-1.5 sm:py-2 gap-1 sm:gap-2"
    style:padding-left={isMobile ? '3.5rem' : undefined}
  >
    <!-- Left: Title + Last Updated + Sync status -->
    <div class="flex items-center gap-1.5 sm:gap-2 min-w-0">
      <input
        type="text"
        value={note?.title ?? ''}
        oninput={onTitleInput}
        autocorrect="on"
        autocapitalize="words"
        spellcheck="true"
        inputmode="text"
        aria-label={$_('component.editor.title_input')}
        class="text-lg font-semibold bg-transparent border-0 outline-none rounded px-1 min-w-0 flex-1 focus:ring-1 focus:ring-ring"
      />
      {#if note?.content_encrypted}
        <Lock size={14} class="flex-shrink-0 text-muted-foreground" />
      {/if}
      {#if isMobile}
        <span class="flex-shrink-0">
          {#if saveStatus === 'saving'}
            <Loader2 size={16} class="animate-spin text-muted-foreground" />
          {:else if saveStatus === 'saved'}
            <Check size={16} class="text-success" />
          {:else if saveStatus === 'unsaved'}
            <AlertCircle size={16} class="text-warning" />
          {/if}
        </span>
      {/if}
      {#if note?.updated_at}
        <span
          class="text-xs text-muted-foreground whitespace-nowrap flex-shrink-0 hidden sm:inline"
          title={new Date(note.updated_at).toLocaleString()}
        >
          {$_('component.editor.last_updated', {
            values: { date: formatRelativeTime(note.updated_at, $_) },
          })}
        </span>
      {/if}
      <!-- Offline/Sync status pill -->
      {#if syncing}
        <div
          class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-blue-500"
        >
          <RefreshCw size={12} class="animate-spin" />
          <span
            >{syncProgress.total > 0
              ? `${syncProgress.current}/${syncProgress.total}`
              : 'Sync...'}</span
          >
        </div>
      {:else if !isOnline && !isEncryptionUnlocked}
        <div
          class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-600"
        >
          <Lock size={12} />
          <span>Gesperrt</span>
        </div>
      {:else if !isOnline && pendingCount > 0}
        <div
          class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-500"
        >
          <WifiOff size={12} />
          <span>{pendingCount}</span>
        </div>
      {:else if !isOnline}
        <div
          class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-500"
        >
          <WifiOff size={12} />
          <span>Offline</span>
        </div>
      {/if}
    </div>

    <!-- Center + Right: wrapped together so the ⋮ button stays inline on mobile -->
    <div class="flex items-center gap-1 flex-1 min-w-0 sm:contents">
      <div class="flex items-center justify-center gap-1 min-w-0 flex-1">
        <div class="toolbar-scroll-wrapper">
          <div
            class="toolbar-buttons flex items-center gap-1"
            role="toolbar"
            aria-label={$_('component.editor.toolbar.editor_toolbar')}
            use:scrollFade
          >
            <!-- Save button -->
            <button
              type="button"
              onclick={onSave}
              disabled={saveStatus === 'saved' || saveStatus === 'saving'}
              class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.save')}
            >
              <Save size={16} />
            </button>

            <!-- Upload image button -->
            <button
              type="button"
              onclick={onUpload}
              disabled={uploading}
              class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.upload')}
            >
              <ImagePlus size={16} />
            </button>

            <!-- Divider -->
            <div class="w-px h-6 bg-border mx-1 flex-shrink-0"></div>

            <!-- History button -->
            <button
              type="button"
              onclick={onShowHistory}
              class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.history')}
            >
              <History size={16} />
            </button>

            <!-- Focus Mode toggle - hidden on mobile -->
            {#if !isMobile}
              <button
                type="button"
                onclick={onToggleFocus}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                class:bg-accent={focusModeActive}
                aria-label={$_('component.editor.toolbar.focus_mode')}
                aria-pressed={focusModeActive}
              >
                {#if focusModeActive}
                  <Minimize2 size={16} />
                {:else}
                  <Maximize2 size={16} />
                {/if}
              </button>
            {/if}
          </div>
        </div>
      </div>

      <!-- Right: More menu button (third grid column on desktop, inline on mobile) -->
      <button
        onclick={handleMoreMenuClick}
        class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn sm:justify-self-end"
        aria-label={$_('component.editor.toolbar.more_options')}
        aria-expanded={showMoreMenu}
        aria-haspopup="menu"
        title={$_('component.editor.toolbar.more_options')}
      >
        <MoreVertical size={16} />
      </button>
    </div>
  </div>
</div>
