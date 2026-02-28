/**
 * Tab Store for Multi-Note Editing
 *
 * Manages tabs and tab groups for multi-note editing.
 * Uses Svelte 5 runes for reactive state.
 * Server-persisted via open_tabs in user_preferences.
 */

import { updateOpenTabsPreference } from '$lib/api/preferences';
import type { OpenTabsPreference } from '$lib/api/types';

export interface Tab {
  id: string;
  noteId: string;
  title: string;
  isDirty: boolean;
}

export interface TabGroup {
  id: string;
  tabs: Tab[];
  activeTabId: string | null;
  width?: number; // For split pane resizing
}

interface TabState {
  groups: TabGroup[];
  activeGroupId: string;
}

// Generate unique IDs
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// Initial state with one empty group
const initialGroup: TabGroup = {
  id: 'main',
  tabs: [],
  activeTabId: null,
};

// State
const state = $state<TabState>({
  groups: [initialGroup],
  activeGroupId: 'main',
});

// Server-sync state
let initialized = $state(false);
let isHydrating = false;
let lastPersistedJSON = '';
let persistTimer: ReturnType<typeof setTimeout> | null = null;

// Getters
export function getState(): TabState {
  return state;
}

export function getActiveGroup(): TabGroup | undefined {
  return state.groups.find((g) => g.id === state.activeGroupId);
}

export function getActiveTab(): Tab | undefined {
  const group = getActiveGroup();
  if (!group || !group.activeTabId) return undefined;
  return group.tabs.find((t) => t.id === group.activeTabId);
}

export function getActiveNoteId(): string | undefined {
  return getActiveTab()?.noteId;
}

export function getTabs(): Tab[] {
  return getActiveGroup()?.tabs ?? [];
}

export function getGroups(): TabGroup[] {
  return state.groups;
}

export function isTabDirty(tabId: string): boolean {
  for (const group of state.groups) {
    const tab = group.tabs.find((t) => t.id === tabId);
    if (tab) return tab.isDirty;
  }
  return false;
}

export function findTabByNoteId(noteId: string): { tab: Tab; group: TabGroup } | undefined {
  for (const group of state.groups) {
    const tab = group.tabs.find((t) => t.noteId === noteId);
    if (tab) return { tab, group };
  }
  return undefined;
}

export function isInitialized(): boolean {
  return initialized;
}

// Actions
export function openTab(noteId: string, title: string, groupId?: string): string {
  const targetGroupId = groupId ?? state.activeGroupId;
  const group = state.groups.find((g) => g.id === targetGroupId);

  if (!group) {
    console.error(`Group ${targetGroupId} not found`);
    return '';
  }

  // Check if tab already exists in this group
  const existingTab = group.tabs.find((t) => t.noteId === noteId);
  if (existingTab) {
    // Activate existing tab
    group.activeTabId = existingTab.id;
    state.activeGroupId = targetGroupId;
    return existingTab.id;
  }

  // Create new tab
  const newTab: Tab = {
    id: generateId(),
    noteId,
    title,
    isDirty: false,
  };

  group.tabs = [...group.tabs, newTab];
  group.activeTabId = newTab.id;
  state.activeGroupId = targetGroupId;

  return newTab.id;
}

export function closeTab(tabId: string, groupId?: string): void {
  const targetGroupId = groupId ?? state.activeGroupId;
  const group = state.groups.find((g) => g.id === targetGroupId);

  if (!group) return;

  const tabIndex = group.tabs.findIndex((t) => t.id === tabId);
  if (tabIndex === -1) return;

  const wasActive = group.activeTabId === tabId;

  // Remove tab — assign new array to ensure $derived reactivity
  group.tabs = group.tabs.filter((t) => t.id !== tabId);

  // Update active tab
  if (wasActive) {
    if (group.tabs.length > 0) {
      // Activate adjacent tab (prefer right, then left)
      const newIndex = Math.min(tabIndex, group.tabs.length - 1);
      group.activeTabId = group.tabs[newIndex].id;
    } else {
      group.activeTabId = null;
    }
  }

  // Remove empty groups (except the main one)
  if (group.tabs.length === 0 && group.id !== 'main' && state.groups.length > 1) {
    state.groups = state.groups.filter((g) => g.id !== group.id);

    // Switch to another group
    if (state.activeGroupId === targetGroupId) {
      state.activeGroupId = state.groups[0].id;
    }
  }
}

