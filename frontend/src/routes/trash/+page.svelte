<script lang="ts">
  import { RotateCcw, Trash, Trash2 } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import { formatRelativeTime } from '$lib/utils/time';

  onMount(() => {
    trash.loadTrash();
  });

  const trashedNotes = $derived(trash.trash.notes);
  const trashCount = $derived(trash.trash.count);
  const isLoading = $derived(trash.trash.isLoading);
  const error = $derived(trash.trash.error);

  function truncateContent(content: string, maxLength = 150): string {
    if (content.length <= maxLength) return content;
    return content.substring(0, maxLength) + '...';
  }

  async function handleRestore(id: string, title: string) {
    const success = await trash.restoreNote(id);
    if (success) {
      toast.success($_('page.trash.restored', { values: { title } }));
      trash.loadTrashCount();
      notes.loadNotes();
      tree.loadTree();
    } else {
      toast.error($_('page.trash.restore_failed'));
    }
  }

  async function handlePermanentDelete(id: string, title: string) {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('dialog.permanent_delete_confirm'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });

    if (!confirmed) return;

    const success = await trash.permanentlyDeleteNote(id);
    if (success) {
      toast.success($_('page.trash.deleted_permanently', { values: { title } }));
    } else {
      toast.error($_('page.trash.delete_failed'));
    }
  }

  async function handleEmptyTrash() {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('dialog.empty_trash_confirm'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });

    if (!confirmed) return;

    const success = await trash.emptyTrash();
    if (success) {
      toast.success($_('page.trash.emptied'));
    } else {
      toast.error($_('page.trash.empty_failed'));
    }
  }
</script>

<div class="h-full overflow-y-auto">
  <div class="max-w-7xl mx-auto px-6 py-8">
    <!-- Header -->
    <div
      class="ui-panel p-4 mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex items-center gap-2 sm:gap-3 min-w-0">
        <MobileSidebarInlineToggle />
        <Trash2 size={28} class="text-muted-foreground" />
        <h1 class="text-2xl font-bold">{$_('page.trash.title')}</h1>
        {#if trashCount > 0}
          <span class="text-muted-foreground">({trashCount})</span>
        {/if}
      </div>

      {#if trashCount > 0}
        <button
          onclick={handleEmptyTrash}
          class="ui-button ui-button-danger self-start sm:self-auto"
        >
          {$_('page.trash.empty_button')}
        </button>
      {/if}
    </div>

    <!-- Loading State -->
    {#if isLoading}
      <div class="text-center py-12 text-muted-foreground" role="status" aria-live="polite">
        {$_('page.trash.loading')}
      </div>
    {:else if error}
      <!-- Error State -->
      <div class="text-center py-12 text-destructive">
        {error}
      </div>
    {:else if trashedNotes.length === 0}
      <!-- Empty State -->
      <div class="ui-empty-state py-20">
        <Trash2 size={64} class="mx-auto text-muted-foreground/50 mb-4" />
        <h2 class="text-xl font-semibold text-muted-foreground mb-2">
          {$_('page.trash.empty_title')}
        </h2>
        <p class="text-muted-foreground">{$_('page.trash.empty_description')}</p>
      </div>
    {:else}
      <!-- Trash List -->
      <div
        class="space-y-3 overflow-auto"
        style="max-height: calc(var(--app-viewport-height, 100dvh) - 250px);"
      >
        {#each trashedNotes as note (note.id)}
          <div class="ui-panel p-4">
            <!-- Note Header -->
            <div class="flex items-start justify-between mb-2">
              <h3 class="font-semibold text-lg line-clamp-2 flex-1">{note.title}</h3>
            </div>

            <!-- Metadata -->
            <div class="text-sm text-muted-foreground mb-2">
              <span>{note.folder_path}</span>
              <span class="mx-2">•</span>
              <span
                >{note.deleted_at
                  ? formatRelativeTime(note.deleted_at, $_)
                  : $_('page.trash.unknown_time')}</span
              >
            </div>

            <!-- Content Preview -->
            <p class="text-sm text-muted-foreground mb-4 line-clamp-3">
              {truncateContent(note.content)}
            </p>

            <!-- Actions -->
            <div class="flex gap-2">
              <button
                onclick={() => handleRestore(note.id, note.title)}
                class="ui-button ui-button-primary flex-1"
              >
                <RotateCcw size={16} />
                {$_('page.trash.restore')}
              </button>
              <button
                onclick={() => handlePermanentDelete(note.id, note.title)}
                class="ui-button ui-button-danger px-3"
              >
                <Trash size={16} />
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .line-clamp-2 {
    display: -webkit-box;
    line-clamp: 2;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .line-clamp-3 {
    display: -webkit-box;
    line-clamp: 3;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
