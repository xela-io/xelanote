<script lang="ts">
  import { ImagePlus, ListTodo, Plus, Table2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { bottomsheet } from '$lib/actions/bottomsheet';

  interface Props {
    onInsertTask: () => void;
    onInsertTable: () => void;
    onUpload: () => void;
    onClose: () => void;
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
    onInsertTask,
    onInsertTable,
    onUpload,
    onClose,
    isMobile: _isMobile = false,
    triggerRect = null,
  }: Props = $props();

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

<div
  class="fixed inset-0 z-40 bg-black/50 sm:bg-transparent"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<div
  class="fixed z-50 p-4
    pt-3
    bottom-0 left-0 right-0 rounded-t-2xl animate-bottom-sheet mobile-glass-sheet
    sm:bottom-auto sm:left-auto sm:right-auto sm:w-56 sm:rounded-lg sm:shadow-lg sm:border sm:border-border sm:animate-none sm:bg-popover sm:backdrop-blur-none"
  style={desktopStyle}
  role="menu"
  aria-label={$_('component.editor.table_insert.insert')}
  tabindex="-1"
  onkeydown={handleKeydown}
  onclick={handleMenuClick}
  use:bottomsheet={{ onClose }}
>
  <div class="w-12 h-1 bg-muted rounded-full mx-auto mb-3 sm:hidden"></div>

  <div class="space-y-1">
    <div
      class="px-2.5 pt-0.5 pb-1.5 text-xs font-medium text-muted-foreground flex items-center gap-1.5"
    >
      <Plus size={14} />
      <span>{$_('component.editor.table_insert.insert')}</span>
    </div>

    <button
      type="button"
      onclick={() => {
        onInsertTask();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-2.5 py-2 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <ListTodo size={18} />
      {$_('component.editor.toolbar.task')}
    </button>

    <button
      type="button"
      onclick={() => {
        onInsertTable();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-2.5 py-2 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <Table2 size={18} />
      {$_('component.editor.toolbar.table')}
    </button>

    <button
      type="button"
      onclick={() => {
        onUpload();
        onClose();
      }}
      class="w-full flex items-center gap-3 px-2.5 py-2 text-left hover:bg-accent rounded-md transition-colors"
      role="menuitem"
    >
      <ImagePlus size={18} />
      {$_('component.editor.toolbar.upload')}
    </button>
  </div>
</div>
