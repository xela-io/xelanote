<script lang="ts">
  import { AlertCircle, Link, Loader2 } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { beforeNavigate, goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { deleteNote, permanentlyDeleteNote } from '$lib/api';
  import * as journal from '$lib/stores/journal.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  // Get note ID from URL params
  // $page.params.id ist immer definiert bei [id] route, aber TypeScript kennt das nicht
  const noteId = $derived($page.params.id);

  // Dynamic import state
  let EditorComponent = $state<ComponentType | null>(null);
  let RecipeEditorComponent = $state<ComponentType | null>(null);
  let CanvasEditorComponent = $state<ComponentType | null>(null);
  let editorLoading = $state(false);
  let editorLoadError = $state<string | null>(null);
  let retryCount = $state(0);
  const MAX_RETRIES = 3;

  // Check if the current note is a recipe or canvas.
  // Only trust currentNote's note_type when it matches the URL noteId.
  // Without this guard, navigating away from a canvas would keep isCanvas=true
  // because currentNote is updated asynchronously by loadNote.
  const currentNote = $derived(notes.getCurrentNote());
  const backlinks = $derived(notes.getBacklinks());
  const noteLoaded = $derived(currentNote?.id === noteId);
  const isRecipe = $derived(noteLoaded && currentNote?.note_type === 'recipe');
  const isCanvas = $derived(noteLoaded && currentNote?.note_type === 'canvas');
  const useCanvasDesktopWorkspace = $derived(isCanvas && !ui.getIsMobile());

  // Auto-cleanup: delete empty journal notes when navigating away
  let didAutoDelete = false;

  beforeNavigate(() => {
    if (didAutoDelete) return;

    const note = notes.getCurrentNote();
    if (note?.note_type === 'journal' && !notes.getIsDirty() && !note.content?.trim()) {
      didAutoDelete = true;

      // Soft-delete + permanent-delete (fire-and-forget)
      deleteNote(note.id)
        .then(() => permanentlyDeleteNote(note.id))
        .catch((err) => {
          if (import.meta.env.DEV) {
            console.warn('Auto-delete empty journal failed:', err);
          }
        });

      // Invalidate year cache so heatmap re-fetches
      if (note.journal_date) {
        const year = parseInt(note.journal_date.substring(0, 4), 10);
        journal.invalidateYearCache(year);
      }
    }
  });

  // Redirect falls keine ID (sollte nicht passieren, aber für TypeScript-Safety)
  $effect(() => {
    if (!noteId) {
      goto('/');
    }
  });

  // If a note cannot be loaded (404), route away from the broken URL to stop retry loops.
  // Prefer another available note; otherwise fall back to home.
  $effect(() => {
    if (!noteId) return;
    if (notes.getError() !== 'NOT_FOUND') return;
    const fallback = notes.getNotes().find((n) => n.id !== noteId && !n.is_deleted);
    if (fallback) {
      goto(`/note/${fallback.id}`, { replaceState: true });
      return;
    }
    goto('/', { replaceState: true });
  });

  // Sync tree selection from URL so sidebar highlights the correct note after refresh
  $effect(() => {
    if (noteId && tree.getSelectedNoteId() !== noteId) {
      tree.selectNote(noteId);
    }
  });

  // Load editor when noteId changes (with cleanup to prevent race conditions)
  $effect(() => {
    if (noteId) {
      let cancelled = false;

      const load = async () => {
        try {
          editorLoading = true;
          editorLoadError = null;

          // Load both editor components in parallel
          const [editorModule, recipeModule, canvasModule] = await Promise.all([
            import('$lib/components/Editor.svelte'),
            import('$lib/components/RecipeEditor.svelte'),
            import('$lib/components/CanvasEditor.svelte'),
          ]);

          if (cancelled) return; // Don't update if effect was cleaned up

          EditorComponent = loadSvelteComponentFromModule(editorModule, 'Editor');
          RecipeEditorComponent = loadSvelteComponentFromModule(recipeModule, 'RecipeEditor');
          CanvasEditorComponent = loadSvelteComponentFromModule(canvasModule, 'CanvasEditor');
          editorLoading = false;
          retryCount = 0;
        } catch (error) {
          if (cancelled) return;

          console.error('Editor load failed:', error, `(attempt ${retryCount + 1})`);

          if (retryCount < MAX_RETRIES) {
            retryCount++;
            // Exponential backoff: 2s, 4s, 8s (2^1, 2^2, 2^3 seconds)
            const delay = Math.pow(2, retryCount) * 1000;
            setTimeout(() => load(), delay);
          } else {
            editorLoadError = $_('page.note.load_error');
            editorLoading = false;
          }
        }
      };

      load();

      return () => {
        cancelled = true; // Cleanup: cancel pending operations
      };
    }
  });
</script>

<svelte:head>
  <title>{notes.getCurrentNote()?.title ?? $_('page.note.fallback_title')} - xelanote</title>
</svelte:head>

{#if noteId}
  {#if editorLoading}
    <div class="flex items-center justify-center h-screen-safe">
      <div class="text-center">
        <Loader2 class="w-8 h-8 animate-spin mx-auto mb-2" />
        <p class="text-sm text-muted-foreground">{$_('page.note.loading')}</p>
      </div>
    </div>
  {:else if editorLoadError}
    <div class="flex items-center justify-center h-screen-safe">
      <div class="text-center text-red-500">
        <AlertCircle class="w-8 h-8 mx-auto mb-2" />
        <p>{editorLoadError}</p>
        <button
          onclick={() => window.location.reload()}
          class="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90"
        >
          {$_('page.note.reload')}
        </button>
      </div>
    </div>
  {:else if isCanvas && CanvasEditorComponent}
    {#if useCanvasDesktopWorkspace}
      <div class="canvas-workspace">
        <div class="canvas-workspace-main">
          <svelte:boundary>
            <CanvasEditorComponent {noteId} />
            {#snippet failed(error, reset)}
              {@const msg = error instanceof Error ? error.message : ''}
              <div class="flex items-center justify-center h-screen-safe">
                <div class="text-center text-destructive">
                  <AlertCircle class="w-8 h-8 mx-auto mb-2" />
                  <p class="mb-1 font-medium">{$_('error_page.component_crashed')}</p>
                  {#if msg}<p class="mb-4 text-sm text-muted-foreground">{msg}</p>{/if}
                  <div class="flex items-center justify-center gap-2">
                    <button
                      onclick={reset}
                      class="px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 text-sm"
                      >{$_('error_page.retry')}</button
                    >
                    <button
                      onclick={() => window.location.reload()}
                      class="px-4 py-2 border border-border rounded hover:bg-muted text-sm"
                      >{$_('error_page.reload_page')}</button
                    >
                  </div>
                </div>
              </div>
            {/snippet}
          </svelte:boundary>
        </div>
        <aside class="canvas-workspace-sidebar">
          <section class="canvas-workspace-panel">
            <h2 class="canvas-workspace-heading">
              <Link size={14} />
              {$_('page.note.workspace.linked_mentions')}
            </h2>
            {#if backlinks.length > 0}
              <div class="canvas-workspace-links">
                {#each backlinks as backlink (backlink.id)}
                  <a href="/note/{backlink.id}" class="canvas-workspace-link">{backlink.title}</a>
                {/each}
              </div>
            {:else}
              <p class="canvas-workspace-empty">{$_('page.note.workspace.no_backlinks')}</p>
            {/if}
          </section>

          <section class="canvas-workspace-panel">
            <h2 class="canvas-workspace-heading">{$_('page.note.workspace.unlinked_mentions')}</h2>
            <p class="canvas-workspace-empty">
              {$_('page.note.workspace.unlinked_mentions_empty')}
            </p>
          </section>
        </aside>
      </div>
    {:else}
      <svelte:boundary>
        <CanvasEditorComponent {noteId} />
        {#snippet failed(error, reset)}
          {@const msg = error instanceof Error ? error.message : ''}
          <div class="flex items-center justify-center h-screen-safe">
            <div class="text-center text-destructive">
              <AlertCircle class="w-8 h-8 mx-auto mb-2" />
              <p class="mb-1 font-medium">{$_('error_page.component_crashed')}</p>
              {#if msg}<p class="mb-4 text-sm text-muted-foreground">{msg}</p>{/if}
              <div class="flex items-center justify-center gap-2">
                <button
                  onclick={reset}
                  class="px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 text-sm"
                  >{$_('error_page.retry')}</button
                >
                <button
                  onclick={() => window.location.reload()}
                  class="px-4 py-2 border border-border rounded hover:bg-muted text-sm"
                  >{$_('error_page.reload_page')}</button
                >
              </div>
            </div>
          </div>
        {/snippet}
      </svelte:boundary>
    {/if}
  {:else if isRecipe && RecipeEditorComponent}
    <svelte:boundary>
      <RecipeEditorComponent {noteId} />
      {#snippet failed(error, reset)}
        {@const msg = error instanceof Error ? error.message : ''}
        <div class="flex items-center justify-center h-screen-safe">
          <div class="text-center text-destructive">
            <AlertCircle class="w-8 h-8 mx-auto mb-2" />
            <p class="mb-1 font-medium">{$_('error_page.component_crashed')}</p>
            {#if msg}<p class="mb-4 text-sm text-muted-foreground">{msg}</p>{/if}
            <div class="flex items-center justify-center gap-2">
              <button
                onclick={reset}
                class="px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 text-sm"
                >{$_('error_page.retry')}</button
              >
              <button
                onclick={() => window.location.reload()}
                class="px-4 py-2 border border-border rounded hover:bg-muted text-sm"
                >{$_('error_page.reload_page')}</button
              >
            </div>
          </div>
        </div>
      {/snippet}
    </svelte:boundary>
  {:else if EditorComponent}
    <svelte:boundary>
      <EditorComponent {noteId} />
      {#snippet failed(error, reset)}
        {@const msg = error instanceof Error ? error.message : ''}
        <div class="flex items-center justify-center h-screen-safe">
          <div class="text-center text-destructive">
            <AlertCircle class="w-8 h-8 mx-auto mb-2" />
            <p class="mb-1 font-medium">{$_('error_page.component_crashed')}</p>
            {#if msg}<p class="mb-4 text-sm text-muted-foreground">{msg}</p>{/if}
            <div class="flex items-center justify-center gap-2">
              <button
                onclick={reset}
                class="px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 text-sm"
                >{$_('error_page.retry')}</button
              >
              <button
                onclick={() => window.location.reload()}
                class="px-4 py-2 border border-border rounded hover:bg-muted text-sm"
                >{$_('error_page.reload_page')}</button
              >
            </div>
          </div>
        </div>
      {/snippet}
    </svelte:boundary>
  {/if}
{:else}
  <div class="flex items-center justify-center h-full text-muted-foreground">
    {$_('page.note.no_id')}
  </div>
{/if}

<style>
  .canvas-workspace {
    display: flex;
    height: 100vh;
    height: 100dvh;
    background: var(--color-background);
  }

  .canvas-workspace-main {
    flex: 1;
    min-width: 0;
    border-right: 1px solid var(--color-border);
  }

  .canvas-workspace-sidebar {
    width: 320px;
    flex-shrink: 0;
    overflow-y: auto;
    background: color-mix(in oklch, var(--color-sidebar-background) 88%, var(--color-background));
    padding: 12px;
  }

  .canvas-workspace-panel {
    padding: 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: color-mix(in oklch, var(--color-card) 92%, transparent);
  }

  .canvas-workspace-panel + .canvas-workspace-panel {
    margin-top: 12px;
  }

  .canvas-workspace-heading {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0 0 10px;
    color: var(--color-foreground);
    font-size: 0.875rem;
    font-weight: 600;
  }

  .canvas-workspace-links {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .canvas-workspace-link {
    display: block;
    padding: 6px 8px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-size: 0.875rem;
    color: var(--color-foreground);
    background: color-mix(in oklch, var(--color-accent) 70%, transparent);
  }

  .canvas-workspace-link:hover {
    background: color-mix(in oklch, var(--color-accent) 88%, transparent);
  }

  .canvas-workspace-empty {
    margin: 0;
    color: var(--color-muted-foreground);
    font-size: 0.875rem;
  }
</style>
