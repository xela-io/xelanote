<script lang="ts">
  import { X } from 'lucide-svelte';
  import { fly } from 'svelte/transition';
  import { _ } from 'svelte-i18n';

  import * as toast from '$lib/stores/toast.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  const toasts = $derived(toast.toastState.toasts);

  const bottomOffset = $derived(
    ui.getIsMobile() && !ui.getIsKeyboardOpen()
      ? 'calc(4.5rem + var(--safe-area-inset-bottom))'
      : 'max(1rem, env(safe-area-inset-bottom, 0))'
  );

  function getTypeClasses(type: string): string {
    switch (type) {
      case 'success':
        return 'bg-primary text-primary-foreground';
      case 'error':
        return 'bg-destructive text-destructive-foreground';
      case 'warning':
        return 'bg-yellow-500 text-white';
      case 'info':
      default:
        return 'bg-accent text-accent-foreground';
    }
  }
</script>

<!-- Toast Container -->
<div
  role="region"
  aria-label={$_('accessibility.notifications')}
  class="fixed left-4 right-4 sm:left-auto sm:right-4 z-50 flex flex-col gap-2 pointer-events-none"
  style:bottom={bottomOffset}
>
  {#each toasts as t (t.id)}
    <div
      transition:fly={{ x: 200, duration: 300 }}
      role={t.type === 'error' ? 'alert' : 'status'}
      aria-live={t.type === 'error' ? 'assertive' : 'polite'}
      aria-atomic="true"
      class="pointer-events-auto flex items-center gap-3 rounded-lg shadow-lg px-4 py-3 sm:min-w-[300px] max-w-full sm:max-w-[500px] {getTypeClasses(
        t.type
      )}"
    >
      <!-- Message -->
      <div class="flex-1">
        {t.message}
      </div>

      <!-- Action Button -->
      {#if t.action}
        <button
          class="px-3 py-1 rounded bg-white bg-opacity-20 hover:bg-opacity-30 font-medium transition-colors"
          onclick={() => {
            t.action?.handler();
            toast.removeToast(t.id);
          }}
        >
          {t.action.label}
        </button>
      {/if}

      <!-- Close Button -->
      <button
        class="p-1 hover:bg-white hover:bg-opacity-20 rounded transition-colors"
        onclick={() => toast.removeToast(t.id)}
        aria-label={$_('accessibility.close_notification')}
      >
        <X size={18} />
      </button>
    </div>
  {/each}
</div>
