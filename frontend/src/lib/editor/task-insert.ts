import type { Text } from '@codemirror/state';
import type { EditorView } from '@codemirror/view';

interface ListBoundary {
  startLine: number;
  endLine: number;
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

export function insertTask(editorView: EditorView) {
  const doc = editorView.state.doc;
  const selection = editorView.state.selection.main;
  const cursorLine = doc.lineAt(selection.from);

  // Find the nearest task list by scanning upward from cursor
  // This handles cases where cursor is below or within a task list
  let nearestTaskListEnd = -1;
  for (let i = cursorLine.number; i >= 1; i--) {
    const line = doc.line(i);
    if (/^\s*[-*+]\s*\[[xX ]\]/.test(line.text)) {
      nearestTaskListEnd = i;
      break;
    }
  }

  // If we found a task list nearby, get its full boundaries
  const tasksInList: Array<{ lineNum: number; isChecked: boolean }> = [];

  if (nearestTaskListEnd > 0) {
    const boundary = findTaskListBoundary(doc, nearestTaskListEnd);

    // Find all tasks in this list boundary
    for (let i = boundary.startLine; i <= boundary.endLine; i++) {
      const line = doc.line(i);
      const match = /^(\s*[-*+]\s*)\[([xX ])\]/.exec(line.text);
      if (match) {
        tasksInList.push({
          lineNum: i,
          isChecked: match[2].toLowerCase() === 'x',
        });
      }
    }
  }

  // Find the first checked task
  const firstCheckedTask = tasksInList.find((t) => t.isChecked);

  // If there are checked tasks AND cursor is at or after the first checked task
  // (including below the list), insert the new task BEFORE the first checked task
  if (firstCheckedTask && cursorLine.number >= firstCheckedTask.lineNum) {
    const targetLine = doc.line(firstCheckedTask.lineNum);
    const text = '- [ ] \n';

    editorView.dispatch({
      changes: { from: targetLine.from, insert: text },
      // Position cursor at end of new task (before the newline)
      selection: { anchor: targetLine.from + text.length - 1 },
    });
  } else {
    // Original behavior: insert at cursor position
    const insertPos = selection.from === selection.to ? selection.from : cursorLine.to;

    const isAtLineStart = insertPos === cursorLine.from;
    const isEmptyLine = cursorLine.text.trim() === '';

    // Auf neuer Zeile einfügen, außer Zeile ist leer oder Cursor am Anfang
    const prefix = isAtLineStart || isEmptyLine ? '' : '\n';
    const text = `${prefix}- [ ] `;

    editorView.dispatch({
      changes: { from: insertPos, to: insertPos, insert: text },
      selection: { anchor: insertPos + text.length },
    });
  }

  editorView.focus();
}
