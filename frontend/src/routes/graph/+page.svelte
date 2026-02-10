<script lang="ts">
  import { ArrowLeft } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import Graph from '$lib/components/Graph.svelte';
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
    <Graph />
  </main>
</div>
