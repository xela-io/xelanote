<script lang="ts">
  import { Sparkles, Plus, Loader2, AlertCircle, ChevronDown, ChevronRight } from 'lucide-svelte';
  import * as api from '$lib/api';
  import type { TagSuggestion } from '$lib/api';
  import * as toast from '$lib/stores/toast.svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    noteId: string;
    isEncrypted: boolean;
    plaintextContent?: string;
    existingTagNames: string[];
    onAddTag: (tagName: string) => Promise<void>;
  }

  const { noteId, isEncrypted, plaintextContent, existingTagNames, onAddTag }: Props = $props();

  let expanded = $state(false);
  let loading = $state(false);
  let suggestions = $state<TagSuggestion[]>([]);
  let error = $state<string | null>(null);
  let aiDisabled = $state(false);
  let hasGenerated = $state(false);

  // Filter out already existing tags from suggestions
  const filteredSuggestions = $derived(() => {
    const existingSet = new Set(existingTagNames.map((t) => t.toLowerCase()));
    return suggestions.filter((s) => !existingSet.has(s.name.toLowerCase()));
  });

  async function generateSuggestions() {
    if (loading) return;

    // For encrypted notes, check if we have plaintext content
    if (isEncrypted && !plaintextContent) {
      error = $_('tagSuggestions.privacyNote');
      return;
    }

    loading = true;
    error = null;
    aiDisabled = false;

    try {
      const content = isEncrypted ? plaintextContent : undefined;
      suggestions = await api.suggestTags(noteId, content);
      hasGenerated = true;
    } catch (e: unknown) {
      console.error('Failed to generate tag suggestions:', e);
      const err = e as { status?: number; message?: string };
      if (err.status === 403 || err.message?.includes('AI features not enabled')) {
        aiDisabled = true;
      } else {
        error = err.message || $_('tagSuggestions.error');
      }
    } finally {
      loading = false;
    }
  }

  async function handleSelectTag(tagName: string) {
    try {
      await onAddTag(tagName);
      // Tag is now in the input field for editing - don't remove from suggestions
      // and don't show toast since user still needs to confirm
    } catch (e: unknown) {
      console.error('Failed to select tag:', e);
      toast.error(e instanceof Error ? e.message : 'Failed to select tag');
    }
  }

  function toggleExpanded() {
    expanded = !expanded;
    // Auto-generate on first expand if no suggestions yet
    if (expanded && !hasGenerated && !loading) {
      generateSuggestions();
    }
  }

  // Score to opacity mapping (0.5-1.0 range)
  function scoreToOpacity(score: number): number {
    return 0.5 + score * 0.5;
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
    <Sparkles size={14} />
    <span>{$_('tagSuggestions.title')}</span>
    {#if loading}
      <Loader2 size={14} class="animate-spin ml-auto" />
    {/if}
  </button>

  {#if expanded}
    <div class="pl-6 space-y-2">
      <!-- Privacy note for encrypted notes -->
      {#if isEncrypted && !plaintextContent}
        <div class="text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
          <AlertCircle size={12} />
          <span>{$_('tagSuggestions.privacyNote')}</span>
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
      {#if filteredSuggestions().length > 0}
        <div class="flex flex-wrap gap-1.5">
          {#each filteredSuggestions() as suggestion (suggestion.name)}
            <button
              type="button"
              onclick={() => handleSelectTag(suggestion.name)}
              class="inline-flex items-center gap-1 px-2 py-0.5 bg-accent/80 hover:bg-accent rounded-full text-sm transition-colors"
              style="opacity: {scoreToOpacity(suggestion.score)}"
              title={$_('tagSuggestions.add') + ` (${Math.round(suggestion.score * 100)}%)`}
            >
              <Plus size={12} />
              <span>{suggestion.name}</span>
              {#if suggestion.is_new}
                <span class="text-xs px-1 py-0.5 bg-primary/20 text-primary rounded">
                  {$_('tagSuggestions.newTag')}
                </span>
              {/if}
            </button>
          {/each}
        </div>
      {:else if hasGenerated && !loading}
        <div class="text-xs text-muted-foreground">
          {$_('tagSuggestions.noSuggestions')}
        </div>
      {/if}

      <!-- Regenerate button -->
      {#if !aiDisabled && (hasGenerated || !isEncrypted || plaintextContent)}
        <button
          type="button"
          onclick={generateSuggestions}
          disabled={loading}
          class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-accent rounded-md transition-colors disabled:opacity-50"
        >
          {#if loading}
            <Loader2 size={12} class="animate-spin" />
            <span>{$_('tagSuggestions.suggesting')}</span>
          {:else}
            <Sparkles size={12} />
            <span
              >{hasGenerated ? $_('tagSuggestions.regenerate') : $_('tagSuggestions.suggest')}</span
            >
          {/if}
        </button>
      {/if}
    </div>
  {/if}
</div>
