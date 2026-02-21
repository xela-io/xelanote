// Content extraction utilities: headings, due dates, wikilinks

import { isValidDueDate } from './duedate-plugin';

// Table of Contents types and extraction
export interface TocEntry {
  level: number;
  text: string;
  slug: string;
}

export function extractHeadings(content: string): TocEntry[] {
  const headings: TocEntry[] = [];
  const lines = content.split('\n');
  const slugCounts = new Map<string, number>();

  for (const line of lines) {
    const match = line.match(/^(#{1,6})\s+(.+)$/);
    if (match) {
      const level = match[1].length;
      const text = match[2].trim();
      let slug = text
        .toLowerCase()
        .replace(/[^\w\s-]/g, '')
        .replace(/\s+/g, '-');

      const count = slugCounts.get(slug) || 0;
      slugCounts.set(slug, count + 1);
      if (count > 0) {
        slug = `${slug}-${count}`;
      }

      headings.push({ level, text, slug });
    }
  }
  return headings;
}

// Extract all valid @due(YYYY-MM-DD) dates from content (ignoring code blocks)
export function extractDueDates(content: string): string[] {
  const dates: string[] = [];
  let inCodeBlock = false;
  let inInlineCode = false;

  const lines = content.split('\n');
  for (const line of lines) {
    if (line.trimStart().startsWith('```')) {
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) continue;

    let i = 0;
    while (i < line.length) {
      if (line[i] === '`') {
        inInlineCode = !inInlineCode;
        i++;
        continue;
      }
      if (inInlineCode) {
        i++;
        continue;
      }
      if (line[i] === '@' && line.slice(i, i + 5) === '@due(') {
        const closeIdx = line.indexOf(')', i + 5);
        if (closeIdx !== -1 && closeIdx <= i + 15) {
          const dateStr = line.slice(i + 5, closeIdx);
          if (isValidDueDate(dateStr)) {
            dates.push(dateStr);
          }
        }
      }
      i++;
    }
    inInlineCode = false;
  }

  return dates;
}

// Detailed due date info for server sync (matches backend parser.DueDate struct)
export interface DueDateInfo {
  due_date: string;
  line_text: string;
  line_index: number;
  is_task_item: boolean;
  is_completed: boolean;
}

const dueDateCleanupRegex = /@due\([^)]*\)/g;
const checkboxRegex = /^\s*(?:[-*+]|\d+[.)]) \[([xX ])\]\s*/;
const listPrefixRegex = /^\s*(?:[-*+]|\d+[.)]) (?:\[[xX ]\]\s*)?/;

// Extract due dates with full metadata for server sync (used by encrypted notes)
export function extractDueDatesDetailed(content: string): DueDateInfo[] {
  const results: DueDateInfo[] = [];
  const lines = content.split('\n');
  let inCodeBlock = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trimStart();

    if (trimmed.startsWith('```')) {
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) continue;

    const dateRegex = /@due\((\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\)/g;
    let match;
    const matches: string[] = [];
    while ((match = dateRegex.exec(line)) !== null) {
      matches.push(match[1]);
    }
    if (matches.length === 0) continue;

    const cbMatch = checkboxRegex.exec(line);
    const isTask = cbMatch !== null;
    const isCompleted = isTask && (cbMatch![1] === 'x' || cbMatch![1] === 'X');

    for (const dateStr of matches) {
      if (!isValidDueDate(dateStr)) continue;

      let cleanText = line.replace(dueDateCleanupRegex, '');
      if (isTask) {
        cleanText = cleanText.replace(listPrefixRegex, '');
      }
      cleanText = cleanText.trim();

      results.push({
        due_date: dateStr,
        line_text: cleanText,
        line_index: i,
        is_task_item: isTask,
        is_completed: isCompleted,
      });
    }
  }

  return results;
}

// Parse wikilinks from content (for extracting link targets)
export function extractWikilinks(content: string): Array<{ title: string; alias?: string }> {
  const links: Array<{ title: string; alias?: string }> = [];
  const regex = /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g;
  let match;

  while ((match = regex.exec(content)) !== null) {
    links.push({
      title: match[1].trim(),
      alias: match[2]?.trim(),
    });
  }

  return links;
}
