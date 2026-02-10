<script lang="ts">
  import { Sun, Moon } from 'lucide-svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import * as settings from '$lib/stores/settings.svelte';

  const isDark = $derived(ui.getCurrentTheme().variant === 'dark');

  async function toggle() {
    const newTheme = isDark ? 'gruvbox-light' : 'gruvbox-dark';
    await settings.setThemePreference(newTheme);
  }
</script>

<button
  onclick={toggle}
  class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground flex items-center toolbar-btn"
  title={isDark ? 'Helles Theme' : 'Dunkles Theme'}
  aria-label={isDark ? 'Zum hellen Theme wechseln' : 'Zum dunklen Theme wechseln'}
>
  {#if isDark}
    <Sun size={16} />
  {:else}
    <Moon size={16} />
  {/if}
</button>
