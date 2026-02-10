<script lang="ts">
  import { FilePlus, FileText,PenLine, Search } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import CreateNoteDialog from '$lib/components/CreateNoteDialog.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { formatRelativeTime } from '$lib/utils/time';

  let showCreateNoteDialog = $state(false);

  async function handleCreateNoteConfirm(title: string) {
    const folderPath = folders.getSelectedFolder() || '/';
    try {
      const note = await notes.createNote(title, '', folderPath);
      await folders.loadFolders();
      goto(`/note/${note.id}`);
    } catch (e) {
      console.error('Failed to create note:', e);
    }
  }
</script>

<svelte:head>
  <title>xelanote</title>
</svelte:head>

<div class="h-full flex items-center justify-center">
  <div class="text-center max-w-md px-4">
    <div class="mb-8">
      <PenLine size={64} class="mx-auto text-muted-foreground mb-4" />
      <h1 class="text-2xl font-bold mb-2">{$_('page.home.welcome_title')}</h1>
      <p class="text-muted-foreground">
        {$_('page.home.welcome_subtitle')}
      </p>
    </div>

    <div class="space-y-3">
      <button
        onclick={() => (showCreateNoteDialog = true)}
        class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90"
      >
        <FilePlus size={20} />
        {$_('page.home.create_new_note')}
      </button>

      <button
        onclick={() => ui.toggleQuickSwitcher()}
        class="w-full flex items-center justify-center gap-2 px-4 py-3 border border-border rounded-lg hover:bg-accent"
      >
        <Search size={20} />
        {$_('page.home.open_quick_search')}
        <span class="text-xs text-muted-foreground ml-2">Ctrl+P</span>
      </button>
    </div>

    {#if notes.getRecentNotes(5).length > 0}
      <div class="mt-8 w-full">
        <div class="text-xs text-muted-foreground uppercase tracking-wider mb-3">
          {$_('page.home.recently_edited')}
        </div>
        <div class="space-y-1">
          {#each notes.getRecentNotes(5) as note (note.id)}
            <button
              onclick={() => goto(`/note/${note.id}`)}
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-accent text-left group"
            >
              <FileText size={16} class="text-muted-foreground flex-shrink-0" />
              <span class="flex-1 truncate text-sm">{note.title}</span>
              <span
                class="text-xs text-muted-foreground opacity-0 group-hover:opacity-100 [@media(pointer:coarse)]:opacity-100 transition-opacity"
              >
                {formatRelativeTime(note.updated_at, $_)}
              </span>
            </button>
          {/each}
        </div>
      </div>
    {:else if notes.getNotes().length > 0}
      <div class="mt-8 text-sm text-muted-foreground">
        {$_('page.home.notes_available', { values: { count: notes.getNotes().length } })}
      </div>
    {/if}
  </div>
</div>

<CreateNoteDialog
  open={showCreateNoteDialog}
  folderPath={folders.getSelectedFolder() || '/'}
  onClose={() => (showCreateNoteDialog = false)}
  onCreate={handleCreateNoteConfirm}
/>
