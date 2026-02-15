<script lang="ts">
  import { ArrowLeft, Menu } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as ui from '$lib/stores/ui.svelte';

  import Logo from './Logo.svelte';

  function goBack() {
    if (ui.canGoBack()) {
      window.history.back();
    } else {
      goto('/');
    }
  }
</script>

<div class="flex items-center px-4 py-2 border-b border-border bg-background">
  {#if ui.getIsStandalone()}
    <button
      onclick={goBack}
      class="min-h-12 min-w-12 flex items-center justify-center -ml-2 rounded-md hover:bg-accent toolbar-btn"
      aria-label={$_('editor.back')}
    >
      <ArrowLeft size={20} />
    </button>
  {/if}
  <button
    onclick={() => ui.setSidebarOpen(true)}
    class="min-h-12 min-w-12 flex items-center justify-center rounded-md hover:bg-accent toolbar-btn"
    class:-ml-2={!ui.getIsStandalone()}
    aria-label="Menü öffnen"
  >
    <Menu size={20} />
  </button>
  <span class="ml-2"><Logo size="sm" /></span>
</div>