export function activateTab(tabId: string, groupId?: string): void {
  const targetGroupId = groupId ?? state.activeGroupId;
  const group = state.groups.find((g) => g.id === targetGroupId);

  if (!group) return;

  const tab = group.tabs.find((t) => t.id === tabId);
  if (!tab) return;

  group.activeTabId = tabId;
  state.activeGroupId = targetGroupId;
}

export function setTabDirty(tabId: string, isDirty: boolean): void {
  for (const group of state.groups) {
    const tab = group.tabs.find((t) => t.id === tabId);
    if (tab) {
      tab.isDirty = isDirty;
      return;
    }
  }
}

export function updateTabTitle(tabId: string, title: string): void {
  for (const group of state.groups) {
    const tab = group.tabs.find((t) => t.id === tabId);
    if (tab) {
      tab.title = title;
      return;
    }
  }
}

export function moveTab(
  tabId: string,
  fromGroupId: string,
  toGroupId: string,
  toIndex?: number
): void {
  const fromGroup = state.groups.find((g) => g.id === fromGroupId);
  const toGroup = state.groups.find((g) => g.id === toGroupId);

  if (!fromGroup || !toGroup) return;

  const tabIndex = fromGroup.tabs.findIndex((t) => t.id === tabId);
  if (tabIndex === -1) return;

  const tab = fromGroup.tabs[tabIndex];

  // Remove from source — assign new array
  fromGroup.tabs = fromGroup.tabs.filter((t) => t.id !== tabId);

  // Add to target — assign new array
  if (toIndex !== undefined && toIndex >= 0 && toIndex <= toGroup.tabs.length) {
    const newTabs = [...toGroup.tabs];
    newTabs.splice(toIndex, 0, tab);
    toGroup.tabs = newTabs;
  } else {
    toGroup.tabs = [...toGroup.tabs, tab];
  }

  // Update active tab in target group
  toGroup.activeTabId = tab.id;

  // Clean up source group if empty
  if (fromGroup.tabs.length === 0 && fromGroup.id !== 'main' && state.groups.length > 1) {
    state.groups = state.groups.filter((g) => g.id !== fromGroup.id);
  } else if (fromGroup.activeTabId === tabId) {
    // Select another tab in source group
    fromGroup.activeTabId = fromGroup.tabs.length > 0 ? fromGroup.tabs[0].id : null;
  }

  state.activeGroupId = toGroupId;
}

export function reorderTabs(groupId: string, fromIndex: number, toIndex: number): void {
  const group = state.groups.find((g) => g.id === groupId);
  if (!group) return;

  if (fromIndex < 0 || fromIndex >= group.tabs.length) return;
  if (toIndex < 0 || toIndex >= group.tabs.length) return;

  const newTabs = [...group.tabs];
  const [tab] = newTabs.splice(fromIndex, 1);
  newTabs.splice(toIndex, 0, tab);
  group.tabs = newTabs;
}

// Split pane operations
export function createSplitGroup(): string {
  const newGroup: TabGroup = {
    id: generateId(),
    tabs: [],
    activeTabId: null,
  };

  state.groups = [...state.groups, newGroup];
  return newGroup.id;
}

export function activateGroup(groupId: string): void {
  if (state.groups.some((g) => g.id === groupId)) {
    state.activeGroupId = groupId;
  }
}

export function setGroupWidth(groupId: string, width: number): void {
  const group = state.groups.find((g) => g.id === groupId);
  if (group) {
    group.width = width;
  }
}

// Close all tabs (e.g., on logout)
export function closeAllTabs(): void {
  state.groups = [{ ...initialGroup, tabs: [], activeTabId: null }];
  state.activeGroupId = 'main';
}

// Check if any tabs have unsaved changes
export function hasUnsavedChanges(): boolean {
  return state.groups.some((g) => g.tabs.some((t) => t.isDirty));
}

// Get all dirty tabs
export function getDirtyTabs(): Tab[] {
  return state.groups.flatMap((g) => g.tabs.filter((t) => t.isDirty));
}

