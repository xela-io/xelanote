<script lang="ts">
  import { Info, Search } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as graph from '$lib/stores/graph.svelte';

  let searchQuery = $state('');
  let showInfo = $state(false);

  function handleSearch() {
    graph.setSearchFilter(searchQuery || null);
  }
</script>

<div class="px-4 pt-4 pb-3 border-b border-border/70 bg-background/80 backdrop-blur-sm">
  <div class="flex items-center gap-2 rounded-xl border border-border/70 bg-card/70 px-3 py-2 shadow-sm">
    <Search size={17} class="text-muted-foreground" />
    <input
      bind:value={searchQuery}
      oninput={handleSearch}
      type="text"
      placeholder={$_('component.graph.search_placeholder')}
      class="flex-1 bg-transparent outline-none text-sm"
    />
    <kbd class="hidden sm:inline-flex px-2 py-0.5 rounded border border-border text-[11px] text-muted-foreground"
      >Ctrl+P</kbd
    >
    <button
      onclick={() => (showInfo = !showInfo)}
      class="p-1.5 hover:bg-accent rounded-lg transition-colors"
      title={$_('component.graph.info')}
      aria-label={$_('component.graph.info')}
    >
      <Info size={17} />
    </button>
  </div>
</div>

{#if showInfo && graph.getMetadata()}
  <div class="mx-4 mb-3 p-3 rounded-xl border border-border/70 bg-card/65 backdrop-blur-sm text-sm">
    <p><strong>{$_('component.graph.nodes')}:</strong> {graph.getMetadata()?.node_count}</p>
    <p><strong>{$_('component.graph.connections')}:</strong> {graph.getMetadata()?.edge_count}</p>
    {#if graph.getMetadata()?.truncated}
      <p class="text-warning mt-2">{$_('component.graph.truncated_warning')}</p>
    {/if}
    <div class="mt-2 flex flex-wrap gap-4 text-xs text-muted-foreground">
      <span class="inline-flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-primary"></span>
        {$_('component.graph.existing_notes')}
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-destructive"></span>
        {$_('component.graph.unresolved_links')}
      </span>
    </div>
  </div>
{/if}
