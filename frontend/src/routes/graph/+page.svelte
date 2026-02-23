<script lang="ts">
  import { ArrowLeft } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import GraphSkeleton from '$lib/components/skeletons/GraphSkeleton.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  let GraphComponent = $state<ComponentType | null>(null);

  onMount(async () => {
    const module = await import('$lib/components/Graph.svelte');
    GraphComponent = loadSvelteComponentFromModule(module, 'Graph');
  });
</script>

<div class="flex flex-col h-screen-safe bg-background">
  <header class="flex items-center gap-2 sm:gap-4 p-4 border-b border-border">
    <MobileSidebarInlineToggle />
    <a href="/" class="p-2 hover:bg-accent rounded-md transition-colors">
      <ArrowLeft size={18} />
    </a>
    <h1 class="text-xl font-semibold">{$_('page.graph.title')}</h1>
    <div class="text-sm text-muted-foreground hidden sm:block">
      {$_('page.graph.hint')}
    </div>
  </header>

  <main class="flex-1 overflow-hidden">
    {#if GraphComponent}
      <svelte:boundary>
        <GraphComponent />
        {#snippet failed(error, reset)}
          {@const msg = error instanceof Error ? error.message : ''}
          <div class="flex items-center justify-center h-full">
            <div class="text-center text-muted-foreground">
              <p class="mb-2">{$_('error_page.component_crashed')}</p>
              {#if msg}<p class="mb-4 text-sm">{msg}</p>{/if}
              <button
                onclick={reset}
                class="px-4 py-2 bg-primary text-primary-foreground rounded text-sm hover:bg-primary/90"
                >{$_('error_page.retry')}</button
              >
            </div>
          </div>
        {/snippet}
      </svelte:boundary>
    {:else}
      <GraphSkeleton />
    {/if}
  </main>
</div>
