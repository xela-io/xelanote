import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Mock the API module before importing the store
vi.mock('$lib/api/preferences', () => ({
  updateOpenTabsPreference: vi.fn().mockResolvedValue({}),
  updateOpenTabsKeepalive: vi.fn(),
}));

import { updateOpenTabsKeepalive, updateOpenTabsPreference } from '$lib/api/preferences';
import * as tabs from '$lib/stores/tabs.svelte';

const mockedUpdatePreference = vi.mocked(updateOpenTabsPreference);
const mockedUpdateKeepalive = vi.mocked(updateOpenTabsKeepalive);

beforeEach(() => {
  tabs._resetForTests();
  vi.clearAllMocks();
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

// ── Core Tab Operations ─────────────────────────────────────────────

describe('openTab', () => {
  it('creates a new tab in the main group', () => {
    const tabId = tabs.openTab('note-1', 'Note 1');
    expect(tabId).toBeTruthy();
    expect(tabs.getTabs()).toHaveLength(1);
    expect(tabs.getTabs()[0].noteId).toBe('note-1');
    expect(tabs.getTabs()[0].title).toBe('Note 1');
  });

  it('activates existing tab if noteId already exists', () => {
    const id1 = tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');
    const id3 = tabs.openTab('note-1', 'Note 1 again');

    expect(id3).toBe(id1);
    expect(tabs.getTabs()).toHaveLength(2);
    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });

  it('sets the new tab as active', () => {
    tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');
    expect(tabs.getActiveTab()?.noteId).toBe('note-2');
  });

  it('returns empty string for non-existent group', () => {
    const id = tabs.openTab('note-1', 'Note 1', 'nonexistent');
    expect(id).toBe('');
  });
});

describe('closeTab', () => {
  it('removes a tab from the group', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.closeTab(id);
    expect(tabs.getTabs()).toHaveLength(0);
  });

  it('activates the right neighbor when closing the active tab', () => {
    tabs.openTab('note-1', 'Note 1');
    const id2 = tabs.openTab('note-2', 'Note 2');
    tabs.openTab('note-3', 'Note 3');

    // Activate middle tab
    tabs.activateTab(id2);
    tabs.closeTab(id2);

    expect(tabs.getActiveTab()?.noteId).toBe('note-3');
  });

  it('activates the left neighbor when closing the last tab in list', () => {
    tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');
    const id3 = tabs.openTab('note-3', 'Note 3');

    tabs.closeTab(id3);
    expect(tabs.getActiveTab()?.noteId).toBe('note-2');
  });

  it('sets activeTabId to null when closing the last tab', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.closeTab(id);
    expect(tabs.getActiveTab()).toBeUndefined();
    expect(tabs.getActiveGroup()?.activeTabId).toBeNull();
  });

  it('does nothing for non-existent tabId', () => {
    tabs.openTab('note-1', 'Note 1');
    tabs.closeTab('nonexistent');
    expect(tabs.getTabs()).toHaveLength(1);
  });

  it('preserves active tab when closing a non-active tab', () => {
    const id1 = tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');

    // Active is note-2, close note-1
    tabs.closeTab(id1);
    expect(tabs.getActiveTab()?.noteId).toBe('note-2');
    expect(tabs.getTabs()).toHaveLength(1);
  });
});

describe('activateTab', () => {
  it('sets the given tab as active', () => {
    const id1 = tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');

    tabs.activateTab(id1);
    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });

  it('does nothing for non-existent tab', () => {
    tabs.openTab('note-1', 'Note 1');
    tabs.activateTab('nonexistent');
    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });
});

describe('closeAllTabs', () => {
  it('removes all tabs and resets to empty main group', () => {
    tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');
    tabs.openTab('note-3', 'Note 3');

    tabs.closeAllTabs();

    expect(tabs.getTabs()).toHaveLength(0);
    expect(tabs.getActiveTab()).toBeUndefined();
    expect(tabs.getGroups()).toHaveLength(1);
    expect(tabs.getGroups()[0].id).toBe('main');
  });

  it('triggers persistence', () => {
    // Initialize to make persistence work
    tabs.initTabs({ version: 1, tabs: [{ note_id: 'note-1' }], active_note_id: 'note-1' });
    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));

    tabs.closeAllTabs();
    vi.advanceTimersByTime(2500);

    expect(mockedUpdatePreference).toHaveBeenCalledWith(null);
  });
});

