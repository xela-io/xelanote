<script lang="ts">
  import { ChevronDown, ChevronUp, Lock, RefreshCw, Sparkles } from 'lucide-svelte';
  import { untrack } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import * as api from '$lib/api';
  import { isEncryptionUnlocked } from '$lib/stores/encryption.svelte';
  import * as toast from '$lib/stores/toast.svelte';

  import Button from './Button.svelte';

  interface Props {
    note: Note;
    /** Decrypted content for encrypted notes (provided by parent) */
    decryptedContent?: string;
    onSummaryUpdated?: (summary: string) => void;
  }

  const { note, decryptedContent, onSummaryUpdated }: Props = $props();

  let loading = $state(false);
  let expanded = $state(true);
  let error = $state<string | null>(null);
  let streamingSummary = $state('');
  // Local state to store newly generated summary (ensures reactivity after generation)
  let generatedSummary = $state<string | null>(null);
  // Track note ID to reset local state when note changes
  let lastNoteId = $state(untrack(() => note.id));

  // Reset local state when switching notes
  $effect(() => {
    if (note.id !== lastNoteId) {
      lastNoteId = note.id;
      generatedSummary = null;
      error = null;
      streamingSummary = '';
    }
  });

  // Get the current summary (either plaintext or needs decryption)
  const summary = $derived.by(() => {
    // If we just generated a summary, use that (ensures immediate display)
    if (generatedSummary !== null) {
      return generatedSummary;
    }
    if (note.summary_encrypted && note.encrypted_summary) {
      // Encrypted summary - would need decryption by parent
      return null;
    }
    return note.summary || null;
  });

  const hasSummary = $derived(!!summary || (note.summary_encrypted && note.encrypted_summary));
  const isEncrypted = $derived(note.content_encrypted || false);
  const kekAvailable = $derived(isEncryptionUnlocked());
  const canGenerate = $derived(!isEncrypted || (isEncrypted && !!decryptedContent));

  // Show streaming text while generating, otherwise show saved summary
  const displaySummary = $derived(loading && streamingSummary ? streamingSummary : summary);

  async function generateSummary() {
    loading = true;
    error = null;
    streamingSummary = '';

    try {
      const plaintextContent = isEncrypted ? decryptedContent : undefined;

      if (isEncrypted && !plaintextContent) {
        error = $_('summary.encrypted_note');
        loading = false;
        return;
      }

      await api.summarizeNoteStream(
        note.id,
        // onToken - called for each token
        (token) => {
          streamingSummary += token;
        },
        // onComplete - called when done
        (finalSummary) => {
          loading = false;
          streamingSummary = '';
          // Store locally to ensure immediate display (Svelte reactivity workaround)
          generatedSummary = finalSummary;
          if (onSummaryUpdated) {
            onSummaryUpdated(finalSummary);
          }
          toast.success($_('summary.title'));
        },
        // onError - called on error
        (errorMsg) => {
          loading = false;
          streamingSummary = '';
          error = errorMsg || $_('summary.error');
          toast.error($_('summary.error'));
        },
        plaintextContent
      );
    } catch (err) {
      console.error('Failed to generate summary:', err);
      error = $_('summary.error');
      toast.error($_('summary.error'));
      loading = false;
    }
  }

  function toggleExpanded() {
    expanded = !expanded;
  }
</script>

<div class="summary-panel rounded-xl border border-border/60 bg-background/35 p-3.5">
  <button
    type="button"
    class="flex w-full items-center justify-between text-left rounded-lg px-1 py-0.5"
    onclick={toggleExpanded}
  >
    <div class="flex items-center gap-2">
      <Sparkles size={16} class="text-primary" />
      <span class="text-sm font-medium text-foreground">{$_('summary.title')}</span>
      {#if note.summary_generated_at}
        <span class="text-xs text-muted-foreground" title={$_('summary.tooltip')}>
          {new Date(note.summary_generated_at).toLocaleDateString()}
        </span>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      {#if isEncrypted && !kekAvailable}
        <Lock size={14} class="text-muted-foreground" />
      {/if}
      {#if expanded}
        <ChevronUp size={16} class="text-muted-foreground" />
      {:else}
        <ChevronDown size={16} class="text-muted-foreground" />
      {/if}
    </div>
  </button>

  {#if expanded}
    <div class="mt-3 space-y-3">
      {#if loading && streamingSummary}
        <!-- Streaming: show text as it arrives -->
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <RefreshCw size={12} class="animate-spin" />
            <span>{$_('summary.generating')}</span>
          </div>
          <p class="text-sm text-muted-foreground leading-relaxed">
            {streamingSummary}<span class="animate-pulse">▊</span>
          </p>
        </div>
      {:else if loading}
        <!-- Loading: waiting for first token -->
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <RefreshCw size={14} class="animate-spin" />
          <span>{$_('summary.generating')}</span>
        </div>
      {:else if error}
        <p class="text-sm text-destructive">{error}</p>
      {:else if hasSummary && displaySummary}
        <p class="text-sm text-muted-foreground leading-relaxed">{displaySummary}</p>
      {:else if isEncrypted && !kekAvailable}
        <p class="text-sm text-muted-foreground italic">{$_('summary.encrypted_note')}</p>
      {:else}
        <p class="text-sm text-muted-foreground italic">{$_('summary.empty')}</p>
      {/if}

      <div class="flex justify-end pt-1">
        <Button
          variant="ghost"
          size="sm"
          icon={RefreshCw}
          onclick={generateSummary}
          disabled={loading || !canGenerate}
        >
          {$_('summary.regenerate')}
        </Button>
      </div>
    </div>
  {/if}
</div>

<style>
  .summary-panel {
    transition:
      border-color var(--duration-fast) var(--ease-default),
      background-color var(--duration-fast) var(--ease-default);
  }

  .summary-panel:hover {
    border-color: color-mix(in oklch, var(--color-border), var(--color-primary) 12%);
    background: var(--surface-panel-bg-soft);
  }

  @media (max-width: 639px) {
    .summary-panel {
      border-color: color-mix(in oklch, var(--color-border), transparent 68%);
      background: color-mix(in oklch, var(--color-background), transparent 48%);
      padding: 0.75rem;
      border-radius: 0.9rem;
      backdrop-filter: none;
    }

    .summary-panel:hover {
      border-color: color-mix(in oklch, var(--color-border), transparent 58%);
      background: color-mix(in oklch, var(--color-background), transparent 42%);
    }
  }
</style>
