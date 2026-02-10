/**
 * Utility for reordering task items in CodeMirror documents.
 * Provides atomic line movement operations for drag & drop.
 */

import type { Text, ChangeSpec } from '@codemirror/state';

export interface TaskInfo {
  index: number; // Global task index (matches data-task-index)
  lineNum: number; // 1-based line number
  lineFrom: number; // Char offset start
  lineTo: number; // Char offset end
  text: string; // Line content
}

/**
 * Find all tasks in the document with their positions.
 * Tasks are lines matching: ^\s*[-*+]\s*\[[xX ]\]
 */
export function getTasksInDocument(doc: Text): TaskInfo[] {
  const tasks: TaskInfo[] = [];
  const taskPattern = /^(\s*[-*+]\s*)\[([xX ])\]/;

  for (let i = 1; i <= doc.lines; i++) {
    const line = doc.line(i);
    if (taskPattern.test(line.text)) {
      tasks.push({
        index: tasks.length,
        lineNum: i,
        lineFrom: line.from,
        lineTo: line.to,
        text: line.text,
      });
    }
  }

  return tasks;
}

/**
 * Calculate CodeMirror ChangeSpec for moving a task from one position to another.
 * Returns changes that can be applied atomically with editorView.dispatch().
 *
 * @param doc - The CodeMirror document
 * @param fromTaskIndex - The global task index of the task to move (data-task-index)
 * @param toTaskIndex - The global task index of the target position
 * @returns Array of ChangeSpec objects for atomic application
 */
export function calculateMoveChanges(
  doc: Text,
  fromTaskIndex: number,
  toTaskIndex: number
): ChangeSpec[] {
  if (fromTaskIndex === toTaskIndex) {
    return [];
  }

  const tasks = getTasksInDocument(doc);

  // Validate indices
  if (fromTaskIndex < 0 || fromTaskIndex >= tasks.length) {
    console.warn(`Invalid fromTaskIndex: ${fromTaskIndex}, tasks count: ${tasks.length}`);
    return [];
  }
  if (toTaskIndex < 0 || toTaskIndex >= tasks.length) {
    console.warn(`Invalid toTaskIndex: ${toTaskIndex}, tasks count: ${tasks.length}`);
    return [];
  }

  const fromTask = tasks[fromTaskIndex];
  const toTask = tasks[toTaskIndex];

  const currentLine = doc.line(fromTask.lineNum);
  const lineText = currentLine.text;

  // CodeMirror ChangeSpec: changes must be sorted by position (ascending)
  // and positions are always relative to the ORIGINAL document
  let changes: ChangeSpec[];

  if (toTaskIndex < fromTaskIndex) {
    // Moving up: insert at target, delete at current (higher position)
    const targetLine = doc.line(toTask.lineNum);
    const insertPos = targetLine.from;

    // Delete includes the newline after
    const deleteFrom = currentLine.from;
    const deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);

    // Changes sorted by position (insert first = lower position)
    changes = [
      { from: insertPos, to: insertPos, insert: lineText + '\n' },
      { from: deleteFrom, to: deleteTo, insert: '' },
    ];
  } else {
    // Moving down: delete at current, insert at target (higher position)
    const targetLine = doc.line(toTask.lineNum);

    // Delete current line including ONE newline (before or after)
    let deleteFrom: number;
    let deleteTo: number;

    if (currentLine.from > 0) {
      // Not first line: delete preceding newline
      deleteFrom = currentLine.from - 1;
      deleteTo = currentLine.to;
    } else {
      // First line: delete following newline if exists
      deleteFrom = currentLine.from;
      deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);
    }

    // Insert after target line
    const insertPos = targetLine.to;

    // Changes sorted by position (delete first = lower position)
    changes = [
      { from: deleteFrom, to: deleteTo, insert: '' },
      { from: insertPos, to: insertPos, insert: '\n' + lineText },
    ];
  }

  return changes;
}