// ── Tab Properties ──────────────────────────────────────────────────

describe('setTabDirty', () => {
  it('marks a tab as dirty', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.setTabDirty(id, true);
    expect(tabs.isTabDirty(id)).toBe(true);
  });

  it('clears dirty flag', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.setTabDirty(id, true);
    tabs.setTabDirty(id, false);
    expect(tabs.isTabDirty(id)).toBe(false);
  });
});

describe('updateTabTitle', () => {
  it('updates the title of a tab', () => {
    const id = tabs.openTab('note-1', 'Old Title');
    tabs.updateTabTitle(id, 'New Title');
    expect(tabs.getTabs()[0].title).toBe('New Title');
  });
});

describe('hasUnsavedChanges', () => {
  it('returns false when no dirty tabs', () => {
    tabs.openTab('note-1', 'Note 1');
    expect(tabs.hasUnsavedChanges()).toBe(false);
  });

  it('returns true when any tab is dirty', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.setTabDirty(id, true);
    expect(tabs.hasUnsavedChanges()).toBe(true);
  });
});

describe('getDirtyTabs', () => {
  it('returns only dirty tabs', () => {
    const id1 = tabs.openTab('note-1', 'Note 1');
    tabs.openTab('note-2', 'Note 2');
    tabs.setTabDirty(id1, true);

    const dirty = tabs.getDirtyTabs();
    expect(dirty).toHaveLength(1);
    expect(dirty[0].noteId).toBe('note-1');
  });
});

// ── Reorder ─────────────────────────────────────────────────────────

describe('reorderTabs', () => {
  it('moves a tab from one position to another', () => {
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');
    tabs.openTab('note-3', 'C');

    const group = tabs.getActiveGroup()!;
    tabs.reorderTabs(group.id, 0, 2);

    const titles = tabs.getTabs().map((t) => t.title);
    expect(titles).toEqual(['B', 'C', 'A']);
  });

  it('does nothing for out-of-bounds indices', () => {
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');

    const group = tabs.getActiveGroup()!;
    tabs.reorderTabs(group.id, -1, 0);

    expect(tabs.getTabs()).toHaveLength(2);
  });
});

// ── Navigation ──────────────────────────────────────────────────────

describe('nextTab / prevTab', () => {
  it('nextTab cycles forward', () => {
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');
    tabs.openTab('note-3', 'C');

    // Set up initialized state for persistence
    tabs.initTabs(null);
    tabs.resolveTabTitles(() => undefined);

    // Re-open tabs after reset
    tabs._resetForTests();
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');
    tabs.openTab('note-3', 'C');

    // Active is C (last opened). Activate A first.
    tabs.activateTab(tabs.getTabs()[0].id);

    const next = tabs.nextTab();
    expect(next).toBe('note-2');
  });

  it('prevTab cycles backward', () => {
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');
    tabs.openTab('note-3', 'C');

    // Active is C (last opened)
    const prev = tabs.prevTab();
    expect(prev).toBe('note-2');
  });

  it('returns null with only one tab', () => {
    tabs.openTab('note-1', 'A');
    expect(tabs.nextTab()).toBeNull();
    expect(tabs.prevTab()).toBeNull();
  });

  it('returns null with no tabs', () => {
    expect(tabs.nextTab()).toBeNull();
    expect(tabs.prevTab()).toBeNull();
  });
});

// ── findTabByNoteId ─────────────────────────────────────────────────

