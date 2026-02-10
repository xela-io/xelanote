<script lang="ts">
  import { Info,Search } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as graph from '$lib/stores/graph.svelte';

  let searchQuery = $state('');
  let showInfo = $state(false);

  function handleSearch() {
    graph.setSearchFilter(searchQuery || null);
  }
</script>

<div class="flex items-center gap-2 p-4 border-b border-border bg-background">
  <!-- Search -->
  <div class="flex-1 flex items-center gap-2 px-3 py-2 bg-muted/50 border border-border rounded-md">
    <Search size={18} class="text-muted-foreground" />
    <input
      bind:value={searchQuery}
      oninput={handleSearch}
      type="text"
      placeholder={$_('component.graph.search_placeholder')}
      class="flex-1 bg-transparent outline-none"
    />
  </div>

  <!-- Info button -->
  <button
    onclick={() => (showInfo = !showInfo)}
    class="p-2 hover:bg-accent rounded-md transition-colors"
    title={$_('component.graph.info')}
  >
    <Info size={18} />
  </button>
</div>

{#if showInfo && graph.getMetadata()}
  <div class="p-4 border-b border-border bg-muted/30 text-sm">
    <p><strong>{$_('component.graph.nodes')}:</strong> {graph.getMetadata()?.node_count}</p>
    <p><strong>{$_('component.graph.connections')}:</strong> {graph.getMetadata()?.edge_count}</p>
    {#if graph.getMetadata()?.truncated}
      <p class="text-amber-600 mt-2">{$_('component.graph.truncated_warning')}</p>
    {/if}
    <div class="mt-2 flex gap-4 text-xs text-muted-foreground">
      <span>{$_('component.graph.existing_notes')}</span>
      <span>{$_('component.graph.unresolved_links')}</span>
    </div>
  </div>
{/if}
