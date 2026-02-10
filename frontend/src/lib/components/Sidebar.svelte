<script lang="ts">
  import {
    CalendarClock,
    ChevronLeft,
    ChevronRight,
    FilePlus,
    FolderPlus,
    LogOut,
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
  import type { DropPosition,TouchDragData } from '$lib/actions/touchdrag';
  import { touchdrag } from '$lib/actions/touchdrag';
  import { getConfig } from '$lib/api';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as sharing from '$lib/stores/sharing.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { validateDrop } from '$lib/utils/tree-drop-validation';

  import ChangelogDialog from './ChangelogDialog.svelte';
  import CreateFolderDialog from './CreateFolderDialog.svelte';
  import CreateNoteDialog from './CreateNoteDialog.svelte';
  import FeedbackDialog from './FeedbackDialog.svelte';
  import JournalButton from './JournalButton.svelte';
  import Logo from './Logo.svelte';
  import RecipeButton from './RecipeButton.svelte';
  import ThemeSelector from './ThemeSelector.svelte';
  import UnifiedTree from './UnifiedTree.svelte';
  import VirtualizedTree from './VirtualizedTree.svelte';

  let appVersion = $state('');
  let isDropZoneActive = $state(false);
  let touchDragActive = $state(false);
  let showCreateNoteDialog = $state(false);
  let showCreateFolderDialog = $state(false);
  let showFeedbackDialog = $state(false);
  let showChangelogDialog = $state(false);

  // Resize state
  let isResizing = $state(false);
  let startX = 0;
  let startWidth = 0;

  function handleResizeStart(e: PointerEvent) {
    if (ui.getIsMobile()) return;
    e.preventDefault();
    isResizing = true;
    startX = e.clientX;
    startWidth = ui.getSidebarWidth();
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';
  }

  function handleResizeMove(e: PointerEvent) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    ui.setSidebarWidth(startWidth + delta);
  }

  function handleResizeEnd() {
    isResizing = false;
    document.body.style.userSelect = '';
    document.body.style.cursor = '';
  }

  function handleResizeDblClick() {
    ui.setSidebarWidth(256);
  }

  // Responsive icon sizes: 20px on mobile, 18px on desktop
  const mainIconSize = $derived(ui.getIsMobile() ? 20 : 18);
  const smallIconSize = $derived(ui.getIsMobile() ? 18 : 16);

  // Feature flags (reactive)
  const journalEnabled = $derived(features.getJournalFeatureEnabled());
  const recipeEnabled = $derived(features.getRecipeFeatureEnabled());

  // Restore expanded state on mount
  onMount(() => {
    tree.loadExpandedStateFromStorage();

    // Load trash count and shared items (notes + folders)
    trash.loadTrashCount();
    sharing.loadAllShared();

    // Load journal feature state
    features.loadJournalFeature();

    // Load recipe feature state
    features.loadRecipeFeature();

    // Load app version from config
    getConfig()
      .then((config) => {
        appVersion = config.version || '';
      })
      .catch(() => {});

    // Auto-refresh trash count every 30 seconds
    const interval = setInterval(() => {
      trash.loadTrashCount();
    }, 30000);

    return () => clearInterval(interval);
  });

  // Load tree when authenticated (reactive)
  $effect(() => {
    if (auth.isAuthenticated()) {
      tree.loadTree();
    }
  });

  // Drop zone handlers for moving folders to top-level
  function handleDropZoneDragOver(e: DragEvent) {
    if (e.dataTransfer?.types.includes('application/x-xelanote-item')) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      isDropZoneActive = true;
    }
  }

  function handleDropZoneDragLeave() {
    isDropZoneActive = false;
  }

  async function handleDropZoneDrop(e: DragEvent) {
    e.preventDefault();
    isDropZoneActive = false;

    const data = e.dataTransfer?.getData('application/x-xelanote-item');
    if (!data) return;

    try {
      const dragData = JSON.parse(data);

      if (dragData.type === 'folder') {
        // Move folder to root level (parent_id = 1, path = "/")
        await tree.moveFolder(dragData.id, '/');
      } else if (dragData.type === 'note') {
        // Move note to root folder
        await notes.moveNote(dragData.id, '/');
        await tree.loadTree();
      }
    } catch (err) {
      console.error('Failed to move to top level:', err);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('page.sidebar.error_moving_to_top_level'),
        variant: 'danger',
      });
    }
  }

  // Touch Drag & Drop handler - bridges touch events to existing store functions
  async function handleTouchDrop(
    dragData: TouchDragData,
    targetData: TouchDragData,
    position: DropPosition
  ) {
    const validation = validateDrop(
      {
        type: dragData.type,
        id: dragData.id,
        path: dragData.path,
        folder_path: dragData.folder_path,
      },
      {
        type: targetData.type,
        id: targetData.id,
        path: targetData.path,
        folder_path: targetData.folder_path,
      },
      position
    );

    if (!validation.valid) {
      if (validation.errorKey) {
        await dialog.alert({
          title: $_('common.error'),
          message: $_(validation.errorKey),
          variant: 'warning',
        });
      }
      return;
    }

    try {
      // Root drop zone: move to top level
      if (targetData.type === 'root-dropzone') {
        if (dragData.type === 'folder') {
          await tree.moveFolder(Number(dragData.id), '/');
        } else {
          await notes.moveNote(dragData.id, '/');
          await tree.loadTree();
        }
        return;
      }

      // Reorder: same-type siblings at before/after positions
      if (position === 'before' || position === 'after') {
        // Folder reorder (siblings)
        if (dragData.type === 'folder' && targetData.type === 'folder') {
          const reordered = await handleTouchFolderReorder(
            Number(dragData.id),
            Number(targetData.id),
            position
          );
          if (reordered) return;
          // Not siblings - fall through to move into folder
        }

        // Note reorder (same folder)
        if (dragData.type === 'note' && targetData.type === 'note') {
          if (dragData.folder_path === targetData.folder_path) {
            await handleTouchNoteReorder(dragData.id, targetData.id, position);
            return;
          }
          // Cross-folder: move note to target note's folder
          await notes.moveNote(dragData.id, targetData.folder_path!);
          await tree.loadTree();
          return;
        }
      }

      // Move into folder
      if (targetData.type === 'folder') {
        if (dragData.type === 'note') {
          await notes.moveNote(dragData.id, targetData.path!);
          await tree.loadTree();
        } else if (dragData.type === 'folder') {
          await tree.moveFolder(Number(dragData.id), targetData.path!);
        }
      }
    } catch (err) {
      console.error('Failed to move:', err);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('component.tree.move_error'),
        variant: 'danger',
      });
    }
  }

  async function handleTouchFolderReorder(
    draggedId: number,
    targetId: number,
    position: 'before' | 'after'
  ): Promise<boolean> {
    const parent = tree.findParentOfNodeById('folder', targetId);
    if (!parent) return false;

    const draggedParent = tree.findParentOfNodeById('folder', draggedId);
    if (!draggedParent || draggedParent !== parent) return false; // Not siblings

    const siblings = parent.children.filter((c) => c.type === 'folder') as tree.FolderTreeNode[];
    const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
    const targetIndex = siblings.findIndex((s) => s.id === targetId);
    if (draggedIndex === -1 || targetIndex === -1) return false;

    const newOrder = [...siblings];
    const [draggedItem] = newOrder.splice(draggedIndex, 1);
    const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
    newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

    const folderIds = newOrder.map((s) => s.id).filter((id) => id !== 0);
    const parentID: number | null = parent.id === 0 ? 1 : parent.id || null;
    await tree.reorderFolders(parentID, folderIds);
    return true;
  }

  async function handleTouchNoteReorder(
    draggedId: string,
    targetId: string,
    position: 'before' | 'after'
  ) {
    const parent = tree.findParentOfNodeById('note', targetId);
    if (!parent) return;

    const siblings = parent.children.filter((c) => c.type === 'note') as tree.NoteTreeNode[];
    const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
    const targetIndex = siblings.findIndex((s) => s.id === targetId);
    if (draggedIndex === -1 || targetIndex === -1) return;

    const newOrder = [...siblings];
    const [draggedItem] = newOrder.splice(draggedIndex, 1);
    const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
    newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

    const noteIds = newOrder.map((s) => s.id);
    const folderPath = parent.path || '/';
    await tree.reorderNotes(folderPath, noteIds);
  }

  function handleCreateNote() {
    showCreateNoteDialog = true;
  }

  function handleCreateNoteConfirm(title: string) {
    const selectedPath = tree.getSelectedFolderPath();
    const folderPath = selectedPath || '/';
    notes.createNote(title, '', folderPath).then((note) => {
      // Reload tree to update counts
      tree.loadTree();
      goto(`/note/${note.id}`);
      ui.closeSidebarOnMobile();
    });
  }

  function handleCreateFolder() {
    showCreateFolderDialog = true;
  }

  function handleCreateFolderConfirm(path: string) {
    tree.createFolder(path).catch(async (err) => {
      console.error('Fehler beim Erstellen:', err);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('page.sidebar.error_creating_folder', { values: { error: err.message } }),
        variant: 'danger',
      });
    });
  }

  async function handleLogout() {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('page.sidebar.confirm_logout'),
      confirmText: $_('common.logout'),
      cancelText: $_('dialog.cancel'),
    });

    if (!confirmed) return;

    try {
      // Stop auto-lock timer
      autoLock.stopAutoLock();

      await auth.logoutAsync();
      // Force reload to clear all state and prevent race conditions
      window.location.href = '/login';
    } catch (err) {
      console.error('Logout failed:', err);
      // Even if backend logout fails, we're logged out locally
      window.location.href = '/login';
    }
  }
