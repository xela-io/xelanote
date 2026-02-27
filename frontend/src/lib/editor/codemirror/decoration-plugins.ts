// CodeMirror decoration plugins: wikilink, color tag, task bracket, due date, list indent

import { Compartment } from '@codemirror/state';
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from '@codemirror/view';

import { FEATURE_FLAGS } from '$lib/config';

import { emptyExtension } from '../focus-mode-extensions';
import { isValidDueDate } from '../markdown';

// Wikilink decoration
const wikilinkMatcher = /\[\[([^\]|]+)(\|[^\]]+)?\]\]/g;
const wikilinkDecoration = Decoration.mark({ class: 'cm-wikilink' });

function getWikilinkDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = wikilinkMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }
  }

  return Decoration.set(
    decorations.map((d) => wikilinkDecoration.range(d.from, d.to)),
    true
  );
}

export const wikilinkPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getWikilinkDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getWikilinkDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Color tag decoration
const colorOpenMatcher = /\{color:[^}]+\}/g;
const colorCloseMatcher = /\{\/color\}/g;
const colorTagDecoration = Decoration.mark({ class: 'cm-color-tag' });

function getColorTagDecorations(view: EditorView): DecorationSet {
  if (!FEATURE_FLAGS.colorSyntax) {
    return Decoration.set([]);
  }

  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);

    let match;
    while ((match = colorOpenMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }

    while ((match = colorCloseMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }
  }

  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => colorTagDecoration.range(d.from, d.to)),
    true
  );
}

export const colorTagPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getColorTagDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getColorTagDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Task bracket decoration - highlights [ ] and [x] checkboxes with accent color
const taskBracketDecoration = Decoration.mark({ class: 'cm-task-bracket' });

function getTaskBracketDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];
  const taskBracketMatcher = /\[[ xX]\]/g;

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = taskBracketMatcher.exec(text)) !== null) {
      const absFrom = from + match.index;
      const line = view.state.doc.lineAt(absFrom);
      if (/^\s*[-*+]\s/.test(line.text) || /^\s*\d+[.)]\s/.test(line.text)) {
        decorations.push({
          from: absFrom,
          to: absFrom + match[0].length,
        });
      }
    }
  }

  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => taskBracketDecoration.range(d.from, d.to)),
    true
  );
}

export const taskBracketPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getTaskBracketDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getTaskBracketDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Due date decoration - highlights @due(YYYY-MM-DD) syntax
const dueDateDecoration = Decoration.mark({ class: 'cm-due-date' });
const dueDateMatcher = /@due\((\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\)/g;

/** The live preview compartment, shared with the main editor module. */
export const livePreviewCompartment = new Compartment();

function getDueDateDecorations(view: EditorView): DecorationSet {
  if (!FEATURE_FLAGS.dueDateSyntax) {
    return Decoration.set([]);
  }

  const livePreviewExt = livePreviewCompartment.get(view.state);
  const livePreviewEnabled = livePreviewExt !== undefined && livePreviewExt !== emptyExtension;
  const activeLines = new Set<number>();
  if (livePreviewEnabled) {
    for (const range of view.state.selection.ranges) {
      const fromLine = view.state.doc.lineAt(range.from).number;
      const toLine = view.state.doc.lineAt(range.to).number;
      for (let line = fromLine; line <= toLine; line += 1) {
        activeLines.add(line);
      }
    }
  }

  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = dueDateMatcher.exec(text)) !== null) {
      const matchFrom = from + match.index;
      const lineNumber = view.state.doc.lineAt(matchFrom).number;
      if (livePreviewEnabled && !activeLines.has(lineNumber)) {
        continue;
      }
      const dateStr = match[1];
      if (isValidDueDate(dateStr)) {
        decorations.push({
          from: matchFrom,
          to: matchFrom + match[0].length,
        });
      }
    }
  }

  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => dueDateDecoration.range(d.from, d.to)),
    true
  );
}

export const dueDatePlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getDueDateDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getDueDateDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// List hanging indent via inline styles
const listIndentPattern = /^(\s*)([-*+]|\d+[.)])\s/;
const taskListPattern = /^(\s*)([-*+]|\d+[.)])\s\[[ xX]\]\s/;
const blankLinePattern = /^\s*$/;

import { computeNestLevel } from '../live-preview/line-primitives';

function buildNestPart(rawIndent: string): string {
  const nestLevel = computeNestLevel(rawIndent);
  return nestLevel > 0 ? ` + ${nestLevel} * var(--live-preview-nest-indent)` : '';
}

function buildListIndentDecorations(view: EditorView): DecorationSet {
  const items: Array<{ pos: number; style: string }> = [];

  for (const { from, to } of view.visibleRanges) {
    let pos = from;
    // Track nest level of the current task item for continuation lines
    let inTaskItemNestLevel = -1; // -1 = not in a task item
    while (pos <= to) {
      const line = view.state.doc.lineAt(pos);
      const text = line.text;

      if (blankLinePattern.test(text)) {
        inTaskItemNestLevel = -1;
        pos = line.to + 1;
        continue;
      }

      const taskMatch = taskListPattern.exec(text);
      if (taskMatch) {
        const nestPart = buildNestPart(taskMatch[1]);
        items.push({
          pos: line.from,
          style: `padding-left: calc(var(--live-preview-marker-column-width) + var(--live-preview-marker-gap)${nestPart}); text-indent: calc(-1 * (var(--live-preview-marker-column-width) + var(--live-preview-marker-gap)));`,
        });
        inTaskItemNestLevel = computeNestLevel(taskMatch[1]);
        pos = line.to + 1;
        continue;
      }

      if (inTaskItemNestLevel >= 0 && !listIndentPattern.test(text)) {
        const nestPart =
          inTaskItemNestLevel > 0
            ? ` + ${inTaskItemNestLevel} * var(--live-preview-nest-indent)`
            : '';
        items.push({
          pos: line.from,
          style: `padding-left: calc(var(--live-preview-marker-column-width) + var(--live-preview-marker-gap)${nestPart});`,
        });
        pos = line.to + 1;
        continue;
      }

      inTaskItemNestLevel = -1;
      const match = listIndentPattern.exec(text);
      if (match) {
        items.push({
          pos: line.from,
          style: `padding-left: ${match[0].length}ch; text-indent: -${match[0].length}ch;`,
        });
      }
      pos = line.to + 1;
    }
  }

  return Decoration.set(
    items.map((item) =>
      Decoration.line({
        attributes: {
          style: item.style,
        },
      }).range(item.pos)
    ),
    true
  );
}

export const listIndentPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = buildListIndentDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = buildListIndentDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// First-line title decoration — always applied (edit, split, live modes)
const firstLineTitleDecoration = Decoration.line({ class: 'cm-first-line-title' });

function getFirstLineTitleDecoration(view: EditorView): DecorationSet {
  if (view.state.doc.lines === 0) return Decoration.set([]);
  const firstLine = view.state.doc.line(1);
  if (firstLine.length === 0) return Decoration.set([]);
  return Decoration.set([firstLineTitleDecoration.range(firstLine.from)]);
}

export const firstLineTitlePlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getFirstLineTitleDecoration(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged) {
        this.decorations = getFirstLineTitleDecoration(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);
