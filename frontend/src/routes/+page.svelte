<script lang="ts">
  import { Clock3, FilePlus, FileText, Search, Sparkles } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import CreateNoteDialog from '$lib/components/CreateNoteDialog.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { formatRelativeTime } from '$lib/utils/time';

  let showCreateNoteDialog = $state(false);
  const recentNotes = $derived(notes.getRecentNotes(5));
  const totalNotes = $derived(notes.getNotes().length);

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

<div class="relative h-full overflow-hidden bg-background">
  <div
    class="pointer-events-none absolute inset-0 opacity-70 [background:radial-gradient(50rem_24rem_at_70%_30%,color-mix(in_oklch,var(--color-primary),transparent_82%),transparent),radial-gradient(32rem_18rem_at_20%_15%,color-mix(in_oklch,var(--color-sidebar-accent),transparent_65%),transparent)]"
  ></div>
  <div
    class="pointer-events-none absolute inset-0 opacity-15 [background-image:linear-gradient(to_right,var(--color-border)_1px,transparent_1px),linear-gradient(to_bottom,var(--color-border)_1px,transparent_1px)] [background-size:28px_28px]"
  ></div>

  <div class="relative h-full overflow-y-auto">
    <div class="mx-auto flex min-h-full w-full max-w-7xl items-center px-4 py-8 sm:px-6 lg:px-10">
      <div
        class="grid w-full grid-cols-1 gap-5 lg:grid-cols-[1.1fr_0.9fr] lg:gap-6 xl:scale-[1.03] xl:origin-center"
      >
        <section
          class="relative overflow-hidden rounded-2xl border border-border/60 bg-card/70 p-5 shadow-sm backdrop-blur-sm sm:p-6"
        >
          <div
            class="pointer-events-none absolute -right-12 -top-10 h-40 w-40 rounded-full bg-primary/10 blur-2xl"
          ></div>

          <div class="relative">
            <div
              class="mb-4 inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/50 px-3 py-1 text-xs font-medium text-muted-foreground"
            >
              <Sparkles size={13} class="text-primary" />
              xelanote
            </div>

            <h1 class="text-balance text-2xl font-semibold tracking-tight sm:text-3xl">
              {$_('page.home.welcome_title')}
            </h1>
            <p class="mt-2 max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
              {$_('page.home.welcome_subtitle')}
            </p>

            <div class="mt-5 grid gap-3 sm:max-w-md">
              <button
                onclick={() => (showCreateNoteDialog = true)}
                class="group w-full rounded-xl border border-primary/25 bg-primary px-4 py-3 text-primary-foreground shadow-[0_8px_24px_-16px_var(--color-primary)] transition hover:brightness-105"
              >
                <span class="flex items-center justify-center gap-2 font-medium">
                  <FilePlus size={18} />
                  {$_('page.home.create_new_note')}
                </span>
              </button>

              <button
                onclick={() => ui.toggleQuickSwitcher()}
                class="w-full rounded-xl border border-border/70 bg-background/40 px-4 py-3 text-left transition hover:bg-accent/40"
              >
                <span class="flex items-center gap-2.5">
                  <Search size={18} class="text-muted-foreground" />
                  <span class="flex-1 text-sm sm:text-[0.95rem]">
                    {$_('page.home.open_quick_search')}
                  </span>
                  <span
                    class="rounded-md border border-border/70 bg-background/70 px-2 py-0.5 text-[11px] font-medium tracking-wide text-muted-foreground"
                  >
                    Ctrl+P
                  </span>
                </span>
              </button>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <div
                class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
              >
                <FileText size={14} class="text-primary/85" />
                <span>
                  {$_('page.home.notes_available', { values: { count: totalNotes } })}
                </span>
              </div>
              <div
                class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
              >
                <Clock3 size={14} class="text-primary/85" />
                <span>{$_('page.home.recently_edited')}</span>
              </div>
            </div>
          </div>
        </section>

        <section
          class="rounded-2xl border border-border/60 bg-card/60 p-4 shadow-sm backdrop-blur-sm sm:p-5"
        >
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <div class="text-[11px] uppercase tracking-[0.12em] text-muted-foreground">
                {$_('page.home.recently_edited')}
              </div>
              <div class="mt-1 text-sm font-medium text-foreground">
                {recentNotes.length > 0 ? `${recentNotes.length} items` : '—'}
              </div>
            </div>
          </div>

          {#if recentNotes.length > 0}
            <div class="space-y-1.5">
              {#each recentNotes as note (note.id)}
                <button
                  onclick={() => goto(`/note/${note.id}`)}
                  class="group w-full rounded-xl border border-transparent bg-background/25 px-3 py-2.5 text-left transition hover:border-border/60 hover:bg-accent/30"
                >
                  <span class="flex items-center gap-3">
                    <span
                      class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/50"
                    >
                      <FileText size={15} class="text-primary/90" />
                    </span>
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-sm font-medium">{note.title}</span>
                      <span class="block text-xs text-muted-foreground">
                        {formatRelativeTime(note.updated_at, $_)}
                      </span>
                    </span>
                  </span>
                </button>
              {/each}
            </div>
          {:else if totalNotes > 0}
            <div
              class="rounded-xl border border-dashed border-border/60 bg-background/25 p-4 text-sm text-muted-foreground"
            >
              {$_('page.home.notes_available', { values: { count: totalNotes } })}
            </div>
          {:else}
            <div
              class="rounded-xl border border-dashed border-border/60 bg-background/25 p-5 text-sm text-muted-foreground"
            >
              Noch keine Notizen vorhanden. Erstelle deine erste Notiz oder nutze die Schnellsuche.
            </div>
          {/if}
        </section>
      </div>
    </div>
  </div>
</div>

<CreateNoteDialog
  open={showCreateNoteDialog}
  folderPath={folders.getSelectedFolder() || '/'}
  onClose={() => (showCreateNoteDialog = false)}
  onCreate={handleCreateNoteConfirm}
/>
