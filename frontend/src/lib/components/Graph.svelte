<script lang="ts">
  import { Loader } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import * as graph from '$lib/stores/graph.svelte';

  import GraphCanvas from './GraphCanvas.svelte';
  import GraphControls from './GraphControls.svelte';

  onMount(async () => {
    await graph.loadGlobalGraph();
  });
</script>

<div class="flex flex-col h-full bg-background">
  <GraphControls />
  <div class="flex-1 relative p-3 pt-0">
    {#if graph.getLoading()}
      <div class="absolute inset-0 flex items-center justify-center gap-2 bg-background/85 backdrop-blur-sm z-20">
        <Loader class="animate-spin" size={24} />
        <span>{$_('component.graph.loading')}</span>
      </div>
    {:else if graph.getError()}
      <div class="absolute inset-0 p-4 text-destructive bg-background/85 backdrop-blur-sm z-20">
        <p>{$_('component.graph.error_loading')}</p>
        <p class="text-sm">{graph.getError()}</p>
      </div>
    {/if}

    <div class="relative h-full overflow-hidden rounded-2xl border border-border/70 shadow-sm">
      <GraphCanvas nodes={graph.getNodes()} edges={graph.getEdges()} />
    </div>
  </div>
</div>
