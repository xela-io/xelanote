import { syntaxTree } from '@codemirror/language';
import type { EditorView } from '@codemirror/view';

const LIST_ITEM_RE = /^\s*(?:[-*+]|\d+[.)])\s+/;
const TASK_LINE_RE = /^(\s*(?:[-*+]|\d+[.)]) )(\[[xX ]\])(\s+)(.*)$/;

export interface CollapsedTaskGroup {
  key: string;
  firstHiddenLine: number;
  lastHiddenLine: number;
  hiddenLineSet: Set<number>;
  completedCount: number;
  expanded: boolean;
}

export interface CollapseInfo {
  groupsByLine: Map<number, CollapsedTaskGroup>;
  keys: Set<string>;
}

export function collectCollapseInfo(view: EditorView, expandedGroups: Set<string>): CollapseInfo {
  const tree = syntaxTree(view.state);
  const doc = view.state.doc;
  const groupsByLine = new Map<number, CollapsedTaskGroup>();
  const keys = new Set<string>();

  tree.iterate({
    enter: (node) => {
      if (node.type.name !== 'BulletList' && node.type.name !== 'OrderedList') return;

      const listCursor = node.node.cursor();
      const taskItems: Array<{ line: number; checked: boolean }> = [];
      if (!listCursor.firstChild()) return;

      do {
        if (listCursor.type.name !== 'ListItem') continue;
        const line = doc.lineAt(listCursor.from);
        const lineText = line.text;

        const itemCursor = listCursor.node.cursor();
        let markerFrom = -1;
        let markerTo = -1;
        let checked = false;
        if (itemCursor.firstChild()) {
          do {
            if (itemCursor.type.name === 'TaskMarker') {
              markerFrom = itemCursor.from;
              markerTo = itemCursor.to;
              checked = doc.sliceString(itemCursor.from, itemCursor.to).toLowerCase() === '[x]';
              break;
            }
          } while (itemCursor.nextSibling());
        }

        if (markerFrom === -1 || markerTo === -1) continue;
        const taskBody = lineText.slice(markerTo - line.from).trim();
        if (taskBody.length === 0) continue;
        taskItems.push({ line: line.number, checked });
      } while (listCursor.nextSibling());

      let trailingChecked = 0;
      for (let i = taskItems.length - 1; i >= 0; i--) {
        if (!taskItems[i].checked) break;
        trailingChecked++;
      }

      if (trailingChecked === 0) return;

      const hiddenLines = taskItems.slice(taskItems.length - trailingChecked).map((item) => item.line);
      const firstHiddenLine = hiddenLines[0];
      const lastHiddenLine = hiddenLines[hiddenLines.length - 1];
      const key = `${node.from}:${node.to}:${firstHiddenLine}`;
      keys.add(key);
      const group: CollapsedTaskGroup = {
        key,
        firstHiddenLine,
        lastHiddenLine,
        hiddenLineSet: new Set(hiddenLines),
        completedCount: hiddenLines.length,
        expanded: expandedGroups.has(key),
      };
      for (const lineNumber of hiddenLines) {
        groupsByLine.set(lineNumber, group);
      }
    },
  });

  const lines: string[] = [];
  for (let i = 1; i <= view.state.doc.lines; i++) {
    lines.push(view.state.doc.line(i).text);
  }

  let i = 0;
  while (i < lines.length) {
    if (!LIST_ITEM_RE.test(lines[i])) {
      i++;
      continue;
    }

    const blockStart = i + 1;
    let blockEndIndex = i;
    while (blockEndIndex + 1 < lines.length && LIST_ITEM_RE.test(lines[blockEndIndex + 1])) {
      blockEndIndex++;
    }
    const blockEnd = blockEndIndex + 1;

    const taskLines: Array<{ line: number; checked: boolean }> = [];
    for (let line = blockStart; line <= blockEnd; line++) {
      const text = lines[line - 1];
      const match = TASK_LINE_RE.exec(text);
      if (!match) continue;
      const taskBody = match[4].trim();
      if (!taskBody) continue;
      taskLines.push({ line, checked: match[2].toLowerCase() === '[x]' });
    }

    let trailingChecked = 0;
    for (let taskIndex = taskLines.length - 1; taskIndex >= 0; taskIndex--) {
      if (!taskLines[taskIndex].checked) break;
      trailingChecked++;
    }

    if (trailingChecked > 0) {
      const hiddenLines = taskLines.slice(taskLines.length - trailingChecked).map((task) => task.line);
      const firstHiddenLine = hiddenLines[0];
      const lastHiddenLine = hiddenLines[hiddenLines.length - 1];
      const key = `${blockStart}:${blockEnd}:${firstHiddenLine}`;
      keys.add(key);
      const group: CollapsedTaskGroup = {
        key,
        firstHiddenLine,
        lastHiddenLine,
        hiddenLineSet: new Set(hiddenLines),
        completedCount: hiddenLines.length,
        expanded: expandedGroups.has(key),
      };
      for (const lineNumber of hiddenLines) {
        groupsByLine.set(lineNumber, group);
      }
    }

    i = blockEndIndex + 1;
  }

  return { groupsByLine, keys };
}
