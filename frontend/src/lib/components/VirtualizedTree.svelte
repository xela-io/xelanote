<script lang="ts">
  import type { SvelteVirtualizer } from '@tanstack/svelte-virtual';
  import { createVirtualizer } from '@tanstack/svelte-virtual';
  import { Info } from 'lucide-svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import type { FlatTreeItem } from '$lib/stores/tree.svelte';
  import * as tree from '$lib/stores/tree.svelte';

  import UnifiedTree from './UnifiedTree.svelte';

  // Scroll container reference
  let scrollElement = $state<HTMLDivElement | null>(null);

  // Virtualizer state
  let virtualizerValue = $state<SvelteVirtualizer<HTMLDivElement, Element> | null>(null);

  // Info banner dismissal
  const BANNER_DISMISSED_KEY = 'xelanote_virtual_tree_banner_dismissed';
  let dismissedBanner = $state(false);

  // Keyboard navigation state (Phase 4.3)
  let focusedIndex = $state<number>(-1);

  // Initialize focusedIndex to selected item (if any)
  $effect(() => {
    if (focusedIndex === -1 && itemCount > 0) {
      const selectedNoteId = tree.getSelectedNoteId();
      const selectedFolder = tree.getSelectedFolderPath();

      if (selectedNoteId || selectedFolder) {
        const index = flatItems.findIndex((item) => {
          if (selectedNoteId && item.node.type === 'note') {
            return item.node.id === selectedNoteId;
          }
          if (selectedFolder && item.node.type === 'folder') {
            return item.node.path === selectedFolder;
          }
          return false;
        });

        if (index >= 0) {
          focusedIndex = index;
        }
      }
    }
  });

  $effect(() => {
    try {
      const stored = localStorage.getItem(BANNER_DISMISSED_KEY);
      dismissedBanner = stored === 'true';
    } catch (_e) {
      dismissedBanner = false;
    }
  });

  function dismissBanner() {
    dismissedBanner = true;
    try {
      localStorage.setItem(BANNER_DISMISSED_KEY, 'true');
    } catch (e) {
      console.error('Failed to save banner dismissal:', e);
    }
  }

  // Get flattened tree items
  const flatItems = $derived(tree.getFlattenedTree());
  const itemCount = $derived(flatItems.length);

  type DisplayRow =
    | {
        kind: 'section';
        id: string;
        label: string;
      }
    | {
        kind: 'item';
        id: string;
        flatIndex: number;
        item: FlatTreeItem;
      };

  function isJournalTopLevelFlatItem(item: FlatTreeItem) {
    return item.level === 0 && item.node.type === 'folder' && item.node.path === '/Journal';
  }

  const displayRows = $derived.by<DisplayRow[]>(() => {
    const rows: DisplayRow[] = [];
    let insertedNotes = false;
    let insertedJournal = false;

    flatItems.forEach((item, flatIndex) => {
      if (item.level === 0) {
        if (isJournalTopLevelFlatItem(item)) {
          if (!insertedJournal) {
            rows.push({
              kind: 'section',
              id: 'section-journal',
              label: $_('page.sidebar.section_journal'),
            });
            insertedJournal = true;
          }
        } else if (!insertedNotes) {
          rows.push({
            kind: 'section',
            id: 'section-notes',
            label: $_('page.sidebar.section_notes'),
          });
          insertedNotes = true;
        }
      }

      rows.push({
        kind: 'item',
        id: `${item.node.type}-${String(item.node.id)}-${flatIndex}`,
        flatIndex,
        item,
      });
    });

    return rows;
  });

  const displayRowCount = $derived(displayRows.length);
  const flatIndexToDisplayIndex = $derived.by(() => {
    const indexMap = new SvelteMap<number, number>();
    displayRows.forEach((row, displayIndex) => {
      if (row.kind === 'item') {
        indexMap.set(row.flatIndex, displayIndex);
      }
    });
    return indexMap;
  });

  // Create virtualizer when items are available
  $effect(() => {
    if (displayRowCount > 0 && scrollElement) {
      const store = createVirtualizer({
        count: displayRowCount,
        getScrollElement: () => scrollElement!,
        estimateSize: (index) => (displayRows[index]?.kind === 'section' ? 24 : 36),
        overscan: 5,
      });

      const unsubscribe = store.subscribe((value) => {
        virtualizerValue = value;
      });

      return unsubscribe;
    } else {
      virtualizerValue = null;
    }
  });

  // Derived values for rendering
  const virtualItems = $derived(virtualizerValue?.getVirtualItems() ?? []);
  const totalSize = $derived(virtualizerValue?.getTotalSize() ?? 0);

  // Auto-scroll to selected item (Phase 4.1)
  $effect(() => {
    const selectedNoteId = tree.getSelectedNoteId();
    const selectedFolder = tree.getSelectedFolderPath();

    if ((selectedNoteId || selectedFolder) && virtualizerValue) {
      // Find index of selected item in flat tree
      const index = flatItems.findIndex((item) => {
        if (selectedNoteId && item.node.type === 'note') {
          return item.node.id === selectedNoteId;
        }
        if (selectedFolder && item.node.type === 'folder') {
          return item.node.path === selectedFolder;
        }
        return false;
      });

      if (index >= 0) {
        const displayIndex = flatIndexToDisplayIndex.get(index) ?? index;
        // Check if item is already visible (avoid unnecessary scrolling)
        const visibleRange = virtualizerValue.getVirtualItems() as Array<{ index: number }>;
        const isVisible = visibleRange.some((v) => v.index === displayIndex);

        if (!isVisible) {
          virtualizerValue.scrollToIndex(displayIndex, {
            align: 'center',
            behavior: 'smooth',
          });
        }
      }
    }
  });

  // Handle expand events from UnifiedTree (for scroll restoration - Phase 4.2)
  function handleExpand(event: CustomEvent<{ nodePath: string }>) {
    const { nodePath } = event.detail;

    // Find current item position
    const itemIndex = flatItems.findIndex(
      (item) => item.node.type === 'folder' && item.node.path === nodePath
    );

    if (itemIndex >= 0 && virtualizerValue) {
      const displayIndex = flatIndexToDisplayIndex.get(itemIndex) ?? itemIndex;
      // Scroll to keep folder in view after expansion
      virtualizerValue.scrollToIndex(displayIndex, {
        align: 'start',
        behavior: 'auto',
      });
    }
  }

  // Keyboard Navigation (Phase 4.3)
  function handleKeyDown(e: KeyboardEvent) {
    if (itemCount === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        focusedIndex = Math.min(focusedIndex + 1, itemCount - 1);
        scrollToFocusedItem();
        break;

      case 'ArrowUp':
        e.preventDefault();
        focusedIndex = Math.max(focusedIndex - 1, 0);
        scrollToFocusedItem();
        break;

      case 'Home':
        e.preventDefault();
        focusedIndex = 0;
        scrollToFocusedItem();
        break;

      case 'End':
        e.preventDefault();
        focusedIndex = itemCount - 1;
        scrollToFocusedItem();
        break;

      case 'Enter':
        e.preventDefault();
        selectFocusedItem();
        break;

      case ' ':
        e.preventDefault();
        toggleFocusedItem();
        break;
    }
  }

  function scrollToFocusedItem() {
    if (focusedIndex >= 0 && focusedIndex < itemCount && virtualizerValue) {
      const displayIndex = flatIndexToDisplayIndex.get(focusedIndex) ?? focusedIndex;
      virtualizerValue.scrollToIndex(displayIndex, {
        align: 'auto',
        behavior: 'smooth',
      });
    }
  }

  function selectFocusedItem() {
    if (focusedIndex < 0 || focusedIndex >= itemCount) return;

    const item = flatItems[focusedIndex];
    if (item.node.type === 'folder') {
      tree.selectFolder(item.node.path);
    } else {
      tree.selectNote(item.node.id);
    }
  }

  function toggleFocusedItem() {
    if (focusedIndex < 0 || focusedIndex >= itemCount) return;

    const item = flatItems[focusedIndex];
    if (item.node.type === 'folder') {
      tree.toggleExpanded(item.node.path);
    }
  }