// ── Server Sync ──────────────────────────────────────────────────────

/**
 * Initialize tabs from server-persisted open_tabs preference.
 * Creates Tab objects with empty titles (resolved later by resolveTabTitles).
 */
export function initTabs(pref: OpenTabsPreference | null | undefined): void {
  isHydrating = true;

  const mainGroup = state.groups.find((g) => g.id === 'main');
  if (!mainGroup) return;

  if (!pref || !pref.tabs || pref.tabs.length === 0) {
    mainGroup.tabs = [];
    mainGroup.activeTabId = null;
    isHydrating = false;
    return;
  }

  mainGroup.tabs = pref.tabs.map((entry) => ({
    id: generateId(),
    noteId: entry.note_id,
    title: '',
    isDirty: false,
  }));

  // Set active tab
  if (pref.active_note_id) {
    const activeTab = mainGroup.tabs.find((t) => t.noteId === pref.active_note_id);
    mainGroup.activeTabId = activeTab?.id ?? mainGroup.tabs[0]?.id ?? null;
  } else {
    mainGroup.activeTabId = mainGroup.tabs[0]?.id ?? null;
  }

  // Cache initial state to avoid writing it back
  lastPersistedJSON = buildPersistedJSON();
}

/**
 * Resolve tab titles from the notes list. Removes tabs for deleted notes.
 * Called after notes.loadNotes() completes.
 */
export function resolveTabTitles(getNoteById: (id: string) => { title: string } | undefined): void {
  const mainGroup = state.groups.find((g) => g.id === 'main');
  if (!mainGroup) return;

  // Filter out tabs whose notes no longer exist and resolve titles
  const validTabs: Tab[] = [];
  for (const tab of mainGroup.tabs) {
    const note = getNoteById(tab.noteId);
    if (note) {
      tab.title = note.title || '';
      validTabs.push(tab);
    }
  }

  const removedCount = mainGroup.tabs.length - validTabs.length;
  if (removedCount > 0) {
    mainGroup.tabs = validTabs;
  }

  // Fix active tab if it was removed
  if (mainGroup.activeTabId) {
    const stillExists = mainGroup.tabs.some((t) => t.id === mainGroup.activeTabId);
    if (!stillExists) {
      mainGroup.activeTabId = mainGroup.tabs[0]?.id ?? null;
    }
  }

  // Update persisted JSON baseline
  lastPersistedJSON = buildPersistedJSON();

  isHydrating = false;
  initialized = true;
}

/**
 * Sync tab state with the current route.
 * Opens or activates the tab for the given noteId.
 */
export function syncTabWithRoute(noteId: string, title: string): void {
  if (!initialized) return;

  const existing = findTabByNoteId(noteId);
  if (existing) {
    existing.tab.title = title || existing.tab.title;
    activateTab(existing.tab.id, existing.group.id);
  } else {
    openTab(noteId, title);
  }

  persistTabs();
}

/**
 * Sync dirty state from the note store to the tab.
 */
export function syncDirtyState(noteId: string, isDirty: boolean): void {
  const existing = findTabByNoteId(noteId);
  if (existing) {
    existing.tab.isDirty = isDirty;
  }
}

/**
 * Close a tab with optional save and return the next note ID to navigate to.
 */
export async function closeTabAndNavigate(
  tabId: string,
  saveFn?: () => Promise<void>
): Promise<{ nextNoteId: string | null }> {
  const mainGroup = state.groups.find((g) => g.id === 'main');
  if (!mainGroup) return { nextNoteId: null };

  const tab = mainGroup.tabs.find((t) => t.id === tabId);
  if (!tab) return { nextNoteId: null };

  // If dirty and saveFn provided, try to save first
  if (tab.isDirty && saveFn) {
    try {
      await Promise.race([
        saveFn(),
        new Promise((_, reject) => setTimeout(() => reject(new Error('save timeout')), 5000)),
      ]);
    } catch {
      // Save failed or timed out — close anyway (auto-save retry will handle it)
    }
  }

  // Find what the next active tab will be before closing
  const tabIndex = mainGroup.tabs.findIndex((t) => t.id === tabId);
  let nextNoteId: string | null = null;

  if (mainGroup.tabs.length > 1) {
    const nextIndex = tabIndex < mainGroup.tabs.length - 1 ? tabIndex + 1 : tabIndex - 1;
    nextNoteId = mainGroup.tabs[nextIndex].noteId;
  }

  closeTab(tabId);
  persistTabs();

  return { nextNoteId };
}

