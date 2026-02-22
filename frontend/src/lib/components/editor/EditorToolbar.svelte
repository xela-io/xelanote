<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import {
    AlertCircle,
    Check,
    Columns,
    Edit,
    Eye,
    History,
    ImagePlus,
    ListTodo,
    Loader2,
    Lock,
    Maximize2,
    Menu,
    Minimize2,
    MoreVertical,
    RefreshCw,
    Save,
    ScanEye,
    Table2,
    Wand2,
    WifiOff,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import { FEATURE_FLAGS } from '$lib/config';
  import { formatRelativeTime } from '$lib/utils/time';

  import SpellCheckToggle from '../SpellCheckToggle.svelte';

  type EditorMode = 'edit' | 'split' | 'preview' | 'live';

  interface Props {
    note?: Note | null;
    editorView?: EditorView | undefined;
    isMobile?: boolean;
    editorMode?: EditorMode;
    autoSaveStatus?: 'saving' | 'saved' | 'error' | 'idle' | 'pending';
    autoSaveEnabled?: boolean;
    isDirty?: boolean;
    isSaving?: boolean;
    uploading?: boolean;
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
    onOpenSidebar: () => void;
    onSetEditorMode: (mode: EditorMode) => void;
    onInsertTask: () => void;
    onInsertTable: () => void;
    onSave: () => void;
    onUpload: () => void;
    onShowHistory: () => void;
    onToggleFocus: () => void;
    onToggleAutosave: () => void;
    onAIActions: (rect: DOMRect) => void;
    onOpenMoreMenu: (rect: DOMRect) => void;
  }

  const {
    note = null,
    editorView,
    isMobile = false,
    editorMode = 'edit',
    autoSaveStatus = 'idle',
    autoSaveEnabled = false,
    isDirty = false,
    isSaving = false,
    uploading = false,
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
    onOpenSidebar,
    onSetEditorMode,
    onInsertTask,
    onInsertTable,
    onSave,
    onUpload,
    onShowHistory,
    onToggleFocus,
    onToggleAutosave,
    onAIActions,
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

  function handleAIActionsClick(e: MouseEvent) {
    if (!aiEnabled) {
      return;
    }
    onAIActions((e.currentTarget as HTMLElement).getBoundingClientRect());
  }

  function handleMoreMenuClick(e: MouseEvent) {
    onOpenMoreMenu((e.currentTarget as HTMLElement).getBoundingClientRect());
  }

  function cycleEditorMode() {
    const modes: EditorMode[] = isMobile
      ? ['live', 'edit', 'preview']
      : ['live', 'edit', 'preview', 'split'];
    const currentIndex = modes.indexOf(editorMode);
    const nextMode = modes[(currentIndex + 1) % modes.length] ?? modes[0];
    onSetEditorMode(nextMode);
  }

  function getModeLabel(mode: EditorMode): string {
    if (mode === 'live') return $_('component.editor.toolbar.mode_live');
    if (mode === 'edit') return $_('component.editor.toolbar.mode_edit');
    if (mode === 'preview') return $_('component.editor.toolbar.mode_preview');
    return $_('component.editor.toolbar.mode_split');
  }
</script>

<!-- Toolbar (fixed header, not in scroll container) -->
<div class="flex-shrink-0 z-10 border-b border-border bg-background">
  <!-- Mobile: flex-col stacked | Desktop: 3-column grid for true centering -->
  <div
    class="flex flex-col sm:grid sm:grid-cols-[minmax(120px,1fr)_minmax(0,auto)_1fr] sm:items-center px-4 py-2 gap-2"
  >
    <!-- Left: Journal title (read-only) or metadata + Sync status -->
    <div class="flex items-center gap-2 min-w-0">
      {#if note?.note_type === 'journal'}
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

    <!-- Center + Right: wrapped together so the ⋮ button stays inline on mobile -->
    <div class="flex items-center gap-1 sm:contents">
      <div class="flex items-center justify-center gap-1 min-w-0 flex-1">
        <div class="toolbar-scroll-wrapper">
          <div
            class="toolbar-buttons flex items-center gap-1"
            role="toolbar"
            aria-label={$_('component.editor.toolbar.editor_toolbar')}
            use:scrollFade
          >
            <!-- Sidebar toggle - always visible on mobile since MobileHeader is hidden on note pages -->
            {#if isMobile}
              <button
                type="button"
                onclick={onOpenSidebar}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                aria-label={$_('component.editor.toolbar.open_menu')}
              >
                <Menu size={16} />
              </button>
            {/if}

            {#if FEATURE_FLAGS.livePreview}
              <button
                type="button"
                onclick={cycleEditorMode}
                class="h-8 w-8 p-0 hover:bg-accent rounded-md toolbar-btn flex-shrink-0 inline-flex items-center justify-center"
                class:bg-accent={editorMode === 'live' ||
                  editorMode === 'edit' ||
                  editorMode === 'preview' ||
                  editorMode === 'split'}
                aria-label={getModeLabel(editorMode)}
                title={getModeLabel(editorMode)}
              >
                {#if editorMode === 'live'}
                  <ScanEye size={16} />
                {:else if editorMode === 'edit'}
                  <Edit size={16} />
                {:else if editorMode === 'preview'}
                  <Eye size={16} />
                {:else}
                  <Columns size={16} />
                {/if}
              </button>
            {/if}

            <!-- Formatting tools - always visible -->
            {#if FEATURE_FLAGS.taskLists}
              <button
                type="button"
                onclick={onInsertTask}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                aria-label={$_('component.editor.toolbar.task')}
              >
                <ListTodo size={16} />
              </button>
            {/if}

            <button
              type="button"
              onclick={onInsertTable}
              class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.table')}
            >
              <Table2 size={16} />
            </button>

            <!-- Divider -->
            <div class="w-px h-6 bg-border mx-1 flex-shrink-0"></div>

            <!-- Save button - always visible -->
            <button
              type="button"
              onclick={onSave}
              disabled={!isDirty || isSaving}
              class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.save')}
            >
              <Save size={16} />
            </button>

            <!-- Upload button - always visible -->
            <button
              type="button"
              onclick={onUpload}
              disabled={uploading}
              class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
              aria-label={$_('component.editor.toolbar.upload')}
            >
              <ImagePlus size={16} />
            </button>

            <!-- History button - always visible -->
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

            <!-- Spell Check toggle - only in edit mode with AI enabled -->
            {#if FEATURE_FLAGS.spellCheck && showSpellCheck}
              <SpellCheckToggle {editorView} />
            {/if}

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

            <!-- Auto-save toggle -->
            <button
              type="button"
              onclick={onToggleAutosave}
              class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
              class:bg-accent={autoSaveEnabled}
              aria-label={$_('component.editor.toolbar.autosave')}
              aria-pressed={autoSaveEnabled}
            >
              {#if autoSaveStatus === 'saving'}
                <Loader2 size={16} class="animate-spin" />
              {:else if autoSaveStatus === 'saved'}
                <Check size={16} class="text-success" />
              {:else if autoSaveStatus === 'error'}
                <AlertCircle size={16} class="text-destructive" />
              {:else}
                <Save size={16} class={autoSaveEnabled ? 'text-primary' : ''} />
              {/if}
            </button>
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
