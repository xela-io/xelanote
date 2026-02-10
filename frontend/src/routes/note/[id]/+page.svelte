<script lang="ts">
  import { AlertCircle,Loader2 } from 'lucide-svelte';
  import type { Component } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { beforeNavigate,goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { deleteNote, permanentlyDeleteNote } from '$lib/api';
  import * as journal from '$lib/stores/journal.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as tree from '$lib/stores/tree.svelte';

  // Get note ID from URL params
  // $page.params.id ist immer definiert bei [id] route, aber TypeScript kennt das nicht
  const noteId = $derived($page.params.id);

  // Dynamic import state
  let EditorComponent = $state<Component<Record<string, unknown>> | null>(null);
  let RecipeEditorComponent = $state<Component<Record<string, unknown>> | null>(null);
  let editorLoading = $state(false);
  let editorLoadError = $state<string | null>(null);
  let retryCount = $state(0);
  const MAX_RETRIES = 3;

  // Check if the current note is a recipe
  const isRecipe = $derived(notes.getCurrentNote()?.note_type === 'recipe');

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
          const [editorModule, recipeModule] = await Promise.all([
            import('$lib/components/Editor.svelte'),
            import('$lib/components/RecipeEditor.svelte'),
          ]);

          if (cancelled) return; // Don't update if effect was cleaned up

          EditorComponent = editorModule.default;
          RecipeEditorComponent = recipeModule.default;
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
  {:else if isRecipe && RecipeEditorComponent}
    <RecipeEditorComponent {noteId} />
  {:else if EditorComponent}
    <EditorComponent {noteId} />
  {/if}
{:else}
  <div class="flex items-center justify-center h-full text-muted-foreground">
    {$_('page.note.no_id')}
  </div>
{/if}