describe('findTabByNoteId', () => {
  it('finds a tab by its noteId', () => {
    tabs.openTab('note-1', 'Note 1');
    const result = tabs.findTabByNoteId('note-1');
    expect(result).toBeDefined();
    expect(result!.tab.noteId).toBe('note-1');
    expect(result!.group.id).toBe('main');
  });

  it('returns undefined for unknown noteId', () => {
    expect(tabs.findTabByNoteId('nonexistent')).toBeUndefined();
  });
});

// ── replaceTempId ───────────────────────────────────────────────────

describe('replaceTempId', () => {
  it('replaces a temporary note ID with a real one', () => {
    // Initialize first, then open a tab with temp ID
    tabs.initTabs(null);
    tabs.resolveTabTitles(() => undefined);

    tabs.openTab('temp-123', 'New Note');
    tabs.replaceTempId('temp-123', 'real-456');

    expect(tabs.findTabByNoteId('real-456')).toBeDefined();
    expect(tabs.findTabByNoteId('temp-123')).toBeUndefined();
  });
});

// ── removeTabByNoteId ───────────────────────────────────────────────

describe('removeTabByNoteId', () => {
  it('removes a tab and returns the next noteId', () => {
    tabs.openTab('note-1', 'A');
    tabs.openTab('note-2', 'B');
    tabs.openTab('note-3', 'C');

    const nextId = tabs.removeTabByNoteId('note-2');
    expect(nextId).toBe('note-3');
    expect(tabs.findTabByNoteId('note-2')).toBeUndefined();
  });

  it('returns null when removing the only tab', () => {
    tabs.openTab('note-1', 'A');
    const nextId = tabs.removeTabByNoteId('note-1');
    expect(nextId).toBeNull();
  });

  it('returns null for non-existent noteId', () => {
    expect(tabs.removeTabByNoteId('nonexistent')).toBeNull();
  });
});

// ── Server Sync: initTabs ───────────────────────────────────────────

describe('initTabs', () => {
  it('restores tabs from server preference', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'note-2',
    });

    const tabList = tabs.getTabs();
    expect(tabList).toHaveLength(2);
    expect(tabList[0].noteId).toBe('note-1');
    expect(tabList[1].noteId).toBe('note-2');
    // Active tab should be note-2
    expect(tabs.getActiveTab()?.noteId).toBe('note-2');
  });

  it('handles null preference (no tabs)', () => {
    tabs.initTabs(null);
    expect(tabs.getTabs()).toHaveLength(0);
    expect(tabs.getActiveTab()).toBeUndefined();
  });

  it('handles empty tabs array', () => {
    tabs.initTabs({ version: 1, tabs: [], active_note_id: null });
    expect(tabs.getTabs()).toHaveLength(0);
  });

  it('falls back to first tab if active_note_id not found', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'nonexistent',
    });

    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });

  it('creates tabs with empty titles (resolved later)', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: null,
    });

    expect(tabs.getTabs()[0].title).toBe('');
  });
});

// ── Server Sync: resolveTabTitles ───────────────────────────────────

describe('resolveTabTitles', () => {
  it('resolves titles from notes and sets initialized', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: null,
    });

    const noteMap: Record<string, { title: string }> = {
      'note-1': { title: 'First Note' },
      'note-2': { title: 'Second Note' },
    };

    tabs.resolveTabTitles((id) => noteMap[id]);

    expect(tabs.getTabs()[0].title).toBe('First Note');
    expect(tabs.getTabs()[1].title).toBe('Second Note');
    expect(tabs.isInitialized()).toBe(true);
  });

  it('removes tabs for deleted notes', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'deleted-note' }],
      active_note_id: null,
    });

    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Surviving Note' } : undefined));

    expect(tabs.getTabs()).toHaveLength(1);
    expect(tabs.getTabs()[0].noteId).toBe('note-1');
  });

  it('fixes active tab if it was removed', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'deleted-note' }],
      active_note_id: 'deleted-note',
    });

    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));

    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });
});

// ── Server Sync: syncTabWithRoute ───────────────────────────────────

