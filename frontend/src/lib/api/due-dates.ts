import { request } from './client';
import type { DueDateItem } from './types';

export async function getDueDates(showCompleted = false): Promise<DueDateItem[]> {
  const params = showCompleted ? '?show_completed=true' : '';
  const data = await request<{ due_dates: DueDateItem[] }>(`/due-dates${params}`);
  return data.due_dates;
}
