<script lang="ts">
  import { BookOpen, Edit, Eye, Folder, Loader2, Users } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  onMount(() => {
    sharing.loadAllShared();
  });

  function handleNoteClick(noteId: string) {
    goto(`/note/${noteId}`);
    ui.closeSidebarOnMobile();
  }

  function handleFolderClick(folderId: number) {
    goto(`/shared/folder/${folderId}`);
    ui.closeSidebarOnMobile();
  }

  function handleCollectionClick(collectionId: number) {
    goto(`/shared/collection/${collectionId}`);
    ui.closeSidebarOnMobile();
  }

  // Group notes by owner
  const groupedNotes = $derived.by(() => {
    const notes = sharing.getSharedNotes();
    const groups = new SvelteMap<string, typeof notes>();
    for (const note of notes) {
      const existing = groups.get(note.shared_by) ?? [];
      existing.push(note);
      groups.set(note.shared_by, existing);
    }
    return groups;
  });

  // Group folders by owner
  const groupedFolders = $derived.by(() => {
    const folders = sharing.getSharedFolders();
    const groups = new SvelteMap<string, typeof folders>();
    for (const folder of folders) {
      const existing = groups.get(folder.shared_by) ?? [];
      existing.push(folder);
      groups.set(folder.shared_by, existing);
    }
    return groups;
  });

  // Group collections by owner
  const groupedCollections = $derived.by(() => {
    const collections = sharing.getSharedCollections();
    const groups = new SvelteMap<string, typeof collections>();
    for (const coll of collections) {
      const existing = groups.get(coll.shared_by) ?? [];
      existing.push(coll);
      groups.set(coll.shared_by, existing);
    }
    return groups;
  });

  const hasAnyShared = $derived(
    sharing.getSharedNotes().length > 0 ||
      sharing.getSharedFolders().length > 0 ||
      sharing.getSharedCollections().length > 0
  );
</script>

<div class="p-4">
  <div class="ui-panel mb-4 flex items-center gap-2 sm:gap-3 p-4">
    <MobileSidebarInlineToggle />
    <h1 class="text-2xl font-bold flex items-center gap-2">
      <Users size={24} />
      {$_('sharing.shared_with_me')}
    </h1>
  </div>

  {#if sharing.getIsLoading()}
    <div class="flex items-center gap-2 text-muted-foreground">
      <Loader2 size={18} class="animate-spin" />
      <span>{$_('common.loading')}</span>
    </div>
  {:else if !hasAnyShared}
    <p class="text-muted-foreground">{$_('sharing.no_shared_notes')}</p>
  {:else}
    <div class="space-y-6">
      <!-- Shared Folders -->
      {#if sharing.getSharedFolders().length > 0}
        <div class="ui-panel p-4">
          <h2 class="ui-kicker mb-2">
            {$_('sharing.shared_folders')}
          </h2>
          {#each [...groupedFolders.entries()] as [owner, folders] (owner)}
            <div class="mb-3">
              <h3 class="ui-kicker mb-1 opacity-80">
                {$_('sharing.shared_by', { values: { name: owner } })}
              </h3>
              <div class="space-y-1.5">
                {#each folders as folder (folder.id)}
                  <button
                    onclick={() => handleFolderClick(folder.id)}
                    class="ui-list-item w-full flex items-center gap-3 px-3 py-2 text-left"
                  >
                    <Folder size={16} class="text-muted-foreground flex-shrink-0" />
                    <div class="flex-1 min-w-0">
                      <div class="font-medium truncate">{folder.name}</div>
                      <div class="text-xs text-muted-foreground">
                        {$_('sharing.notes_in_folder', { values: { count: folder.note_count } })}
                      </div>
                    </div>
                    <span
                      class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {folder.share_role ===
                      'editor'
                        ? 'bg-primary/10 text-primary'
                        : 'bg-muted text-muted-foreground'}"
                    >
                      {#if folder.share_role === 'editor'}
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
            </div>
          {/each}
        </div>
      {/if}

      <!-- Shared Collections (Cookbooks) -->
      {#if sharing.getSharedCollections().length > 0}
        <div class="ui-panel p-4">
          <h2 class="ui-kicker mb-2">
            {$_('sharing.shared_collections')}
          </h2>
          {#each [...groupedCollections.entries()] as [owner, collections] (owner)}
            <div class="mb-3">
              <h3 class="ui-kicker mb-1 opacity-80">
                {$_('sharing.shared_by', { values: { name: owner } })}
              </h3>
              <div class="space-y-1.5">
                {#each collections as coll (coll.id)}
                  <button
                    onclick={() => handleCollectionClick(coll.id)}
                    class="ui-list-item w-full flex items-center gap-3 px-3 py-2 text-left"
                  >
                    <BookOpen size={16} class="text-muted-foreground flex-shrink-0" />
                    <div class="flex-1 min-w-0">
                      <div class="font-medium truncate">{coll.name}</div>
                      <div class="text-xs text-muted-foreground">
                        {coll.recipe_count}
                        {$_('page.recipes.servings_label', { default: 'recipes' })}
                      </div>
                    </div>
                    <span
                      class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {coll.share_role ===
                      'editor'
                        ? 'bg-primary/10 text-primary'
                        : 'bg-muted text-muted-foreground'}"
                    >
                      {#if coll.share_role === 'editor'}
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
            </div>
          {/each}
        </div>
      {/if}

      <!-- Shared Notes -->
      {#if sharing.getSharedNotes().length > 0}
        {#each [...groupedNotes.entries()] as [owner, notes] (owner)}
          <div>
            <h2 class="ui-kicker mb-2">
              {$_('sharing.shared_by', { values: { name: owner } })}
            </h2>
            <div class="ui-panel p-3 space-y-1.5">
              {#each notes as note (note.id)}
                <button
                  onclick={() => handleNoteClick(note.id)}
                  class="ui-list-item w-full flex items-center gap-3 px-3 py-2 text-left"
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
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
