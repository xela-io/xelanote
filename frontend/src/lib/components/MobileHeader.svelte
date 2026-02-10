<script lang="ts">
  import { Menu } from 'lucide-svelte';

  import * as ui from '$lib/stores/ui.svelte';

  import Logo from './Logo.svelte';
</script>

<div class="flex items-center px-4 py-2 border-b border-border bg-background">
  <button
    onclick={() => ui.setSidebarOpen(true)}
    class="p-2 -ml-2 rounded-md hover:bg-accent toolbar-btn"
    aria-label="Menü öffnen"
  >
    <Menu size={20} />
  </button>
  <span class="ml-2"><Logo size="sm" /></span>
</div>