</script>

<!-- Info Banner (sticky, dismissable) -->
{#if itemCount > 500 && !dismissedBanner}
  <div
    class="sticky top-0 z-10 bg-warning/10 border-b border-warning p-2 text-sm flex items-center gap-2"
  >
    <Info size={14} class="text-warning shrink-0" />
    <span class="flex-1"
      >Virtuelles Scrollen aktiv. Drag-and-Drop auf sichtbare Items beschränkt.</span
    >
    <button
      onclick={dismissBanner}
      class="text-muted-foreground hover:text-foreground shrink-0"
      aria-label="Banner schließen"
    >
      ×
    </button>
  </div>
{/if}

<!-- Virtual scrolling container -->
<div
  bind:this={scrollElement}
  class="overflow-auto flex-1"
  role="tree"
  aria-label="Notizen und Ordner"
  tabindex="0"
  onkeydown={handleKeyDown}
>
  {#if virtualizerValue && itemCount > 0}
    <!-- Virtual rendering -->
    <div style="height: {totalSize}px; width: 100%; position: relative;">
      {#each virtualItems as virtualRow (virtualRow.key)}
        {@const row = displayRows[virtualRow.index]}
        {#if row?.kind === 'section'}
          <div
            class="virtual-section-label"
            class:section-divider={row.id === 'section-journal'}
            style="position: absolute; top: 0; left: 0; width: 100%; transform: translateY({virtualRow.start}px);"
            role="presentation"
          >
            {row.label}
          </div>
        {:else if row?.kind === 'item'}
          {@const item = row.item}
          {@const isFocused = focusedIndex === row.flatIndex}
          <div
            style="position: absolute; top: 0; left: 0; width: 100%; transform: translateY({virtualRow.start}px);"
            role="treeitem"
            aria-level={item.level + 1}
            aria-expanded={item.node.type === 'folder' ? item.node.isExpanded : undefined}
            aria-selected={isFocused}
            tabindex={isFocused ? 0 : -1}
          >
            <div style="padding-left: {item.level * 12}px">
              <UnifiedTree
                node={item.node}
                depth={item.level}
                isVirtualized={true}
                on:expand={handleExpand}
              />
            </div>
          </div>
        {/if}
      {/each}
    </div>
  {:else if itemCount > 0}
    <!-- SSR/Hydration Fallback: Show first 20 items -->
    <div class="space-y-0">
      {#each displayRows.slice(0, 24) as row, index (index)}
        {#if row.kind === 'section'}
          <div
            class="virtual-section-label"
            class:section-divider={row.id === 'section-journal'}
            role="presentation"
          >
            {row.label}
          </div>
        {:else}
          <div style="padding-left: {row.item.level * 12}px">
            <UnifiedTree node={row.item.node} depth={row.item.level} isVirtualized={true} />
          </div>
        {/if}
      {/each}
    </div>
  {:else}
    <!-- Empty state -->
    <div class="p-4 text-center text-muted-foreground">
      <p>Keine Notizen oder Ordner vorhanden</p>
    </div>
  {/if}
</div>

<style>
  /* Focus ring for keyboard navigation */
  [role='tree']:focus {
    outline: none;
  }

  [role='treeitem'][aria-selected='true'] {
    outline: 2px solid var(--color-primary, #3b82f6);
    outline-offset: -2px;
    border-radius: var(--radius-sm);
  }

  /* Ensure focus is visible for accessibility */
  [role='tree']:focus-visible {
    box-shadow: inset 0 0 0 2px var(--color-ring, #3b82f6);
  }

  .virtual-section-label {
    font-size: 0.68rem;
    line-height: 1.2;
    font-weight: 600;
    letter-spacing: 0.045em;
    text-transform: uppercase;
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 48%
    );
    padding: 0.5rem 0.5rem 0.22rem;
    background: color-mix(in oklch, var(--color-sidebar-background), transparent 6%);
  }

  .virtual-section-label.section-divider {
    border-top: 1px solid color-mix(in oklch, var(--color-sidebar-border), transparent 22%);
    margin-top: 0.45rem;
    padding-top: 0.62rem;
  }
</style>
