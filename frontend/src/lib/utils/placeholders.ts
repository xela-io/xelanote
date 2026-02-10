/**
 * Expands predefined placeholders in content.
 * Supports: {{date}}, {{time}}, {{cursor}}
 *
 * @param content - Content with placeholders
 * @returns Object with expanded content and optional cursor position
 */
export function expandPlaceholders(content: string): {
  content: string;
  cursorPos?: number;
} {
  let expanded = content;
  const now = new Date();

  // {{date}} → YYYY-MM-DD (e.g., 2026-01-18)
  expanded = expanded.replace(/\{\{date\}\}/g, now.toISOString().split('T')[0]);

  // {{time}} → HH:MM (e.g., 14:30)
  const timeStr = now.toTimeString().substring(0, 5);
  expanded = expanded.replace(/\{\{time\}\}/g, timeStr);

  // {{cursor}} → Sets cursor position (ONLY FIRST occurrence)
  const cursorIndex = expanded.indexOf('{{cursor}}');
  if (cursorIndex !== -1) {
    // Remove ONLY first {{cursor}}, keep others as literal text
    expanded = expanded.replace('{{cursor}}', '');

    // If multiple {{cursor}}, warn in console (dev only)
    if (expanded.indexOf('{{cursor}}') !== -1) {
      console.warn(
        'Multiple {{cursor}} placeholders found. Only first one is used for positioning.'
      );
    }

    return { content: expanded, cursorPos: cursorIndex };
  }

  return { content: expanded };
}

/**
 * Validates if content contains valid placeholders only.
 * @param content - Content to validate
 * @returns true if valid, false otherwise
 */
export function hasValidPlaceholders(content: string): boolean {
  const validPattern = /\{\{(date|time|cursor)\}\}/g;
  const allMatches = content.match(/\{\{[^}]+\}\}/g) || [];
  return allMatches.every((match) => validPattern.test(match));
}
