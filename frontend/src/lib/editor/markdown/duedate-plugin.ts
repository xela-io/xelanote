// Due date syntax plugin for markdown-it: @due(YYYY-MM-DD)

import type MarkdownIt from 'markdown-it';
import type StateInline from 'markdown-it/lib/rules_inline/state_inline.mjs';
import type Token from 'markdown-it/lib/token.mjs';

import { FEATURE_FLAGS } from '$lib/config';

const DUE_DATE_REGEX = /^\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])$/;

/**
 * Validate a date string in YYYY-MM-DD format.
 * Checks both format and overflow (e.g. Feb 30 is invalid).
 */
export function isValidDueDate(dateStr: string): boolean {
  if (!DUE_DATE_REGEX.test(dateStr)) return false;
  const d = new Date(dateStr + 'T00:00:00');
  if (isNaN(d.getTime())) return false;
  const [y, m, day] = dateStr.split('-').map(Number);
  return d.getFullYear() === y && d.getMonth() + 1 === m && d.getDate() === day;
}

/**
 * Determine the status of a due date relative to today.
 */
export function getDueDateStatus(dateStr: string): 'overdue' | 'today' | 'soon' | 'future' {
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  const due = new Date(dateStr + 'T00:00:00');
  const diffMs = due.getTime() - now.getTime();
  const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24));
  if (diffDays < 0) return 'overdue';
  if (diffDays === 0) return 'today';
  if (diffDays <= 3) return 'soon';
  return 'future';
}

function dueDateRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.dueDateSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  if (state.src.charCodeAt(start) !== 0x40 /* @ */) return false;
  if (start + 5 >= max) return false;
  if (state.src.slice(start, start + 5) !== '@due(') return false;

  const closePos = state.src.indexOf(')', start + 5);
  if (closePos === -1 || closePos > start + 15) return false;

  const dateStr = state.src.slice(start + 5, closePos);
  if (!isValidDueDate(dateStr)) return false;

  if (!silent) {
    const token = state.push('due_date', 'span', 0);
    token.content = dateStr;
  }

  state.pos = closePos + 1;
  return true;
}

/** Register due date plugin with a MarkdownIt instance. */
export function register(md: MarkdownIt, escapeHtml: (s: string) => string): void {
  if (!FEATURE_FLAGS.dueDateSyntax) return;

  md.inline.ruler.before('link', 'due_date', dueDateRule);

  md.renderer.rules.due_date = (tokens: Token[], idx: number): string => {
    const dateStr = tokens[idx].content;
    const status = getDueDateStatus(dateStr);
    return `<span class="due-date due-date-${status}" data-due-date="${escapeHtml(dateStr)}">${escapeHtml(dateStr)}</span>`;
  };
}
