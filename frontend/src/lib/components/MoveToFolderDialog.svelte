<script lang="ts">
  import { FolderPlus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';

  interface Props {
    noteId: string;
    currentFolder: string;
    onClose: () => void;
  }

  const { noteId, currentFolder, onClose }: Props = $props();

  // eslint-disable-next-line svelte/prefer-writable-derived -- needs local edits via bind:value
  let selectedFolder = $state('');
  let newFolderPath = $state('');
  let showNewFolder = $state(false);
  let isMoving = $state(false);

  // Get unique folder paths from the store
  const folderList = $derived.by(() => {
    const foldersData = folders.getFolders();
    const paths = foldersData.map((f) => f.path);
    // Add root if not present
    if (!paths.includes('/')) {
      return ['/', ...paths];
    }
    return paths;
  });

  $effect(() => {
    selectedFolder = currentFolder;
  });

  async function handleMove() {
    const targetFolder = showNewFolder ? newFolderPath : selectedFolder;

    if (!targetFolder) {
      return;
    }

    // Validate folder path
    if (!targetFolder.startsWith('/')) {
      await dialog.alert({
        title: $_('common.error'),
        message: $_('component.move_dialog.path_must_start_with_slash'),
        variant: 'warning',
      });
      return;
    }

    if (targetFolder === currentFolder) {
      onClose();
      return;
    }

    isMoving = true;
    try {
      await notes.moveNote(noteId, targetFolder);
      // Reload folders to update counts
      await folders.loadFolders();
      onClose();
    } catch (e) {
      console.error('Failed to move note:', e);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('component.move_dialog.move_error'),
        variant: 'danger',
      });
    } finally {
      isMoving = false;
    }
  }
</script>

<BaseDialog open={true} title={$_('component.move_dialog.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <div class="text-sm text-muted-foreground">
        {$_('component.move_dialog.current_folder')}: <span class="font-mono">{currentFolder}</span>
      </div>

      {#if !showNewFolder}
        <!-- Existing folders -->
        <DialogField forId="folder-select" label={$_('component.move_dialog.select_target') + ':'}>
          <select id="folder-select" bind:value={selectedFolder} class="ui-select">
            {#each folderList as folder (folder)}
              <option value={folder}>
                {folder === '/' ? $_('component.move_dialog.root_folder') : folder}
              </option>
            {/each}
          </select>
        </DialogField>

        <button
          type="button"
          onclick={() => (showNewFolder = true)}
          class="ui-button ui-button-ghost text-sm px-0 py-0"
        >
          <FolderPlus size={16} />
          {$_('component.move_dialog.create_new_folder')}
        </button>
      {:else}
        <!-- New folder input -->
        <DialogField
          forId="new-folder-input"
          label={$_('component.move_dialog.new_folder_path') + ':'}
          help={$_('component.move_dialog.path_example')}
        >
          <input
            id="new-folder-input"
            type="text"
            bind:value={newFolderPath}
            placeholder={$_('component.move_dialog.path_placeholder')}
            class="ui-input"
          />
        </DialogField>

        <button
          type="button"
          onclick={() => (showNewFolder = false)}
          class="ui-button ui-button-ghost text-sm px-0 py-0"
        >
          {$_('component.move_dialog.select_existing')}
        </button>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button type="button" onclick={onClose} class="ui-button ui-button-secondary text-sm">
        {$_('common.cancel')}
      </button>
      <button
        type="button"
        onclick={handleMove}
        disabled={isMoving}
        class="ui-button ui-button-primary text-sm"
      >
        {isMoving ? $_('component.move_dialog.moving') : $_('component.move_dialog.move')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