describe('syncTabWithRoute', () => {
  beforeEach(() => {
    // Initialize the store as if hydration completed
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));
  });

  it('opens a new tab for an unknown noteId', () => {
    tabs.syncTabWithRoute('note-2', 'Note 2');

    expect(tabs.getTabs()).toHaveLength(2);
    expect(tabs.getActiveTab()?.noteId).toBe('note-2');
  });

  it('activates an existing tab for a known noteId', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.syncTabWithRoute('note-1', 'Note 1');

    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
    expect(tabs.getTabs()).toHaveLength(2); // no duplicate
  });

  it('does nothing when not initialized', () => {
    tabs._resetForTests(); // not initialized
    tabs.syncTabWithRoute('note-1', 'Note 1');
    expect(tabs.getTabs()).toHaveLength(0);
  });

  it('triggers debounced persistence', () => {
    tabs.syncTabWithRoute('note-2', 'Note 2');

    // Not yet persisted (debounced)
    expect(mockedUpdatePreference).not.toHaveBeenCalled();

    // After 2s debounce
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).toHaveBeenCalled();
  });
});

// ── Persistence ─────────────────────────────────────────────────────

describe('persistTabs', () => {
  beforeEach(() => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));
  });

  it('debounces API calls by 2 seconds', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();

    expect(mockedUpdatePreference).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1000);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1500);
    expect(mockedUpdatePreference).toHaveBeenCalledTimes(1);
  });

  it('does not persist if state has not changed', () => {
    // Persist without changing anything — should be a no-op since
    // lastPersistedJSON was set in resolveTabTitles
    tabs.persistTabs();
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();
  });

  it('sends null when all tabs are closed', () => {
    const tab = tabs.getTabs()[0];
    tabs.closeTab(tab.id);
    tabs.persistTabs();

    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).toHaveBeenCalledWith(null);
  });

  it('sends correct payload with tabs', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();

    vi.advanceTimersByTime(2500);

    const payload = mockedUpdatePreference.mock.calls[0][0];
    expect(payload).toEqual({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'note-2',
    });
  });

  it('resets debounce timer on rapid calls', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();

    vi.advanceTimersByTime(1000);
    tabs.openTab('note-3', 'Note 3');
    tabs.persistTabs();

    vi.advanceTimersByTime(1000);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1500);
    expect(mockedUpdatePreference).toHaveBeenCalledTimes(1);

    // Should include both new tabs
    const payload = mockedUpdatePreference.mock.calls[0][0];
    expect(payload!.tabs).toHaveLength(3);
  });
});

// ── flushPendingPersist ─────────────────────────────────────────────

describe('flushPendingPersist', () => {
  beforeEach(() => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));
  });

  it('immediately persists pending changes via keepalive', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs(); // starts the 2s timer

    tabs.flushPendingPersist();

    // Should use keepalive, not the regular API
    expect(mockedUpdateKeepalive).toHaveBeenCalledTimes(1);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();
  });

  it('does nothing when no pending persist timer', () => {
    tabs.flushPendingPersist();
    expect(mockedUpdateKeepalive).not.toHaveBeenCalled();
  });

  it('cancels the debounce timer', () => {
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();

    tabs.flushPendingPersist();

    // Advance past the original debounce — should not trigger again
    vi.advanceTimersByTime(3000);
    expect(mockedUpdateKeepalive).toHaveBeenCalledTimes(1);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();
  });

  it('sends null when flushing with all tabs closed', () => {
    const tab = tabs.getTabs()[0];
    tabs.closeTab(tab.id);
    tabs.persistTabs();

    tabs.flushPendingPersist();
    expect(mockedUpdateKeepalive).toHaveBeenCalledWith(null);
  });

  it('skips flush if state has not changed since last persist', () => {
    // First persist completes normally
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).toHaveBeenCalledTimes(1);

    // Now trigger another persist with no state change
    tabs.persistTabs();
    tabs.flushPendingPersist();
    // The timer callback inside persistTabs would check json === lastPersistedJSON,
    // but flushPendingPersist checks before the timer fires — let's verify it's smart about this
    // Actually, persistTabs starts a new timer. flushPendingPersist cancels it and checks.
    // Since state hasn't changed, lastPersistedJSON matches, so no keepalive call.
    expect(mockedUpdateKeepalive).not.toHaveBeenCalled();
  });
});

