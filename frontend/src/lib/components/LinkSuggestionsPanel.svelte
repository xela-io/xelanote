<script lang="ts">
  import { AlertCircle, ChevronDown, ChevronRight, Link2, Loader2, Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { LinkSuggestion, NoteTitleInfo } from '$lib/api';
  import * as api from '$lib/api';
  import { extractWikilinks } from '$lib/editor/markdown';
  import * as toast from '$lib/stores/toast.svelte';

  interface Props {
    noteId: string;
    isEncrypted: boolean;
    plaintextContent?: string;
    onInsertLink: (term: string, targetTitle: string) => void;
  }

  const { noteId, isEncrypted, plaintextContent, onInsertLink }: Props = $props();

  let expanded = $state(false);
  let loading = $state(false);
  let loadingTitles = $state(false);
  let suggestions = $state<LinkSuggestion[]>([]);
  let noteTitles = $state<NoteTitleInfo[]>([]);
  let error = $state<string | null>(null);
  let aiDisabled = $state(false);
  let hasGenerated = $state(false);

  // Get the content to analyze
  const content = $derived(isEncrypted ? plaintextContent : plaintextContent);

  async function loadNoteTitles() {
    if (noteTitles.length > 0) return; // Already loaded

    loadingTitles = true;
    try {
      noteTitles = await api.getNoteTitles();
    } catch (e: unknown) {
      console.error('Failed to load note titles:', e);
      error = e instanceof Error ? e.message : 'Failed to load note titles';
    } finally {
      loadingTitles = false;
    }
  }

  async function generateSuggestions() {
    if (loading || !content) return;

    // P0 privacy hardening: never send encrypted-note plaintext to the server.
    if (isEncrypted) {
      aiDisabled = true;
      error = $_('ai.encrypted_processing_disabled');
      return;
    }

    loading = true;
    error = null;
    aiDisabled = false;

    try {
      // Make sure we have titles loaded
      await loadNoteTitles();

      // Extract existing wikilinks from content
      const existingLinks = extractWikilinks(content || '');
      const existingTitles = existingLinks.map((link) => link.title);

      // Get only unencrypted titles for the LLM
      const titleStrings = noteTitles.map((t) => t.title);

      // Call API
      const contentToSend = isEncrypted ? plaintextContent : undefined;
      suggestions = await api.suggestLinks(noteId, contentToSend, titleStrings, existingTitles);
      hasGenerated = true;
    } catch (e: unknown) {
      console.error('Failed to generate link suggestions:', e);
      const err = e as { status?: number; message?: string };
      if (err.status === 403 && err.message?.includes('encrypted notes')) {
        aiDisabled = true;
        error = $_('ai.encrypted_processing_disabled');
      } else if (err.status === 403 || err.message?.includes('AI features not enabled')) {
        aiDisabled = true;
      } else {
        error = err.message || $_('linkSuggestions.error');
      }
    } finally {
      loading = false;
    }
  }

  function handleInsertLink(suggestion: LinkSuggestion) {
    onInsertLink(suggestion.term, suggestion.target_title);
    // Remove from suggestions after inserting
    suggestions = suggestions.filter((s) => s.term !== suggestion.term);
    toast.success(`Wikilink [[${suggestion.target_title}]] eingefügt`);
  }

  function toggleExpanded() {
    expanded = !expanded;
    // Auto-load titles and generate on first expand
    if (expanded && !hasGenerated && !loading) {
      generateSuggestions();
    }
  }

  // Confidence to badge style
  function confidenceBadge(confidence: number): string {
    if (confidence >= 0.8) return 'bg-success/15 text-success';
    if (confidence >= 0.5)
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
    return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
  }
</script>

<div class="space-y-2">
  <!-- Header (clickable to expand) -->
  <button
    type="button"
    onclick={toggleExpanded}
    class="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors w-full text-left"
  >
    {#if expanded}
      <ChevronDown size={14} />
    {:else}
      <ChevronRight size={14} />
    {/if}
    <Link2 size={14} />
    <span>{$_('linkSuggestions.title')}</span>
    {#if loading || loadingTitles}
      <Loader2 size={14} class="animate-spin ml-auto" />
    {/if}
  </button>

  {#if expanded}
    <div class="pl-6 space-y-2">
      <!-- Privacy note for encrypted notes -->
      {#if isEncrypted}
        <div class="text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
          <AlertCircle size={12} />
          <span>{$_('ai.encrypted_processing_disabled')}</span>
        </div>
      {/if}

      <!-- AI disabled notice -->
      {#if aiDisabled}
        <p class="text-sm text-destructive">AI features not enabled for this note</p>
      {/if}

      <!-- Error message -->
      {#if error}
        <div class="text-xs text-destructive flex items-center gap-1">
          <AlertCircle size={12} />
          <span>{error}</span>
        </div>
      {/if}

      <!-- Suggestions list -->
      {#if suggestions.length > 0}
        <div class="space-y-1">
          {#each suggestions as suggestion (suggestion.term + suggestion.target_title)}
            <div class="flex items-center justify-between gap-2 p-2 bg-muted/50 rounded-md">
              <div class="flex-1 min-w-0">
                <div class="text-sm truncate">
                  <span class="text-muted-foreground">&ldquo;{suggestion.term}&rdquo;</span>
                  <span class="mx-1">&rarr;</span>
                  <span class="font-medium">[[{suggestion.target_title}]]</span>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <span
                  class="text-xs px-1.5 py-0.5 rounded {confidenceBadge(suggestion.confidence)}"
                >
                  {Math.round(suggestion.confidence * 100)}%
                </span>
                <button
                  type="button"
                  onclick={() => handleInsertLink(suggestion)}
                  class="inline-flex items-center gap-1 px-2 py-1 text-xs bg-primary text-primary-foreground hover:bg-primary/90 rounded transition-colors"
                >
                  <Plus size={12} />
                  {$_('linkSuggestions.insert')}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {:else if hasGenerated && !loading}
        <div class="text-xs text-muted-foreground">
          {$_('linkSuggestions.noSuggestions')}
        </div>
      {/if}

      <!-- Regenerate button -->
      {#if !aiDisabled && (hasGenerated || content)}
        <button
          type="button"
          onclick={generateSuggestions}
          disabled={loading || loadingTitles}
          class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-accent rounded-md transition-colors disabled:opacity-50"
        >
          {#if loading || loadingTitles}
            <Loader2 size={12} class="animate-spin" />
            <span>{$_('linkSuggestions.suggesting')}</span>
          {:else}
            <Link2 size={12} />
            <span
              >{hasGenerated
                ? $_('linkSuggestions.regenerate')
                : $_('linkSuggestions.suggest')}</span
            >
          {/if}
        </button>
      {/if}
    </div>
  {/if}
</div>
