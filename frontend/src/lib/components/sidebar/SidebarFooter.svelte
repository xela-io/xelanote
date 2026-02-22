<script lang="ts">
  import { MessageSquareWarning, Shield } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as auth from '$lib/stores/auth.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';

  interface Props {
    isMobile: boolean;
    smallIconSize: number;
    onShowFeedback: () => void;
    /** Callback invoked after navigating (e.g. to close mobile sidebar). */
    onNavigate?: () => void;
  }

  const { isMobile, smallIconSize, onShowFeedback, onNavigate }: Props = $props();

  function navigateTo(path: string) {
    goto(path);
    onNavigate?.();
  }
</script>

{#if auth.isAdmin() || errorReporter.getServiceAvailable()}
  <div
    class="border-t border-sidebar-border shrink-0 {isMobile ? 'pb-safe' : ''}"
  >
    <div
      class="flex items-center {isMobile ? 'px-2 py-2 gap-1' : 'px-1.5 py-1.5 gap-0.5'}"
    >
      {#if auth.isAdmin()}
        <button
          onclick={() => navigateTo('/admin')}
          class="rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile
            ? 'p-2 toolbar-btn'
            : 'p-1.5'} shrink-0"
          title={$_('page.sidebar.admin')}
          aria-label={$_('page.sidebar.admin')}
        >
          <Shield size={smallIconSize} />
        </button>
      {/if}
      {#if errorReporter.getServiceAvailable()}
        <button
          onclick={onShowFeedback}
          class="rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile
            ? 'p-2 toolbar-btn'
            : 'p-1.5'} shrink-0"
          title={$_('feedback.sidebar_button')}
          aria-label={$_('feedback.sidebar_button')}
        >
          <MessageSquareWarning size={smallIconSize} />
        </button>
      {/if}
    </div>
  </div>
{/if}
