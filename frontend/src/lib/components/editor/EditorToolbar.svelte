<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import {
    AlertCircle,
    Check,
    History,
    Loader2,
    Lock,
    Maximize2,
    Minimize2,
    MoreVertical,
    Plus,
    RefreshCw,
    Save,
    Wand2,
    WifiOff,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import { FEATURE_FLAGS } from '$lib/config';
  import { formatRelativeTime } from '$lib/utils/time';

  import SpellCheckToggle from '../SpellCheckToggle.svelte';
  import EditorModeSelector from './EditorModeSelector.svelte';

  type EditorMode = 'edit' | 'split' | 'preview' | 'live';

  interface Props {
    note?: Note | null;
    editorView?: EditorView | undefined;
    isMobile?: boolean;
    editorMode?: EditorMode;
    autoSaveStatus?: 'saving' | 'saved' | 'error' | 'idle' | 'pending';
    isDirty?: boolean;
    isSaving?: boolean;
    showAIActionsDropdown?: boolean;
    showMoreMenu?: boolean;
    aiEnabled?: boolean;
    syncing?: boolean;
    syncProgress?: { current: number; total: number };
    pendingCount?: number;
    isOnline?: boolean;
    isEncryptionUnlocked?: boolean;
    focusModeActive?: boolean;
    showSpellCheck?: boolean;
    showInsertMenu?: boolean;
    onSetEditorMode: (mode: EditorMode) => void;
    onSave: () => void;
    onShowHistory: () => void;
    onToggleFocus: () => void;
    onAIActions: (rect: DOMRect) => void;
    onOpenInsertMenu: (rect: DOMRect) => void;
    onOpenMoreMenu: (rect: DOMRect) => void;
  }

  const {
    note = null,
    editorView,
    isMobile = false,
    editorMode = 'edit',
    autoSaveStatus = 'idle',
    isDirty = false,
    isSaving = false,
    showAIActionsDropdown = false,
    showMoreMenu = false,
    aiEnabled = false,
    syncing = false,
    syncProgress = { current: 0, total: 0 },
    pendingCount = 0,
    isOnline = true,
    isEncryptionUnlocked = true,
    focusModeActive = false,
    showSpellCheck = false,
    showInsertMenu = false,
    onSetEditorMode,
    onSave,
    onShowHistory,
    onToggleFocus,
    onAIActions,
    onOpenInsertMenu,
    onOpenMoreMenu,
  }: Props = $props();

  const showSaveButton = $derived.by(() => !isMobile || isDirty || isSaving);
  const showHistoryButton = $derived.by(() => !isMobile);
  const showAIPillContent = $derived.by(
    () =>
      (aiEnabled && (editorMode === 'edit' || editorMode === 'split' || editorMode === 'live')) ||
      (!isMobile && FEATURE_FLAGS.spellCheck && showSpellCheck) ||
      !isMobile
  );
  const showSaveHistoryPill = $derived.by(() => showSaveButton || showHistoryButton);

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

  function handleAIActionsClick(e: MouseEvent) {
    if (!aiEnabled) {
      return;
    }
    onAIActions((e.currentTarget as HTMLElement).getBoundingClientRect());
  }

  function handleMoreMenuClick(e: MouseEvent) {
    onOpenMoreMenu((e.currentTarget as HTMLElement).getBoundingClientRect());
  }

  function handleInsertMenuClick(e: MouseEvent) {
    onOpenInsertMenu((e.currentTarget as HTMLElement).getBoundingClientRect());
  }
</script>

<!-- Toolbar (fixed header, not in scroll container) -->
<div class="flex-shrink-0 z-10 border-b border-border/70 bg-background/85 backdrop-blur-md">
  <!-- Mobile: single row | Desktop: 3-column grid for true centering -->
  <div
    class="flex items-center sm:grid sm:grid-cols-[minmax(120px,1fr)_minmax(0,auto)_1fr] sm:items-center px-2 sm:px-4 py-2 sm:py-2.5 gap-1 sm:gap-2"
  >
    <!-- Left: metadata (desktop only shows full info, mobile shows compact) -->
    <div class="flex items-center gap-1.5 sm:gap-2 min-w-0 shrink-0 sm:shrink">
      <MobileSidebarInlineToggle />
      {#if note?.note_type === 'journal' && !isMobile}
        <span
          class="text-lg font-semibold px-1 min-w-0 flex-1 truncate cursor-default opacity-70"
          aria-label={$_('component.editor.title_input')}
        >
          {note.title}
        </span>
      {/if}
      {#if note?.content_encrypted}
        <Lock size={14} class="flex-shrink-0 text-muted-foreground" />
      {/if}
      {#if isMobile}
        <span class="flex-shrink-0">
          {#if autoSaveStatus === 'saving'}
            <Loader2 size={16} class="animate-spin text-muted-foreground" />
          {:else if autoSaveStatus === 'saved'}
            <Check size={16} class="text-success" />
          {:else if autoSaveStatus === 'error'}
            <AlertCircle size={16} class="text-destructive" />
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

    <!-- Center + Right: toolbar buttons + more menu -->
    <div class="flex items-center gap-1 flex-1 min-w-0 sm:contents">
      <div class="flex items-center justify-center gap-1 min-w-0 flex-1">
        <div class="toolbar-scroll-wrapper">
          <div
            class="toolbar-buttons flex items-center gap-1.5"
            role="toolbar"
            aria-label={$_('component.editor.toolbar.editor_toolbar')}
            use:scrollFade
          >
            <div class="toolbar-group-pill">
              {#if FEATURE_FLAGS.livePreview}
                <EditorModeSelector {editorMode} {isMobile} {onSetEditorMode} />
              {/if}

              <button
                type="button"
                onclick={handleInsertMenuClick}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn inline-flex items-center gap-1"
                class:bg-accent={showInsertMenu}
                aria-label={$_('component.editor.table_insert.insert')}
                aria-expanded={showInsertMenu}
                aria-haspopup="menu"
                title={$_('component.editor.table_insert.insert')}
              >
                <Plus size={16} />
                <span class="hidden sm:inline text-xs">
                  {$_('component.editor.table_insert.insert')}
                </span>
              </button>
            </div>

            {#if showSaveHistoryPill}
              <div class="toolbar-group-pill">
                {#if showSaveButton}
                  <button
                    type="button"
                    onclick={onSave}
                    disabled={!isDirty || isSaving}
                    class="p-2 rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn inline-flex items-center gap-1 hover:bg-accent"
                    class:bg-primary={isDirty && !isSaving}
                    class:text-primary-foreground={isDirty && !isSaving}
                    aria-label={$_('component.editor.toolbar.save')}
                    title={$_('component.editor.toolbar.save')}
                  >
                    {#if isSaving}
                      <Loader2 size={16} class="animate-spin" />
                    {:else}
                      <Save size={16} />
                    {/if}
                    {#if isDirty || isSaving}
                      <span class="hidden md:inline text-xs">
                        {$_('component.editor.toolbar.save_short')}
                      </span>
                    {/if}
                  </button>
                {/if}

                {#if showHistoryButton}
                  <button
                    type="button"
                    onclick={onShowHistory}
                    class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn hidden sm:inline-flex"
                    aria-label={$_('component.editor.toolbar.history')}
                    title={$_('component.editor.toolbar.history')}
                  >
                    <History size={16} />
                  </button>
                {/if}
              </div>
            {/if}

            {#if showAIPillContent}
              <div class="toolbar-group-pill">
                <!-- AI Actions button - only when AI enabled -->
                {#if aiEnabled && (editorMode === 'edit' || editorMode === 'split' || editorMode === 'live')}
                  <div class="flex-shrink-0">
                    <button
                      type="button"
                      onclick={handleAIActionsClick}
                      class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                      class:bg-accent={showAIActionsDropdown}
                      aria-label={$_('component.editor.ai_actions')}
                      title={$_('component.editor.ai_actions_tooltip')}
                      aria-expanded={showAIActionsDropdown}
                      aria-haspopup="menu"
                    >
                      <Wand2 size={16} />
                    </button>
                  </div>
                {/if}

                <!-- Spell Check toggle - only in edit mode with AI enabled -->
                {#if FEATURE_FLAGS.spellCheck && showSpellCheck && !isMobile}
                  <SpellCheckToggle {editorView} />
                {/if}

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

<style>
  .toolbar-group-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.125rem;
    padding: 0.16rem;
    border-radius: 0.7rem;
    border: 1px solid var(--surface-panel-border-strong);
    background: var(--surface-panel-bg);
    box-shadow: inset 0 1px 0 var(--surface-panel-inset-highlight);
    flex-shrink: 0;
  }

  :global(.toolbar-btn) {
    transition:
      background-color var(--duration-fast) var(--ease-default),
      color var(--duration-fast) var(--ease-default),
      border-color var(--duration-fast) var(--ease-default);
    border-radius: 0.55rem;
  }

  @media (max-width: 639px) {
    .toolbar-group-pill {
      gap: 0;
      border-radius: 0.5rem;
      padding: 0.125rem;
    }
  }
</style>
