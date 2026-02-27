import type { Text } from '@codemirror/state';
import type { EditorView } from '@codemirror/view';

import { computeNestLevel } from './live-preview/line-primitives';
import {
  buildTaskTree,
  computeDownwardPropagations,
  computeUpwardPropagations,
  findNodeByLine,
  getSubtreeLineRange,
} from './task-nesting';

interface TaskInfo {
  lineNum: number;
  from: number;
  to: number;
  isChecked: boolean;
  lineFrom: number;
  lineTo: number;
  indent: string;
}

interface ListBoundary {
  startLine: number;
  endLine: number;
}

interface ToggleTaskOptions {
  editorView?: EditorView | null;
  checkboxIndex: number;
  checked: boolean;
  getContent: () => string;
  setContent: (content: string) => void;
  scheduleAutoSave: () => void;
  queueTaskEvent: (
    noteId: string,
    taskText: string,
    index: number,
    status: 'completed' | 'reopened'
  ) => void;
  noteId?: string;
  log?: (...args: unknown[]) => void;
}

interface ToggleTaskByLineOptions extends Omit<ToggleTaskOptions, 'checkboxIndex'> {
  lineNumber: number;
}

function findTaskCheckboxIndexByLine(content: string, lineNumber: number): number {
  const lines = content.split('\n');
  let taskIndex = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineMatch = /^(\s*(?:[-*+]|\d+[.)]) )\[([xX ])\]/.exec(line);
    if (!lineMatch) continue;
    const taskBody = line.substring(lineMatch[0].length).trim();
    if (!taskBody) continue;
    if (i + 1 === lineNumber) return taskIndex;
    taskIndex++;
  }

  return -1;
}

// Matches any list item marker (unordered: -, *, + or ordered: 1. / 1))
const LIST_ITEM_RE = /^\s*(?:[-*+]|\d+[.)])/;

function findTaskListBoundary(doc: Text, taskLineNum: number): ListBoundary {
  let startLine = taskLineNum;
  let endLine = taskLineNum;

  // Scan upward - find all contiguous list items
  for (let i = taskLineNum - 1; i >= 1; i--) {
    const line = doc.line(i);
    const text = line.text;

    // Empty line or heading = boundary
    if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;

    // Any list item continues the list
    if (LIST_ITEM_RE.test(text)) {
      startLine = i;
    } else {
      // Non-list, non-empty line = boundary
      break;
    }
  }

  // Scan downward - find all contiguous list items
  for (let i = taskLineNum + 1; i <= doc.lines; i++) {
    const line = doc.line(i);
    const text = line.text;

    // Empty line or heading = boundary
    if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;

    // Any list item continues the list
    if (LIST_ITEM_RE.test(text)) {
      endLine = i;
    } else {
      // Non-list, non-empty line = boundary
      break;
    }
  }

  return { startLine, endLine };
}

function findTaskListBoundaryFromString(lines: string[], taskLineIndex: number): ListBoundary {
  let startLine = taskLineIndex;
  let endLine = taskLineIndex;

  // Scan upward
  for (let i = taskLineIndex - 1; i >= 0; i--) {
    const text = lines[i];
    if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;
    if (LIST_ITEM_RE.test(text)) {
      startLine = i;
    } else {
      break;
    }
  }

  // Scan downward
  for (let i = taskLineIndex + 1; i < lines.length; i++) {
    const text = lines[i];
    if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;
    if (LIST_ITEM_RE.test(text)) {
      endLine = i;
    } else {
      break;
    }
  }

  // Convert to 1-based line numbers for consistency
  return { startLine: startLine + 1, endLine: endLine + 1 };
}

function calculateTargetPosition(
  tasksInList: TaskInfo[],
  currentTask: TaskInfo,
  isNowChecked: boolean,
  log?: (...args: unknown[]) => void
): number {
  // Only top-level tasks participate in sort ordering.
  // Child tasks (nestLevel > 0) stay in place — they move with their parent.
  if (computeNestLevel(currentTask.indent) > 0) {
    return currentTask.lineNum;
  }

  // Filter to top-level tasks only for sort calculation
  const topLevelTasks = tasksInList.filter((t) => computeNestLevel(t.indent) === 0);

  // Sort tasks by line number (they should already be, but ensure it)
  const sortedTasks = [...topLevelTasks].sort((a, b) => a.lineNum - b.lineNum);

  if (sortedTasks.length === 0) return currentTask.lineNum;

  // Create the target order: unchecked first (preserve relative order), then checked
  // Apply the NEW checked state to the current task
  const withNewState = sortedTasks.map((t) => ({
    ...t,
    isChecked: t.lineNum === currentTask.lineNum ? isNowChecked : t.isChecked,
  }));

  const uncheckedInOrder = withNewState.filter((t) => !t.isChecked);
  const checkedInOrder = withNewState.filter((t) => t.isChecked);
  const targetOrder = [...uncheckedInOrder, ...checkedInOrder];

  // Find the index of the current task in the target order
  const targetIndex = targetOrder.findIndex((t) => t.lineNum === currentTask.lineNum);

  if (targetIndex === -1 || targetIndex >= sortedTasks.length) return currentTask.lineNum;

  log?.(
    '[TaskSort] calculateTargetPosition:',
    'unchecked:',
    uncheckedInOrder.length,
    'checked:',
    checkedInOrder.length,
    'targetIndex:',
    targetIndex,
    'sortedTasks[targetIndex].lineNum:',
    sortedTasks[targetIndex].lineNum
  );

  // The target line is the line of the task currently at that index
  return sortedTasks[targetIndex].lineNum;
}