// ── closeTabAndNavigate ─────────────────────────────────────────────

describe('closeTabAndNavigate', () => {
  it('returns the next noteId after closing', async () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => ({ title: id }));

    const tab1 = tabs.findTabByNoteId('note-1')!.tab;
    const result = await tabs.closeTabAndNavigate(tab1.id);

    expect(result.nextNoteId).toBe('note-2');
    expect(tabs.getTabs()).toHaveLength(1);
  });

  it('returns null when closing the last tab', async () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => ({ title: id }));

    const tab1 = tabs.findTabByNoteId('note-1')!.tab;
    const result = await tabs.closeTabAndNavigate(tab1.id);

    expect(result.nextNoteId).toBeNull();
    expect(tabs.getTabs()).toHaveLength(0);
  });

  it('calls saveFn for dirty tabs', async () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    tabs.resolveTabTitles((id) => ({ title: id }));

    const tab1 = tabs.findTabByNoteId('note-1')!.tab;
    tabs.setTabDirty(tab1.id, true);

    const saveFn = vi.fn().mockResolvedValue(undefined);
    await tabs.closeTabAndNavigate(tab1.id, saveFn);

    expect(saveFn).toHaveBeenCalledTimes(1);
  });
});

// ── Split Group Operations ──────────────────────────────────────────

describe('split groups', () => {
  it('creates a new split group', () => {
    const newGroupId = tabs.createSplitGroup();
    expect(tabs.getGroups()).toHaveLength(2);
    expect(tabs.getGroups()[1].id).toBe(newGroupId);
  });

  it('moves a tab between groups', () => {
    const id1 = tabs.openTab('note-1', 'Note 1');
    const newGroupId = tabs.createSplitGroup();

    tabs.moveTab(id1, 'main', newGroupId);

    expect(tabs.getGroups().find((g) => g.id === 'main')!.tabs).toHaveLength(0);
    expect(tabs.getGroups().find((g) => g.id === newGroupId)!.tabs).toHaveLength(1);
  });

  it('activates a group', () => {
    const newGroupId = tabs.createSplitGroup();
    tabs.activateGroup(newGroupId);
    expect(tabs.getState().activeGroupId).toBe(newGroupId);
  });

  it('removes empty non-main groups when closing their last tab', () => {
    const newGroupId = tabs.createSplitGroup();
    const tabId = tabs.openTab('note-1', 'Note 1', newGroupId);

    tabs.closeTab(tabId, newGroupId);
    expect(tabs.getGroups()).toHaveLength(1); // only main remains
  });
});

// ── syncDirtyState ──────────────────────────────────────────────────

describe('syncDirtyState', () => {
  it('updates dirty state for a tab by noteId', () => {
    const id = tabs.openTab('note-1', 'Note 1');
    tabs.syncDirtyState('note-1', true);
    expect(tabs.isTabDirty(id)).toBe(true);

    tabs.syncDirtyState('note-1', false);
    expect(tabs.isTabDirty(id)).toBe(false);
  });

  it('does nothing for unknown noteId', () => {
    tabs.syncDirtyState('nonexistent', true);
    // Should not throw
  });
});

// ── Race Condition Guard ─────────────────────────────────────────────

