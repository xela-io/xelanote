<script lang="ts">
  import {
    CalendarClock,
    FileText,
    Moon,
    MoreHorizontal,
    PanelLeft,
    Search,
    Settings,
    Sun,
    Trash2,
    Users,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { bottomsheet } from '$lib/actions/bottomsheet';
  import * as features from '$lib/stores/features.svelte';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  let showMoreSheet = $state(false);

  const isNotesActive = $derived(
    page.url.pathname === '/' || page.url.pathname.startsWith('/note/')
  );

  const journalEnabled = $derived(
    features.getJournalFeatureEnabled() && features.getJournalFeatureLoaded()
  );
  const recipeEnabled = $derived(
    features.getRecipeFeatureEnabled() && features.getRecipeFeatureLoaded()
  );
  const isDark = $derived(ui.getCurrentTheme().variant === 'dark');

  function handleNotesClick() {
    goto('/');
  }

  function handleSearchClick() {
    ui.toggleQuickSwitcher();
  }

  function handleMoreClick() {
    showMoreSheet = true;
  }

  function closeSheet() {
    showMoreSheet = false;
  }

  function handleSheetKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      closeSheet();
    }
  }

  function handleNotesTree() {
    ui.setSidebarOpen(true);
    closeSheet();
  }

  function handleThemeToggle() {
    ui.toggleTheme();
    closeSheet();
  }
</script>

<!-- Bottom Navigation Bar -->
<nav
  class="mobile-bottom-nav fixed bottom-0 left-0 right-0 z-30 bg-background border-t border-border pb-safe"
  aria-label={$_('nav.bottom_navigation')}
>
  <div class="flex items-center h-14 px-2">
    <!-- Nav Tabs -->
    <button
      onclick={handleNotesClick}
      class="flex-1 flex flex-col items-center justify-center gap-0.5 min-h-12"
      style="-webkit-tap-highlight-color: transparent"
    >
      <span
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full transition-colors {isNotesActive
          ? 'bg-primary/12 text-primary'
          : 'text-muted-foreground'}"
      >
        <FileText size={20} />
        <span class="text-[10px] font-medium">{$_('nav.notes')}</span>
      </span>
    </button>

    <button
      onclick={handleSearchClick}
      class="flex-1 flex flex-col items-center justify-center gap-0.5 min-h-12 text-muted-foreground"
      style="-webkit-tap-highlight-color: transparent"
    >
      <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full">
        <Search size={20} />
        <span class="text-[10px] font-medium">{$_('page.sidebar.search')}</span>
      </span>
    </button>

    <button
      onclick={handleMoreClick}
      class="flex-1 flex flex-col items-center justify-center gap-0.5 min-h-12 text-muted-foreground"
      style="-webkit-tap-highlight-color: transparent"
    >
      <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full">
        <MoreHorizontal size={20} />
        <span class="text-[10px] font-medium">{$_('nav.more')}</span>
      </span>
    </button>
  </div>
</nav>

