/**
 * Nested task tree construction, propagation, and subtree range utilities.
 */

import { computeNestLevel } from './live-preview/line-primitives';

export interface TaskNode {
  lineNumber: number; // 1-based
  indentLength: number; // raw character count of leading whitespace
  nestLevel: number; // computed via computeNestLevel
  checked: boolean;
  children: TaskNode[];
  parent: TaskNode | null;
}

export interface PropagationChange {
  lineNumber: number;
  newChecked: boolean;
}

/**
 * Build a forest of TaskNodes from a flat array of task items.
 * Uses a stack-based algorithm: for each task, pop the stack until
 * `stack-top.indentLength < current.indentLength`, making stack-top the parent.
 */
export function buildTaskTree(
  tasks: Array<{ lineNumber: number; indent: string; checked: boolean }>
): TaskNode[] {
  const roots: TaskNode[] = [];
  const stack: TaskNode[] = [];

  for (const task of tasks) {
    const node: TaskNode = {
      lineNumber: task.lineNumber,
      indentLength: task.indent.length,
      nestLevel: computeNestLevel(task.indent),
      checked: task.checked,
      children: [],
      parent: null,
    };

    // Pop stack until we find a parent with strictly less indent
    while (stack.length > 0 && stack[stack.length - 1].indentLength >= node.indentLength) {
      stack.pop();
    }

    if (stack.length > 0) {
      const parent = stack[stack.length - 1];
      node.parent = parent;
      parent.children.push(node);
    } else {
      roots.push(node);
    }

    stack.push(node);
  }

  return roots;
}

/**
 * Find a node in the tree by line number.
 */
export function findNodeByLine(roots: TaskNode[], lineNumber: number): TaskNode | null {
  for (const root of roots) {
    const found = findNodeInTree(root, lineNumber);
    if (found) return found;
  }
  return null;
}

function findNodeInTree(node: TaskNode, lineNumber: number): TaskNode | null {
  if (node.lineNumber === lineNumber) return node;
  for (const child of node.children) {
    const found = findNodeInTree(child, lineNumber);
    if (found) return found;
  }
  return null;
}

/**
 * Compute downward propagation: set all descendants to newChecked.
 */
export function computeDownwardPropagations(
  node: TaskNode,
  newChecked: boolean
): PropagationChange[] {
  const changes: PropagationChange[] = [];

  function recurse(n: TaskNode) {
    for (const child of n.children) {
      if (child.checked !== newChecked) {
        changes.push({ lineNumber: child.lineNumber, newChecked });
      }
      recurse(child);
    }
  }

  recurse(node);
  return changes;
}

/**
 * Compute upward propagation after toggling a node:
 * - Checked: if all siblings now checked → parent becomes checked, recurse up
 * - Unchecked: if parent was checked → parent becomes unchecked, recurse up
 */
export function computeUpwardPropagations(
  node: TaskNode,
  newChecked: boolean
): PropagationChange[] {
  const changes: PropagationChange[] = [];

  let current = node;
  let currentNewChecked = newChecked;

  while (current.parent) {
    const parent = current.parent;

    if (currentNewChecked) {
      // Check if all siblings (including current, using its new state) are checked
      const allSiblingsChecked = parent.children.every((child) =>
        child.lineNumber === current.lineNumber ? currentNewChecked : child.checked
      );
      if (allSiblingsChecked && !parent.checked) {
        changes.push({ lineNumber: parent.lineNumber, newChecked: true });
        current = parent;
        currentNewChecked = true;
      } else {
        break;
      }
    } else {
      // Unchecking: if parent was checked, uncheck it
      if (parent.checked) {
        changes.push({ lineNumber: parent.lineNumber, newChecked: false });
        current = parent;
        currentNewChecked = false;
      } else {
        break;
      }
    }
  }

  return changes;
}

/**
 * Get the line range (0-based indices) of a task and all its nested children.
 * Works on raw document lines, not just tasks.
 *
 * @param lines - Array of all document lines
 * @param taskLineIndex - 0-based index of the task line
 * @returns {start, end} inclusive 0-based line indices
 */
export function getSubtreeLineRange(
  lines: string[],
  taskLineIndex: number
): { start: number; end: number } {
  const taskLine = lines[taskLineIndex];
  const taskIndentMatch = taskLine.match(/^(\s*)/);
  const taskIndent = taskIndentMatch ? taskIndentMatch[1].length : 0;

  let end = taskLineIndex;
  for (let i = taskLineIndex + 1; i < lines.length; i++) {
    const line = lines[i];
    // Empty line ends the subtree
    if (line.trim() === '') break;
    const indentMatch = line.match(/^(\s*)/);
    const indent = indentMatch ? indentMatch[1].length : 0;
    // Line with indent <= task's indent ends the subtree
    if (indent <= taskIndent) break;
    end = i;
  }

  return { start: taskLineIndex, end };
}

/**
 * Get subtree line range from a CodeMirror document (1-based line numbers).
 */
export function getSubtreeLineRange1Based(
  getLineText: (lineNum: number) => string,
  totalLines: number,
  taskLineNum: number
): { startLine: number; endLine: number } {
  const taskLine = getLineText(taskLineNum);
  const taskIndentMatch = taskLine.match(/^(\s*)/);
  const taskIndent = taskIndentMatch ? taskIndentMatch[1].length : 0;

  let endLine = taskLineNum;
  for (let i = taskLineNum + 1; i <= totalLines; i++) {
    const line = getLineText(i);
    if (line.trim() === '') break;
    const indentMatch = line.match(/^(\s*)/);
    const indent = indentMatch ? indentMatch[1].length : 0;
    if (indent <= taskIndent) break;
    endLine = i;
  }

  return { startLine: taskLineNum, endLine };
}
