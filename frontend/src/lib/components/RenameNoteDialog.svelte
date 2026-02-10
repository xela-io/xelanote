<script lang="ts">
  import { _ } from 'svelte-i18n';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as tree from '$lib/stores/tree.svelte';

  interface Props {
    open: boolean;
    noteId: string;
    currentTitle: string;
    onClose: () => void;
  }

  const { open, noteId, currentTitle, onClose }: Props = $props();

  // eslint-disable-next-line svelte/prefer-writable-derived -- needs local edits via bind:value
  let newTitle = $state('');
  let isRenaming = $state(false);
  let errorMessage = $state<string | null>(null);

  $effect(() => {
    newTitle = currentTitle;
  });

  async function handleRename() {
    errorMessage = null;

    if (!newTitle.trim()) {
      errorMessage = $_('component.rename_note_dialog.error_empty');
      return;
    }

    if (newTitle.trim() === currentTitle) {
      onClose();
      return;
    }

    isRenaming = true;
    try {
      await api.renameNote(noteId, newTitle.trim());
      await notes.loadNotes();
      await tree.loadTree();
      onClose();
    } catch (e) {
      console.error('Failed to rename note:', e);
      errorMessage = $_('component.rename_note_dialog.error_rename', {
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

<BaseDialog {open} title={$_('component.rename_note_dialog.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <div class="text-sm text-muted-foreground">
        {$_('component.rename_note_dialog.current_title')}:
        <span class="font-mono">{currentTitle}</span>
      </div>

      <div class="space-y-2">
        <label for="new-title-input" class="text-sm font-medium"
          >{$_('component.rename_note_dialog.new_title')}:</label
        >
        <input
          id="new-title-input"
          type="text"
          bind:value={newTitle}
          placeholder={$_('component.rename_note_dialog.placeholder')}
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        />
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
      {isRenaming
        ? $_('component.rename_note_dialog.renaming')
        : $_('component.rename_note_dialog.rename')}
    </button>
  {/snippet}
</BaseDialog>
