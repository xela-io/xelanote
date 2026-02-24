<script lang="ts">
  import { _ } from 'svelte-i18n';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
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

      <DialogField
        forId="new-title-input"
        label={$_('component.rename_note_dialog.new_title') + ':'}
        error={errorMessage}
      >
        <input
          id="new-title-input"
          type="text"
          bind:value={newTitle}
          placeholder={$_('component.rename_note_dialog.placeholder')}
          class="ui-input"
        />
      </DialogField>
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button type="button" onclick={onClose} class="ui-button ui-button-secondary text-sm">
        {$_('common.cancel')}
      </button>
      <button
        type="button"
        onclick={handleRename}
        disabled={isRenaming}
        class="ui-button ui-button-primary text-sm"
      >
        {isRenaming
          ? $_('component.rename_note_dialog.renaming')
          : $_('component.rename_note_dialog.rename')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
