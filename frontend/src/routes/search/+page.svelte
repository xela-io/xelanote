<script lang="ts">
  import type { SvelteVirtualizer } from '@tanstack/svelte-virtual';
  import { createVirtualizer } from '@tanstack/svelte-virtual';
  import { FileText, Lock, Search } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import type { SearchResult } from '$lib/api';
  import { search } from '$lib/api';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import {
    buildIndex,
    cancelBuild,
    getIndexProgress,
    getIndexState,
    searchEncrypted,
  } from '$lib/stores/search-index.svelte';

  // Parse snippet to safely render search highlights (defense in depth)
  type SnippetPart = { text: string; highlighted: boolean };
  function parseSnippet(snippet: string): SnippetPart[] {
    const parts: SnippetPart[] = [];
    const regex = /<mark>(.*?)<\/mark>/g;
    let lastIndex = 0;
    let match;

    while ((match = regex.exec(snippet)) !== null) {
      // Add text before the match
      if (match.index > lastIndex) {
        parts.push({ text: snippet.slice(lastIndex, match.index), highlighted: false });
      }
      // Add the highlighted text
      parts.push({ text: match[1], highlighted: true });
      lastIndex = regex.lastIndex;
    }

    // Add any remaining text
    if (lastIndex < snippet.length) {
      parts.push({ text: snippet.slice(lastIndex), highlighted: false });
    }

    return parts;
  }

  function getDisplayTitle(result: SearchResult): string {
    if (result.title_encrypted && result.encrypted_title) {
      const decrypted = encryption.decryptTitle(result.encrypted_title);
      return decrypted ?? result.id.substring(0, 8) + '...';
    }
    return result.title || result.id.substring(0, 8) + '...';
  }

  /**
   * Merge server and client results with 2:1 interleaving.
   * Deduplicates by note ID, preferring client snippet for duplicates.
   */
  function mergeResults(server: SearchResult[], client: SearchResult[]): SearchResult[] {
    const serverIds = new Set(server.map((r) => r.id));
    const uniqueClient = client.filter((r) => !serverIds.has(r.id));

    // For duplicates: prefer client snippet (real content match)
    for (const cr of client) {
      if (serverIds.has(cr.id) && cr.snippet) {
        const idx = server.findIndex((s) => s.id === cr.id);
        if (idx !== -1) server[idx].snippet = cr.snippet;
      }
    }

    // Interleaving: 2 server results, then 1 client result
    const merged: SearchResult[] = [];
    let si = 0,
      ci = 0;
    while (si < server.length || ci < uniqueClient.length) {
      if (si < server.length) merged.push(server[si++]);
      if (si < server.length) merged.push(server[si++]);
      if (ci < uniqueClient.length) merged.push(uniqueClient[ci++]);
    }
    return merged;
  }

  const query = $derived(page.url.searchParams.get('q') ?? '');
  let results = $state<SearchResult[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let scrollElement = $state<HTMLDivElement | null>(null);

  // Virtualizer state - updated via effect
  let virtualizerValue = $state<SvelteVirtualizer<HTMLDivElement, Element> | null>(null);

  $effect(() => {
    if (query) {
      performSearch(query);
    }
  });

  // Subscribe to virtualizer store when it changes
  $effect(() => {
    if (results.length > 0 && scrollElement) {
      const store = createVirtualizer({
        count: results.length,
        getScrollElement: () => scrollElement!,
        estimateSize: () => 80,
        overscan: 5,
      });

      const unsubscribe = store.subscribe((value) => {
        virtualizerValue = value;
      });

      return unsubscribe;
    } else {
      virtualizerValue = null;
    }
  });

  async function performSearch(q: string) {
    loading = true;
    error = null;
    try {
      const serverResponse = await search(q, 50);
      const clientResults = searchEncrypted(q, 50);

      results = mergeResults(serverResponse.results, clientResults);
    } catch (e) {
      error = e instanceof Error ? e.message : $_('page.search.search_failed');
      results = [];
    } finally {
      loading = false;
    }
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const input = form.querySelector('input') as HTMLInputElement;
    if (input.value.trim()) {
      goto(`/search?q=${encodeURIComponent(input.value)}`);
    }
  }

  const virtualItems = $derived(virtualizerValue?.getVirtualItems() ?? []);
  const totalSize = $derived(virtualizerValue?.getTotalSize() ?? 0);
</script>

<svelte:head>
  <title>{$_('page.search.page_title', { values: { query } })}</title>
</svelte:head>

<div class="h-full overflow-auto p-6">
  <div class="max-w-2xl mx-auto">
    <div class="mb-6 flex items-center gap-2 sm:gap-3">
      <MobileSidebarInlineToggle />
      <h1 class="text-2xl font-bold">{$_('page.search.title')}</h1>
    </div>

    <form onsubmit={handleSubmit} class="mb-6">
      <div class="ui-panel p-2 flex gap-2">
        <div class="flex-1 relative">
          <Search
            size={18}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
          />
          <input
            type="text"
            value={query}
            placeholder={$_('page.search.placeholder')}
            aria-label={$_('page.search.input_label')}
            autocomplete="off"
            inputmode="search"
            class="ui-input w-full pl-10 pr-4 py-2"
          />
        </div>
        <button type="submit" class="ui-button ui-button-primary">
          {$_('page.search.submit')}
        </button>
      </div>
    </form>

    {#if getIndexState() === 'building'}
      <div
        class="ui-panel-soft text-sm text-muted-foreground p-3 mb-4 flex items-center justify-between"
      >
        <span
          >{$_('page.search.building_index')} ({getIndexProgress().current}/{getIndexProgress()
            .total})</span
        >
        <button onclick={() => cancelBuild()} class="ui-button ui-button-ghost px-2 py-1 text-xs">
          {$_('page.search.cancel')}
        </button>
      </div>
    {:else if getIndexState() === 'error'}
      <div
        class="ui-panel-soft text-sm text-destructive bg-destructive/10 p-3 mb-4 flex items-center justify-between"
      >
        <span>{$_('page.search.index_error')}</span>
        <button onclick={() => buildIndex()} class="ui-button ui-button-ghost px-2 py-1 text-xs">
          {$_('page.search.retry')}
        </button>
      </div>
    {/if}

    {#if loading}
      <div class="ui-empty-state ui-empty-state-compact py-8" role="status" aria-live="polite">
        {$_('page.search.searching')}
      </div>
    {:else if error}
      <div class="ui-empty-state ui-empty-state-compact py-8 text-destructive">
        {error}
      </div>
    {:else if query && results.length === 0}
      <div class="ui-empty-state ui-empty-state-compact py-8">
        {$_('page.search.no_results', { values: { query } })}
      </div>
    {:else if results.length > 0}
      <div class="text-sm text-muted-foreground mb-4">
        {$_('page.search.results_count', { values: { count: results.length, query } })}
      </div>

      <!-- Virtual scrolling container -->
      <div
        bind:this={scrollElement}
        class="overflow-auto"
        style="max-height: calc(var(--app-viewport-height, 100dvh) - 300px);"
      >
        <div style="height: {totalSize}px; width: 100%; position: relative;">
          {#each virtualItems as virtualRow (virtualRow.key)}
            {@const result = results[virtualRow.index]}
            <a
              href="/note/{result.id}?highlight={encodeURIComponent(query)}"
              class="ui-list-item block p-4"
              style="position: absolute; top: 0; left: 0; width: 100%; transform: translateY({virtualRow.start}px);"
            >
              <div class="flex items-start gap-3">
                {#if result.encrypted}
                  <Lock size={20} class="text-muted-foreground shrink-0 mt-0.5" />
                {:else}
                  <FileText size={20} class="text-muted-foreground shrink-0 mt-0.5" />
                {/if}
                <div class="min-w-0">
                  <h2 class="font-medium truncate">{getDisplayTitle(result)}</h2>
                  <p class="text-sm text-muted-foreground mt-1 line-clamp-2">
                    {#if result.snippet}
                      {#each parseSnippet(result.snippet) as part, pi (pi)}
                        {#if part.highlighted}<mark>{part.text}</mark>{:else}{part.text}{/if}
                      {/each}
                    {:else if result.matched_keywords?.length}
                      {$_('page.search.keyword_match')}: {result.matched_keywords.join(', ')}
                    {:else if result.encrypted}
                      {$_('page.search.encrypted_note')}
                    {/if}
                  </p>
                </div>
              </div>
            </a>
          {/each}
        </div>
      </div>
    {:else}
      <div class="ui-empty-state ui-empty-state-compact py-8">
        {$_('page.search.enter_query')}
      </div>
    {/if}
  </div>
</div>
