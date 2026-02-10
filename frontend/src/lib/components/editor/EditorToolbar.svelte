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
    Wand2,
    WifiOff,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import { FEATURE_FLAGS } from '$lib/config';

  import Breadcrumb from '../Breadcrumb.svelte';
  import SpellCheckToggle from '../SpellCheckToggle.svelte';

  type EditorMode = 'edit' | 'split' | 'preview';

  export let note: Note | null = null;
  export let editorView: EditorView | undefined;
  export let isMobile = false;
  export let editorMode: EditorMode = 'edit';
  export let autoSaveStatus: 'saving' | 'saved' | 'error' | 'idle' | 'pending' = 'idle';
  export let autoSaveEnabled = false;
  export let isDirty = false;
  export let isSaving = false;
  export let uploading = false;
  export let showAIActionsDropdown = false;
  export let showMoreMenu = false;
  export let aiEnabled = false;
  export let syncing = false;
  export let syncProgress = { current: 0, total: 0 };
  export let pendingCount = 0;
  export let isOnline = true;
  export let isEncryptionUnlocked = true;
  export let focusModeActive = false;
  export let showSpellCheck = false;

  export let onTitleInput: (event: Event) => void;
  export let onOpenSidebar: () => void;
  export let onSetEditorMode: (mode: EditorMode) => void;
  export let onInsertTask: () => void;
  export let onSave: () => void;
  export let onUpload: () => void;
  export let onShowHistory: () => void;
  export let onToggleFocus: () => void;
  export let onToggleAutosave: () => void;
  export let onAIActions: (rect: DOMRect) => void;
  export let onOpenMoreMenu: (rect: DOMRect) => void;

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
</script>

<!-- Toolbar (fixed header, not in scroll container) -->
<div class="flex-shrink-0 z-10 border-b border-border bg-background">
  <!-- Breadcrumb Navigation - hidden on mobile -->
  {#if !isMobile && note}
    <div class="px-4 pt-2">
      <Breadcrumb folderPath={note.folder_path} noteTitle={note.title} />
    </div>
  {/if}

  <!-- Mobile: Two-row layout | Desktop: Single-row layout -->
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between px-4 py-2 gap-2">
    <!-- Title (auto-sized on mobile so save icon sits next to text, limited on desktop) -->
    <div class="flex items-center gap-1 flex-shrink min-w-0 sm:max-w-[30%]">
      <input
        type="text"
        value={note?.title ?? ''}
        oninput={onTitleInput}
        autocorrect="on"
        autocapitalize="words"
        spellcheck="true"
        inputmode="text"
        aria-label={$_('component.editor.title_input')}
        class="text-lg font-semibold bg-transparent border-0 outline-none focus:ring-1 focus:ring-ring rounded px-1 min-w-0 {isMobile
          ? ''
          : 'w-full'}"
        style={isMobile ? `width: ${Math.max((note?.title ?? '').length, 2) + 1}ch` : ''}
      />
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
    </div>

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

    <!-- Buttons (horizontally scrollable with fade indicator) + fixed More button -->
    <div class="flex items-center gap-1 flex-1 min-w-0">
      <div class="toolbar-scroll-wrapper">
        <div class="toolbar-buttons flex items-center gap-1" use:scrollFade>
          <!-- Sidebar toggle - always visible on mobile since MobileHeader is hidden on note pages -->
          {#if isMobile}
            <button
              type="button"
              onclick={onOpenSidebar}
              class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
              aria-label="Menü öffnen"
            >
              <Menu size={16} />
            </button>
          {/if}

          <!-- Editor mode toggles - always visible -->
          <div
            class="flex rounded-md border border-border flex-shrink-0"
            role="group"
            aria-label={$_('component.editor.toolbar.mode_group')}
          >
            <button
              type="button"
              onclick={() => onSetEditorMode('edit')}
              class="p-2 hover:bg-accent rounded-l-md toolbar-btn"
              class:rounded-r-md={isMobile}
              class:bg-accent={editorMode === 'edit'}
              aria-label={$_('component.editor.toolbar.mode_edit')}
              aria-pressed={editorMode === 'edit'}
            >
              <Edit size={16} />
            </button>
            {#if !isMobile}
              <button
                type="button"
                onclick={() => onSetEditorMode('split')}
                class="p-2 hover:bg-accent border-x border-border toolbar-btn"
                class:bg-accent={editorMode === 'split'}
                aria-label={$_('component.editor.toolbar.mode_split')}
                aria-pressed={editorMode === 'split'}
              >
                <Columns size={16} />
              </button>
            {/if}
            <button
              type="button"
              onclick={() => onSetEditorMode('preview')}
              class="p-2 hover:bg-accent rounded-r-md toolbar-btn"
              class:border-l={isMobile}
              class:border-border={isMobile}
              class:bg-accent={editorMode === 'preview'}
              aria-label={$_('component.editor.toolbar.mode_preview')}
              aria-pressed={editorMode === 'preview'}
            >
              <Eye size={16} />
            </button>
          </div>

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
          {#if aiEnabled && (editorMode === 'edit' || editorMode === 'split')}
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
      <!-- More menu button - fixed right, always visible outside scroll area -->
      <button
        onclick={handleMoreMenuClick}
        class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
        aria-expanded={showMoreMenu}
        aria-haspopup="menu"
        title={$_('component.editor.toolbar.more_options')}
      >
        <MoreVertical size={16} />
      </button>
    </div>
  </div>
</div>
