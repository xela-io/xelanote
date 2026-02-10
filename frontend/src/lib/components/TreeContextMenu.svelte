<script lang="ts">
  import { Edit3, Palette, Share2,Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { TreeNode } from '$lib/stores/tree.svelte';

  interface Props {
    node: TreeNode;
    position: { x: number; y: number };
    onClose: () => void;
    onRename: () => void;
    onDelete: () => void;
    onColorPicker: () => void;
    onShare?: () => void;
  }

  const { node, position, onClose, onRename, onDelete, onColorPicker, onShare }: Props = $props();

  let menuRef: HTMLDivElement | undefined = $state();

  const isRoot = $derived(node.type === 'folder' && node.path === '/');

  // Initial position to avoid flash at 0,0
  const initialStyle = $derived(`left: ${position.x}px; top: ${position.y}px;`);

  // Clamp menu to viewport after render
  $effect(() => {
    if (!menuRef) return;
    const rect = menuRef.getBoundingClientRect();
    const pad = 8;
    if (rect.right > window.innerWidth - pad) {
      menuRef.style.left = `${window.innerWidth - rect.width - pad}px`;
    }
    if (rect.bottom > window.innerHeight - pad) {
      menuRef.style.top = `${window.innerHeight - rect.height - pad}px`;
    }
  });

  // Autofocus first menu item
  $effect(() => {
    if (menuRef) {
      const firstItem = menuRef.querySelector<HTMLElement>('[role="menuitem"]');
      firstItem?.focus();
    }
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    }
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

<!-- Menu: Bottom sheet on mobile, positioned on desktop -->
<div
  bind:this={menuRef}
  class="fixed z-50 bg-background p-4
		bottom-0 left-0 right-0 rounded-t-2xl animate-bottom-sheet
		sm:bottom-auto sm:left-auto sm:right-auto sm:w-48 sm:rounded-lg sm:shadow-lg sm:border sm:border-border sm:animate-none sm:p-2"
  style={initialStyle}
  role="menu"
  aria-label={$_('component.tree.context_menu.more_options')}
  tabindex="-1"
  onkeydown={handleKeydown}
  onclick={handleMenuClick}
>
  <!-- Mobile handle -->
  <div class="w-12 h-1 bg-muted rounded-full mx-auto mb-4 sm:hidden"></div>

  <div class="space-y-1">
    <!-- Color - always shown -->
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
      {$_('component.tree.context_menu.color')}
    </button>

    {#if !isRoot}
      <!-- Rename - folders and notes, not root -->
      <button
        type="button"
        onclick={() => {
          onRename();
          onClose();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <Edit3 size={18} />
        {$_('component.tree.context_menu.rename')}
      </button>

      <!-- Share -->
      {#if onShare}
        <button
          type="button"
          onclick={() => {
            onShare();
            onClose();
          }}
          class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
          role="menuitem"
        >
          <Share2 size={18} />
          {$_('sharing.share')}
        </button>
      {/if}
    {/if}

    <hr class="my-2 border-border" />

    <!-- Delete - folders (not root) and notes -->
    {#if !isRoot}
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
        {$_('component.tree.context_menu.delete')}
      </button>
    {/if}
  </div>
</div>