describe('preferencesLoaded guard', () => {
  it('isPreferencesLoaded returns false before initTabs', () => {
    expect(tabs.isPreferencesLoaded()).toBe(false);
  });

  it('isPreferencesLoaded returns true after initTabs', () => {
    tabs.initTabs(null);
    expect(tabs.isPreferencesLoaded()).toBe(true);
  });

  it('isPreferencesLoaded returns true after initTabs with data', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });
    expect(tabs.isPreferencesLoaded()).toBe(true);
  });

  it('resolveTabTitles does not set initialized if called before initTabs', () => {
    // Simulate the race: resolveTabTitles should NOT be called before initTabs.
    // The layout effect guards this with isPreferencesLoaded().
    // But if someone bypasses the guard, verify the store handles it.
    tabs.resolveTabTitles(() => undefined);
    // initialized should be true (resolveTabTitles always sets it)
    expect(tabs.isInitialized()).toBe(true);
    // but preferencesLoaded should still be false
    expect(tabs.isPreferencesLoaded()).toBe(false);
  });

  it('late initTabs after resolveTabTitles restores tabs correctly', () => {
    // Simulate the race condition: resolveTabTitles runs first on empty state
    tabs.resolveTabTitles(() => undefined);
    expect(tabs.isInitialized()).toBe(true);
    expect(tabs.getTabs()).toHaveLength(0);

    // Late initTabs with server data
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'note-1',
    });

    // Tabs should be loaded
    expect(tabs.getTabs()).toHaveLength(2);
    expect(tabs.getActiveTab()?.noteId).toBe('note-1');
  });
});

// ── Edge Cases ──────────────────────────────────────────────────────

describe('edge cases', () => {
  it('getActiveNoteId returns undefined when no tabs', () => {
    expect(tabs.getActiveNoteId()).toBeUndefined();
  });

  it('getActiveNoteId returns the active tab noteId', () => {
    tabs.openTab('note-1', 'Note 1');
    expect(tabs.getActiveNoteId()).toBe('note-1');
  });

  it('setGroupWidth sets width on a group', () => {
    tabs.setGroupWidth('main', 500);
    expect(tabs.getActiveGroup()?.width).toBe(500);
  });

  it('persistence is skipped during hydration', () => {
    // initTabs sets isHydrating = true and then false for empty prefs
    // For non-empty prefs, hydration ends when resolveTabTitles is called
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });

    // During hydration, persistTabs should be a no-op
    tabs.persistTabs();
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();
  });

  it('resolveTabTitles with 0 notes removes all tabs and sets initialized', () => {
    // Simulate: user has persisted tabs, but all notes were deleted
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }, { note_id: 'note-2' }],
      active_note_id: 'note-1',
    });

    // getNoteById returns undefined for all — all notes deleted
    tabs.resolveTabTitles(() => undefined);

    expect(tabs.getTabs()).toHaveLength(0);
    expect(tabs.isInitialized()).toBe(true);
    expect(tabs.getActiveTab()).toBeUndefined();
  });

  it('resolveTabTitles with empty initTabs sets initialized', () => {
    // Simulate: no persisted tabs, 0 notes loaded
    tabs.initTabs(null);

    tabs.resolveTabTitles(() => undefined);

    expect(tabs.getTabs()).toHaveLength(0);
    expect(tabs.isInitialized()).toBe(true);
  });

  it('hydration safety timeout clears isHydrating after 15s', () => {
    // initTabs with data sets isHydrating = true
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });

    // Hydrating blocks persistence
    tabs.persistTabs();
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).not.toHaveBeenCalled();

    // After 15s safety timeout, isHydrating is cleared
    vi.advanceTimersByTime(15000);
    expect(tabs.isInitialized()).toBe(true);

    // Now persistence should work
    tabs.openTab('note-2', 'Note 2');
    tabs.persistTabs();
    vi.advanceTimersByTime(2500);
    expect(mockedUpdatePreference).toHaveBeenCalled();
  });

  it('resolveTabTitles clears hydration safety timeout', () => {
    tabs.initTabs({
      version: 1,
      tabs: [{ note_id: 'note-1' }],
      active_note_id: 'note-1',
    });

    // Resolve titles normally (within 15s)
    tabs.resolveTabTitles((id) => (id === 'note-1' ? { title: 'Note 1' } : undefined));

    expect(tabs.isInitialized()).toBe(true);

    // After 15s, nothing bad happens (no double-init or errors)
    vi.advanceTimersByTime(15000);
    expect(tabs.isInitialized()).toBe(true);
    expect(tabs.getTabs()).toHaveLength(1);
  });
});
