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

<div class="flex h-full flex-col bg-background">
  <GraphControls />
  <div class="relative flex-1 px-4 pb-4 pt-0 sm:px-6 sm:pb-6">
    {#if graph.getLoading()}
      <div
        class="absolute inset-0 z-20 flex items-center justify-center gap-2 bg-background/85 backdrop-blur-sm"
      >
        <Loader class="animate-spin" size={24} />
        <span>{$_('component.graph.loading')}</span>
      </div>
    {:else if graph.getError()}
      <div
        class="absolute inset-0 z-20 grid place-items-center bg-background/85 p-4 backdrop-blur-sm"
      >
        <div class="ui-panel-soft max-w-md p-4 text-destructive">
          <p class="font-medium">{$_('component.graph.error_loading')}</p>
          <p class="mt-1 text-sm">{graph.getError()}</p>
        </div>
      </div>
    {/if}

    <div class="ui-panel relative h-full overflow-hidden p-0">
      <GraphCanvas nodes={graph.getNodes()} edges={graph.getEdges()} />

      {#if !graph.getLoading() && !graph.getError() && graph.getNodes().length === 0}
        <div class="absolute inset-0 grid place-items-center p-4">
          <div class="ui-panel-soft max-w-sm p-4">
            <div class="ui-empty-state ui-empty-state-compact">
              <span class="text-sm font-medium text-foreground"
                >{$_('component.graph.empty_title')}</span
              >
              <p class="text-xs text-muted-foreground">
                {$_('component.graph.empty_description')}
              </p>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
