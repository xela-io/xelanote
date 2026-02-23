<script lang="ts">
  import { PanelLeft } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as ui from '$lib/stores/ui.svelte';
</script>

<div class="w-9 shrink-0 md:hidden">
  <button
    type="button"
    onclick={() => ui.toggleSidebar()}
    class="toolbar-btn inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-all duration-200 hover:bg-accent active:scale-95"
    style="-webkit-tap-highlight-color: transparent"
    aria-label={ui.getSidebarOpen() ? $_('page.sidebar.close_drawer') : $_('nav.notes_tree')}
    title={ui.getSidebarOpen() ? $_('page.sidebar.close_drawer') : $_('nav.notes_tree')}
  >
    <PanelLeft size={16} />
  </button>
</div>
