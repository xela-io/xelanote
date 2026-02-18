// Journal store using Svelte 5 runes
// Manages journal entries and calendar navigation

import { SvelteDate, SvelteMap, SvelteSet } from 'svelte/reactivity';

import { goto } from '$app/navigation';
import {
  ApiError,
  getJournalCalendar,
  getJournalEntries,
  getJournalYearCalendar,
  type JournalEntry,
  lookupJournal,
} from '$lib/api';

import { getJournalFeatureEnabled } from './features.svelte';
import * as notesStore from './notes.svelte';
import * as tree from './tree.svelte';

// Journal folder path
const JOURNAL_FOLDER = '/Journal';

// State
let currentDate = $state<string>(formatDate(new Date()));
let calendarYear = $state(new Date().getFullYear());
let calendarMonth = $state(new Date().getMonth() + 1);
let calendarDates = $state<string[]>([]);
let calendarLoading = $state(false);
let journalLoading = $state(false);
let lastError = $state<string | null>(null);
let entries = $state<JournalEntry[]>([]);
let entriesLoading = $state(false);

// Year calendar state
const yearCache = new SvelteMap<number, Set<string>>();
const prevDecCache = new SvelteMap<number, string[]>();
let yearCacheVersion = $state(0); // Reactive trigger for non-reactive Map mutations
let yearCalendarYear = $state(new Date().getFullYear());
let yearCalendarLoading = $state(false);
let yearCalendarError = $state<string | null>(null);

/**
 * Format a Date object to YYYY-MM-DD string (local timezone).
 */
export function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/**
 * Get today's date formatted as YYYY-MM-DD.
 */
export function getTodayDate(): string {
  return formatDate(new SvelteDate());
}

// Getters
export function getCurrentDate() {
  return currentDate;
}

export function getCalendarYear() {
  return calendarYear;
}

export function getCalendarMonth() {
  return calendarMonth;
}

export function getCalendarDates() {
  return calendarDates;
}

export function getCalendarLoading() {
  return calendarLoading;
}

export function getJournalLoading() {
  return journalLoading;
}

export function getLastError() {
  return lastError;
}

export function getEntries() {
  return entries;
}

export function getEntriesLoading() {
  return entriesLoading;
}

// Year calendar getters
export function getYearCalendarYear() {
  return yearCalendarYear;
}

export function getYearCalendarLoading() {
  return yearCalendarLoading;
}

export function getYearCalendarError() {
  return yearCalendarError;
}

export function getYearDatesSet(): Set<string> {
  void yearCacheVersion; // Read reactive trigger so Svelte tracks this dependency
  return yearCache.get(yearCalendarYear) ?? new SvelteSet();
}

export function getPrevDecDates(): string[] {
  void yearCacheVersion;
  return prevDecCache.get(yearCalendarYear) ?? [];
}

/**
 * Set the current date for the journal view.
 */
export function setCurrentDate(date: string) {
  currentDate = date;
}

/**
 * Load calendar data for a specific month.
 */
export async function loadCalendar(year: number, month: number) {
  if (!getJournalFeatureEnabled()) return;

  calendarLoading = true;
  lastError = null;
  try {
    const response = await getJournalCalendar(year, month);
    calendarYear = response.year;
    calendarMonth = response.month;
    calendarDates = response.dates ?? [];
  } catch (error) {
    console.error('Failed to load journal calendar:', error);
    lastError = 'Kalender konnte nicht geladen werden';
    calendarDates = [];
  } finally {
    calendarLoading = false;
  }
}

/**
 * Load year calendar data. Uses yearCache to avoid re-fetching.
 * Also loads Dec of previous year for streak calculation across year boundaries.
 * Pass force=true to bypass cache (e.g. when returning to journal page after deletions).
 */
export async function loadYearCalendar(year: number, force = false) {
  if (!getJournalFeatureEnabled()) return;

  yearCalendarYear = year;

  // Check cache (skip when forced)
  if (!force && yearCache.has(year)) return;

  yearCalendarLoading = true;
  yearCalendarError = null;
  try {
    // Fetch year data + Dec of previous year in parallel
    const [yearResponse, decResponse] = await Promise.all([
      getJournalYearCalendar(year),
      getJournalCalendar(year - 1, 12),
    ]);

    yearCache.set(year, new SvelteSet(yearResponse.dates ?? []));
    prevDecCache.set(year, decResponse.dates ?? []);
    yearCacheVersion++;
  } catch (error) {
    console.error('Failed to load year calendar:', error);
    yearCalendarError = 'Year calendar could not be loaded';
  } finally {
    yearCalendarLoading = false;
  }
}

