<script lang="ts">
  import { CalendarClock, Home, PanelLeft, Settings, Trash2, Users } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/state';
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
  import SidebarHeader from './sidebar/SidebarHeader.svelte';
  import ThemeSelector from './ThemeSelector.svelte';
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

  // Icon strip width constant
  const ICON_STRIP_WIDTH = 40;

  // Feature flags (reactive)
  const journalEnabled = $derived(features.getJournalFeatureEnabled());
  const recipeEnabled = $derived(features.getRecipeFeatureEnabled());
  const graphEnabled = $derived(features.getGraphFeatureEnabled());

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

  const iconStripButtonBase =
    'relative p-2 rounded-lg text-sidebar-foreground transition-colors hover:bg-sidebar-accent';
  const iconStripButtonSubtle =
    'relative p-2 rounded-lg text-sidebar-foreground transition-colors hover:bg-sidebar-accent/50';

  function isPathActive(path: string) {
    return page.url.pathname === path;
  }

  function iconStripButtonClass(active: boolean, subtle = false) {
    return `icon-strip-nav-button ${subtle ? iconStripButtonSubtle : iconStripButtonBase} ${active ? 'active bg-sidebar-accent shadow-sm' : ''}`;
  }

  function isJournalTopLevelNode(node: tree.TreeNode) {
    return node.type === 'folder' && node.path === '/Journal';
  }

  function getTopLevelNonJournalNodes() {
    return (tree.getTreeData()?.children ?? []).filter((child) => !isJournalTopLevelNode(child));
  }

  function getTopLevelJournalNode() {
    return (tree.getTreeData()?.children ?? []).find((child) => isJournalTopLevelNode(child));
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

{#if ui.getIsMobile()}
  <!-- Mobile: Two-column drawer (icon strip + tree panel), mirrors desktop layout -->
  <aside
    aria-label={$_('accessibility.sidebar')}
    class="flex flex-row border-r border-border bg-sidebar-background h-full
      fixed inset-y-0 left-0 z-50 w-[85vw] max-w-xs shadow-xl transition-transform duration-200
      {ui.getSidebarOpen() ? 'translate-x-0' : '-translate-x-full'}"
    use:swipe={{
      direction: 'left',
      edge: 'none',
      onSwipe: () => ui.setSidebarOpen(false),
      enabled: () => ui.getIsMobile() && ui.getSidebarOpen() && !touchDragActive,
    }}
  >
    {#if ui.getSidebarOpen()}
      <!-- Left icon strip — matches desktop -->
      <nav
        class="flex flex-col items-center h-full shrink-0 border-r border-sidebar-border py-2 gap-1"
        style="width: {ICON_STRIP_WIDTH}px; padding-top: calc(var(--safe-area-inset-top) + 2.5rem)"
        aria-label={$_('accessibility.sidebar')}
      >
        <!-- Home -->
        <button
          onclick={() => {
            goto('/');
            ui.closeSidebarOnMobile();
          }}
          class={iconStripButtonClass(isPathActive('/'))}
          title={$_('page.home.title')}
          aria-label={$_('page.home.title')}
        >
          <Home size={smallIconSize} />
        </button>

        <!-- Due Dates -->
        <button
          onclick={() => {
            goto('/due-dates');
            ui.closeSidebarOnMobile();
          }}
          class={iconStripButtonClass(isPathActive('/due-dates'))}
          title={$_('page.due_dates.title')}
          aria-label={$_('page.due_dates.title')}
        >
          <CalendarClock size={smallIconSize} />
        </button>

        <!-- Shared -->
        <button
          onclick={() => {
            goto('/shared');
            ui.closeSidebarOnMobile();
          }}
          class={iconStripButtonClass(isPathActive('/shared'))}
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

        <!-- Trash -->
        <button
          onclick={() => {
            goto('/trash');
            ui.closeSidebarOnMobile();
          }}
          class={iconStripButtonClass(isPathActive('/trash'))}
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

        <div class="h-px w-6 my-1 bg-sidebar-border/80"></div>

        <!-- Journal (if enabled) -->
        {#if journalEnabled}
          <div class="icon-strip-item">
            <JournalButton iconOnly />
          </div>
        {/if}

        <!-- Recipe (if enabled) -->
        {#if recipeEnabled}
          <div class="icon-strip-item">
            <RecipeButton iconOnly />
          </div>
        {/if}

        <!-- Spacer -->
        <div class="flex-1"></div>

        <!-- Theme toggle -->
        <ThemeSelector />

        <!-- Settings -->
        <button
          onclick={() => {
            goto('/settings');
            ui.closeSidebarOnMobile();
          }}
          class={iconStripButtonClass(isPathActive('/settings'), true)}
          title={$_('page.sidebar.settings')}
          aria-label={$_('page.sidebar.settings')}
        >
          <Settings size={smallIconSize} />
        </button>
      </nav>

      <!-- Right panel: header + tree -->
      <div class="flex flex-col flex-1 min-w-0">
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
          onBindSortDropdownRef={(el) => (sortDropdownRef = el)}
        />

        <!-- Notes Tree (main content - maximized space) -->
        <nav
          class="flex-1 min-h-0 overflow-y-auto overscroll-y-contain thin-scrollbar px-2 py-2"
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
            <div class="px-4 py-2 text-sm text-muted-foreground" role="status">
              {$_('common.loading')}
            </div>
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
              {#if getTopLevelNonJournalNodes().length > 0}
                <div class="sidebar-tree-section-label">{$_('page.sidebar.section_notes')}</div>
                {#each getTopLevelNonJournalNodes() as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
                  <UnifiedTree node={child} />
                {/each}
              {/if}

              {#if getTopLevelJournalNode()}
                <div class="sidebar-tree-section-label section-divider mt-2">
                  {$_('page.sidebar.section_journal')}
                </div>
                <UnifiedTree node={getTopLevelJournalNode()!} />
              {/if}
            {/if}
          {:else}
            <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.no_data')}</div>
          {/if}

          <!-- Footer (Admin/Feedback only) -->
          <SidebarFooter
            isMobile={true}
            {smallIconSize}
            onShowFeedback={() => (showFeedbackDialog = true)}
            onNavigate={() => ui.closeSidebarOnMobile()}
          />
        </nav>
      </div>
    {/if}
  </aside>
{:else}
  <!-- Desktop: Two-column layout (icon strip + collapsible panel) -->
  <aside
    aria-label={$_('accessibility.sidebar')}
    class="flex flex-row border-r border-border bg-sidebar-background h-full relative shrink-0"
    style="width: {ui.getSidebarOpen() ? ui.getSidebarWidth() : ICON_STRIP_WIDTH}px"
  >
    <!-- Left icon strip — always visible -->
    <nav
      class="flex flex-col items-center h-full shrink-0 border-r border-sidebar-border py-2 gap-1"
      style="width: {ICON_STRIP_WIDTH}px"
      aria-label={$_('accessibility.sidebar')}
    >
      <!-- Toggle panel expand/collapse -->
      <button
        onclick={() => ui.toggleSidebar()}
        class={iconStripButtonClass(ui.getSidebarOpen())}
        title={$_(
          ui.getSidebarOpen() ? 'page.sidebar.collapse_sidebar' : 'page.sidebar.expand_sidebar'
        )}
        aria-label={$_(
          ui.getSidebarOpen() ? 'page.sidebar.collapse_sidebar' : 'page.sidebar.expand_sidebar'
        )}
      >
        <PanelLeft size={smallIconSize} />
      </button>

      <!-- Home -->
      <button
        onclick={() => goto('/')}
        class={iconStripButtonClass(isPathActive('/'))}
        title={$_('page.home.title')}
        aria-label={$_('page.home.title')}
      >
        <Home size={smallIconSize} />
      </button>

      <!-- Due Dates -->
      <button
        onclick={() => goto('/due-dates')}
        class={iconStripButtonClass(isPathActive('/due-dates'))}
        title={$_('page.due_dates.title')}
        aria-label={$_('page.due_dates.title')}
      >
        <CalendarClock size={smallIconSize} />
      </button>

      <!-- Shared -->
      <button
        onclick={() => goto('/shared')}
        class={iconStripButtonClass(isPathActive('/shared'))}
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

      <!-- Trash -->
      <button
        onclick={() => goto('/trash')}
        class={iconStripButtonClass(isPathActive('/trash'))}
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

      <div class="h-px w-6 my-1 bg-sidebar-border/80"></div>

      <!-- Journal (if enabled) -->
      {#if journalEnabled}
        <div class="icon-strip-item">
          <JournalButton iconOnly />
        </div>
      {/if}

      <!-- Recipe (if enabled) -->
      {#if recipeEnabled}
        <div class="icon-strip-item">
          <RecipeButton iconOnly />
        </div>
      {/if}

      <!-- Spacer -->
      <div class="flex-1"></div>

      <!-- Theme toggle -->
      <ThemeSelector />

      <!-- Settings -->
      <button
        onclick={() => goto('/settings')}
        class={iconStripButtonClass(isPathActive('/settings'), true)}
        title={$_('page.sidebar.settings')}
        aria-label={$_('page.sidebar.settings')}
      >
        <Settings size={smallIconSize} />
      </button>
    </nav>

    <!-- Collapsible main panel -->
    {#if ui.getSidebarOpen()}
      <div class="flex flex-col flex-1 min-w-0 relative">
        <!-- Header toolbar -->
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
          onToggleSearch={() => ui.toggleQuickSwitcher()}
          onOpenGraph={() => goto('/graph')}
          {graphEnabled}
          onCollapseSidebar={() => ui.toggleSidebar()}
        />

        <!-- Notes Tree -->
        <nav
          class="flex-1 min-h-0 overflow-y-auto overscroll-y-contain thin-scrollbar px-1.5 py-1.5"
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
            <div class="px-4 py-2 text-sm text-muted-foreground" role="status">
              {$_('common.loading')}
            </div>
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
              {#if getTopLevelNonJournalNodes().length > 0}
                <div class="sidebar-tree-section-label">{$_('page.sidebar.section_notes')}</div>
                {#each getTopLevelNonJournalNodes() as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
                  <UnifiedTree node={child} />
                {/each}
              {/if}

              {#if getTopLevelJournalNode()}
                <div class="sidebar-tree-section-label section-divider mt-2">
                  {$_('page.sidebar.section_journal')}
                </div>
                <UnifiedTree node={getTopLevelJournalNode()!} />
              {/if}
            {/if}
          {:else}
            <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.no_data')}</div>
          {/if}
        </nav>

        <!-- Footer (Admin/Feedback only) -->
        <SidebarFooter
          isMobile={false}
          {smallIconSize}
          onShowFeedback={() => (showFeedbackDialog = true)}
        />

        <!-- Resize handle -->
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
      </div>
    {/if}
  </aside>
{/if}

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

  .icon-strip-item {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-strip-nav-button {
    position: relative;
  }

  .icon-strip-nav-button.active::before {
    content: '';
    position: absolute;
    left: -5px;
    top: 5px;
    bottom: 5px;
    width: 3px;
    border-radius: 999px;
    background: var(--color-primary);
    opacity: 1;
    box-shadow: 0 0 8px color-mix(in oklch, var(--color-primary), transparent 60%);
  }

  .icon-strip-nav-button.active {
    background: color-mix(in oklch, var(--color-sidebar-accent), var(--color-primary) 10%);
  }

  .sidebar-tree-section-label {
    font-size: 0.68rem;
    line-height: 1;
    font-weight: 600;
    letter-spacing: 0.045em;
    text-transform: uppercase;
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 48%
    );
    padding: 0.55rem 0.5rem 0.32rem;
  }

  .sidebar-tree-section-label.section-divider {
    border-top: 1px solid color-mix(in oklch, var(--color-sidebar-border), transparent 22%);
    margin-top: 0.5rem;
    padding-top: 0.65rem;
  }
</style>
