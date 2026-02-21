<script lang="ts">
  import {
    CalendarClock,
    ChevronLeft,
    ChevronRight,
    FilePlus,
    FolderPlus,
    MessageSquareWarning,
    Network,
    Search,
    Settings,
    Shield,
    Trash2,
    Users,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { swipe } from '$lib/actions/swipe';
  import type { DropPosition, TouchDragData } from '$lib/actions/touchdrag';
  import { touchdrag } from '$lib/actions/touchdrag';
  import {
    handleCreateFolderConfirm as handleCreateFolderConfirmAction,
    handleCreateNoteConfirm as handleCreateNoteConfirmAction,
  } from '$lib/components/sidebar/sidebar-actions';
  import {
    handleDropZoneDragLeave as handleDropZoneDragLeaveAction,
    handleDropZoneDragOver as handleDropZoneDragOverAction,
    handleDropZoneDrop as handleDropZoneDropAction,
    handleTouchDrop as handleTouchDropAction,
  } from '$lib/components/sidebar/sidebar-dnd';
  import { handleSidebarEscape } from '$lib/components/sidebar/sidebar-escape';
  import { initSidebarOnMount } from '$lib/components/sidebar/sidebar-init';
  import {
    handleSidebarResizeDblClick,
    handleSidebarResizeEnd,
    handleSidebarResizeMove,
    handleSidebarResizeStart,
  } from '$lib/components/sidebar/sidebar-resize';
  import * as auth from '$lib/stores/auth.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import type { SortMode } from '$lib/stores/tree.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  import CreateFolderDialog from './CreateFolderDialog.svelte';
  import CreateNoteDialog from './CreateNoteDialog.svelte';
  import FeedbackDialog from './FeedbackDialog.svelte';
  import JournalButton from './JournalButton.svelte';
  import RecipeButton from './RecipeButton.svelte';
  import SidebarFooter from './sidebar/SidebarFooter.svelte';
  import ThemeSelector from './ThemeSelector.svelte';
  import SidebarHeader from './sidebar/SidebarHeader.svelte';
  import UnifiedTree from './UnifiedTree.svelte';
  import VirtualizedTree from './VirtualizedTree.svelte';

  let isDropZoneActive = $state(false);
  let touchDragActive = $state(false);
  let showCreateNoteDialog = $state(false);
  let showCreateFolderDialog = $state(false);
  let showFeedbackDialog = $state(false);
  let showSortDropdown = $state(false);
  let sortDropdownRef = $state<HTMLDivElement | null>(null);

  const sortOptions: { mode: SortMode; labelKey: string }[] = [
    { mode: 'manual', labelKey: 'page.sidebar.sort_manual' },
    { mode: 'updated', labelKey: 'page.sidebar.sort_updated' },
    { mode: 'title', labelKey: 'page.sidebar.sort_title' },
    { mode: 'created', labelKey: 'page.sidebar.sort_created' },
  ];

  function handleSortSelect(mode: SortMode) {
    tree.setSortMode(mode);
    showSortDropdown = false;
  }

  function handleClickOutsideSort(e: MouseEvent) {
    if (sortDropdownRef && !sortDropdownRef.contains(e.target as Node)) {
      showSortDropdown = false;
    }
  }

  // Resize state
  let isResizing = $state(false);
  let startX = 0;
  let startWidth = 0;

  function handleResizeStart(e: PointerEvent) {
    handleSidebarResizeStart(e, {
      getIsMobile: ui.getIsMobile,
      getSidebarWidth: ui.getSidebarWidth,
      setSidebarWidth: ui.setSidebarWidth,
      setActive: (value) => {
        isResizing = value;
      },
      setStartX: (value) => {
        startX = value;
      },
      setStartWidth: (value) => {
        startWidth = value;
      },
    });
  }

  function handleResizeMove(e: PointerEvent) {
    if (!isResizing) return;
    handleSidebarResizeMove(
      e,
      {
        getIsMobile: ui.getIsMobile,
        getSidebarWidth: ui.getSidebarWidth,
        setSidebarWidth: ui.setSidebarWidth,
        setActive: (value) => {
          isResizing = value;
        },
        setStartX: (value) => {
          startX = value;
        },
        setStartWidth: (value) => {
          startWidth = value;
        },
      },
      startX,
      startWidth
    );
  }

  function handleResizeEnd() {
    handleSidebarResizeEnd({
      getIsMobile: ui.getIsMobile,
      getSidebarWidth: ui.getSidebarWidth,
      setSidebarWidth: ui.setSidebarWidth,
      setActive: (value) => {
        isResizing = value;
      },
      setStartX: (value) => {
        startX = value;
      },
      setStartWidth: (value) => {
        startWidth = value;
      },
    });
  }

  function handleResizeDblClick() {
    handleSidebarResizeDblClick({
      getIsMobile: ui.getIsMobile,
      getSidebarWidth: ui.getSidebarWidth,
      setSidebarWidth: ui.setSidebarWidth,
      setActive: (value) => {
        isResizing = value;
      },
      setStartX: (value) => {
        startX = value;
      },
      setStartWidth: (value) => {
        startWidth = value;
      },
    });
  }

  // Responsive icon sizes: 20px on mobile, 18px on desktop
  const mainIconSize = $derived(ui.getIsMobile() ? 20 : 18);
  const smallIconSize = $derived(ui.getIsMobile() ? 18 : 16);

  // Feature flags (reactive)
  const journalEnabled = $derived(features.getJournalFeatureEnabled());
  const recipeEnabled = $derived(features.getRecipeFeatureEnabled());

  // Restore expanded state on mount
  onMount(() =>
    initSidebarOnMount({
      loadExpandedState: tree.loadExpandedStateFromStorage,
      loadTrashCount: trash.loadTrashCount,
      loadShared: sharing.loadAllShared,
      loadJournalFeature: features.loadJournalFeature,
      loadRecipeFeature: features.loadRecipeFeature,
      loadCanvasFeature: features.loadCanvasFeature,
      startInterval: (handler, ms) => window.setInterval(handler, ms),
      clearInterval: (id) => window.clearInterval(id),
    })
  );

  // Load tree when authenticated (reactive)
  $effect(() => {
    if (auth.isAuthenticated()) {
      tree.loadTree();
    }
  });

  // Drop zone handlers for moving folders to top-level
  function handleDropZoneDragOver(e: DragEvent) {
    handleDropZoneDragOverAction(e, (active) => {
      isDropZoneActive = active;
    });
  }

  function handleDropZoneDragLeave() {
    handleDropZoneDragLeaveAction((active) => {
      isDropZoneActive = active;
    });
  }

  async function handleDropZoneDrop(e: DragEvent) {
    await handleDropZoneDropAction(
      e,
      {
        moveFolder: tree.moveFolder,
        moveNote: notes.moveNote,
        loadTree: tree.loadTree,
        reorderFolders: tree.reorderFolders,
        reorderNotes: tree.reorderNotes,
        findParentOfNodeById: tree.findParentOfNodeById,
        getSortMode: tree.getSortMode,
        getFolderChildren: (parent: tree.FolderTreeNode) =>
          parent.children.filter((c): c is tree.FolderTreeNode => c.type === 'folder'),
        getNoteChildren: (parent: tree.FolderTreeNode) =>
          parent.children.filter((c): c is tree.NoteTreeNode => c.type === 'note'),
        alert: dialog.alert,
        strings: {
          errorTitle: $_('common.error'),
          errorMovingTopLevel: $_('page.sidebar.error_moving_to_top_level'),
          moveError: $_('component.tree.move_error'),
        },
      },
      (active) => {
        isDropZoneActive = active;
      }
    );
  }

  // Touch Drag & Drop handler - bridges touch events to existing store functions
  async function handleTouchDrop(
    dragData: TouchDragData,
    targetData: TouchDragData,
    position: DropPosition
  ) {
    await handleTouchDropAction(
      dragData,
      targetData,
      position,
      {
        moveFolder: tree.moveFolder,
        moveNote: notes.moveNote,
        loadTree: tree.loadTree,
        reorderFolders: tree.reorderFolders,
        reorderNotes: tree.reorderNotes,
        findParentOfNodeById: tree.findParentOfNodeById,
        getSortMode: tree.getSortMode,
        getFolderChildren: (parent: tree.FolderTreeNode) =>
          parent.children.filter((c): c is tree.FolderTreeNode => c.type === 'folder'),
        getNoteChildren: (parent: tree.FolderTreeNode) =>
          parent.children.filter((c): c is tree.NoteTreeNode => c.type === 'note'),
        alert: dialog.alert,
        strings: {
          errorTitle: $_('common.error'),
          errorMovingTopLevel: $_('page.sidebar.error_moving_to_top_level'),
          moveError: $_('component.tree.move_error'),
        },
      },
      (key) => $_(key)
    );
  }

  function handleCreateNote() {
    showCreateNoteDialog = true;
  }

  const sidebarActionDeps = {
    getSelectedFolderPath: tree.getSelectedFolderPath,
    createNote: notes.createNote,
    createFolder: tree.createFolder,
    loadTree: tree.loadTree,
    closeSidebarOnMobile: ui.closeSidebarOnMobile,
    goto,
    confirm: dialog.confirm,
    alert: dialog.alert,
    strings: {
      errorTitle: $_('common.error'),
      createFolderError: (error: string) =>
        $_('page.sidebar.error_creating_folder', { values: { error } }),
    },
  };

  function handleCreateNoteConfirm(title: string, noteType?: string) {
    handleCreateNoteConfirmAction(title, sidebarActionDeps, noteType);
  }

  function handleCreateFolder() {
    showCreateFolderDialog = true;
  }

  function handleCreateFolderConfirm(path: string) {
    handleCreateFolderConfirmAction(path, sidebarActionDeps);
  }
</script>

<!-- Escape handler for mobile drawer -->
<svelte:window
  onkeydown={(e) => {
    handleSidebarEscape(e, {
      isMobile: ui.getIsMobile,
      isOpen: ui.getSidebarOpen,
      isQuickSwitcherOpen: ui.getQuickSwitcherOpen,
      close: () => ui.setSidebarOpen(false),
    });
  }}
  onclick={(e) => {
    if (showSortDropdown) handleClickOutsideSort(e);
  }}
/>

<aside
  aria-label={$_('accessibility.sidebar')}
  class="flex flex-col border-r border-border bg-sidebar-background h-full
		{ui.getIsMobile()
    ? 'fixed inset-y-0 left-0 z-50 w-[85vw] max-w-xs shadow-xl transition-transform duration-200 ' +
      (ui.getSidebarOpen() ? 'translate-x-0' : '-translate-x-full')
    : ui.getSidebarOpen()
      ? 'relative shrink-0'
      : 'relative w-12 shrink-0 transition-[width,opacity] duration-200'}"
  style={!ui.getIsMobile() && ui.getSidebarOpen() ? `width: ${ui.getSidebarWidth()}px` : undefined}
  use:swipe={{
    direction: 'left',
    edge: 'none',
    onSwipe: () => ui.setSidebarOpen(false),
    enabled: () => ui.getIsMobile() && ui.getSidebarOpen() && !touchDragActive,
  }}
>
  {#if ui.getIsMobile()}
    <!-- Mobile: Only render content when drawer is open -->
    {#if ui.getSidebarOpen()}
      <!-- Safe area spacer for iOS PWA standalone mode -->
      <div class="pt-safe shrink-0"></div>
      <!-- Header with creation buttons -->
      <SidebarHeader
        isMobile={true}
        {mainIconSize}
        {showSortDropdown}
        currentSortMode={tree.getSortMode()}
        {sortOptions}
        {sortDropdownRef}
        onCreateNote={handleCreateNote}
        onCreateFolder={handleCreateFolder}
        onToggleSortDropdown={() => (showSortDropdown = !showSortDropdown)}
        onSortSelect={handleSortSelect}
        onCloseSidebar={() => ui.setSidebarOpen(false)}
        onBindSortDropdownRef={(el) => (sortDropdownRef = el)}
        onLogoClick={() => ui.closeSidebarOnMobile()}
      />

      <!-- Notes Tree (main content - maximized space) -->
      <nav
        class="flex-1 min-h-0 overflow-y-auto overscroll-y-contain px-2 py-2 thin-scrollbar"
        aria-label={$_('accessibility.notes_tree')}
        use:touchdrag={{
          holdDuration: 300,
          enabled: () => ui.getIsMobile() && !settings.getVirtualTreeEnabled(),
          onDrop: handleTouchDrop,
          onDragStart: () => (touchDragActive = true),
          onDragEnd: () => (touchDragActive = false),
        }}
      >
        {#if tree.getIsLoading()}
          <div class="px-4 py-2 text-sm text-muted-foreground" role="status">{$_('common.loading')}</div>
        {:else if tree.getTreeData()}
          <!-- Drop zone for moving folders/notes to top level -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <!-- Intentional: Drag-drop zone is inherently pointer-based. Keyboard alternative exists via MoveToFolderDialog context menu. -->
          <div
            class="drop-zone"
            class:active={isDropZoneActive}
            class:touch-drag-visible={touchDragActive}
            data-drag-type="root-dropzone"
            data-drag-id="root"
            ondragover={handleDropZoneDragOver}
            ondragleave={handleDropZoneDragLeave}
            ondrop={handleDropZoneDrop}
          >
            <div class="drop-zone-hint">
              {$_('page.sidebar.drag_to_top_level')}
            </div>
          </div>

          <!-- Render tree (virtual or non-virtual based on settings) -->
          {#if settings.getVirtualTreeEnabled()}
            <VirtualizedTree />
          {:else}
            <!-- Render top-level folders and notes (non-virtual) -->
            {#each tree.getTreeData()?.children ?? [] as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
              <UnifiedTree node={child} />
            {/each}
          {/if}
        {:else}
          <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.no_data')}</div>
        {/if}

        <!-- Due Dates, Shared, Trash + Controls -->
        <SidebarFooter
          isMobile={true}
          {smallIconSize}
          onShowFeedback={() => (showFeedbackDialog = true)}
          onNavigate={() => ui.closeSidebarOnMobile()}
        />
      </nav>

      <!-- Journal Button (if enabled) - Mobile -->
      {#if journalEnabled}
        <div class="border-t border-sidebar-border px-2 py-2 shrink-0">
          <JournalButton />
        </div>
      {/if}

      <!-- Recipe Button (if enabled) - Mobile -->
      {#if recipeEnabled}
        <div class="border-t border-sidebar-border px-2 py-2 shrink-0">
          <RecipeButton />
        </div>
      {/if}
    {/if}
  {:else}
    <!-- Desktop: Normal sidebar behavior -->
    {#if ui.getSidebarOpen()}
      <!-- Header with creation buttons -->
      <SidebarHeader
        isMobile={false}
        {mainIconSize}
        {showSortDropdown}
        currentSortMode={tree.getSortMode()}
        {sortOptions}
        {sortDropdownRef}
        onCreateNote={handleCreateNote}
        onCreateFolder={handleCreateFolder}
        onToggleSortDropdown={() => (showSortDropdown = !showSortDropdown)}
        onSortSelect={handleSortSelect}
        onBindSortDropdownRef={(el) => (sortDropdownRef = el)}
      />

      <!-- Keep collapse action always reachable, even on very narrow sidebar widths -->
      <button
        onclick={() => ui.toggleSidebar()}
        class="absolute top-1/2 right-1 z-20 -translate-y-1/2 p-1.5 rounded-full hover:bg-sidebar-accent/50 text-sidebar-foreground border border-sidebar-border bg-sidebar-background/90 backdrop-blur-sm shadow-sm"
        title={$_('page.sidebar.collapse_sidebar')}
        aria-label={$_('page.sidebar.collapse_sidebar')}
      >
        <ChevronLeft size={mainIconSize} />
      </button>

      <!-- Notes Tree (main content - maximized space) -->
      <nav
        class="flex-1 min-h-0 overflow-y-auto overscroll-y-contain px-2 py-2 thin-scrollbar"
        aria-label={$_('accessibility.notes_tree')}
        use:touchdrag={{
          holdDuration: 300,
          enabled: () => ui.getIsMobile() && !settings.getVirtualTreeEnabled(),
          onDrop: handleTouchDrop,
          onDragStart: () => (touchDragActive = true),
          onDragEnd: () => (touchDragActive = false),
        }}
      >
        {#if tree.getIsLoading()}
          <div class="px-4 py-2 text-sm text-muted-foreground" role="status">{$_('common.loading')}</div>
        {:else if tree.getTreeData()}
          <!-- Drop zone for moving folders/notes to top level -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <!-- Intentional: Drag-drop zone is inherently pointer-based. Keyboard alternative exists via MoveToFolderDialog context menu. -->
          <div
            class="drop-zone"
            class:active={isDropZoneActive}
            class:touch-drag-visible={touchDragActive}
            data-drag-type="root-dropzone"
            data-drag-id="root"
            ondragover={handleDropZoneDragOver}
            ondragleave={handleDropZoneDragLeave}
            ondrop={handleDropZoneDrop}
          >
            <div class="drop-zone-hint">
              {$_('page.sidebar.drag_to_top_level')}
            </div>
          </div>

          <!-- Render tree (virtual or non-virtual based on settings) -->
          {#if settings.getVirtualTreeEnabled()}
            <VirtualizedTree />
          {:else}
            <!-- Render top-level folders and notes (non-virtual) -->
            {#each tree.getTreeData()?.children ?? [] as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
              <UnifiedTree node={child} />
            {/each}
          {/if}
        {:else}
          <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.no_data')}</div>
        {/if}

        <!-- Due Dates, Shared, Trash + Controls -->
        <SidebarFooter
          isMobile={false}
          {smallIconSize}
          onShowFeedback={() => (showFeedbackDialog = true)}
        />
      </nav>

      <!-- Journal Button (if enabled) - Desktop -->
      {#if journalEnabled}
        <div class="border-t border-sidebar-border px-2 py-2 shrink-0">
          <JournalButton />
        </div>
      {/if}

      <!-- Recipe Button (if enabled) - Desktop -->
      {#if recipeEnabled}
        <div class="border-t border-sidebar-border px-2 py-2 shrink-0">
          <RecipeButton />
        </div>
      {/if}

      <!-- Resize handle -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="resize-handle"
        class:active={isResizing}
        role="separator"
        aria-orientation="vertical"
        aria-label={$_('accessibility.resize_sidebar')}
        onpointerdown={handleResizeStart}
        onpointermove={handleResizeMove}
        onpointerup={handleResizeEnd}
        onpointercancel={handleResizeEnd}
        ondblclick={handleResizeDblClick}
      ></div>
    {:else}
      <!-- Collapsed sidebar -->
      <nav class="flex flex-col items-center h-full min-h-0 overflow-y-auto py-3 gap-1.5" aria-label={$_('accessibility.sidebar')}>
        <!-- Top actions -->
        <button
          onclick={() => ui.toggleSidebar()}
          class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('page.sidebar.expand_sidebar')}
          aria-label={$_('page.sidebar.expand_sidebar')}
        >
          <ChevronRight size={mainIconSize} />
        </button>
        <button
          onclick={handleCreateNote}
          class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('page.sidebar.new_note')}
          aria-label={$_('page.sidebar.new_note')}
        >
          <FilePlus size={mainIconSize} />
        </button>
        <button
          onclick={handleCreateFolder}
          class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('page.sidebar.new_folder')}
          aria-label={$_('page.sidebar.new_folder')}
        >
          <FolderPlus size={mainIconSize} />
        </button>
        <button
          onclick={() => ui.toggleQuickSwitcher()}
          class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title="{$_('page.sidebar.search')} (Ctrl+P)"
          aria-label={$_('page.sidebar.search')}
        >
          <Search size={mainIconSize} />
        </button>
        {#if features.getGraphFeatureEnabled()}
          <button
            onclick={() => goto('/graph')}
            class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
            title="{$_('page.sidebar.graph')} (Ctrl+G)"
            aria-label={$_('page.sidebar.graph')}
          >
            <Network size={mainIconSize} />
          </button>
        {/if}

        <!-- Spacer -->
        <div class="flex-1"></div>

        <!-- Bottom actions -->
        <button
          onclick={() => goto('/due-dates')}
          class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('page.due_dates.title')}
          aria-label={$_('page.due_dates.title')}
        >
          <CalendarClock size={smallIconSize} />
        </button>
        <button
          onclick={() => goto('/shared')}
          class="relative p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('sharing.shared_with_me')}
          aria-label={$_('sharing.shared_with_me')}
        >
          <Users size={smallIconSize} />
          {#if sharing.getTotalSharedCount() > 0}
            <span
              class="absolute -top-0.5 -right-0.5 bg-primary text-primary-foreground text-[8px] font-medium w-3.5 h-3.5 rounded-full flex items-center justify-center"
            >
              {sharing.getTotalSharedCount()}
            </span>
          {/if}
        </button>
        <button
          onclick={() => goto('/trash')}
          class="relative p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
          title={$_('page.sidebar.trash')}
          aria-label={$_('page.sidebar.trash')}
        >
          <Trash2 size={smallIconSize} />
          {#if trash.getTrashCount() > 0}
            <span
              class="absolute -top-0.5 -right-0.5 bg-destructive text-destructive-foreground text-[8px] font-medium w-3.5 h-3.5 rounded-full flex items-center justify-center"
            >
              {trash.getTrashCount()}
            </span>
          {/if}
        </button>
        <div class="w-6 border-t border-sidebar-border my-0.5"></div>
        <ThemeSelector />
        <button
          onclick={() => goto('/settings')}
          class="p-2 rounded hover:bg-sidebar-accent/50 text-sidebar-foreground"
          title={$_('page.sidebar.settings')}
          aria-label={$_('page.sidebar.settings')}
        >
          <Settings size={smallIconSize} />
        </button>
        {#if auth.isAdmin()}
          <button
            onclick={() => goto('/admin')}
            class="p-2 rounded hover:bg-sidebar-accent/50 text-sidebar-foreground"
            title={$_('page.sidebar.admin')}
            aria-label={$_('page.sidebar.admin')}
          >
            <Shield size={smallIconSize} />
          </button>
        {/if}
        {#if errorReporter.getServiceAvailable()}
          <button
            onclick={() => {
              showFeedbackDialog = true;
            }}
            class="p-2 rounded hover:bg-sidebar-accent/50 text-sidebar-foreground"
            title={$_('feedback.sidebar_button')}
            aria-label={$_('feedback.sidebar_button')}
          >
            <MessageSquareWarning size={smallIconSize} />
          </button>
        {/if}
      </nav>
    {/if}
  {/if}
</aside>

<!-- Create Note Dialog -->
{#if showCreateNoteDialog}
  <CreateNoteDialog
    open={true}
    folderPath={tree.getSelectedFolderPath() || '/'}
    onClose={() => (showCreateNoteDialog = false)}
    onCreate={handleCreateNoteConfirm}
  />
{/if}

<!-- Create Folder Dialog -->
{#if showCreateFolderDialog}
  <CreateFolderDialog
    open={true}
    onClose={() => (showCreateFolderDialog = false)}
    onCreate={handleCreateFolderConfirm}
  />
{/if}

<!-- Feedback Dialog -->
{#if showFeedbackDialog}
  <FeedbackDialog open={true} onClose={() => (showFeedbackDialog = false)} />
{/if}

<style>
  .drop-zone {
    margin: 4px 8px;
    padding: 0;
    border: 1px dashed transparent;
    border-radius: var(--radius-sm);
    text-align: center;
    transition: all var(--duration-fast) var(--ease-default);
    opacity: 0;
    height: 0;
    overflow: hidden;
  }

  .drop-zone.active {
    opacity: 1;
    height: auto;
    padding: 8px 12px;
    border-color: var(--color-primary);
    background: rgba(59, 130, 246, 0.05);
    background: color-mix(in oklch, var(--color-primary), transparent 90%);
  }

  .drop-zone-hint {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
    pointer-events: none;
    font-weight: 500;
  }

  .drop-zone.active .drop-zone-hint {
    color: var(--color-primary);
  }

  /* Touch devices: show drop zone only during active touch drag.
     Use sticky + negative margin so it overlays the first item
     instead of pushing the tree content down (prevents layout shift). */
  .drop-zone.touch-drag-visible {
    position: sticky;
    top: 0;
    z-index: 5;
    opacity: 1;
    height: 48px;
    margin-bottom: -48px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-color: var(--color-border);
    background: var(--color-sidebar-background);
  }

  /* Touch drag: highlight root drop zone when hovered */
  :global(.drop-zone.touch-drop-into) {
    border-color: var(--color-primary, #3b82f6) !important;
    background: color-mix(in oklch, var(--color-primary), transparent 90%) !important;
    opacity: 1 !important;
  }

  .resize-handle {
    position: absolute;
    top: 0;
    right: -3px;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 10;
    transition: background-color var(--duration-fast) var(--ease-default);
  }

  .resize-handle:hover,
  .resize-handle.active {
    background-color: var(--color-primary, oklch(0.65 0.15 155));
  }
</style>