</script>

<!-- Escape handler for mobile drawer -->
<svelte:window
  onkeydown={(e) => {
    if (
      e.key === 'Escape' &&
      ui.getIsMobile() &&
      ui.getSidebarOpen() &&
      !ui.getQuickSwitcherOpen()
    ) {
      ui.setSidebarOpen(false);
    }
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
      : 'relative w-12 shrink-0 transition-all duration-200'}"
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
      <!-- Header with creation buttons -->
      <div
        class="flex items-center justify-between px-4 py-2.5 border-b border-sidebar-border shrink-0"
      >
        <a
          href="/"
          onclick={() => ui.closeSidebarOnMobile()}
          class="hover:opacity-80 transition-opacity"
        >
          <Logo size="md" />
        </a>
        <div class="flex items-center gap-0.5">
          <button
            onclick={handleCreateNote}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors toolbar-btn"
            title={$_('page.sidebar.new_note')}
            aria-label={$_('page.sidebar.new_note')}
          >
            <FilePlus size={mainIconSize} />
          </button>
          <button
            onclick={handleCreateFolder}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors toolbar-btn"
            title={$_('page.sidebar.new_folder')}
            aria-label={$_('page.sidebar.new_folder')}
          >
            <FolderPlus size={mainIconSize} />
          </button>
          <button
            onclick={() => ui.setSidebarOpen(false)}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground toolbar-btn"
            title={$_('page.sidebar.close_drawer')}
            aria-label={$_('page.sidebar.close_drawer')}
          >
            <ChevronLeft size={mainIconSize} />
          </button>
        </div>
      </div>

      <!-- Search row -->
      <div class="flex items-center gap-1 px-2 py-1.5 border-b border-sidebar-border shrink-0">
        <button
          onclick={() => ui.toggleQuickSwitcher()}
          class="flex-1 flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
          title="{$_('page.sidebar.search')} (Ctrl+P)"
          aria-label={$_('page.sidebar.search')}
        >
          <Search size={16} />
          <span class="text-muted-foreground">{$_('page.sidebar.search')}...</span>
          <span class="ml-auto text-xs text-muted-foreground">Ctrl+P</span>
        </button>
        {#if features.getGraphFeatureEnabled()}
          <button
            onclick={() => {
              goto('/graph');
              ui.closeSidebarOnMobile();
            }}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors toolbar-btn"
            title="{$_('page.sidebar.graph')} (Ctrl+G)"
            aria-label={$_('page.sidebar.graph')}
          >
            <Network size={16} />
          </button>
        {/if}
      </div>

      <!-- Notes Tree (main content - maximized space) -->
      <div
        class="flex-1 min-h-0 overflow-y-auto px-2 py-2"
        use:touchdrag={{
          holdDuration: 300,
          enabled: () => ui.getIsMobile() && !settings.getVirtualTreeEnabled(),
          onDrop: handleTouchDrop,
          onDragStart: () => (touchDragActive = true),
          onDragEnd: () => (touchDragActive = false),
        }}
      >
        {#if tree.getIsLoading()}
          <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.loading')}</div>
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

        <!-- Due Dates, Shared, Trash as virtual folders at bottom of tree -->
        <div class="mt-2 pt-2 border-t border-sidebar-border mx-1 space-y-0.5">
          <button
            onclick={() => {
              goto('/due-dates');
              ui.closeSidebarOnMobile();
            }}
            class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
            title={$_('page.due_dates.title')}
            aria-label={$_('page.due_dates.title')}
          >
            <CalendarClock size={16} class="text-muted-foreground flex-shrink-0" />
            <span class="flex-1 truncate">{$_('page.due_dates.title')}</span>
          </button>
          <button
            onclick={() => {
              goto('/shared');
              ui.closeSidebarOnMobile();
            }}
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
            onclick={() => {
              goto('/trash');
              ui.closeSidebarOnMobile();
            }}
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
      </div>

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

      <!-- Footer: User info + Controls -->
      <div class="border-t border-sidebar-border shrink-0 pb-safe">
        <!-- User info - compact -->
        {#if auth.getCurrentUser()}
          <div class="px-4 py-1.5 text-xs text-muted-foreground truncate">
            {auth.getCurrentUser()?.email}{#if appVersion}<button
                onclick={() => (showChangelogDialog = true)}
                class="opacity-60 hover:opacity-100 hover:text-primary transition-all cursor-pointer"
                title="Changelog"
              >
                · {appVersion}</button
              >{/if}
          </div>
        {/if}

        <!-- Controls - compact row -->
        <div class="px-2 py-2 flex items-center gap-1">
          <ThemeSelector />
          <button
            onclick={() => {
              goto('/settings');
              ui.closeSidebarOnMobile();
            }}
            class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground toolbar-btn"
            title={$_('page.sidebar.settings')}
            aria-label={$_('page.sidebar.settings')}
          >
            <Settings size={smallIconSize} />
          </button>
          {#if auth.isAdmin()}
            <button
              onclick={() => {
                goto('/admin');
                ui.closeSidebarOnMobile();
              }}
              class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground toolbar-btn"
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
              class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground toolbar-btn"
              title={$_('feedback.sidebar_button')}
              aria-label={$_('feedback.sidebar_button')}
            >
              <MessageSquareWarning size={smallIconSize} />
            </button>
          {/if}
          <div class="flex-1"></div>
          <button
            onclick={handleLogout}
            class="p-2 rounded-lg hover:bg-sidebar-accent/50 hover:text-red-500 text-sidebar-foreground toolbar-btn"
            title={$_('common.logout')}
            aria-label={$_('common.logout')}
          >
            <LogOut size={smallIconSize} />
          </button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- Desktop: Normal sidebar behavior -->
    {#if ui.getSidebarOpen()}
      <!-- Header with creation buttons -->
      <div
        class="flex items-center justify-between px-4 py-2.5 border-b border-sidebar-border shrink-0"
      >
        <a href="/" class="hover:opacity-80 transition-opacity">
          <Logo size="md" />
        </a>
        <div class="flex items-center gap-0.5">
          <button
            onclick={handleCreateNote}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors"
            title={$_('page.sidebar.new_note')}
            aria-label={$_('page.sidebar.new_note')}
          >
            <FilePlus size={mainIconSize} />
          </button>
          <button
            onclick={handleCreateFolder}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors"
            title={$_('page.sidebar.new_folder')}
            aria-label={$_('page.sidebar.new_folder')}
          >
            <FolderPlus size={mainIconSize} />
          </button>
          <button
            onclick={() => ui.toggleSidebar()}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground"
            title={$_('page.sidebar.collapse_sidebar')}
            aria-label={$_('page.sidebar.collapse_sidebar')}
          >
            <ChevronLeft size={mainIconSize} />
          </button>
        </div>
      </div>

      <!-- Search row -->
      <div class="flex items-center gap-1 px-2 py-1.5 border-b border-sidebar-border shrink-0">
        <button
          onclick={() => ui.toggleQuickSwitcher()}
          class="flex-1 flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
          title="{$_('page.sidebar.search')} (Ctrl+P)"
          aria-label={$_('page.sidebar.search')}
        >
          <Search size={16} />
          <span class="text-muted-foreground">{$_('page.sidebar.search')}...</span>
          <span class="ml-auto text-xs text-muted-foreground">Ctrl+P</span>
        </button>
        {#if features.getGraphFeatureEnabled()}
          <button
            onclick={() => goto('/graph')}
            class="p-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground transition-colors"
            title="{$_('page.sidebar.graph')} (Ctrl+G)"
            aria-label={$_('page.sidebar.graph')}
          >
            <Network size={16} />
          </button>
        {/if}
      </div>

      <!-- Notes Tree (main content - maximized space) -->
      <div
        class="flex-1 min-h-0 overflow-y-auto px-2 py-2"
        use:touchdrag={{
          holdDuration: 300,
          enabled: () => ui.getIsMobile() && !settings.getVirtualTreeEnabled(),
          onDrop: handleTouchDrop,
          onDragStart: () => (touchDragActive = true),
          onDragEnd: () => (touchDragActive = false),
        }}
      >
        {#if tree.getIsLoading()}
          <div class="px-4 py-2 text-sm text-muted-foreground">{$_('common.loading')}</div>
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

        <!-- Due Dates, Shared, Trash as virtual folders at bottom of tree -->
        <div class="mt-2 pt-2 border-t border-sidebar-border mx-1 space-y-0.5">
          <button
            onclick={() => goto('/due-dates')}
            class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-sidebar-accent text-sidebar-foreground text-sm transition-colors"
            title={$_('page.due_dates.title')}
            aria-label={$_('page.due_dates.title')}
          >
            <CalendarClock size={16} class="text-muted-foreground flex-shrink-0" />
            <span class="flex-1 truncate">{$_('page.due_dates.title')}</span>
          </button>
          <button
            onclick={() => goto('/shared')}
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
            onclick={() => goto('/trash')}
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
      </div>

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

      <!-- Footer: User info + Controls -->
      <div class="border-t border-sidebar-border shrink-0 pb-safe">
        <!-- User info - compact -->
        {#if auth.getCurrentUser()}
          <div class="px-4 py-1.5 text-xs text-muted-foreground truncate">
            {auth.getCurrentUser()?.email}{#if appVersion}<button
                onclick={() => (showChangelogDialog = true)}
                class="opacity-60 hover:opacity-100 hover:text-primary transition-all cursor-pointer"
                title="Changelog"
              >
                · {appVersion}</button
              >{/if}
          </div>
        {/if}

        <!-- Controls - compact row -->
        <div class="px-2 py-2 flex items-center gap-0.5">
          <ThemeSelector />
          <button
            onclick={() => goto('/settings')}
            class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground"
            title={$_('page.sidebar.settings')}
            aria-label={$_('page.sidebar.settings')}
          >
            <Settings size={smallIconSize} />
          </button>
          {#if auth.isAdmin()}
            <button
              onclick={() => goto('/admin')}
              class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground"
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
              class="p-2 rounded-lg hover:bg-sidebar-accent/50 text-sidebar-foreground"
              title={$_('feedback.sidebar_button')}
              aria-label={$_('feedback.sidebar_button')}
            >
              <MessageSquareWarning size={smallIconSize} />
            </button>
          {/if}
          <div class="flex-1"></div>
          <button
            onclick={handleLogout}
            class="p-2 rounded-lg hover:bg-sidebar-accent/50 hover:text-red-500 text-sidebar-foreground"
            title={$_('common.logout')}
            aria-label={$_('common.logout')}
          >
            <LogOut size={smallIconSize} />
          </button>
        </div>
      </div>

      <!-- Resize handle -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="resize-handle"
        class:active={isResizing}
        onpointerdown={handleResizeStart}
        onpointermove={handleResizeMove}
        onpointerup={handleResizeEnd}
        onpointercancel={handleResizeEnd}
        ondblclick={handleResizeDblClick}
      ></div>
    {:else}
      <!-- Collapsed sidebar -->
      <div class="flex flex-col items-center h-full py-3 gap-1.5">
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
        <button
          onclick={handleLogout}
          class="p-2 rounded hover:bg-sidebar-accent/50 hover:text-red-500 text-sidebar-foreground"
          title={$_('common.logout')}
          aria-label={$_('common.logout')}
        >
          <LogOut size={smallIconSize} />
        </button>
      </div>
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

<!-- Changelog Dialog -->
{#if showChangelogDialog}
  <ChangelogDialog open={true} version={appVersion} onClose={() => (showChangelogDialog = false)} />
{/if}

<style>
  .drop-zone {
    margin: 4px 8px;
    padding: 0;
    border: 1px dashed transparent;
    border-radius: 4px;
    text-align: center;
    transition: all 0.15s ease;
    opacity: 0;
    height: 0;
    overflow: hidden;
  }

  .drop-zone.active {
    opacity: 1;
    height: auto;
    padding: 8px 12px;
    border-color: var(--text-accent, #3b82f6);
    background: var(--bg-accent-subtle, rgba(59, 130, 246, 0.05));
  }

  .drop-zone-hint {
    font-size: 0.75rem;
    color: var(--text-muted, #6b7280);
    pointer-events: none;
    font-weight: 500;
  }

  .drop-zone.active .drop-zone-hint {
    color: var(--text-accent, #3b82f6);
  }

  /* Touch devices: show drop zone only during active touch drag */
  .drop-zone.touch-drag-visible {
    opacity: 1;
    height: auto;
    min-height: 48px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    justify-content: center;
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
    transition: background-color 0.15s ease;
  }

  .resize-handle:hover,
  .resize-handle.active {
    background-color: var(--color-primary, oklch(0.65 0.15 155));
  }
</style>
