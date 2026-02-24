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

<div class="px-4 pt-4 pb-3 sm:px-6">
  <div class="ui-panel-soft flex items-center gap-2 px-3 py-2">
    <Search size={17} class="text-muted-foreground" />
    <input
      bind:value={searchQuery}
      oninput={handleSearch}
      type="text"
      placeholder={$_('component.graph.search_placeholder')}
      class="ui-input flex-1 border-0 bg-transparent px-0 py-0 text-sm shadow-none"
    />
    <kbd
      class="hidden sm:inline-flex px-2 py-0.5 rounded border border-border text-[11px] text-muted-foreground"
      >Ctrl+P</kbd
    >
    <button
      onclick={() => (showInfo = !showInfo)}
      class="ui-icon-button ui-icon-button-sm"
      title={$_('component.graph.info')}
      aria-label={$_('component.graph.info')}
    >
      <Info size={17} />
    </button>
  </div>
</div>

{#if showInfo && graph.getMetadata()}
  <div class="ui-panel-soft mx-4 mb-3 p-3 text-sm sm:mx-6">
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
