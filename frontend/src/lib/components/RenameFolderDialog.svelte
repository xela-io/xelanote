<script lang="ts">
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import * as tree from '$lib/stores/tree.svelte';

  interface Props {
    open: boolean;
    folderId: number;
    currentName: string;
    onClose: () => void;
  }

  const { open, folderId, currentName, onClose }: Props = $props();

  // eslint-disable-next-line svelte/prefer-writable-derived -- needs local edits via bind:value
  let newName = $state('');
  let isRenaming = $state(false);
  let errorMessage = $state<string | null>(null);
  let nameInput: HTMLInputElement | null = null;

  $effect(() => {
    newName = currentName;
  });

  async function handleRename() {
    errorMessage = null;

    // Validation
    if (!newName.trim()) {
      errorMessage = $_('component.rename_dialog.error_empty');
      return;
    }

    if (newName === currentName) {
      onClose();
      return;
    }

    if (newName.includes('/')) {
      errorMessage = $_('component.rename_dialog.error_slash');
      return;
    }

    if (newName.includes('..')) {
      errorMessage = $_('component.rename_dialog.error_dotdot');
      return;
    }

    isRenaming = true;
    try {
      await tree.renameFolder(folderId, newName.trim());
      onClose();
    } catch (e) {
      console.error('Failed to rename folder:', e);
      errorMessage = $_('component.rename_dialog.error_rename', {
        values: { error: e instanceof Error ? e.message : 'Unknown error' },
      });
    } finally {
      isRenaming = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleRename();
    }
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

<BaseDialog {open} title={$_('component.rename_dialog.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <div class="text-sm text-muted-foreground">
        {$_('component.rename_dialog.current_name')}: <span class="font-mono">{currentName}</span>
      </div>

      <div class="space-y-2">
        <label for="new-name-input" class="text-sm font-medium"
          >{$_('component.rename_dialog.new_name')}:</label
        >
        <input
          id="new-name-input"
          type="text"
          bind:value={newName}
          bind:this={nameInput}
          placeholder={$_('component.rename_dialog.placeholder')}
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        />
        <p class="text-xs text-muted-foreground">
          {$_('component.rename_dialog.hint')}
        </p>
      </div>

      {#if errorMessage}
        <div class="text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-2 rounded">
          {errorMessage}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('common.cancel')}
    </button>
    <button
      type="button"
      onclick={handleRename}
      disabled={isRenaming}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {isRenaming ? $_('component.rename_dialog.renaming') : $_('component.rename_dialog.rename')}
    </button>
  {/snippet}
</BaseDialog>
