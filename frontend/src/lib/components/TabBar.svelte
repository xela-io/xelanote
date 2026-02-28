<script lang="ts">
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as autosave from '$lib/stores/autosave.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as tabs from '$lib/stores/tabs.svelte';

  let dragTabId: string | null = $state(null);
  let dragOverIndex: number | null = $state(null);

  const tabList = $derived(tabs.getTabs());
  const activeTab = $derived(tabs.getActiveTab());

  function handleTabClick(noteId: string) {
    goto(`/note/${noteId}`);
  }

  function handleMiddleClick(e: MouseEvent, tabId: string) {
    if (e.button === 1) {
      e.preventDefault();
      handleCloseTab(tabId);
    }
  }

  async function handleCloseTab(tabId: string) {
    // Only navigate if closing the currently active tab
    const isActive = activeTab?.id === tabId;
    if (!isActive) {
      // Non-active tab: just remove it, stay on current note
      tabs.closeTab(tabId);
      tabs.persistTabs();
      return;
    }

    const saveFn = autosave.getAutoSaveEnabled()
      ? async () => {
          await notes.saveNote();
        }
      : undefined;
    const { nextNoteId } = await tabs.closeTabAndNavigate(tabId, saveFn);
    if (nextNoteId) {
      goto(`/note/${nextNoteId}`);
    } else {
      goto('/');
    }
  }

  function resolveTitle(tab: tabs.Tab): string {
    if (tab.title) return tab.title;
    const note = notes.getNoteById(tab.noteId);
    if (note?.title) return note.title;
    return $_('component.tabs.untitled');
  }

  // Drag & drop reordering
  function handleDragStart(e: DragEvent, tabId: string) {
    dragTabId = tabId;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', tabId);
    }
  }

  function handleDragOver(e: DragEvent, index: number) {
    e.preventDefault();
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'move';
    }
    dragOverIndex = index;
  }

  function handleDragLeave() {
    dragOverIndex = null;
  }

  function handleDrop(e: DragEvent, toIndex: number) {
    e.preventDefault();
    dragOverIndex = null;

    if (!dragTabId) return;
    const mainGroup = tabs.getActiveGroup();
    if (!mainGroup) return;

    const fromIndex = mainGroup.tabs.findIndex((t) => t.id === dragTabId);
    if (fromIndex === -1 || fromIndex === toIndex) return;

    tabs.reorderTabsAndPersist(mainGroup.id, fromIndex, toIndex);
    dragTabId = null;
  }

  function handleDragEnd() {
    dragTabId = null;
    dragOverIndex = null;
  }
</script>

<div class="tab-bar" role="tablist" aria-label={$_('component.tabs.label')}>
  {#each tabList as tab, index (tab.id)}
    <div
      role="tab"
      class="tab"
      class:active={activeTab?.id === tab.id}
      class:dragging={dragTabId === tab.id}
      class:drag-over={dragOverIndex === index && dragTabId !== tab.id}
      aria-selected={activeTab?.id === tab.id}
      title={resolveTitle(tab)}
      onclick={() => handleTabClick(tab.noteId)}
      onmousedown={(e) => handleMiddleClick(e, tab.id)}
      onkeydown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') handleTabClick(tab.noteId);
      }}
      tabindex="0"
      draggable="true"
      ondragstart={(e) => handleDragStart(e, tab.id)}
      ondragover={(e) => handleDragOver(e, index)}
      ondragleave={handleDragLeave}
      ondrop={(e) => handleDrop(e, index)}
      ondragend={handleDragEnd}
    >
      {#if tab.isDirty}
        <span class="dirty-dot" title={$_('component.tabs.unsaved')}></span>
      {/if}
      <span class="tab-title">{resolveTitle(tab)}</span>
      <button
        class="close-btn"
        aria-label={$_('component.tabs.close')}
        onclick={(e) => {
          e.stopPropagation();
          handleCloseTab(tab.id);
        }}
        onmousedown={(e) => e.stopPropagation()}
        ondragstart={(e) => e.preventDefault()}
        draggable="false"
        tabindex="-1"
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path
            d="M4 4L10 10M10 4L4 10"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          ></path>
        </svg>
      </button>
    </div>
  {/each}
</div>

<style>
  .tab-bar {
    display: flex;
    align-items: stretch;
    overflow-x: auto;
    overflow-y: hidden;
    background: var(--color-sidebar-background, var(--color-background));
    border-bottom: 1px solid var(--color-border);
    min-height: 36px;
    scrollbar-width: thin;
    -webkit-overflow-scrolling: touch;
  }

  .tab-bar::-webkit-scrollbar {
    height: 3px;
  }

  .tab-bar::-webkit-scrollbar-thumb {
    background: var(--color-border);
    border-radius: 2px;
  }

  .tab {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 8px;
    min-width: 0;
    max-width: 200px;
    min-height: 36px;
    border: none;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--color-muted-foreground);
    font-size: 13px;
    line-height: 1;
    cursor: pointer;
    white-space: nowrap;
    position: relative;
    flex-shrink: 0;
    transition:
      background 0.1s,
      border-color 0.1s,
      color 0.1s;
    /* Mobile touch target */
    touch-action: manipulation;
  }

  @media (max-width: 768px) {
    .tab {
      min-height: 44px;
      padding: 0 12px;
    }
  }

  .tab:hover {
    background: color-mix(in oklch, var(--color-foreground), transparent 92%);
    color: var(--color-foreground);
  }

  .tab.active {
    background: var(--color-background);
    color: var(--color-foreground);
    border-bottom-color: var(--color-primary);
  }

  .tab.dragging {
    opacity: 0.5;
  }

  .tab.drag-over {
    border-left: 2px solid var(--color-primary);
  }

  .tab-title {
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 160px;
  }

  .dirty-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--color-primary);
    flex-shrink: 0;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--color-muted-foreground);
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
    opacity: 0;
    transition:
      opacity 0.1s,
      background 0.1s;
  }

  .tab:hover .close-btn,
  .tab.active .close-btn {
    opacity: 1;
  }

  .close-btn:hover {
    background: color-mix(in oklch, var(--color-foreground), transparent 85%);
    color: var(--color-foreground);
  }
</style>
