<script lang="ts">
  import { FolderPlus } from 'lucide-svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import { _ } from 'svelte-i18n';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

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
        <div class="space-y-2">
          <label for="folder-select" class="text-sm font-medium"
            >{$_('component.move_dialog.select_target')}:</label
          >
          <select
            id="folder-select"
            bind:value={selectedFolder}
            class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
          >
            {#each folderList as folder (folder)}
              <option value={folder}>
                {folder === '/' ? $_('component.move_dialog.root_folder') : folder}
              </option>
            {/each}
          </select>
        </div>

        <button
          type="button"
          onclick={() => (showNewFolder = true)}
          class="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <FolderPlus size={16} />
          {$_('component.move_dialog.create_new_folder')}
        </button>
      {:else}
        <!-- New folder input -->
        <div class="space-y-2">
          <label for="new-folder-input" class="text-sm font-medium"
            >{$_('component.move_dialog.new_folder_path')}:</label
          >
          <input
            id="new-folder-input"
            type="text"
            bind:value={newFolderPath}
            placeholder={$_('component.move_dialog.path_placeholder')}
            class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <p class="text-xs text-muted-foreground">
            {$_('component.move_dialog.path_example')}
          </p>
        </div>

        <button
          type="button"
          onclick={() => (showNewFolder = false)}
          class="text-sm text-muted-foreground hover:text-foreground"
        >
          {$_('component.move_dialog.select_existing')}
        </button>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('common.cancel')}
    </button>
    <button
      type="button"
      onclick={handleMove}
      disabled={isMoving}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {isMoving ? $_('component.move_dialog.moving') : $_('component.move_dialog.move')}
    </button>
  {/snippet}
</BaseDialog>
