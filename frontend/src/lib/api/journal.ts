import { request } from './client';
import type {
  JournalCalendarResponse,
  JournalEntriesResponse,
  JournalLookupResponse,
  JournalYearCalendarResponse,
} from './types';

/**
 * Check if a journal exists for a specific date.
 * Returns the note ID if it exists.
 */
export async function lookupJournal(date: string): Promise<JournalLookupResponse> {
  return request(`/journal?date=${date}`);
}

/**
 * Get calendar data (dates with journal entries) for a specific month.
 */
export async function getJournalCalendar(
  year: number,
  month: number
): Promise<JournalCalendarResponse> {
  return request(`/journal/calendar?year=${year}&month=${month}`);
}

/**
 * Get calendar data (dates with journal entries) for a full year.
 */
export async function getJournalYearCalendar(year: number): Promise<JournalYearCalendarResponse> {
  return request(`/journal/calendar/year?year=${year}`, { cache: 'no-store' });
}

/**
 * Get all journal entries for the current user.
 */
export async function getJournalEntries(): Promise<JournalEntriesResponse> {
  return request('/journal/entries');
}
