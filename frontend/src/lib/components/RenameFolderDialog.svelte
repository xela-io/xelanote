<script lang="ts">
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
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

      <DialogField
        forId="new-name-input"
        label={$_('component.rename_dialog.new_name') + ':'}
        help={$_('component.rename_dialog.hint')}
        error={errorMessage}
      >
        <input
          id="new-name-input"
          type="text"
          bind:value={newName}
          bind:this={nameInput}
          placeholder={$_('component.rename_dialog.placeholder')}
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
        {isRenaming ? $_('component.rename_dialog.renaming') : $_('component.rename_dialog.rename')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
