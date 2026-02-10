<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { X, ChevronUp, ChevronDown, CaseSensitive, Replace } from 'lucide-svelte';
  import type { EditorView } from '@codemirror/view';
  import {
    performSearch,
    goToNextMatch,
    goToPreviousMatch,
    performReplace,
    performReplaceAll,
    clearSearch,
    getMatchCount,
  } from '$lib/editor/find-replace';

  interface Props {
    editorView: EditorView | undefined;
    initialQuery?: string;
    showReplace?: boolean;
    isReadOnly?: boolean;
    onClose: () => void;
    onMatchChange?: (info: { current: number; total: number }) => void;
    onQueryChange?: (query: string, caseSensitive: boolean) => void;
  }

  const {
    editorView,
    initialQuery = '',
    showReplace: initialShowReplace = false,
    isReadOnly = false,
    onClose,
    onMatchChange,
    onQueryChange,
  }: Props = $props();

  let searchQuery = $state(initialQuery);
  let replaceQuery = $state('');
  let caseSensitive = $state(false);
  let showReplaceRow = $state(initialShowReplace);
  let matchInfo = $state({ current: 0, total: 0 });
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let searchInputRef: HTMLInputElement | null = $state(null);

  // Focus the search input on mount
  $effect(() => {
    if (searchInputRef) {
      searchInputRef.focus();
      if (initialQuery) {
        searchInputRef.select();
      }
    }
  });

  // Initialize with initialQuery
  $effect(() => {
    if (initialQuery && editorView) {
      searchQuery = initialQuery;
      executeSearch(initialQuery);
    }
  });

  function executeSearch(query: string) {
    // Notify parent of query change (for preview highlighting)
    onQueryChange?.(query, caseSensitive);

    if (!editorView) return;

    performSearch(editorView, query, { caseSensitive });

    // Update match count
    updateMatchCount();
  }

  function updateMatchCount() {
    if (!editorView) return;

    const info = getMatchCount(editorView);
    matchInfo = info;
    onMatchChange?.(info);
  }

  function handleSearchInput(e: Event) {
    const value = (e.target as HTMLInputElement).value;
    searchQuery = value;

    // Debounced search
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      executeSearch(value);
    }, 150);
  }

  function flushDebounce() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    executeSearch(searchQuery);
  }

  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      flushDebounce();
      if (e.shiftKey) {
        handlePrev();
      } else {
        handleNext();
      }
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleEscape();
    }
  }

  function handleReplaceKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleReplaceSingle();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleEscape();
    }
  }

  function handleNext() {
    if (!editorView) return;
    goToNextMatch(editorView);
    updateMatchCount();
  }

  function handlePrev() {
    if (!editorView) return;
    goToPreviousMatch(editorView);
    updateMatchCount();
  }

  function handleCaseSensitiveToggle() {
    caseSensitive = !caseSensitive;
    executeSearch(searchQuery);
  }

  function handleReplaceSingle() {
    if (!editorView || !searchQuery) return;
    performReplace(editorView, searchQuery, replaceQuery, { caseSensitive });
    updateMatchCount();
  }

  function handleReplaceAll() {
    if (!editorView || !searchQuery) return;
    performReplaceAll(editorView, searchQuery, replaceQuery, { caseSensitive });
    updateMatchCount();
  }

  function handleEscape() {
    if (showReplaceRow) {
      // First escape: close replace row
      showReplaceRow = false;
    } else {
      // Second escape: close entire bar
      handleClose();
    }
  }

  function handleClose() {
    flushDebounce();
    if (editorView) {
      clearSearch(editorView);
    }
    onClose();
  }

  function toggleReplaceRow() {
    showReplaceRow = !showReplaceRow;
  }

  // Expose method for parent to toggle replace
  export function setShowReplace(show: boolean) {
    showReplaceRow = show;
  }
</script>

