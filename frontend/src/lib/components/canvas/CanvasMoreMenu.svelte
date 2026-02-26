<script lang="ts">
  import { Download, FolderInput, Share2, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { bottomsheet } from '$lib/actions/bottomsheet';

  interface Props {
    onExport: () => void;
    onShare?: () => void;
    onMove: () => void;
    onDelete: () => void;
    onClose: () => void;
    isEncrypted?: boolean;
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
    onExport,
    onShare,
    onMove,
    onDelete,
    onClose,
    isEncrypted = false,
    triggerRect = null,
  }: Props = $props();

  // Compute desktop position from trigger button rect
  const desktopStyle = $derived.by(() => {
    if (!triggerRect) return '';
    const top = triggerRect.bottom + 4;
    const right = window.innerWidth - triggerRect.right;
    return `top: ${top}px; right: ${right}px;`;
  });

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
  class="fixed z-50 p-4
		bottom-0 left-0 right-0 rounded-t-2xl animate-bottom-sheet mobile-glass-sheet
		sm:bottom-auto sm:left-auto sm:right-auto sm:w-56 sm:rounded-lg sm:shadow-lg sm:border sm:border-border sm:animate-none ui-panel"
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
    <div class="px-3 pt-1 pb-1 ui-form-section-title sm:hidden">
      {$_('component.editor.toolbar.more_options')}
    </div>
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
      {$_('component.canvas.toolbar.export_canvas')}
    </button>

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
        onMove();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <FolderInput size={18} />
      {$_('component.editor.toolbar.move')}
    </button>

    <hr class="my-2 border-border" />
    <div class="px-3 pt-1 pb-1 ui-form-section-title">
      {$_('component.editor.toolbar.section_danger')}
    </div>

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
  </div>
</div>