/**
 * Navigate to the next tab (cyclic).
 * Returns the noteId to navigate to.
 */
export function nextTab(): string | null {
  const tabs = getTabs();
  if (tabs.length <= 1) return null;

  const activeTab = getActiveTab();
  if (!activeTab) return tabs[0]?.noteId ?? null;

  const currentIndex = tabs.findIndex((t) => t.id === activeTab.id);
  const nextIndex = (currentIndex + 1) % tabs.length;
  activateTab(tabs[nextIndex].id);
  persistTabs();
  return tabs[nextIndex].noteId;
}

/**
 * Navigate to the previous tab (cyclic).
 * Returns the noteId to navigate to.
 */
export function prevTab(): string | null {
  const tabs = getTabs();
  if (tabs.length <= 1) return null;

  const activeTab = getActiveTab();
  if (!activeTab) return tabs[0]?.noteId ?? null;

  const currentIndex = tabs.findIndex((t) => t.id === activeTab.id);
  const prevIndex = (currentIndex - 1 + tabs.length) % tabs.length;
  activateTab(tabs[prevIndex].id);
  persistTabs();
  return tabs[prevIndex].noteId;
}

/**
 * Replace a temporary note ID with a real one (offline create -> sync).
 */
export function replaceTempId(tempId: string, realId: string): void {
  const existing = findTabByNoteId(tempId);
  if (existing) {
    existing.tab.noteId = realId;
    persistTabs();
  }
}

/**
 * Remove a tab by noteId (e.g., when a note is deleted).
 * Returns the next noteId to navigate to, or null.
 */
export function removeTabByNoteId(noteId: string): string | null {
  const existing = findTabByNoteId(noteId);
  if (!existing) return null;

  const mainGroup = state.groups.find((g) => g.id === 'main');
  if (!mainGroup) return null;

  const tabIndex = mainGroup.tabs.findIndex((t) => t.id === existing.tab.id);
  let nextNoteId: string | null = null;

  if (mainGroup.tabs.length > 1) {
    const nextIndex = tabIndex < mainGroup.tabs.length - 1 ? tabIndex + 1 : tabIndex - 1;
    nextNoteId = mainGroup.tabs[nextIndex].noteId;
  }

  closeTab(existing.tab.id, existing.group.id);
  persistTabs();

  return nextNoteId;
}

// ── Persistence ──────────────────────────────────────────────────────

function buildPersistedJSON(): string {
  const mainGroup = state.groups.find((g) => g.id === 'main');
  if (!mainGroup || mainGroup.tabs.length === 0) return '';

  const activeTab = mainGroup.activeTabId
    ? mainGroup.tabs.find((t) => t.id === mainGroup.activeTabId)
    : null;

  const payload: OpenTabsPreference = {
    version: 1,
    tabs: mainGroup.tabs.map((t) => ({ note_id: t.noteId })),
    active_note_id: activeTab?.noteId ?? null,
  };

  return JSON.stringify(payload);
}

/**
 * Persist tabs to server (debounced 2s).
 * No-op during hydration. Skips if state hasn't changed.
 */
export function persistTabs(): void {
  if (isHydrating) return;

  if (persistTimer) {
    clearTimeout(persistTimer);
  }

  persistTimer = setTimeout(() => {
    persistTimer = null;

    const json = buildPersistedJSON();
    if (json === lastPersistedJSON) return;

    lastPersistedJSON = json;

    const payload: OpenTabsPreference | null = json ? JSON.parse(json) : null;
    updateOpenTabsPreference(payload).catch((err) => {
      console.warn('[tabs] Failed to persist tabs:', err);
    });
  }, 2000);
}

/**
 * Reorder tabs and trigger persist.
 */
export function reorderTabsAndPersist(groupId: string, fromIndex: number, toIndex: number): void {
  reorderTabs(groupId, fromIndex, toIndex);
  persistTabs();
}
