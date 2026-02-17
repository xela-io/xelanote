<script lang="ts">
  import { RefreshCw } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';
</script>

<div
  role="status"
  aria-live="polite"
  class="fixed top-3 right-3 z-50 inline-flex items-center gap-2 rounded-full bg-blue-600/95 px-3 py-1.5 text-xs font-medium text-white shadow-lg backdrop-blur-sm"
>
  <RefreshCw size={14} class="animate-spin shrink-0" />
  <span>{$_('common.session_restoring')}</span>
</div>