/**
 * Navigate to previous year in heatmap.
 */
export async function previousYear() {
  await loadYearCalendar(yearCalendarYear - 1);
}

/**
 * Navigate to next year in heatmap.
 */
export async function nextYear() {
  await loadYearCalendar(yearCalendarYear + 1);
}

/**
 * Load all journal entries for the current user.
 */
export async function loadEntries() {
  if (!getJournalFeatureEnabled()) return;

  entriesLoading = true;
  lastError = null;
  try {
    const response = await getJournalEntries();
    entries = response.entries ?? [];
  } catch (error) {
    console.error('Failed to load journal entries:', error);
    lastError = 'Journal-Einträge konnten nicht geladen werden';
    entries = [];
  } finally {
    entriesLoading = false;
  }
}

/**
 * Navigate to previous month in calendar.
 */
export async function previousMonth() {
  let year = calendarYear;
  let month = calendarMonth - 1;
  if (month < 1) {
    month = 12;
    year--;
  }
  await loadCalendar(year, month);
}

/**
 * Navigate to next month in calendar.
 */
export async function nextMonth() {
  let year = calendarYear;
  let month = calendarMonth + 1;
  if (month > 12) {
    month = 1;
    year++;
  }
  await loadCalendar(year, month);
}

/**
 * Navigate to today's month in calendar.
 */
export async function goToToday() {
  const today = new SvelteDate();
  await loadCalendar(today.getFullYear(), today.getMonth() + 1);
  currentDate = getTodayDate();
}

/**
 * Check if a journal exists for a date.
 */
export async function checkJournalExists(
  date: string
): Promise<{ exists: boolean; noteId: string | null }> {
  try {
    const response = await lookupJournal(date);
    return { exists: response.exists, noteId: response.note_id || null };
  } catch (error) {
    console.error('Failed to check journal:', error);
    return { exists: false, noteId: null };
  }
}

/**
 * Generate a title for a journal entry based on the date.
 */
function generateJournalTitle(date: string): string {
  const d = new Date(date + 'T00:00:00');
  const options: Intl.DateTimeFormatOptions = {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  };
  return d.toLocaleDateString('de-DE', options);
}

/**
 * Open or create a journal for a specific date.
 * Returns the note ID if successful, or null if failed.
 */
export async function openJournalForDate(date: string): Promise<string | null> {
  if (!getJournalFeatureEnabled()) {
    lastError = 'Journal-Feature ist nicht aktiviert';
    return null;
  }

  journalLoading = true;
  lastError = null;
  currentDate = date;

  try {
    // First check if journal exists
    const { exists, noteId } = await checkJournalExists(date);

    if (exists && noteId) {
      // Journal exists - navigate to it
      goto(`/note/${noteId}`);
      journalLoading = false;
      return noteId;
    }

    // Journal doesn't exist - create it
    const title = generateJournalTitle(date);

    // Create journal note (notesStore.createNote handles encryption automatically)
    const note = await notesStore.createNote(title, '', JOURNAL_FOLDER, {
      note_type: 'journal',
      journal_date: date,
    });

    // Navigate to the new note
    goto(`/note/${note.id}`);

    // Refresh tree to show the Journal folder (created on first entry)
    await tree.loadTree();

    // Refresh calendar to show the new entry
    await loadCalendar(calendarYear, calendarMonth);

    // Invalidate year cache — will re-fetch from backend on next journal page visit.
    // We don't optimistically add because the note is empty at this point;
    // the heatmap should only reflect entries the user actually wrote.
    const dateYear = parseInt(date.substring(0, 4), 10);
    invalidateYearCache(dateYear);

    journalLoading = false;
    return note.id;
  } catch (error) {
    if (error instanceof Error && error.message === 'ENCRYPTION_LOCKED') {
      journalLoading = false;
      return null;
    }
    console.error('Failed to open/create journal:', error);
    if (error instanceof ApiError && error.status === 409) {
      // Journal already exists (race condition) - try to load it
      const { noteId } = await checkJournalExists(date);
      if (noteId) {
        goto(`/note/${noteId}`);
        journalLoading = false;
        return noteId;
      }
    }
    lastError = error instanceof Error ? error.message : 'Journal konnte nicht erstellt werden';
    journalLoading = false;
    return null;
  }
}

