<script lang="ts">
  import {
    ArrowUpDown,
    Check,
    ChevronLeft,
    FilePlus,
    FolderPlus,
  } from 'lucide-svelte';
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
    onCloseSidebar?: () => void;
    onBindSortDropdownRef: (el: HTMLDivElement | null) => void;
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
    onCloseSidebar,
    onBindSortDropdownRef,
    onLogoClick,
  }: Props = $props();

  let localSortDropdownRef = $state<HTMLDivElement | null>(null);

  $effect(() => {
    onBindSortDropdownRef(localSortDropdownRef);
  });
</script>

<div
  class="flex items-center justify-between px-4 py-2.5 border-b border-sidebar-border shrink-0"
>
  <a
    href="/"
    onclick={onLogoClick}
    class="hover:opacity-80 transition-opacity"
  >
    <Logo size="md" />
  </a>
  <div class="flex items-center gap-0.5">
    <button
      onclick={onCreateNote}
      class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors {isMobile ? 'toolbar-btn' : ''}"
      title={$_('page.sidebar.new_note')}
      aria-label={$_('page.sidebar.new_note')}
    >
      <FilePlus size={mainIconSize} />
    </button>
    <button
      onclick={onCreateFolder}
      class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors {isMobile ? 'toolbar-btn' : ''}"
      title={$_('page.sidebar.new_folder')}
      aria-label={$_('page.sidebar.new_folder')}
    >
      <FolderPlus size={mainIconSize} />
    </button>
    <div class="relative" bind:this={localSortDropdownRef}>
      <button
        onclick={onToggleSortDropdown}
        class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors {isMobile ? 'toolbar-btn' : ''}"
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
        class="p-1.5 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground {isMobile ? 'toolbar-btn' : ''}"
        title={$_(isMobile ? 'page.sidebar.close_drawer' : 'page.sidebar.collapse_sidebar')}
        aria-label={$_(isMobile ? 'page.sidebar.close_drawer' : 'page.sidebar.collapse_sidebar')}
      >
        <ChevronLeft size={mainIconSize} />
      </button>
    {/if}
  </div>
</div>
