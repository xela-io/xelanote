/**
 * Tab Store for Desktop App
 *
 * Manages tabs and tab groups for multi-note editing.
 * Uses Svelte 5 runes for reactive state.
 */

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

  group.tabs.push(newTab);
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

  // Remove tab
  group.tabs.splice(tabIndex, 1);

  // Update active tab
  if (group.activeTabId === tabId) {
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
    const groupIndex = state.groups.findIndex((g) => g.id === group.id);
    state.groups.splice(groupIndex, 1);

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

  // Remove from source
  const [tab] = fromGroup.tabs.splice(tabIndex, 1);

  // Add to target
  if (toIndex !== undefined && toIndex >= 0 && toIndex <= toGroup.tabs.length) {
    toGroup.tabs.splice(toIndex, 0, tab);
  } else {
    toGroup.tabs.push(tab);
  }

  // Update active tab in target group
  toGroup.activeTabId = tab.id;

  // Clean up source group if empty
  if (fromGroup.tabs.length === 0 && fromGroup.id !== 'main' && state.groups.length > 1) {
    const groupIndex = state.groups.findIndex((g) => g.id === fromGroup.id);
    state.groups.splice(groupIndex, 1);
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

  const [tab] = group.tabs.splice(fromIndex, 1);
  group.tabs.splice(toIndex, 0, tab);
}

// Split pane operations
export function createSplitGroup(): string {
  const newGroup: TabGroup = {
    id: generateId(),
    tabs: [],
    activeTabId: null,
  };

  state.groups.push(newGroup);
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
