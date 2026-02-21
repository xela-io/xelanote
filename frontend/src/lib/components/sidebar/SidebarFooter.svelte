<script lang="ts">
  import {
    CalendarClock,
    MessageSquareWarning,
    Network,
    Search,
    Settings,
    Shield,
    Trash2,
    Users,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as auth from '$lib/stores/auth.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  import ThemeSelector from '../ThemeSelector.svelte';

  interface Props {
    isMobile: boolean;
    smallIconSize: number;
    onShowFeedback: () => void;
    /** Callback invoked after navigating (e.g. to close mobile sidebar). */
    onNavigate?: () => void;
  }

  const {
    isMobile,
    smallIconSize,
    onShowFeedback,
    onNavigate,
  }: Props = $props();

  function navigateTo(path: string) {
    goto(path);
    onNavigate?.();
  }
</script>

<!-- Due Dates, Shared, Trash as virtual folders at bottom of tree -->
<div class="mt-2 pt-2 border-t border-sidebar-border mx-1 space-y-0.5">
  <button
    onclick={() => navigateTo('/due-dates')}
    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
    title={$_('page.due_dates.title')}
    aria-label={$_('page.due_dates.title')}
  >
    <CalendarClock size={16} class="text-muted-foreground flex-shrink-0" />
    <span class="flex-1 truncate">{$_('page.due_dates.title')}</span>
  </button>
  <button
    onclick={() => navigateTo('/shared')}
    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
    title={$_('sharing.shared_with_me')}
    aria-label={$_('sharing.shared_with_me')}
  >
    <Users size={16} class="text-muted-foreground flex-shrink-0" />
    <span class="flex-1 truncate">{$_('sharing.shared_with_me')}</span>
    {#if sharing.getTotalSharedCount() > 0}
      <span
        class="bg-primary text-primary-foreground text-[10px] font-medium px-1.5 min-w-[18px] h-4 rounded-full flex items-center justify-center"
      >
        {sharing.getTotalSharedCount()}
      </span>
    {/if}
  </button>
  <button
    onclick={() => navigateTo('/trash')}
    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
    title={$_('page.sidebar.trash')}
    aria-label={$_('page.sidebar.trash')}
  >
    <Trash2 size={16} class="text-muted-foreground flex-shrink-0" />
    <span class="flex-1 truncate">{$_('page.sidebar.trash')}</span>
    {#if trash.getTrashCount() > 0}
      <span
        class="bg-destructive text-destructive-foreground text-[10px] font-medium px-1.5 min-w-[18px] h-4 rounded-full flex items-center justify-center"
      >
        {trash.getTrashCount()}
      </span>
    {/if}
  </button>
</div>

{#snippet controls()}
  <!-- Controls bar -->
  <div class="border-t border-sidebar-border shrink-0 {isMobile ? 'pb-safe' : ''}">
    <div class="px-2 py-2 flex items-center gap-{isMobile ? '1' : '0.5'} flex-wrap">
      <button
        onclick={() => {
          ui.toggleQuickSwitcher();
          if (isMobile) ui.closeSidebarOnMobile();
        }}
        class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''} shrink-0"
        title="{$_('page.sidebar.search')} (Ctrl+P)"
        aria-label={$_('page.sidebar.search')}
      >
        <Search size={smallIconSize} />
      </button>
      {#if features.getGraphFeatureEnabled()}
        <button
          onclick={() => navigateTo('/graph')}
          class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''} shrink-0"
          title="{$_('page.sidebar.graph')} (Ctrl+G)"
          aria-label={$_('page.sidebar.graph')}
        >
          <Network size={smallIconSize} />
        </button>
      {/if}
      <div class="shrink-0">
        <ThemeSelector />
      </div>
      <button
        onclick={() => navigateTo('/settings')}
        class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''} shrink-0"
        title={$_('page.sidebar.settings')}
        aria-label={$_('page.sidebar.settings')}
      >
        <Settings size={smallIconSize} />
      </button>
      {#if auth.isAdmin()}
        <button
          onclick={() => navigateTo('/admin')}
          class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''} shrink-0"
          title={$_('page.sidebar.admin')}
          aria-label={$_('page.sidebar.admin')}
        >
          <Shield size={smallIconSize} />
        </button>
      {/if}
      {#if errorReporter.getServiceAvailable()}
        <button
          onclick={onShowFeedback}
          class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''} shrink-0"
          title={$_('feedback.sidebar_button')}
          aria-label={$_('feedback.sidebar_button')}
        >
          <MessageSquareWarning size={smallIconSize} />
        </button>
      {/if}
    </div>
  </div>
{/snippet}

{@render controls()}
