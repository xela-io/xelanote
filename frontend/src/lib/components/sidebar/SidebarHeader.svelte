<script lang="ts">
  import { ArrowUpDown, Check, FilePlus, FolderPlus, Network, Search, X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { SortMode } from '$lib/stores/tree.svelte';

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
  }: Props = $props();

  let localSortDropdownRef = $state<HTMLDivElement | null>(null);

  const iconButtonClass =
    'rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1.5 toolbar-btn';
  const iconButtonDesktopClass =
    'rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors p-1';

  $effect(() => {
    onBindSortDropdownRef(localSortDropdownRef);
  });
</script>

{#if isMobile}
  <!-- Mobile header: toolbar only (no logo — Bottom Nav handles branding) -->
  <div
    class="flex items-center justify-between border-b border-sidebar-border shrink-0 px-2 py-1.5"
  >
    <div class="flex items-center gap-1 w-full min-w-0">
      <button
        onclick={onCreateNote}
        class="inline-flex items-center gap-1.5 rounded-lg border border-sidebar-border bg-sidebar-accent/50 hover:bg-sidebar-accent text-sidebar-foreground transition-colors px-2.5 py-1.5 toolbar-btn shrink-0"
        title={$_('page.sidebar.new_note')}
        aria-label={$_('page.sidebar.new_note')}
      >
        <FilePlus size={mainIconSize} />
        <span class="text-xs font-medium">{$_('page.sidebar.new_note')}</span>
      </button>
      <div class="flex items-center gap-0.5 ml-auto pl-1 border-l border-sidebar-border/70">
        <button
          onclick={onCreateFolder}
          class={iconButtonClass}
          title={$_('page.sidebar.new_folder')}
          aria-label={$_('page.sidebar.new_folder')}
        >
          <FolderPlus size={mainIconSize} />
        </button>
        <div class="relative" bind:this={localSortDropdownRef}>
          <button
            onclick={onToggleSortDropdown}
            class={iconButtonClass}
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
      </div>
    </div>
  </div>
{:else}
  <!-- Desktop header: icon toolbar only, no logo -->
  <div
    class="flex items-center justify-between border-b border-sidebar-border shrink-0 px-2 py-1.5"
  >
    <div class="flex items-center gap-1 min-w-0 flex-1">
      <button
        onclick={onCreateNote}
        class="inline-flex items-center gap-1.5 rounded-lg border border-sidebar-border bg-sidebar-accent/40 hover:bg-sidebar-accent text-sidebar-foreground transition-colors px-2 py-1 shrink-0"
        title={$_('page.sidebar.new_note')}
        aria-label={$_('page.sidebar.new_note')}
      >
        <FilePlus size={mainIconSize} />
        <span class="text-xs font-medium">{$_('page.sidebar.new_note')}</span>
      </button>
      <div class="flex items-center gap-0.5 ml-0.5">
        <button
          onclick={onCreateFolder}
          class={iconButtonDesktopClass}
          title={$_('page.sidebar.new_folder')}
          aria-label={$_('page.sidebar.new_folder')}
        >
          <FolderPlus size={mainIconSize} />
        </button>
        <div class="relative" bind:this={localSortDropdownRef}>
          <button
            onclick={onToggleSortDropdown}
            class={iconButtonDesktopClass}
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
      </div>
      <div class="mx-1 h-4 w-px bg-sidebar-border/70"></div>
      <div class="flex items-center gap-1">
        {#if onToggleSearch}
          <button
            onclick={onToggleSearch}
            class={iconButtonDesktopClass}
            title="{$_('page.sidebar.search')} (Ctrl+P)"
            aria-label={$_('page.sidebar.search')}
          >
            <Search size={mainIconSize} />
          </button>
        {/if}
        {#if graphEnabled && onOpenGraph}
          <button
            onclick={onOpenGraph}
            class={`${iconButtonDesktopClass} opacity-85 hover:opacity-100`}
            title="{$_('page.sidebar.graph')} (Ctrl+G)"
            aria-label={$_('page.sidebar.graph')}
          >
            <Network size={mainIconSize} />
          </button>
        {/if}
      </div>
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
