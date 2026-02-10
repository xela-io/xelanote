import type { Text } from '@codemirror/state';
import type { EditorView } from '@codemirror/view';

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
    if (/^\s*[-*+]/.test(text)) {
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
    if (/^\s*[-*+]/.test(text)) {
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
    if (/^\s*[-*+]/.test(text)) {
      startLine = i;
    } else {
      break;
    }
  }

  // Scan downward
  for (let i = taskLineIndex + 1; i < lines.length; i++) {
    const text = lines[i];
    if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;
    if (/^\s*[-*+]/.test(text)) {
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
  // Sort tasks by line number (they should already be, but ensure it)
  const sortedTasks = [...tasksInList].sort((a, b) => a.lineNum - b.lineNum);

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
  // Match: -, *, + followed by [ ] or [x] or [X]
  const lines = content.split('\n');
  const tasks: TaskInfo[] = [];

  let charOffset = 0;
  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const line = lines[lineIndex];
    const lineMatch = /^(\s*[-*+]\s*)\[([xX ])\]/.exec(line);
    if (lineMatch) {
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

  // Calculate target position
  const targetLineNum = calculateTargetPosition(tasksInList, task, checked, log);
  log?.('[TaskSort] Current line:', task.lineNum, 'Target line:', targetLineNum);

  // Check if we need to move the line
  const needsMove = targetLineNum !== task.lineNum;
  log?.('[TaskSort] Needs move:', needsMove);

  if (useEditorView) {
    // CodeMirror mode - apply changes via dispatch
    const doc = editorView!.state.doc;

    if (!needsMove) {
      editorView!.dispatch({
        changes: { from: task.from, to: task.to, insert: newCheckboxText },
      });
    } else {
      const currentLine = doc.line(task.lineNum);
      const lineText = currentLine.text;
      const toggledLineText =
        lineText.substring(0, task.from - currentLine.from) +
        newCheckboxText +
        lineText.substring(task.to - currentLine.from);

      if (targetLineNum < 1 || targetLineNum > doc.lines) {
        editorView!.dispatch({
          changes: { from: task.from, to: task.to, insert: newCheckboxText },
        });
        scheduleAutoSave();
        return;
      }

      let changes: { from: number; to: number; insert: string }[];

      if (targetLineNum < task.lineNum) {
        const targetLine = doc.line(targetLineNum);
        const insertPos = targetLine.from;
        const deleteFrom = currentLine.from;
        const deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);
        changes = [
          { from: insertPos, to: insertPos, insert: toggledLineText + '\n' },
          { from: deleteFrom, to: deleteTo, insert: '' },
        ];
      } else {
        const targetLine = doc.line(targetLineNum);
        let deleteFrom: number;
        let deleteTo: number;

        if (currentLine.from > 0) {
          deleteFrom = currentLine.from - 1;
          deleteTo = currentLine.to;
        } else {
          deleteFrom = currentLine.from;
          deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);
        }

        const insertPos = targetLine.to;
        changes = [
          { from: deleteFrom, to: deleteTo, insert: '' },
          { from: insertPos, to: insertPos, insert: '\n' + toggledLineText },
        ];
      }

      editorView!.dispatch({
        changes,
        scrollIntoView: true,
      });
    }
  } else {
    // Preview mode - manipulate content string directly
    const currentLineIndex = task.lineNum - 1;
    const currentLineText = lines[currentLineIndex];

    // Toggle the checkbox in the line
    const toggledLineText = currentLineText.replace(/\[([xX ])\]/, newCheckboxText);

    if (!needsMove) {
      // Just toggle, no move
      lines[currentLineIndex] = toggledLineText;
    } else {
      // Remove current line and insert at target position
      const targetLineIndex = targetLineNum - 1;

      // Remove the line
      lines.splice(currentLineIndex, 1);

      // When moving down: we want to insert AFTER the target element
      // splice(index, 0, item) inserts BEFORE index, so we need targetLineIndex
      // (which after removal is the position AFTER the shifted target element)
      // When moving up: we want to insert BEFORE the target, so just use targetLineIndex
      const newTargetIndex = targetLineIndex;

      log?.(
        '[TaskSort] Moving line from index',
        currentLineIndex,
        'to index',
        newTargetIndex
      );

      // Insert at new position
      lines.splice(newTargetIndex, 0, toggledLineText);
    }

    // Update the note content
    const newContent = lines.join('\n');
    log?.('[TaskSort] Updating note content, new length:', newContent.length);
    setContent(newContent);
  }

  // Queue task event for sending after next successful save
  const taskLine = lines[task.lineNum - 1];
  const taskText = taskLine.replace(/^\s*[-*+]\s*\[[xX ]\]\s*/, '').trim();
  if (taskText && noteId) {
    queueTaskEvent(noteId, taskText.substring(0, 500), checkboxIndex, checked ? 'completed' : 'reopened');
  }

  // Trigger auto-save
  scheduleAutoSave();
}
