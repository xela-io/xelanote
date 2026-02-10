<script lang="ts">
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import { AlertTriangle } from 'lucide-svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    open: boolean;
    folderId: number;
    folderName: string;
    folderPath: string;
    noteCount: number;
    onClose: () => void;
  }

  const { open, folderId, folderName, folderPath, noteCount, onClose }: Props = $props();

  let isDeleting = $state(false);
  let errorMessage = $state<string | null>(null);

  async function handleDelete() {
    errorMessage = null;
    isDeleting = true;

    try {
      await tree.deleteFolder(folderId);

      // Show success message with note count info
      if (noteCount > 0) {
        toast.success(
          $_('dialog.delete_folder.success_with_notes', {
            values: { folder: folderName, count: noteCount },
          })
        );
      } else {
        toast.success($_('dialog.delete_folder.success', { values: { folder: folderName } }));
      }
      onClose();
    } catch (e) {
      console.error('Failed to delete folder:', e);

      // Parse Backend-Fehlermeldungen
      const errorMsg = e instanceof Error ? e.message : $_('common.unknown_error');

      if (errorMsg.includes('cannot delete root folder')) {
        errorMessage = $_('dialog.delete_folder.error_root');
      } else if (errorMsg.includes('folder not found') || errorMsg.includes('404')) {
        errorMessage = $_('dialog.delete_folder.error_not_found');
      } else {
        errorMessage = $_('dialog.delete_folder.error_generic', { values: { error: errorMsg } });
      }
    } finally {
      isDeleting = false;
    }
  }
</script>

<BaseDialog {open} title={$_('dialog.delete_folder.title')} {onClose} size="sm" variant="danger">
  {#snippet content()}
    <div class="space-y-4">
      <!-- Warning -->
      <div
        class="flex items-start gap-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 p-3 rounded-md"
      >
        <AlertTriangle size={20} class="text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
        <div class="text-sm">
          <p class="font-semibold text-amber-800 dark:text-amber-200">
            {$_('dialog.delete_folder.warning_permanent')}
          </p>
          {#if noteCount > 0}
            <p class="text-amber-700 dark:text-amber-300 mt-1">
              {$_('dialog.delete_folder.warning_notes', { values: { count: noteCount } })}
            </p>
          {:else}
            <p class="text-amber-700 dark:text-amber-300 mt-1">
              {$_('dialog.delete_folder.empty')}
            </p>
          {/if}
        </div>
      </div>

      <!-- Folder Info -->
      <div class="space-y-2 text-sm">
        <div>
          <span class="text-muted-foreground">{$_('dialog.delete_folder.label_folder')}:</span>
          <span class="font-mono ml-2 font-semibold">{folderName}</span>
        </div>
        <div>
          <span class="text-muted-foreground">{$_('dialog.delete_folder.label_path')}:</span>
          <span class="font-mono ml-2">{folderPath}</span>
        </div>
        {#if noteCount > 0}
          <div>
            <span class="text-muted-foreground">{$_('dialog.delete_folder.label_notes')}:</span>
            <span class="ml-2 font-semibold text-amber-600 dark:text-amber-400">
              {$_('dialog.delete_folder.note_count', { values: { count: noteCount } })}
            </span>
          </div>
        {/if}
      </div>

      <!-- Confirmation Question -->
      <p class="text-sm font-medium">
        {$_('dialog.delete_folder.confirm_question', { values: { folder: folderName } })}
      </p>

      {#if errorMessage}
        <div
          class="text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-3 rounded border border-red-200 dark:border-red-700"
        >
          {errorMessage}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button
      type="button"
      onclick={onClose}
      class="px-4 py-2 text-sm hover:bg-accent rounded-md"
      disabled={isDeleting}
    >
      {$_('dialog.cancel')}
    </button>
    <button
      type="button"
      onclick={handleDelete}
      disabled={isDeleting}
      class="px-4 py-2 text-sm bg-destructive text-destructive-foreground hover:bg-destructive/90 rounded-md disabled:opacity-50 font-medium"
    >
      {isDeleting ? $_('dialog.delete_folder.deleting') : $_('dialog.delete_folder.title')}
    </button>
  {/snippet}
</BaseDialog>