<div class="find-replace-bar" role="search" aria-label={$_('component.editor.find.placeholder')}>
  <!-- Search Row -->
  <div class="flex items-center gap-1">
    {#if !isReadOnly}
      <button
        type="button"
        class="p-1 hover:bg-accent rounded flex-shrink-0"
        onclick={toggleReplaceRow}
        aria-label={$_('component.editor.find.replace')}
        aria-expanded={showReplaceRow}
      >
        <Replace size={14} class={showReplaceRow ? 'text-primary' : ''} />
      </button>
    {/if}

    <input
      bind:this={searchInputRef}
      type="text"
      value={searchQuery}
      oninput={handleSearchInput}
      onkeydown={handleSearchKeydown}
      placeholder={$_('component.editor.find.placeholder')}
      class="flex-1 min-w-0 px-2 py-1 text-sm bg-background border border-input rounded focus:ring-1 focus:ring-ring focus:border-transparent"
      aria-label={$_('component.editor.find.placeholder')}
    />

    <!-- Match counter -->
    <span
      class="text-xs text-muted-foreground whitespace-nowrap min-w-[60px] text-center"
      aria-live="polite"
    >
      {#if searchQuery}
        {#if matchInfo.total === 0}
          {$_('component.editor.find.no_matches')}
        {:else}
          {$_('component.editor.find.match_count', {
            values: { current: matchInfo.current, total: matchInfo.total },
          })}
        {/if}
      {/if}
    </span>

    <!-- Navigation -->
    <button
      type="button"
      class="p-1 hover:bg-accent rounded flex-shrink-0 disabled:opacity-30"
      onclick={handlePrev}
      disabled={matchInfo.total === 0}
      aria-label="Previous match"
    >
      <ChevronUp size={14} />
    </button>
    <button
      type="button"
      class="p-1 hover:bg-accent rounded flex-shrink-0 disabled:opacity-30"
      onclick={handleNext}
      disabled={matchInfo.total === 0}
      aria-label="Next match"
    >
      <ChevronDown size={14} />
    </button>

    <!-- Case Sensitive Toggle -->
    <button
      type="button"
      class="p-1 hover:bg-accent rounded flex-shrink-0"
      class:bg-accent={caseSensitive}
      onclick={handleCaseSensitiveToggle}
      aria-label={$_('component.editor.find.case_sensitive')}
      aria-pressed={caseSensitive}
      title={$_('component.editor.find.case_sensitive')}
    >
      <CaseSensitive size={14} />
    </button>

    <!-- Close -->
    <button
      type="button"
      class="p-1 hover:bg-accent rounded flex-shrink-0"
      onclick={handleClose}
      aria-label={$_('component.editor.find.close')}
    >
      <X size={14} />
    </button>
  </div>

  <!-- Replace Row -->
  {#if showReplaceRow && !isReadOnly}
    <div class="flex items-center gap-1 mt-1">
      <!-- Spacer to align with search input (replace toggle button width) -->
      <div class="w-[26px] flex-shrink-0"></div>

      <input
        type="text"
        bind:value={replaceQuery}
        onkeydown={handleReplaceKeydown}
        placeholder={$_('component.editor.find.replace_placeholder')}
        class="flex-1 min-w-0 px-2 py-1 text-sm bg-background border border-input rounded focus:ring-1 focus:ring-ring focus:border-transparent"
        aria-label={$_('component.editor.find.replace_placeholder')}
      />

      <button
        type="button"
        class="px-2 py-1 text-xs hover:bg-accent rounded flex-shrink-0 disabled:opacity-30"
        onclick={handleReplaceSingle}
        disabled={matchInfo.total === 0}
      >
        {$_('component.editor.find.replace')}
      </button>
      <button
        type="button"
        class="px-2 py-1 text-xs hover:bg-accent rounded flex-shrink-0 disabled:opacity-30"
        onclick={handleReplaceAll}
        disabled={matchInfo.total === 0}
      >
        {$_('component.editor.find.replace_all')}
      </button>
    </div>
  {/if}
</div>

<style>
  .find-replace-bar {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 20;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 0 0 0 0.5rem;
    padding: 0.5rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    max-width: min(400px, calc(100vw - 2rem));
  }
</style>
