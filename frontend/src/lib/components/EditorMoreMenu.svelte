<script lang="ts">
  import {
    Columns,
    Download,
    Edit,
    Eye,
    FolderInput,
    HelpCircle,
    Indent,
    Lock,
    LockOpen,
    Outdent,
    Palette,
    Save,
    ScanEye,
    Share2,
    Sparkles,
    Trash2,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { bottomsheet } from '$lib/actions/bottomsheet';
  import { FEATURE_FLAGS } from '$lib/config';
  import * as autosave from '$lib/stores/autosave.svelte';

  interface Props {
    onDelete: () => void;
    onMove: () => void;
    onExport: () => void;
    onColorPicker: () => void;
    onHelp: () => void;
    onIndent: () => void;
    onOutdent: () => void;
    onAIToggle: () => void;
    onShare?: () => void;
    onEncryptionToggle?: () => void;
    aiEnabled: boolean;
    isEncrypted?: boolean;
    onClose: () => void;
    onSetEditorMode?: (mode: 'edit' | 'preview' | 'split' | 'live') => void;
    editorMode?: 'edit' | 'preview' | 'split' | 'live';
    isMobile?: boolean;
    triggerRect?: {
      top: number;
      right: number;
      bottom: number;
      left: number;
      width: number;
      height: number;
    } | null;
  }

  const {
    onDelete,
    onMove,
    onExport,
    onColorPicker,
    onHelp,
    onIndent,
    onOutdent,
    onAIToggle,
    onShare,
    onEncryptionToggle,
    aiEnabled,
    isEncrypted = false,
    onClose,
    onSetEditorMode,
    editorMode = 'live',
    isMobile = false,
    triggerRect = null,
  }: Props = $props();

  // Compute desktop position from trigger button rect
  const desktopStyle = $derived.by(() => {
    if (!triggerRect) return '';
    const top = triggerRect.bottom + 4;
    const right = window.innerWidth - triggerRect.right;
    return `top: ${top}px; right: ${right}px;`;
  });

  function handleAutoSaveToggle() {
    autosave.setAutoSaveEnabled(!autosave.getAutoSaveEnabled());
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }

  function handleBackdropClick() {
    onClose();
  }

  function handleMenuClick(e: MouseEvent) {
    e.stopPropagation();
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 z-40 bg-black/50 sm:bg-transparent"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<!-- Menu: Bottom sheet on mobile, fixed-positioned near button on desktop -->
<div
  class="fixed z-50 bg-background p-4
		bottom-0 left-0 right-0 rounded-t-2xl animate-bottom-sheet
		sm:bottom-auto sm:left-auto sm:right-auto sm:w-56 sm:rounded-lg sm:shadow-lg sm:border sm:border-border sm:animate-none"
  style={desktopStyle}
  role="menu"
  aria-label={$_('component.editor.toolbar.more_options')}
  tabindex="-1"
  onkeydown={handleKeydown}
  onclick={handleMenuClick}
  use:bottomsheet={{ onClose }}
>
  <!-- Mobile handle -->
  <div class="w-12 h-1 bg-muted rounded-full mx-auto mb-4 sm:hidden"></div>

  <div class="space-y-1">
    {#if onSetEditorMode}
      <div class="px-3 pt-1 pb-1 text-xs font-medium text-muted-foreground">View Mode</div>
      <button
        type="button"
        onclick={() => {
          onSetEditorMode('live');
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        class:bg-accent={editorMode === 'live'}
        role="menuitemradio"
        aria-checked={editorMode === 'live'}
      >
        <ScanEye size={16} />
        {$_('component.editor.toolbar.mode_live')}
      </button>
      <button
        type="button"
        onclick={() => {
          onSetEditorMode('edit');
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        class:bg-accent={editorMode === 'edit'}
        role="menuitemradio"
        aria-checked={editorMode === 'edit'}
      >
        <Edit size={18} />
        {$_('component.editor.toolbar.mode_edit')}
      </button>
      <button
        type="button"
        onclick={() => {
          onSetEditorMode('preview');
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        class:bg-accent={editorMode === 'preview'}
        role="menuitemradio"
        aria-checked={editorMode === 'preview'}
      >
        <Eye size={18} />
        {$_('component.editor.toolbar.mode_preview')}
      </button>
      {#if !isMobile}
        <button
          type="button"
          onclick={() => {
            onSetEditorMode('split');
            onClose();
          }}
          class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
          class:bg-accent={editorMode === 'split'}
          role="menuitemradio"
          aria-checked={editorMode === 'split'}
        >
          <Columns size={18} />
          {$_('component.editor.toolbar.mode_split')}
        </button>
      {/if}
      <hr class="my-2 border-border" />
    {/if}

    <button
      type="button"
      onclick={() => {
        onIndent();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Indent size={18} />
      {$_('component.editor.toolbar.indent')}
    </button>

    <button
      type="button"
      onclick={() => {
        onOutdent();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Outdent size={18} />
      {$_('component.editor.toolbar.outdent')}
    </button>

    <hr class="my-2 border-border" />

    {#if FEATURE_FLAGS.colorSyntax}
      <button
        type="button"
        onclick={() => {
          onColorPicker();
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <Palette size={18} />
        {$_('component.editor.toolbar.color')}
      </button>
    {/if}

    {#if onEncryptionToggle}
      <button
        type="button"
        onclick={() => {
          onEncryptionToggle();
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        {#if isEncrypted}
          <LockOpen size={18} />
          {$_('component.editor.toolbar.decrypt_note')}
        {:else}
          <Lock size={18} />
          {$_('component.editor.toolbar.encrypt_note')}
        {/if}
      </button>
    {/if}

    {#if onShare}
      <button
        type="button"
        onclick={() => {
          onShare();
          onClose();
        }}
        disabled={isEncrypted}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        role="menuitem"
      >
        <Share2 size={18} />
        {$_('component.editor.toolbar.share')}
      </button>
    {/if}

    <button
      type="button"
      onclick={() => {
        onExport();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Download size={18} />
      {$_('component.editor.toolbar.export_note')}
    </button>

    <button
      type="button"
      onclick={() => {
        onMove();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <FolderInput size={18} />
      {$_('component.editor.toolbar.move')}
    </button>

    <button
      type="button"
      onclick={() => {
        onDelete();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left text-destructive hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Trash2 size={18} />
      {$_('component.editor.toolbar.delete_note')}
    </button>

    <hr class="my-2 border-border" />

    <button
      type="button"
      onclick={() => {
        onAIToggle();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      class:text-primary={aiEnabled}
      role="menuitem"
    >
      <Sparkles size={18} />
      {aiEnabled
        ? $_('component.editor.toolbar.ai_disable')
        : $_('component.editor.toolbar.ai_enable')}
    </button>

    <button
      type="button"
      onclick={() => {
        onHelp();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <HelpCircle size={18} />
      {$_('component.editor.toolbar.help')}
    </button>

    <button
      type="button"
      onclick={() => {
        handleAutoSaveToggle();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Save size={18} />
      {autosave.getAutoSaveEnabled()
        ? $_('component.editor.toolbar.autosave_disable')
        : $_('component.editor.toolbar.autosave_enable')}
    </button>
  </div>
</div>
