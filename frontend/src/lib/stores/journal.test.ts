import { beforeEach, describe, expect, it, vi } from 'vitest';

const getJournalCalendar = vi.fn();
const getJournalEntries = vi.fn();
const getJournalYearCalendar = vi.fn();
const lookupJournal = vi.fn();

vi.mock('$lib/api', () => ({
  ApiError: class ApiError extends Error {
    status: number;
    constructor(status = 500) {
      super('api error');
      this.status = status;
    }
  },
  getJournalCalendar,
  getJournalEntries,
  getJournalYearCalendar,
  lookupJournal,
}));

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

const getJournalFeatureEnabled = vi.fn().mockReturnValue(true);
vi.mock('./features.svelte', () => ({
  getJournalFeatureEnabled,
}));

const createNote = vi.fn();
vi.mock('./notes.svelte', () => ({
  createNote,
}));

const loadTree = vi.fn();
vi.mock('./tree.svelte', () => ({
  loadTree,
}));

// Mock SvelteDate/SvelteMap/SvelteSet with native equivalents
vi.mock('svelte/reactivity', () => ({
  SvelteDate: Date,
  SvelteMap: Map,
  SvelteSet: Set,
}));

describe('journal store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    getJournalFeatureEnabled.mockReturnValue(true);
  });

  describe('formatDate', () => {
    it('should format date as YYYY-MM-DD', async () => {
      const store = await import('$lib/stores/journal.svelte');
      expect(store.formatDate(new Date(2026, 0, 5))).toBe('2026-01-05');
      expect(store.formatDate(new Date(2026, 11, 25))).toBe('2026-12-25');
    });

    it('should pad single-digit month and day', async () => {
      const store = await import('$lib/stores/journal.svelte');
      expect(store.formatDate(new Date(2026, 1, 3))).toBe('2026-02-03');
    });
  });

  describe('loadCalendar', () => {
    it('should load calendar data for a month', async () => {
      getJournalCalendar.mockResolvedValue({
        year: 2026,
        month: 2,
        dates: ['2026-02-01', '2026-02-05'],
      });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 2);

      expect(getJournalCalendar).toHaveBeenCalledWith(2026, 2);
      expect(store.getCalendarYear()).toBe(2026);
      expect(store.getCalendarMonth()).toBe(2);
      expect(store.getCalendarDates()).toEqual(['2026-02-01', '2026-02-05']);
      expect(store.getCalendarLoading()).toBe(false);
    });

    it('should handle null dates from API', async () => {
      getJournalCalendar.mockResolvedValue({
        year: 2026,
        month: 3,
        dates: null,
      });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 3);

      expect(store.getCalendarDates()).toEqual([]);
    });

    it('should set error and empty dates on failure', async () => {
      getJournalCalendar.mockRejectedValue(new Error('network'));

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 2);

      expect(store.getLastError()).toBe('Kalender konnte nicht geladen werden');
      expect(store.getCalendarDates()).toEqual([]);
      expect(store.getCalendarLoading()).toBe(false);
    });

    it('should skip when journal feature is disabled', async () => {
      getJournalFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 2);

      expect(getJournalCalendar).not.toHaveBeenCalled();
    });
  });

  describe('previousMonth / nextMonth', () => {
    it('previousMonth should wrap from Jan to Dec of previous year', async () => {
      getJournalCalendar.mockResolvedValue({ year: 2026, month: 1, dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      // Set calendar to January 2026
      await store.loadCalendar(2026, 1);

      getJournalCalendar.mockResolvedValue({ year: 2025, month: 12, dates: [] });
      await store.previousMonth();

      expect(getJournalCalendar).toHaveBeenLastCalledWith(2025, 12);
    });

    it('nextMonth should wrap from Dec to Jan of next year', async () => {
      getJournalCalendar.mockResolvedValue({ year: 2026, month: 12, dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 12);

      getJournalCalendar.mockResolvedValue({ year: 2027, month: 1, dates: [] });
      await store.nextMonth();

      expect(getJournalCalendar).toHaveBeenLastCalledWith(2027, 1);
    });
  });

  describe('loadEntries', () => {
    it('should load journal entries', async () => {
      const mockEntries = [
        { id: '1', date: '2026-02-01', title: 'Day 1' },
        { id: '2', date: '2026-02-02', title: 'Day 2' },
      ];
      getJournalEntries.mockResolvedValue({ entries: mockEntries });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadEntries();

      expect(store.getEntries()).toEqual(mockEntries);
      expect(store.getEntriesLoading()).toBe(false);
    });

    it('should handle null entries from API', async () => {
      getJournalEntries.mockResolvedValue({ entries: null });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadEntries();

      expect(store.getEntries()).toEqual([]);
    });

    it('should set error on failure', async () => {
      getJournalEntries.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/journal.svelte');
      await store.loadEntries();

      expect(store.getLastError()).toBe('Journal-Einträge konnten nicht geladen werden');
      expect(store.getEntries()).toEqual([]);
    });

    it('should skip when journal feature is disabled', async () => {
      getJournalFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/journal.svelte');
      await store.loadEntries();

      expect(getJournalEntries).not.toHaveBeenCalled();
    });
  });

  describe('loadYearCalendar', () => {
    it('should load year data and previous December in parallel', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01', '2026-01-15'] });
      getJournalCalendar.mockResolvedValue({ dates: ['2025-12-28', '2025-12-31'] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);

      expect(getJournalYearCalendar).toHaveBeenCalledWith(2026);
      expect(getJournalCalendar).toHaveBeenCalledWith(2025, 12);
      expect(store.getYearCalendarYear()).toBe(2026);
      expect(store.getYearCalendarLoading()).toBe(false);
    });

    it('should use cache on subsequent calls', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01'] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);
      expect(getJournalYearCalendar).toHaveBeenCalledTimes(1);

      // Second call should use cache
      await store.loadYearCalendar(2026);
      expect(getJournalYearCalendar).toHaveBeenCalledTimes(1); // Still 1
    });

    it('should bypass cache when force=true', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01'] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);
      await store.loadYearCalendar(2026, true);

      expect(getJournalYearCalendar).toHaveBeenCalledTimes(2);
    });

    it('should set error on failure', async () => {
      getJournalYearCalendar.mockRejectedValue(new Error('fail'));
      getJournalCalendar.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);

      expect(store.getYearCalendarError()).toBe('Year calendar could not be loaded');
      expect(store.getYearCalendarLoading()).toBe(false);
    });
  });

  describe('previousYear / nextYear', () => {
    it('previousYear should load year-1', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: [] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      // First load current year
      await store.loadYearCalendar(2026);
      // Then go back
      await store.previousYear();

      expect(store.getYearCalendarYear()).toBe(2025);
    });

    it('nextYear should load year+1', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: [] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);
      await store.nextYear();

      expect(store.getYearCalendarYear()).toBe(2027);
    });
  });

  describe('checkJournalExists', () => {
    it('should return exists=true when journal found', async () => {
      lookupJournal.mockResolvedValue({ exists: true, note_id: 'note-123' });

      const store = await import('$lib/stores/journal.svelte');
      const result = await store.checkJournalExists('2026-02-21');

      expect(lookupJournal).toHaveBeenCalledWith('2026-02-21');
      expect(result).toEqual({ exists: true, noteId: 'note-123' });
    });

    it('should return exists=false when not found', async () => {
      lookupJournal.mockResolvedValue({ exists: false, note_id: '' });

      const store = await import('$lib/stores/journal.svelte');
      const result = await store.checkJournalExists('2026-02-22');

      expect(result).toEqual({ exists: false, noteId: null });
    });

    it('should return exists=false on error', async () => {
      lookupJournal.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/journal.svelte');
      const result = await store.checkJournalExists('2026-02-22');

      expect(result).toEqual({ exists: false, noteId: null });
    });
  });

  describe('hasJournalEntry', () => {
    it('should return true for dates in calendarDates', async () => {
      getJournalCalendar.mockResolvedValue({
        year: 2026,
        month: 2,
        dates: ['2026-02-01', '2026-02-05'],
      });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2026, 2);

      expect(store.hasJournalEntry('2026-02-01')).toBe(true);
      expect(store.hasJournalEntry('2026-02-05')).toBe(true);
      expect(store.hasJournalEntry('2026-02-02')).toBe(false);
    });
  });

  describe('openJournalForDate', () => {
    it('should navigate to existing journal', async () => {
      lookupJournal.mockResolvedValue({ exists: true, note_id: 'note-42' });

      const { goto } = await import('$app/navigation');
      const store = await import('$lib/stores/journal.svelte');
      const noteId = await store.openJournalForDate('2026-02-21');

      expect(noteId).toBe('note-42');
      expect(goto).toHaveBeenCalledWith('/note/note-42');
      expect(store.getJournalLoading()).toBe(false);
    });

    it('should create new journal when none exists', async () => {
      lookupJournal.mockResolvedValue({ exists: false, note_id: '' });
      const mockNote = { id: 'new-note-1' };
      createNote.mockResolvedValue(mockNote);
      loadTree.mockResolvedValue(undefined);
      getJournalCalendar.mockResolvedValue({ year: 2026, month: 2, dates: [] });

      const { goto } = await import('$app/navigation');
      const store = await import('$lib/stores/journal.svelte');
      const noteId = await store.openJournalForDate('2026-02-21');

      expect(noteId).toBe('new-note-1');
      expect(createNote).toHaveBeenCalledWith(
        expect.any(String), // German-formatted title
        '',
        '/Journal',
        { note_type: 'journal', journal_date: '2026-02-21' }
      );
      expect(goto).toHaveBeenCalledWith('/note/new-note-1');
      expect(loadTree).toHaveBeenCalled();
    });

    it('should return null when feature is disabled', async () => {
      getJournalFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/journal.svelte');
      const noteId = await store.openJournalForDate('2026-02-21');

      expect(noteId).toBeNull();
      expect(store.getLastError()).toBe('Journal-Feature ist nicht aktiviert');
    });

    it('should return null on ENCRYPTION_LOCKED error', async () => {
      lookupJournal.mockResolvedValue({ exists: false, note_id: '' });
      createNote.mockRejectedValue(new Error('ENCRYPTION_LOCKED'));

      const store = await import('$lib/stores/journal.svelte');
      const noteId = await store.openJournalForDate('2026-02-21');

      expect(noteId).toBeNull();
      expect(store.getJournalLoading()).toBe(false);
    });
  });

  describe('calculateStreaks', () => {
    it('should return zeros for empty dates', async () => {
      const store = await import('$lib/stores/journal.svelte');
      const result = store.calculateStreaks(new Set(), []);

      expect(result).toEqual({ current: 0, longest: 0, todayDone: false });
    });

    it('should calculate longest streak', async () => {
      const store = await import('$lib/stores/journal.svelte');
      const dates = new Set([
        '2026-01-01',
        '2026-01-02',
        '2026-01-03', // 3-day streak
        '2026-01-10',
        '2026-01-11', // 2-day streak
      ]);

      const result = store.calculateStreaks(dates, []);
      expect(result.longest).toBe(3);
    });

    it('should count todayDone correctly', async () => {
      const store = await import('$lib/stores/journal.svelte');
      const today = store.getTodayDate();
      const dates = new Set([today]);

      const result = store.calculateStreaks(dates, []);
      expect(result.todayDone).toBe(true);
    });

    it('should count current streak from yesterday backwards', async () => {
      const store = await import('$lib/stores/journal.svelte');
      const today = new Date();

      // Build a streak ending yesterday
      const dates = new Set<string>();
      for (let i = 1; i <= 5; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - i);
        dates.add(store.formatDate(d));
      }

      const result = store.calculateStreaks(dates, []);
      expect(result.current).toBe(5);
    });

    it('should include today in current streak if done', async () => {
      const store = await import('$lib/stores/journal.svelte');
      const today = new Date();

      const dates = new Set<string>();
      // Add today
      dates.add(store.formatDate(today));
      // Add yesterday
      const yesterday = new Date(today);
      yesterday.setDate(yesterday.getDate() - 1);
      dates.add(store.formatDate(yesterday));

      const result = store.calculateStreaks(dates, []);
      expect(result.current).toBe(2); // yesterday + today
      expect(result.todayDone).toBe(true);
    });
  });

  describe('invalidateYearCache', () => {
    it('should clear cache for a specific year', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01'] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadYearCalendar(2026);

      // Cache should be populated
      expect(store.getYearDatesSetForYear(2026).size).toBeGreaterThan(0);

      // Invalidate
      store.invalidateYearCache(2026);

      // Cache should be empty
      expect(store.getYearDatesSetForYear(2026).size).toBe(0);
    });
  });

  describe('resetJournalState', () => {
    it('should reset all state to defaults', async () => {
      // First load some data
      getJournalCalendar.mockResolvedValue({
        year: 2025,
        month: 6,
        dates: ['2025-06-01'],
      });
      getJournalEntries.mockResolvedValue({
        entries: [{ id: '1', date: '2025-06-01', title: 'Test' }],
      });

      const store = await import('$lib/stores/journal.svelte');
      await store.loadCalendar(2025, 6);
      await store.loadEntries();

      expect(store.getCalendarDates().length).toBe(1);
      expect(store.getEntries().length).toBe(1);

      // Reset
      store.resetJournalState();

      expect(store.getCalendarDates()).toEqual([]);
      expect(store.getEntries()).toEqual([]);
      expect(store.getCalendarLoading()).toBe(false);
      expect(store.getJournalLoading()).toBe(false);
      expect(store.getLastError()).toBeNull();
      expect(store.getEntriesLoading()).toBe(false);
      expect(store.getYearCalendarLoading()).toBe(false);
      expect(store.getYearCalendarError()).toBeNull();
    });
  });

  describe('setCurrentDate', () => {
    it('should update current date', async () => {
      const store = await import('$lib/stores/journal.svelte');
      store.setCurrentDate('2026-03-15');
      expect(store.getCurrentDate()).toBe('2026-03-15');
    });
  });

  describe('ensureYearCacheLoaded', () => {
    it('should load and cache if not already present', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01'] });
      getJournalCalendar.mockResolvedValue({ dates: ['2025-12-31'] });

      const store = await import('$lib/stores/journal.svelte');
      await store.ensureYearCacheLoaded(2026);

      expect(getJournalYearCalendar).toHaveBeenCalledWith(2026);
      expect(store.getYearDatesSetForYear(2026).has('2026-01-01')).toBe(true);
      expect(store.getPrevDecDatesForYear(2026)).toEqual(['2025-12-31']);
    });

    it('should skip if already cached', async () => {
      getJournalYearCalendar.mockResolvedValue({ dates: ['2026-01-01'] });
      getJournalCalendar.mockResolvedValue({ dates: [] });

      const store = await import('$lib/stores/journal.svelte');
      await store.ensureYearCacheLoaded(2026);
      await store.ensureYearCacheLoaded(2026); // second call

      expect(getJournalYearCalendar).toHaveBeenCalledTimes(1);
    });

    it('should skip when feature is disabled', async () => {
      getJournalFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/journal.svelte');
      await store.ensureYearCacheLoaded(2026);

      expect(getJournalYearCalendar).not.toHaveBeenCalled();
    });
  });
});
