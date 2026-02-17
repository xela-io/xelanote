import type { Extension } from '@codemirror/state';
import { RangeSetBuilder } from '@codemirror/state';
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
  WidgetType,
} from '@codemirror/view';

import { collectTreeFeatures, extractMarkdownLinksFromText } from './live-preview-features';

const lineDecorationCache = new Map<string, Decoration>();

class InlineTextWidget extends WidgetType {
  constructor(
    private readonly text: string,
    private readonly className: string,
    private readonly dataAttrs?: Record<string, string>
  ) {
    super();
  }

  eq(other: InlineTextWidget): boolean {
    return (
      this.text === other.text &&
      this.className === other.className &&
      JSON.stringify(this.dataAttrs ?? {}) === JSON.stringify(other.dataAttrs ?? {})
    );
  }

  toDOM(): HTMLElement {
    const span = document.createElement('span');
    span.className = this.className;
    span.textContent = this.text;
    if (this.dataAttrs) {
      for (const [key, value] of Object.entries(this.dataAttrs)) {
        span.dataset[key] = value;
      }
    }
    return span;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

class HiddenSyntaxWidget extends WidgetType {
  eq(): boolean {
    return true;
  }

  toDOM(): HTMLElement {
    const span = document.createElement('span');
    span.className = 'cm-live-hidden-syntax';
    span.setAttribute('aria-hidden', 'true');
    return span;
  }
}

const hiddenSyntaxDecoration = Decoration.replace({
  widget: new HiddenSyntaxWidget(),
});

function addProtectedWidget(
  builder: RangeSetBuilder<Decoration>,
  protectedRanges: Array<{ from: number; to: number }>,
  fromPos: number,
  toPos: number,
  widget: WidgetType
) {
  builder.add(
    fromPos,
    toPos,
    Decoration.replace({
      widget,
    })
  );
  protectedRanges.push({ from: fromPos, to: toPos });
}

function getLineDecoration(className: string): Decoration {
  const cached = lineDecorationCache.get(className);
  if (cached) return cached;

  const decoration = Decoration.line({ attributes: { class: className } });
  lineDecorationCache.set(className, decoration);
  return decoration;
}

class TaskCheckboxWidget extends WidgetType {
  constructor(
    private readonly checked: boolean,
    private readonly lineNumber: number
  ) {
    super();
  }

  eq(other: TaskCheckboxWidget): boolean {
    return this.checked === other.checked && this.lineNumber === other.lineNumber;
  }

  toDOM(): HTMLElement {
    const wrapper = document.createElement('span');
    wrapper.className = `cm-live-task-checkbox ${this.checked ? 'is-checked' : ''}`;
    wrapper.dataset.checked = String(this.checked);
    wrapper.dataset.line = String(this.lineNumber);
    wrapper.setAttribute('role', 'checkbox');
    wrapper.setAttribute('aria-checked', String(this.checked));
    wrapper.tabIndex = -1;
    const input = document.createElement('input');
    input.type = 'checkbox';
    input.className = 'cm-live-task-checkbox-input';
    input.checked = this.checked;
    input.tabIndex = -1;
    input.setAttribute('aria-hidden', 'true');
    wrapper.appendChild(input);
    return wrapper;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

function isCodeFence(text: string): boolean {
  return /^\s*```/.test(text);
}

function isTableSeparatorLine(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.includes('|') || !trimmed.includes('-')) return false;
  return /^[\s|:-]+$/.test(trimmed);
}

function isTableCandidateLine(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.includes('|')) return false;
  const pipeCount = (trimmed.match(/\|/g) ?? []).length;
  return pipeCount >= 1;
}

interface StructuredLines {
  codeFenceLines: Set<number>;
  codeContentLines: Set<number>;
  tableLines: Set<number>;
}

interface LivePreviewStaticData {
  structuredLines: StructuredLines;
  treeFeatures: ReturnType<typeof collectTreeFeatures>;
}

interface HeadingSection {
  key: string;
  headingLine: number;
  endLine: number;
  collapsed: boolean;
}

interface HeadingInfo {
  headingByLine: Map<number, HeadingSection>;
  sectionByLine: Map<number, HeadingSection>;
  keys: Set<string>;
}

interface LinePrimitives {
  heading: { indentLength: number; level: number; spacingLength: number } | null;
  blockquote: { indentLength: number; spacingLength: number } | null;
  markerPrefixLength: number | null;
  taskRegex: { markerLength: number; markerTokenLength: number; checked: boolean } | null;
  listMarker: { indentLength: number; marker: string; spacingLength: number } | null;
}

type LivePreviewProfilePhase = 'build' | 'tree' | 'structured';

interface LivePreviewProfileSample {
  phase: LivePreviewProfilePhase;
  reason: string;
  ms: number;
}

type LivePreviewProfilerSink = (sample: LivePreviewProfileSample) => void;

let livePreviewProfilerSink: LivePreviewProfilerSink | null = null;

function nowMs(): number {
  return typeof performance !== 'undefined' ? performance.now() : Date.now();
}

function profile<T>(phase: LivePreviewProfilePhase, reason: string, fn: () => T): T {
  const start = nowMs();
  const result = fn();
  const end = nowMs();
  livePreviewProfilerSink?.({
    phase,
    reason,
    ms: end - start,
  });
  return result;
}

export function setLivePreviewProfilerSink(sink: LivePreviewProfilerSink | null): void {
  livePreviewProfilerSink = sink;
}

function parseLinePrimitives(text: string): LinePrimitives {
  const headingMatch = /^(\s{0,3})(#{1,6})(\s+)/.exec(text);
  const blockquoteMatch = /^(\s{0,3})>(\s*)/.exec(text);
  const markerPrefixMatch = /^(\s*(?:[-*+]|\d+[.)])\s+)/.exec(text);
  const taskRegexMatch = /^(\s*(?:[-*+]|\d+[.)]) )(\[[xX ]\])(\s+)/.exec(text);
  const listMarkerMatch = /^(\s*)([-*+]|\d+[.)])(\s+)/.exec(text);

  return {
    heading: headingMatch
      ? {
          indentLength: headingMatch[1].length,
          level: headingMatch[2].length,
          spacingLength: headingMatch[3].length,
        }
      : null,
    blockquote: blockquoteMatch
      ? {
          indentLength: blockquoteMatch[1].length,
          spacingLength: blockquoteMatch[2].length,
        }
      : null,
    markerPrefixLength: markerPrefixMatch ? markerPrefixMatch[1].length : null,
    taskRegex: taskRegexMatch
      ? {
          markerLength: taskRegexMatch[1].length,
          markerTokenLength: taskRegexMatch[2].length,
          checked: taskRegexMatch[2].toLowerCase() === '[x]',
        }
      : null,
    listMarker: listMarkerMatch
      ? {
          indentLength: listMarkerMatch[1].length,
          marker: listMarkerMatch[2],
          spacingLength: listMarkerMatch[3].length,
        }
      : null,
  };
}

const STRUCTURED_LINE_TRIGGER_RE = /[`|:-]/;

function textMayAffectStructuredLines(text: string): boolean {
  return STRUCTURED_LINE_TRIGGER_RE.test(text);
}

function hasAnyLineInRange(lines: Set<number>, fromLine: number, toLine: number): boolean {
  for (let line = fromLine; line <= toLine; line++) {
    if (lines.has(line)) return true;
  }
  return false;
}

function shouldRecomputeStructuredLines(update: ViewUpdate, previous: StructuredLines): boolean {
  if (!update.docChanged) return false;
  if (update.startState.doc.lines !== update.state.doc.lines) return true;

  let needsRecompute = false;
  update.changes.iterChanges((fromA, toA, _fromB, _toB, inserted) => {
    if (needsRecompute) return;

    const removedText = update.startState.doc.sliceString(fromA, toA);
    const insertedText = inserted.toString();
    if (textMayAffectStructuredLines(removedText) || textMayAffectStructuredLines(insertedText)) {
      needsRecompute = true;
      return;
    }

    const oldDoc = update.startState.doc;
    const fromProbe = Math.min(fromA, oldDoc.length);
    const toProbe = Math.min(Math.max(toA - 1, fromA), oldDoc.length);
    const fromLine = oldDoc.lineAt(fromProbe).number;
    const toLine = oldDoc.lineAt(toProbe).number;
    const nearbyFrom = Math.max(1, fromLine - 1);
    const nearbyTo = Math.min(oldDoc.lines, toLine + 1);

    if (
      hasAnyLineInRange(previous.codeFenceLines, nearbyFrom, nearbyTo) ||
      hasAnyLineInRange(previous.tableLines, nearbyFrom, nearbyTo)
    ) {
      needsRecompute = true;
    }
  });

  return needsRecompute;
}

function collectStructuredLines(view: EditorView): StructuredLines {
  const codeFenceLines = new Set<number>();
  const codeContentLines = new Set<number>();
  const tableLines = new Set<number>();
  const lines: string[] = [];

  let inCodeBlock = false;
  for (let i = 1; i <= view.state.doc.lines; i++) {
    const text = view.state.doc.line(i).text;
    lines.push(text);

    if (isCodeFence(text)) {
      codeFenceLines.add(i);
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) {
      codeContentLines.add(i);
    }
  }

  for (let i = 0; i < lines.length; i++) {
    const lineNo = i + 1;
    if (codeFenceLines.has(lineNo) || codeContentLines.has(lineNo)) continue;

    const text = lines[i];
    if (!isTableCandidateLine(text) && !isTableSeparatorLine(text)) continue;

    const prev = i > 0 ? lines[i - 1] : '';
    const next = i < lines.length - 1 ? lines[i + 1] : '';
    const nearSeparator =
      isTableSeparatorLine(text) || isTableSeparatorLine(prev) || isTableSeparatorLine(next);

    if (nearSeparator) {
      tableLines.add(lineNo);
      if (i > 0 && isTableCandidateLine(prev)) tableLines.add(lineNo - 1);
      if (i < lines.length - 1 && isTableCandidateLine(next)) tableLines.add(lineNo + 1);
    }
  }

  return { codeFenceLines, codeContentLines, tableLines };
}

function collectHeadingInfo(view: EditorView, collapsedSections: Set<string>): HeadingInfo {
  const headingByLine = new Map<number, HeadingSection>();
  const sectionByLine = new Map<number, HeadingSection>();
  const keys = new Set<string>();
  const headings: Array<{ line: number; level: number }> = [];

  for (let lineNo = 1; lineNo <= view.state.doc.lines; lineNo++) {
    const text = view.state.doc.line(lineNo).text;
    const match = /^(\s{0,3})(#{1,6})(\s+)/.exec(text);
    if (!match) continue;
    headings.push({ line: lineNo, level: match[2].length });
  }

  for (let i = 0; i < headings.length; i++) {
    const current = headings[i];
    let endLine = view.state.doc.lines;
    for (let j = i + 1; j < headings.length; j++) {
      if (headings[j].level <= current.level) {
        endLine = headings[j].line - 1;
        break;
      }
    }

    const key = `${current.line}:${current.level}:${endLine}`;
    keys.add(key);
    const section: HeadingSection = {
      key,
      headingLine: current.line,
      endLine,
      collapsed: collapsedSections.has(key),
    };
    headingByLine.set(current.line, section);
    if (endLine > current.line) {
      for (let line = current.line + 1; line <= endLine; line++) {
        sectionByLine.set(line, section);
      }
    }
  }

  return { headingByLine, sectionByLine, keys };
}

function getActiveLines(view: EditorView): Set<number> {
  const lines = new Set<number>();

  for (const range of view.state.selection.ranges) {
    let currentLine = view.state.doc.lineAt(range.from);
    lines.add(currentLine.number);

    if (range.empty) continue;

    while (currentLine.to < range.to) {
      const nextFrom = currentLine.to + 1;
      if (nextFrom > view.state.doc.length) break;
      currentLine = view.state.doc.lineAt(nextFrom);
      lines.add(currentLine.number);
    }
  }

  return lines;
}

function activeLinesKey(lines: Set<number>): string {
  return [...lines].sort((a, b) => a - b).join(',');
}

function isInsideRanges(position: number, ranges: Array<{ from: number; to: number }>): boolean {
  return ranges.some((range) => position >= range.from && position < range.to);
}

function buildDecorations(
  view: EditorView,
  staticData: LivePreviewStaticData,
  headingInfo: HeadingInfo,
  reason: string,
  activeLines: Set<number>,
  getLinePrimitives: (lineNumber: number, text: string) => LinePrimitives
): DecorationSet {
  return profile('build', reason, () => {
    const builder = new RangeSetBuilder<Decoration>();
    const seenLines = new Set<number>();
    const doc = view.state.doc;
    const { structuredLines, treeFeatures } = staticData;

    for (const { from, to } of view.visibleRanges) {
      let pos = from;
      while (pos <= to && pos <= doc.length) {
        const line = doc.lineAt(pos);
        pos = line.to + 1;

        if (seenLines.has(line.number)) continue;
        seenLines.add(line.number);

        if (line.length === 0) continue;

        const headingSectionLine = headingInfo.sectionByLine.get(line.number);
        if (headingSectionLine?.collapsed) {
          builder.add(line.from, line.from, getLineDecoration('cm-live-collapsed-line'));
          builder.add(line.from, line.to, hiddenSyntaxDecoration);
          continue;
        }

        const headingSection = headingInfo.headingByLine.get(line.number);
        const showHeadingToggle =
          !!headingSection && headingSection.endLine > headingSection.headingLine;

        const protectedRanges: Array<{ from: number; to: number }> = [];
        const text = line.text;
        const base = line.from;
        const isActiveLine = activeLines.has(line.number);
        const lineClasses: string[] = [];

        const primitives = getLinePrimitives(line.number, text);
        const treeTask = treeFeatures.tasksByLine.get(line.number);
        const taskInfo =
          treeTask && primitives.markerPrefixLength != null
            ? {
                markerLength: primitives.markerPrefixLength,
                from: treeTask.from,
                to: treeTask.to,
                checked: treeTask.checked,
              }
            : primitives.taskRegex
              ? {
                  markerLength: primitives.taskRegex.markerLength,
                  from: base + primitives.taskRegex.markerLength,
                  to:
                    base +
                    primitives.taskRegex.markerLength +
                    primitives.taskRegex.markerTokenLength,
                  checked: primitives.taskRegex.checked,
                }
              : null;
        const listMarkerInfo = !taskInfo ? primitives.listMarker : null;

        if (!isActiveLine) {
          lineClasses.push('cm-live-preview-line');
          if (primitives.heading) {
            lineClasses.push(`cm-live-heading-h${primitives.heading.level}`);
          }
          if (primitives.blockquote) {
            lineClasses.push('cm-live-blockquote');
          }
          if (listMarkerInfo) {
            lineClasses.push('cm-live-list-item');
          }
          if (structuredLines.codeFenceLines.has(line.number)) {
            lineClasses.push('cm-live-code-fence');
          } else if (structuredLines.codeContentLines.has(line.number)) {
            lineClasses.push('cm-live-code-line');
          }
          if (structuredLines.tableLines.has(line.number)) {
            lineClasses.push('cm-live-table-line');
          }
        }

        if (lineClasses.length > 0) {
          builder.add(line.from, line.from, getLineDecoration(lineClasses.join(' ')));
        }

        if (showHeadingToggle && headingSection) {
          builder.add(
            line.from,
            line.from,
            Decoration.widget({
              side: -1,
              widget: new HeadingToggleWidget(headingSection.key, headingSection.collapsed),
            })
          );
        }

        if (!isActiveLine && taskInfo) {
          const markerFrom = base;
          const markerTo = base + taskInfo.markerLength;
          builder.add(markerFrom, markerTo, hiddenSyntaxDecoration);
        }

        if (taskInfo) {
          builder.add(
            taskInfo.from,
            taskInfo.to,
            Decoration.replace({
              widget: new TaskCheckboxWidget(taskInfo.checked, line.number),
            })
          );
        }

        if (isActiveLine) {
          continue;
        }

        if (
          structuredLines.codeFenceLines.has(line.number) ||
          structuredLines.codeContentLines.has(line.number)
        ) {
          continue;
        }
        if (structuredLines.tableLines.has(line.number)) {
          continue;
        }

        if (primitives.heading) {
          const fromPos = base + primitives.heading.indentLength;
          const toPos = fromPos + primitives.heading.level + primitives.heading.spacingLength;
          builder.add(fromPos, toPos, hiddenSyntaxDecoration);
        }

        if (primitives.blockquote) {
          const fromPos = base + primitives.blockquote.indentLength;
          const toPos = fromPos + 1 + primitives.blockquote.spacingLength;
          builder.add(fromPos, toPos, hiddenSyntaxDecoration);
        }

        if (!taskInfo) {
          if (listMarkerInfo) {
            const fromPos = base + listMarkerInfo.indentLength;
            const toPos = fromPos + listMarkerInfo.marker.length + listMarkerInfo.spacingLength;
            const marker = listMarkerInfo.marker;
            const markerWidgetText = /^\d+[.)]$/.test(marker) ? `${marker} ` : '• ';
            addProtectedWidget(
              builder,
              protectedRanges,
              fromPos,
              toPos,
              new InlineTextWidget(markerWidgetText, 'cm-live-list-marker')
            );
          }
        }

        const treeInline = treeFeatures.inlineByLine.get(line.number) ?? [];
        if (treeInline.length > 0) {
          for (const inline of treeInline) {
            addProtectedWidget(
              builder,
              protectedRanges,
              inline.from,
              inline.to,
              new InlineTextWidget(inline.text, inline.className)
            );
          }
        } else {
          if (text.includes('`')) {
            const inlineCodePattern = /`([^`\n]+)`/g;
            let inlineCodeMatch: RegExpExecArray | null;
            while ((inlineCodeMatch = inlineCodePattern.exec(text)) !== null) {
              const fromPos = base + inlineCodeMatch.index;
              const toPos = fromPos + inlineCodeMatch[0].length;
              addProtectedWidget(
                builder,
                protectedRanges,
                fromPos,
                toPos,
                new InlineTextWidget(inlineCodeMatch[1], 'cm-live-preview-code')
              );
            }
          }
        }

        const treeLinks = treeFeatures.linksByLine.get(line.number) ?? [];
        if (treeLinks.length > 0) {
          for (const link of treeLinks) {
            addProtectedWidget(
              builder,
              protectedRanges,
              link.from,
              link.to,
              new InlineTextWidget(link.label, 'cm-live-preview-link', {
                href: link.href,
              })
            );
          }
        } else {
          if (text.includes('[') && text.includes('](')) {
            const fallbackLinks = extractMarkdownLinksFromText(text, base);
            for (const link of fallbackLinks) {
              addProtectedWidget(
                builder,
                protectedRanges,
                link.from,
                link.to,
                new InlineTextWidget(link.label, 'cm-live-preview-link', {
                  href: link.href,
                })
              );
            }
          }
        }

        const treeWikilinks = treeFeatures.wikilinksByLine.get(line.number) ?? [];
        if (treeWikilinks.length > 0) {
          for (const wikilink of treeWikilinks) {
            addProtectedWidget(
              builder,
              protectedRanges,
              wikilink.from,
              wikilink.to,
              new InlineTextWidget(wikilink.label, 'cm-live-preview-wikilink', {
                title: wikilink.title,
              })
            );
          }
        } else {
          if (text.includes('[[')) {
            const wikilinkPattern = /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g;
            let wikilinkMatch: RegExpExecArray | null;
            while ((wikilinkMatch = wikilinkPattern.exec(text)) !== null) {
              const fromPos = base + wikilinkMatch.index;
              const toPos = fromPos + wikilinkMatch[0].length;
              const label = wikilinkMatch[2] ?? wikilinkMatch[1];
              addProtectedWidget(
                builder,
                protectedRanges,
                fromPos,
                toPos,
                new InlineTextWidget(label, 'cm-live-preview-wikilink', {
                  title: wikilinkMatch[1].trim(),
                })
              );
            }
          }
        }

        const treeDueDates = treeFeatures.dueDatesByLine.get(line.number) ?? [];
        if (treeDueDates.length > 0) {
          for (const dueDate of treeDueDates) {
            addProtectedWidget(
              builder,
              protectedRanges,
              dueDate.from,
              dueDate.to,
              new InlineTextWidget(dueDate.date, 'cm-live-preview-due')
            );
          }
        } else {
          if (text.includes('@due(')) {
            const dueDatePattern = /@due\((\d{4}-\d{2}-\d{2})\)/g;
            let dueDateMatch: RegExpExecArray | null;
            while ((dueDateMatch = dueDatePattern.exec(text)) !== null) {
              const fromPos = base + dueDateMatch.index;
              const toPos = fromPos + dueDateMatch[0].length;
              addProtectedWidget(
                builder,
                protectedRanges,
                fromPos,
                toPos,
                new InlineTextWidget(dueDateMatch[1], 'cm-live-preview-due')
              );
            }
          }
        }

        if (treeInline.length === 0) {
          if (text.includes('**') || text.includes('__')) {
            const strongPattern = /(\*\*|__)(.+?)\1/g;
            let strongMatch: RegExpExecArray | null;
            while ((strongMatch = strongPattern.exec(text)) !== null) {
              const fromPos = base + strongMatch.index;
              if (isInsideRanges(fromPos, protectedRanges)) continue;
              const toPos = fromPos + strongMatch[0].length;
              addProtectedWidget(
                builder,
                protectedRanges,
                fromPos,
                toPos,
                new InlineTextWidget(strongMatch[2], 'cm-live-preview-strong')
              );
            }
          }

          if (text.includes('*') || text.includes('_')) {
            const emPattern = /(?<!\*)\*([^*\n]+)\*(?!\*)|(?<!_)_([^_\n]+)_(?!_)/g;
            let emMatch: RegExpExecArray | null;
            while ((emMatch = emPattern.exec(text)) !== null) {
              const fromPos = base + emMatch.index;
              if (isInsideRanges(fromPos, protectedRanges)) continue;
              const toPos = fromPos + emMatch[0].length;
              const emText = emMatch[1] ?? emMatch[2];
              addProtectedWidget(
                builder,
                protectedRanges,
                fromPos,
                toPos,
                new InlineTextWidget(emText, 'cm-live-preview-em')
              );
            }
          }
        }

        if (text.includes('*') || text.includes('_') || text.includes('~') || text.includes('`')) {
          const syntaxPattern = /(\*\*|__|~~|`|\*|_)/g;
          let syntaxMatch: RegExpExecArray | null;
          while ((syntaxMatch = syntaxPattern.exec(text)) !== null) {
            const fromPos = base + syntaxMatch.index;
            if (isInsideRanges(fromPos, protectedRanges)) continue;
            const toPos = fromPos + syntaxMatch[0].length;
            builder.add(fromPos, toPos, hiddenSyntaxDecoration);
          }
        }
      }
    }
    return builder.finish();
  });
}

class HeadingToggleWidget extends WidgetType {
  constructor(
    private readonly sectionKey: string,
    private readonly collapsed: boolean
  ) {
    super();
  }

  eq(other: HeadingToggleWidget): boolean {
    return this.sectionKey === other.sectionKey && this.collapsed === other.collapsed;
  }

  toDOM(): HTMLElement {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'cm-live-heading-toggle';
    button.dataset.section = this.sectionKey;
    button.dataset.collapsed = String(this.collapsed);
    button.setAttribute(
      'aria-label',
      this.collapsed ? 'Expand heading section' : 'Collapse heading section'
    );
    button.textContent = this.collapsed ? '+' : '−';
    return button;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

const livePreviewPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    staticData: LivePreviewStaticData;
    headingInfo: HeadingInfo;
    linePrimitivesCache = new Map<number, { text: string; parsed: LinePrimitives }>();
    activeLines: Set<number>;
    activeLinesSignature: string;
    collapsedHeadingSections = new Set<string>();
    forceRebuild = false;
    headingInfoDirty = false;
    pendingGutterSyncFrame: number | null = null;
    gutterObserver: MutationObserver | null = null;

    constructor(view: EditorView) {
      this.staticData = this.computeStaticData(view, 'init');
      this.headingInfo = profile('structured', 'init', () =>
        collectHeadingInfo(view, this.collapsedHeadingSections)
      );
      this.activeLines = getActiveLines(view);
      this.activeLinesSignature = activeLinesKey(this.activeLines);
      this.decorations = buildDecorations(
        view,
        this.staticData,
        this.headingInfo,
        'init',
        this.activeLines,
        this.getLinePrimitives.bind(this)
      );
      this.setupGutterObserver(view);
      this.syncCollapsedGutter(view);
      this.scheduleCollapsedGutterSync(view);
    }

    update(update: ViewUpdate) {
      const reason = this.describeUpdateReason(update);
      const nextActiveLines = update.selectionSet ? getActiveLines(update.view) : this.activeLines;
      const nextActiveLinesSignature = update.selectionSet
        ? activeLinesKey(nextActiveLines)
        : this.activeLinesSignature;
      const selectionAffectsRendering =
        update.selectionSet && nextActiveLinesSignature !== this.activeLinesSignature;

      if (update.docChanged) {
        this.linePrimitivesCache.clear();
        this.staticData = {
          structuredLines: shouldRecomputeStructuredLines(update, this.staticData.structuredLines)
            ? profile('structured', reason, () => collectStructuredLines(update.view))
            : this.staticData.structuredLines,
          treeFeatures: profile('tree', reason, () => collectTreeFeatures(update.view)),
        };
        this.headingInfo = profile('structured', reason, () =>
          collectHeadingInfo(update.view, this.collapsedHeadingSections)
        );
        this.headingInfoDirty = false;
      } else if (update.viewportChanged) {
        this.staticData = {
          ...this.staticData,
          treeFeatures: profile('tree', reason, () => collectTreeFeatures(update.view)),
        };
      } else if (this.headingInfoDirty) {
        this.headingInfo = profile('structured', reason, () =>
          collectHeadingInfo(update.view, this.collapsedHeadingSections)
        );
        this.headingInfoDirty = false;
      }
      if (
        this.forceRebuild ||
        update.docChanged ||
        selectionAffectsRendering ||
        update.viewportChanged ||
        update.focusChanged
      ) {
        this.forceRebuild = false;
        this.decorations = buildDecorations(
          update.view,
          this.staticData,
          this.headingInfo,
          reason,
          nextActiveLines,
          this.getLinePrimitives.bind(this)
        );
        this.pruneHeadingSections(this.headingInfo.keys);
      }
      this.activeLines = nextActiveLines;
      this.activeLinesSignature = nextActiveLinesSignature;
      this.syncCollapsedGutter(update.view);
      this.scheduleCollapsedGutterSync(update.view);
    }

    toggleHeadingSection(view: EditorView, key: string): boolean {
      if (this.collapsedHeadingSections.has(key)) {
        this.collapsedHeadingSections.delete(key);
      } else {
        this.collapsedHeadingSections.add(key);
      }
      this.headingInfoDirty = true;
      this.forceRebuild = true;
      view.dispatch({});
      return true;
    }

    private pruneHeadingSections(validKeys: Set<string>) {
      for (const key of this.collapsedHeadingSections) {
        if (!validKeys.has(key)) {
          this.collapsedHeadingSections.delete(key);
        }
      }
    }

    private computeStaticData(view: EditorView, reason: string): LivePreviewStaticData {
      return {
        structuredLines: profile('structured', reason, () => collectStructuredLines(view)),
        treeFeatures: profile('tree', reason, () => collectTreeFeatures(view)),
      };
    }

    private getLinePrimitives(lineNumber: number, text: string): LinePrimitives {
      const cached = this.linePrimitivesCache.get(lineNumber);
      if (cached && cached.text === text) return cached.parsed;
      const parsed = parseLinePrimitives(text);
      this.linePrimitivesCache.set(lineNumber, { text, parsed });
      return parsed;
    }

    private describeUpdateReason(update: ViewUpdate): string {
      if (this.forceRebuild) return 'forceRebuild';
      if (update.docChanged) return 'docChanged';
      if (update.selectionSet) return 'selectionSet';
      if (update.viewportChanged) return 'viewportChanged';
      if (update.focusChanged) return 'focusChanged';
      return 'other';
    }

    private syncCollapsedGutter(view: EditorView): void {
      const collapsedLines = new Set<number>();

      for (const [lineNumber, section] of this.headingInfo.sectionByLine) {
        if (section.collapsed) {
          collapsedLines.add(lineNumber);
        }
      }

      const gutterElements = view.dom.querySelectorAll<HTMLElement>(
        '.cm-lineNumbers .cm-gutterElement'
      );
      for (const element of gutterElements) {
        const lineNumber = Number.parseInt(element.textContent?.trim() ?? '', 10);
        if (Number.isInteger(lineNumber) && collapsedLines.has(lineNumber)) {
          element.classList.add('cm-live-collapsed-line');
        } else {
          element.classList.remove('cm-live-collapsed-line');
        }
      }
    }

    private scheduleCollapsedGutterSync(view: EditorView): void {
      if (this.pendingGutterSyncFrame != null && typeof cancelAnimationFrame === 'function') {
        cancelAnimationFrame(this.pendingGutterSyncFrame);
      }

      if (typeof requestAnimationFrame === 'function') {
        this.pendingGutterSyncFrame = requestAnimationFrame(() => {
          this.pendingGutterSyncFrame = null;
          this.syncCollapsedGutter(view);
        });
        return;
      }

      this.syncCollapsedGutter(view);
    }

    private setupGutterObserver(view: EditorView): void {
      if (typeof MutationObserver === 'undefined') return;
      const lineNumberGutter = view.dom.querySelector('.cm-lineNumbers');
      if (!lineNumberGutter) return;

      this.gutterObserver?.disconnect();
      this.gutterObserver = new MutationObserver(() => {
        this.syncCollapsedGutter(view);
      });
      // React when CodeMirror adds/removes gutter rows during scroll/reflow.
      this.gutterObserver.observe(lineNumberGutter, { childList: true, subtree: true });
    }

    destroy(): void {
      if (this.pendingGutterSyncFrame != null && typeof cancelAnimationFrame === 'function') {
        cancelAnimationFrame(this.pendingGutterSyncFrame);
        this.pendingGutterSyncFrame = null;
      }
      this.gutterObserver?.disconnect();
      this.gutterObserver = null;
    }
  },
  {
    decorations: (value) => value.decorations,
  }
);

export function toggleLivePreviewHeadingSection(view: EditorView, key: string): boolean {
  const plugin = view.plugin(livePreviewPlugin);
  if (!plugin) return false;
  return plugin.toggleHeadingSection(view, key);
}

export function createLivePreviewExtension(): Extension {
  return livePreviewPlugin;
}
