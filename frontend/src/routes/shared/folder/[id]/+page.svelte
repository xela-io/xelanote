<script lang="ts">
  import { ArrowLeft, Edit, Eye, Loader2 } from 'lucide-svelte';
  import { onDestroy,onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  const folderId = $derived(Number($page.params.id));

  // Find the folder info from loaded shared folders
  const folderInfo = $derived(sharing.getSharedFolders().find((f) => f.id === folderId));

  onMount(() => {
    // Load shared folders if not yet loaded (e.g. direct navigation)
    if (sharing.getSharedFolders().length === 0) {
      sharing.loadSharedFolders();
    }
    sharing.loadSharedFolderNotes(folderId);
  });

  onDestroy(() => {
    sharing.clearCurrentSharedFolderNotes();
  });

  function handleNoteClick(noteId: string) {
    goto(`/note/${noteId}`);
    ui.closeSidebarOnMobile();
  }

  function handleBack() {
    goto('/shared');
  }
</script>

<svelte:head>
  <title>{folderInfo?.name ?? $_('sharing.shared_folders')} - xelanote</title>
</svelte:head>

<div class="p-4">
  <!-- Header -->
  <div class="mb-4">
    <button
      onclick={handleBack}
      class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors mb-2"
    >
      <ArrowLeft size={16} />
      {$_('sharing.back_to_shared')}
    </button>

    {#if folderInfo}
      <h1 class="text-2xl font-bold">{folderInfo.name}</h1>
      <p class="text-sm text-muted-foreground">
        {$_('sharing.folder_shared_by', { values: { name: folderInfo.shared_by } })}
      </p>
    {:else}
      <h1 class="text-2xl font-bold">{$_('sharing.shared_folders')}</h1>
    {/if}
  </div>

  <!-- Notes list -->
  {#if sharing.getIsLoading()}
    <div class="flex items-center gap-2 text-muted-foreground">
      <Loader2 size={18} class="animate-spin" />
      <span>{$_('common.loading')}</span>
    </div>
  {:else if sharing.getCurrentSharedFolderNotes().length === 0}
    <p class="text-muted-foreground">{$_('sharing.no_shared_notes')}</p>
  {:else}
    <div class="space-y-1">
      {#each sharing.getCurrentSharedFolderNotes() as note (note.id)}
        <button
          onclick={() => handleNoteClick(note.id)}
          class="w-full flex items-center gap-3 px-3 py-2 rounded-md hover:bg-accent transition-colors text-left"
        >
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate">{note.title}</div>
            <div class="text-xs text-muted-foreground">
              {new Date(note.updated_at).toLocaleDateString()}
            </div>
          </div>
          <span
            class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {note.share_role ===
            'editor'
              ? 'bg-primary/10 text-primary'
              : 'bg-muted text-muted-foreground'}"
          >
            {#if note.share_role === 'editor'}
              <Edit size={10} />
              {$_('sharing.editable')}
            {:else}
              <Eye size={10} />
              {$_('sharing.read_only')}
            {/if}
          </span>
        </button>
      {/each}
    </div>
  {/if}
</div>
