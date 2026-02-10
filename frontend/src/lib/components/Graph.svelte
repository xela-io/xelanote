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

<div class="flex flex-col h-full">
  <GraphControls />
  <div class="flex-1 relative">
    {#if graph.getLoading()}
      <div class="absolute inset-0 flex items-center justify-center gap-2 bg-background z-20">
        <Loader class="animate-spin" size={24} />
        <span>{$_('component.graph.loading')}</span>
      </div>
    {:else if graph.getError()}
      <div class="absolute inset-0 p-4 text-destructive bg-background z-20">
        <p>{$_('component.graph.error_loading')}</p>
        <p class="text-sm">{graph.getError()}</p>
      </div>
    {/if}

    <GraphCanvas nodes={graph.getNodes()} edges={graph.getEdges()} />
  </div>
</div>