/**
 * Open today's journal.
 */
export async function openTodayJournal(): Promise<string | null> {
  return openJournalForDate(getTodayDate());
}

/**
 * Check if a specific date has a journal entry.
 */
export function hasJournalEntry(date: string): boolean {
  return calendarDates.includes(date);
}

/**
 * Calculate streak data from year dates and previous December dates.
 */
export function calculateStreaks(
  yearDates: Set<string>,
  prevDecDates: string[]
): { current: number; longest: number; todayDone: boolean } {
  const today = getTodayDate();
  const todayDone = yearDates.has(today);

  // Current streak: count backwards from yesterday
  let current = 0;
  const d = new SvelteDate();
  d.setDate(d.getDate() - 1); // Start from yesterday

  // Count backwards through current year
  while (true) {
    const dateStr = formatDate(d);
    const dateYear = d.getFullYear();

    if (dateYear === parseInt(today.substring(0, 4), 10)) {
      // Same year as "today"
      if (yearDates.has(dateStr)) {
        current++;
        d.setDate(d.getDate() - 1);
      } else {
        break;
      }
    } else if (dateYear === parseInt(today.substring(0, 4), 10) - 1) {
      // Previous year - check prevDecDates
      if (prevDecDates.includes(dateStr)) {
        current++;
        d.setDate(d.getDate() - 1);
      } else {
        break;
      }
    } else {
      break;
    }
  }

  // If today is done, add it to the current streak
  if (todayDone) {
    current++;
  }

  // Longest streak (within the year only)
  const sortedDates = Array.from(yearDates).sort();
  let longest = 0;
  let streak = 0;
  let prevDate: Date | null = null;

  for (const dateStr of sortedDates) {
    const curr = new SvelteDate(dateStr + 'T00:00:00');
    if (prevDate) {
      const diff = (curr.getTime() - prevDate.getTime()) / (1000 * 60 * 60 * 24);
      if (diff === 1) {
        streak++;
      } else {
        streak = 1;
      }
    } else {
      streak = 1;
    }
    if (streak > longest) longest = streak;
    prevDate = curr;
  }

  return { current, longest, todayDone };
}

/**
 * Get year dates for a specific year (independent of navigation state).
 */
export function getYearDatesSetForYear(year: number): Set<string> {
  void yearCacheVersion;
  // eslint-disable-next-line svelte/prefer-svelte-reactivity
  return yearCache.get(year) ?? new Set();
}

/**
 * Get prev December dates for a specific year (independent of navigation state).
 */
export function getPrevDecDatesForYear(year: number): string[] {
  void yearCacheVersion;
  return prevDecCache.get(year) ?? [];
}

/**
 * Populate cache without mutating yearCalendarYear navigation state.
 * Unlike loadYearCalendar(), this does NOT set yearCalendarYear,
 * so the desktop heatmap's year navigation is not affected.
 */
export async function ensureYearCacheLoaded(year: number) {
  if (!getJournalFeatureEnabled()) return;
  if (yearCache.has(year)) return;

  yearCalendarLoading = true;
  yearCalendarError = null;
  try {
    const [yearResponse, decResponse] = await Promise.all([
      getJournalYearCalendar(year),
      getJournalCalendar(year - 1, 12),
    ]);
    yearCache.set(year, new SvelteSet(yearResponse.dates ?? []));
    prevDecCache.set(year, decResponse.dates ?? []);
    yearCacheVersion++;
  } catch (error) {
    console.error('Failed to load year calendar:', error);
    yearCalendarError = 'Year calendar could not be loaded';
  } finally {
    yearCalendarLoading = false;
  }
}

/**
 * Invalidate year cache for a specific year (e.g., when a journal entry is deleted).
 */
export function invalidateYearCache(year: number) {
  yearCache.delete(year);
  prevDecCache.delete(year);
  yearCacheVersion++;
}

/**
 * Reset journal state (called on logout).
 */
export function resetJournalState() {
  currentDate = formatDate(new SvelteDate());
  calendarYear = new SvelteDate().getFullYear();
  calendarMonth = new SvelteDate().getMonth() + 1;
  calendarDates = [];
  calendarLoading = false;
  journalLoading = false;
  lastError = null;
  entries = [];
  entriesLoading = false;
  yearCache.clear();
  prevDecCache.clear();
  yearCacheVersion++;
  yearCalendarYear = new SvelteDate().getFullYear();
  yearCalendarLoading = false;
  yearCalendarError = null;
}