<!-- "More" Bottom Sheet -->
{#if showMoreSheet}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-40 bg-black/50"
    onclick={closeSheet}
    onkeydown={handleSheetKeydown}
    tabindex="-1"
    role="presentation"
  ></div>

  <!-- Sheet -->
  <div
    class="mobile-more-sheet fixed z-50 bottom-0 left-0 right-0 bg-background rounded-t-2xl animate-bottom-sheet p-4"
    role="menu"
    aria-label={$_('nav.more_options')}
    tabindex="-1"
    onkeydown={handleSheetKeydown}
    use:bottomsheet={{ onClose: closeSheet }}
  >
    <!-- Drag Handle -->
    <div class="w-12 h-1 bg-muted rounded-full mx-auto mb-4"></div>

    <div class="space-y-1">
      <div class="mobile-more-sheet-section">{$_('nav.bottom_navigation')}</div>

      <!-- Notes Tree -->
      <button
        type="button"
        onclick={handleNotesTree}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <PanelLeft size={18} />
        {$_('nav.notes_tree')}
      </button>

      <hr class="mobile-more-sheet-divider" />

      <div class="mobile-more-sheet-section">{$_('nav.more_options')}</div>

      <!-- Journal (conditional) -->
      {#if journalEnabled}
        <button
          type="button"
          onclick={() => {
            goto('/journal');
            closeSheet();
          }}
          class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
          role="menuitem"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
            <line x1="16" y1="2" x2="16" y2="6"></line>
            <line x1="8" y1="2" x2="8" y2="6"></line>
            <line x1="3" y1="10" x2="21" y2="10"></line>
          </svg>
          {$_('page.journal.title')}
        </button>
      {/if}

      <!-- Shared Notes -->
      <button
        type="button"
        onclick={() => {
          goto('/shared');
          closeSheet();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <Users size={18} />
        {$_('sharing.shared_with_me')}
        {#if sharing.getTotalSharedCount() > 0}
          <span
            class="ml-auto bg-primary text-primary-foreground text-[10px] font-medium px-1.5 py-0.5 rounded-full"
          >
            {sharing.getTotalSharedCount()}
          </span>
        {/if}
      </button>

      <!-- Due Dates -->
      <button
        type="button"
        onclick={() => {
          goto('/due-dates');
          closeSheet();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <CalendarClock size={18} />
        {$_('page.due_dates.title')}
      </button>

      <!-- Trash -->
      <button
        type="button"
        onclick={() => {
          goto('/trash');
          closeSheet();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <Trash2 size={18} />
        {$_('page.sidebar.trash')}
        {#if trash.getTrashCount() > 0}
          <span
            class="ml-auto bg-destructive text-destructive-foreground text-[10px] font-medium px-1.5 py-0.5 rounded-full"
          >
            {trash.getTrashCount()}
          </span>
        {/if}
      </button>

      <!-- Recipes (conditional) -->
      {#if recipeEnabled}
        <button
          type="button"
          onclick={() => {
            goto('/recipes');
            closeSheet();
          }}
          class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
          role="menuitem"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M15 11h.01"></path>
            <path d="M11 15h.01"></path>
            <path d="M16 16h.01"></path>
            <path d="m2 16 20 6-6-20A20 20 0 0 0 2 16"></path>
            <path d="M5.71 17.11a17.04 17.04 0 0 1 11.4-11.4"></path>
          </svg>
          {$_('page.recipes.title')}
        </button>
      {/if}

      <hr class="mobile-more-sheet-divider" />

      <div class="mobile-more-sheet-section">{$_('page.settings.tabs.appearance')}</div>

      <!-- Theme Toggle -->
      <button
        type="button"
        onclick={handleThemeToggle}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        {#if isDark}
          <Sun size={18} />
        {:else}
          <Moon size={18} />
        {/if}
        {$_('nav.toggle_theme')}
      </button>

      <hr class="mobile-more-sheet-divider" />

      <div class="mobile-more-sheet-section">{$_('page.sidebar.settings')}</div>

      <!-- Settings -->
      <button
        type="button"
        onclick={() => {
          goto('/settings');
          closeSheet();
        }}
        class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
        role="menuitem"
      >
        <Settings size={18} />
        {$_('page.sidebar.settings')}
      </button>
    </div>
  </div>
{/if}

<style>
  .mobile-more-sheet {
    border-top: 1px solid color-mix(in oklch, var(--color-border), transparent 38%);
    box-shadow:
      0 -12px 36px color-mix(in oklch, black, transparent 82%),
      inset 0 1px 0 color-mix(in oklch, white, transparent 94%);
  }

  .mobile-more-sheet-section {
    padding: 0.25rem 0.75rem 0.2rem;
    font-size: 0.68rem;
    line-height: 1;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: color-mix(in oklch, var(--color-muted-foreground), var(--color-primary) 20%);
  }

  .mobile-more-sheet-divider {
    margin-block: 0.55rem;
    border-color: color-mix(in oklch, var(--color-border), transparent 46%);
  }
</style>
