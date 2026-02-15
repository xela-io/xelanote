<script lang="ts">
  import { ArrowLeft } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import GraphSkeleton from '$lib/components/skeletons/GraphSkeleton.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  let GraphComponent = $state<ComponentType | null>(null);

  onMount(async () => {
    const module = await import('$lib/components/Graph.svelte');
    GraphComponent = loadSvelteComponentFromModule(module, 'Graph');
  });
</script>

<div class="flex flex-col h-screen-safe bg-background">
  <header class="flex items-center gap-4 p-4 border-b border-border">
    <a href="/" class="p-2 hover:bg-accent rounded-md transition-colors">
      <ArrowLeft size={18} />
    </a>
    <h1 class="text-xl font-semibold">{$_('page.graph.title')}</h1>
    <div class="text-sm text-muted-foreground">
      {$_('page.graph.hint')}
    </div>
  </header>

  <main class="flex-1 overflow-hidden">
    {#if GraphComponent}
      <GraphComponent />
    {:else}
      <GraphSkeleton />
    {/if}
  </main>
</div>
