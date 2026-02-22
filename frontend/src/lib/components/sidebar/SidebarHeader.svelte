<script lang="ts">
  import { ArrowUpDown, Check, FilePlus, FolderPlus, Network, Search, X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { SortMode } from '$lib/stores/tree.svelte';

  import Logo from '../Logo.svelte';

  interface Props {
    isMobile: boolean;
    mainIconSize: number;
    showSortDropdown: boolean;
    currentSortMode: SortMode;
    sortOptions: { mode: SortMode; labelKey: string }[];
    sortDropdownRef: HTMLDivElement | null;
    onCreateNote: () => void;
    onCreateFolder: () => void;
    onToggleSortDropdown: () => void;
    onSortSelect: (mode: SortMode) => void;
    onBindSortDropdownRef: (el: HTMLDivElement | null) => void;
    /** Desktop: toggle quick-switcher search */
    onToggleSearch?: () => void;
    /** Desktop: navigate to graph view */
    onOpenGraph?: () => void;
    /** Desktop: whether graph feature is enabled */
    graphEnabled?: boolean;
    /** Desktop: collapse the sidebar panel */
    onCollapseSidebar?: () => void;
    /** Mobile only: close the drawer */
    onCloseSidebar?: () => void;
    /** Mobile only: logo click handler */
    onLogoClick?: () => void;
  }

  const {
    isMobile,
    mainIconSize,
    showSortDropdown,
    currentSortMode,
    sortOptions,
    sortDropdownRef: _sortDropdownRef,
    onCreateNote,
    onCreateFolder,
    onToggleSortDropdown,
    onSortSelect,
    onBindSortDropdownRef,
    onToggleSearch,
    onOpenGraph,
    graphEnabled = false,
    onCollapseSidebar,
    onCloseSidebar,
    onLogoClick,
  }: Props = $props();

  let localSortDropdownRef = $state<HTMLDivElement | null>(null);

  $effect(() => {
    onBindSortDropdownRef(localSortDropdownRef);
  });
</script>

{#if isMobile}
  <!-- Mobile header: keep logo + existing layout -->
  <div class="flex items-center justify-between border-b border-sidebar-border shrink-0 px-3 py-2">
    <a href="/" onclick={onLogoClick} class="hover:opacity-80 transition-opacity">
      <Logo size="md" />
    </a>
    <div class="flex items-center gap-0.5">
      <button
        onclick={onCreateNote}
        class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1.5 toolbar-btn"
        title={$_('page.sidebar.new_note')}
        aria-label={$_('page.sidebar.new_note')}
      >
        <FilePlus size={mainIconSize} />
      </button>
      <button
        onclick={onCreateFolder}
        class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1.5 toolbar-btn"
        title={$_('page.sidebar.new_folder')}
        aria-label={$_('page.sidebar.new_folder')}
      >
        <FolderPlus size={mainIconSize} />
      </button>
      <div class="relative" bind:this={localSortDropdownRef}>
        <button
          onclick={onToggleSortDropdown}
          class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1.5 toolbar-btn"
          class:bg-sidebar-accent={showSortDropdown}
          title={$_('page.sidebar.sort_notes')}
          aria-label={$_('page.sidebar.sort_notes')}
        >
          <ArrowUpDown size={mainIconSize} />
        </button>
        {#if showSortDropdown}
          <div
            class="absolute right-0 top-full mt-1 w-44 bg-popover border border-border rounded-lg shadow-lg z-50 py-1"
          >
            {#each sortOptions as opt (opt.mode)}
              <button
                onclick={() => onSortSelect(opt.mode)}
                class="w-full flex items-center gap-2 px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent transition-colors"
              >
                <span class="w-4 flex-shrink-0">
                  {#if currentSortMode === opt.mode}
                    <Check size={14} />
                  {/if}
                </span>
                <span>{$_(opt.labelKey)}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
      {#if onCloseSidebar}
        <button
          onclick={onCloseSidebar}
          class="rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground p-1.5 toolbar-btn"
          title={$_('page.sidebar.close_drawer')}
          aria-label={$_('page.sidebar.close_drawer')}
        >
          <X size={mainIconSize} />
        </button>
      {/if}
    </div>
  </div>
{:else}
  <!-- Desktop header: icon toolbar only, no logo -->
  <div
    class="flex items-center justify-between border-b border-sidebar-border shrink-0 px-2 py-1.5"
  >
    <div class="flex items-center gap-0.5">
      <button
        onclick={onCreateNote}
        class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1"
        title={$_('page.sidebar.new_note')}
        aria-label={$_('page.sidebar.new_note')}
      >
        <FilePlus size={mainIconSize} />
      </button>
      <button
        onclick={onCreateFolder}
        class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1"
        title={$_('page.sidebar.new_folder')}
        aria-label={$_('page.sidebar.new_folder')}
      >
        <FolderPlus size={mainIconSize} />
      </button>
      <div class="relative" bind:this={localSortDropdownRef}>
        <button
          onclick={onToggleSortDropdown}
          class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1"
          class:bg-sidebar-accent={showSortDropdown}
          title={$_('page.sidebar.sort_notes')}
          aria-label={$_('page.sidebar.sort_notes')}
        >
          <ArrowUpDown size={mainIconSize} />
        </button>
        {#if showSortDropdown}
          <div
            class="absolute right-0 top-full mt-0.5 w-44 bg-popover border border-border rounded-lg shadow-lg z-50 py-1"
          >
            {#each sortOptions as opt (opt.mode)}
              <button
                onclick={() => onSortSelect(opt.mode)}
                class="w-full flex items-center gap-2 px-3 py-1.5 text-sm text-popover-foreground hover:bg-accent transition-colors"
              >
                <span class="w-4 flex-shrink-0">
                  {#if currentSortMode === opt.mode}
                    <Check size={14} />
                  {/if}
                </span>
                <span>{$_(opt.labelKey)}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
      {#if graphEnabled && onOpenGraph}
        <button
          onclick={onOpenGraph}
          class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1"
          title="{$_('page.sidebar.graph')} (Ctrl+G)"
          aria-label={$_('page.sidebar.graph')}
        >
          <Network size={mainIconSize} />
        </button>
      {/if}
      {#if onToggleSearch}
        <button
          onclick={onToggleSearch}
          class="rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1"
          title="{$_('page.sidebar.search')} (Ctrl+P)"
          aria-label={$_('page.sidebar.search')}
        >
          <Search size={mainIconSize} />
        </button>
      {/if}
    </div>
    {#if onCollapseSidebar}
      <button
        onclick={onCollapseSidebar}
        class="rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground p-1"
        title={$_('page.sidebar.collapse_sidebar')}
        aria-label={$_('page.sidebar.collapse_sidebar')}
      >
        <X size={mainIconSize} />
      </button>
    {/if}
  </div>
{/if}
