/**
 * Utility for reordering task items in CodeMirror documents.
 * Provides atomic line movement operations for drag & drop.
 */

import type { ChangeSpec, Text } from '@codemirror/state';

import { computeNestLevel } from '$lib/editor/live-preview/line-primitives';
import { getSubtreeLineRange1Based } from '$lib/editor/task-nesting';

export interface TaskInfo {
  index: number; // Global task index (matches data-task-index)
  lineNum: number; // 1-based line number
  lineFrom: number; // Char offset start
  lineTo: number; // Char offset end
  text: string; // Line content
  indentLength: number; // Leading whitespace length
}

/**
 * Find all tasks in the document with their positions.
 * Tasks are lines matching: ^\s*(?:[-*+]|\d+[.)]) \[[xX ]\]
 */
export function getTasksInDocument(doc: Text): TaskInfo[] {
  const tasks: TaskInfo[] = [];
  const taskPattern = /^(\s*(?:[-*+]|\d+[.)]) )\[([xX ])\]/;

  for (let i = 1; i <= doc.lines; i++) {
    const line = doc.line(i);
    const match = taskPattern.exec(line.text);
    if (match) {
      const taskBody = line.text.substring(match[0].length).trim();
      // Match markdown-it-task-lists behavior: no checkbox for empty tasks.
      if (!taskBody) continue;
      const indentMatch = line.text.match(/^(\s*)/);
      tasks.push({
        index: tasks.length,
        lineNum: i,
        lineFrom: line.from,
        lineTo: line.to,
        text: line.text,
        indentLength: indentMatch ? indentMatch[1].length : 0,
      });
    }
  }

  return tasks;
}

/**
 * Calculate CodeMirror ChangeSpec for moving a task from one position to another.
 * Subtree-aware: if the task has nested children, they move as a block.
 * Rejects moves for nested tasks (nestLevel > 0).
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

  // Reject moves for nested tasks — only top-level tasks are draggable
  const fromNestLevel = computeNestLevel(fromTask.text.match(/^(\s*)/)?.[1] ?? '');
  if (fromNestLevel > 0) {
    return [];
  }

  // Get subtree range for the source task
  const getLineText = (lineNum: number) => doc.line(lineNum).text;
  const fromSubtree = getSubtreeLineRange1Based(getLineText, doc.lines, fromTask.lineNum);
  const fromStartLine = doc.line(fromSubtree.startLine);
  const fromEndLine = doc.line(fromSubtree.endLine);

  // Get subtree range for the target task
  const toSubtree = getSubtreeLineRange1Based(getLineText, doc.lines, toTask.lineNum);

  // Collect the text of the entire subtree being moved
  const subtreeLines: string[] = [];
  for (let i = fromSubtree.startLine; i <= fromSubtree.endLine; i++) {
    subtreeLines.push(doc.line(i).text);
  }
  const subtreeText = subtreeLines.join('\n');

  // CodeMirror ChangeSpec: changes must be sorted by position (ascending)
  // and positions are always relative to the ORIGINAL document
  let changes: ChangeSpec[];

  if (toTaskIndex < fromTaskIndex) {
    // Moving up: insert at target, delete at current (higher position)
    const targetLine = doc.line(toSubtree.startLine);
    const insertPos = targetLine.from;

    // Delete the entire subtree including trailing newline
    const deleteFrom = fromStartLine.from;
    const deleteTo = fromEndLine.to + (fromEndLine.to < doc.length ? 1 : 0);

    // Changes sorted by position (insert first = lower position)
    changes = [
      { from: insertPos, to: insertPos, insert: subtreeText + '\n' },
      { from: deleteFrom, to: deleteTo, insert: '' },
    ];
  } else {
    // Moving down: delete at current, insert after target's subtree
    const targetEndLine = doc.line(toSubtree.endLine);

    // Delete current subtree including ONE newline (before or after)
    let deleteFrom: number;
    let deleteTo: number;

    if (fromStartLine.from > 0) {
      // Not first line: delete preceding newline
      deleteFrom = fromStartLine.from - 1;
      deleteTo = fromEndLine.to;
    } else {
      // First line: delete following newline if exists
      deleteFrom = fromStartLine.from;
      deleteTo = fromEndLine.to + (fromEndLine.to < doc.length ? 1 : 0);
    }

    // Insert after target's subtree
    const insertPos = targetEndLine.to;

    // Changes sorted by position (delete first = lower position)
    changes = [
      { from: deleteFrom, to: deleteTo, insert: '' },
      { from: insertPos, to: insertPos, insert: '\n' + subtreeText },
    ];
  }

  return changes;
}