/**
 * Apply checkbox text replacement to a line string.
 */
function toggleCheckboxInLine(line: string, newChecked: boolean): string {
  const replacement = newChecked ? '[x]' : '[ ]';
  return line.replace(/\[([xX ])\]/, replacement);
}

/**
 * Collect all propagation changes (upward + downward) for a toggle.
 */
function collectPropagationChanges(
  tasksInBoundary: TaskInfo[],
  task: TaskInfo,
  checked: boolean
): Array<{ lineNum: number; newChecked: boolean }> {
  const treeInput = tasksInBoundary.map((t) => ({
    lineNumber: t.lineNum,
    indent: t.indent,
    checked: t.isChecked,
  }));

  const roots = buildTaskTree(treeInput);
  const node = findNodeByLine(roots, task.lineNum);
  if (!node) return [];

  const downward = computeDownwardPropagations(node, checked);
  const upward = computeUpwardPropagations(node, checked);

  return [
    ...downward.map((c) => ({ lineNum: c.lineNumber, newChecked: c.newChecked })),
    ...upward.map((c) => ({ lineNum: c.lineNumber, newChecked: c.newChecked })),
  ];
}

export function toggleTaskByIndex(options: ToggleTaskOptions) {
  const {
    editorView,
    checkboxIndex,
    checked,
    getContent,
    setContent,
    scheduleAutoSave,
    queueTaskEvent,
    noteId,
    log,
  } = options;

  log?.('[TaskSort] toggleTaskByIndex called, index:', checkboxIndex, 'checked:', checked);

  const useEditorView = editorView != null;
  const content = useEditorView ? editorView!.state.doc.toString() : getContent();

  if (!content) {
    log?.('[TaskSort] No content available, returning');
    return;
  }

  // Find all task items in the markdown source
  // Match: -, *, +, or ordered (1. / 1)) followed by [ ] or [x] or [X]
  const lines = content.split('\n');
  const tasks: TaskInfo[] = [];

  let charOffset = 0;
  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const line = lines[lineIndex];
    const lineMatch = /^(\s*(?:[-*+]|\d+[.)]) )\[([xX ])\]/.exec(line);
    if (lineMatch) {
      const taskBody = line.substring(lineMatch[0].length).trim();
      // Keep index mapping aligned with markdown-it-task-lists:
      // empty task items ("- [ ] ") are not rendered as checkboxes.
      if (!taskBody) {
        charOffset += line.length + 1;
        continue;
      }
      const indentMatch = line.match(/^(\s*)/);
      tasks.push({
        lineNum: lineIndex + 1, // 1-based
        from: charOffset + lineMatch[1].length,
        to: charOffset + lineMatch[1].length + 3,
        isChecked: lineMatch[2].toLowerCase() === 'x',
        lineFrom: charOffset,
        lineTo: charOffset + line.length,
        indent: indentMatch ? indentMatch[1] : '',
      });
    }
    charOffset += line.length + 1; // +1 for newline
  }

  log?.('[TaskSort] Found', tasks.length, 'tasks in document');

  if (checkboxIndex < 0 || checkboxIndex >= tasks.length) {
    log?.('[TaskSort] Invalid checkbox index, returning');
    return;
  }

  const task = tasks[checkboxIndex];
  const newCheckboxText = checked ? '[x]' : '[ ]';
  const queuedTaskText =
    lines[task.lineNum - 1]
      ?.replace(/^\s*(?:[-*+]|\d+[.)]) \[[xX ]\]\s*/, '')
      .trim()
      .substring(0, 500) ?? '';

  // Find list boundaries
  const boundary = useEditorView
    ? findTaskListBoundary(editorView!.state.doc, task.lineNum)
    : findTaskListBoundaryFromString(lines, task.lineNum - 1);
  log?.('[TaskSort] List boundary:', boundary);

  // Find all tasks within this list boundary
  const tasksInList = tasks.filter(
    (t) => t.lineNum >= boundary.startLine && t.lineNum <= boundary.endLine
  );
  log?.('[TaskSort] Tasks in list:', tasksInList.length);

  // Compute propagation changes (children + parents)
  const propagationChanges = collectPropagationChanges(tasksInList, task, checked);

  // Determine which task drives the sort position.
  // If the clicked task is a child and propagation changed a top-level ancestor,
  // the ancestor determines the move target (subtree moves as a unit).
  let moveTask = task;
  let moveTaskNewChecked = checked;

  if (computeNestLevel(task.indent) > 0) {
    let bestNestLevel = Infinity;
    for (const prop of propagationChanges) {
      const propTask = tasksInList.find((t) => t.lineNum === prop.lineNum);
      if (!propTask) continue;
      const nl = computeNestLevel(propTask.indent);
      if (nl < bestNestLevel) {
        bestNestLevel = nl;
        moveTask = propTask;
        moveTaskNewChecked = prop.newChecked;
      }
    }
  }

  // Calculate target position (only for top-level tasks)
  const targetLineNum = calculateTargetPosition(tasksInList, moveTask, moveTaskNewChecked, log);
  log?.('[TaskSort] Move task line:', moveTask.lineNum, 'Target line:', targetLineNum);

  // Check if we need to move the line
  const needsMove = targetLineNum !== moveTask.lineNum;
  log?.('[TaskSort] Needs move:', needsMove);

  if (useEditorView) {
    // CodeMirror mode - apply changes via dispatch
    const doc = editorView!.state.doc;

    // Build all checkbox toggle changes (own + propagation)
    const toggleChanges: { from: number; to: number; insert: string }[] = [];

    // Own toggle
    toggleChanges.push({ from: task.from, to: task.to, insert: newCheckboxText });

    // Propagation toggles
    for (const prop of propagationChanges) {
      const propTask = tasksInList.find((t) => t.lineNum === prop.lineNum);
      if (propTask) {
        const propText = prop.newChecked ? '[x]' : '[ ]';
        toggleChanges.push({ from: propTask.from, to: propTask.to, insert: propText });
      }
    }

    if (!needsMove) {
      // Sort changes by position descending to avoid offset issues
      toggleChanges.sort((a, b) => a.from - b.from);
      editorView!.dispatch({ changes: toggleChanges });
    } else {
      // Subtree-aware move: get the full range of the moveTask's subtree
      const subtreeRange = getSubtreeLineRange(lines, moveTask.lineNum - 1);
      const subtreeStartLine = subtreeRange.start + 1; // to 1-based
      const subtreeEndLine = subtreeRange.end + 1;
      const subtreeLineCount = subtreeEndLine - subtreeStartLine + 1;

      // Build the toggled subtree text
      const subtreeLines: string[] = [];
      for (let i = subtreeStartLine; i <= subtreeEndLine; i++) {
        let lineText = lines[i - 1];
        // Apply direct toggle to the clicked task
        if (i === task.lineNum) {
          lineText = toggleCheckboxInLine(lineText, checked);
        }
        // Apply propagation toggles
        const prop = propagationChanges.find((p) => p.lineNum === i);
        if (prop) {
          lineText = toggleCheckboxInLine(lineText, prop.newChecked);
        }
        subtreeLines.push(lineText);
      }
      const subtreeText = subtreeLines.join('\n');

      if (targetLineNum < 1 || targetLineNum > doc.lines) {
        // Fall back to just toggling
        toggleChanges.sort((a, b) => a.from - b.from);
        editorView!.dispatch({ changes: toggleChanges });
        scheduleAutoSave();
        return;
      }

      // Get target's subtree range to determine insertion point
      const targetSubtreeRange = getSubtreeLineRange(lines, targetLineNum - 1);
      const targetSubtreeEndLine = targetSubtreeRange.end + 1;

      const currentFirstLine = doc.line(subtreeStartLine);
      const currentLastLine = doc.line(subtreeEndLine);

      let changes: { from: number; to: number; insert: string }[];

      if (targetLineNum < moveTask.lineNum) {
        // Moving up
        const targetLine = doc.line(targetLineNum);
        const insertPos = targetLine.from;
        const deleteFrom = currentFirstLine.from;
        const deleteTo = currentLastLine.to + (currentLastLine.to < doc.length ? 1 : 0);

        // Also apply propagation changes to lines NOT in the subtree
        const externalPropChanges: { from: number; to: number; insert: string }[] = [];
        for (const prop of propagationChanges) {
          if (prop.lineNum >= subtreeStartLine && prop.lineNum <= subtreeEndLine) continue;
          const propTask = tasksInList.find((t) => t.lineNum === prop.lineNum);
          if (propTask) {
            externalPropChanges.push({
              from: propTask.from,
              to: propTask.to,
              insert: prop.newChecked ? '[x]' : '[ ]',
            });
          }
        }

        changes = [
          { from: insertPos, to: insertPos, insert: subtreeText + '\n' },
          { from: deleteFrom, to: deleteTo, insert: '' },
          ...externalPropChanges,
        ];
      } else {
        // Moving down: insert after target's subtree
        const targetEndLine = doc.line(targetSubtreeEndLine);

        let deleteFrom: number;
        let deleteTo: number;

        if (currentFirstLine.from > 0) {
          deleteFrom = currentFirstLine.from - 1;
          deleteTo = currentLastLine.to;
        } else {
          deleteFrom = currentFirstLine.from;
          deleteTo =
            currentLastLine.to + (currentLastLine.to < doc.length ? 1 : 0);
        }

        const insertPos = targetEndLine.to;

        // Also apply propagation changes to lines NOT in the subtree
        const externalPropChanges: { from: number; to: number; insert: string }[] = [];
        for (const prop of propagationChanges) {
          if (prop.lineNum >= subtreeStartLine && prop.lineNum <= subtreeEndLine) continue;
          const propTask = tasksInList.find((t) => t.lineNum === prop.lineNum);
          if (propTask) {
            externalPropChanges.push({
              from: propTask.from,
              to: propTask.to,
              insert: prop.newChecked ? '[x]' : '[ ]',
            });
          }
        }

        changes = [
          { from: deleteFrom, to: deleteTo, insert: '' },
          { from: insertPos, to: insertPos, insert: '\n' + subtreeText },
          ...externalPropChanges,
        ];
      }

      // Sort changes by position for CodeMirror
      changes.sort((a, b) => a.from - b.from);
      editorView!.dispatch({ changes });
    }
  } else {
    // Preview mode - manipulate content string directly
    const currentLineIndex = task.lineNum - 1;
    const moveLineIndex = moveTask.lineNum - 1;

    // Apply propagation toggles first (before moving lines)
    for (const prop of propagationChanges) {
      const propLineIndex = prop.lineNum - 1;
      lines[propLineIndex] = toggleCheckboxInLine(lines[propLineIndex], prop.newChecked);
    }

    // Toggle the checkbox in the clicked task line
    lines[currentLineIndex] = toggleCheckboxInLine(lines[currentLineIndex], checked);

    if (!needsMove) {
      // No move needed — changes already applied above
    } else {
      // Subtree-aware move using the moveTask's subtree
      const subtreeRange = getSubtreeLineRange(lines, moveLineIndex);
      const subtreeLineCount = subtreeRange.end - subtreeRange.start + 1;

      // Extract the subtree lines
      const subtreeLines = lines.splice(subtreeRange.start, subtreeLineCount);

      // Recalculate target index after removal
      const targetLineIndex = targetLineNum - 1;
      // After splice, the target shifts if it was after the removed block
      const adjustedTarget =
        targetLineIndex > subtreeRange.start
          ? targetLineIndex - subtreeLineCount
          : targetLineIndex;

      // For moving down, calculate target's subtree end in the modified array
      let insertAt: number;
      if (targetLineNum > moveTask.lineNum) {
        // Moving down: insert after target's subtree
        const targetSubtreeRange = getSubtreeLineRange(lines, adjustedTarget);
        insertAt = targetSubtreeRange.end + 1;
      } else {
        // Moving up: insert before target
        insertAt = adjustedTarget;
      }

      log?.('[TaskSort] Moving subtree from', subtreeRange.start, 'to', insertAt);

      // Insert at new position
      lines.splice(insertAt, 0, ...subtreeLines);
    }

    // Update the note content
    const newContent = lines.join('\n');
    log?.('[TaskSort] Updating note content, new length:', newContent.length);
    setContent(newContent);
  }

  // Queue task event for sending after next successful save
  if (queuedTaskText && noteId) {
    queueTaskEvent(noteId, queuedTaskText, checkboxIndex, checked ? 'completed' : 'reopened');
  }

  // Trigger auto-save
  scheduleAutoSave();
}

export function toggleTaskByLine(options: ToggleTaskByLineOptions) {
  const { lineNumber, editorView, getContent, ...rest } = options;
  const content = editorView ? editorView.state.doc.toString() : getContent();
  const checkboxIndex = findTaskCheckboxIndexByLine(content, lineNumber);

  if (checkboxIndex === -1) {
    options.log?.('[TaskSort] No task found for line:', lineNumber);
    return;
  }

  toggleTaskByIndex({
    editorView,
    checkboxIndex,
    getContent,
    ...rest,
  });
}
