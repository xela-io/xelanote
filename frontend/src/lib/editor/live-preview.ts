// Live preview: main orchestrator with buildDecorations, plugin class, and public API

import type { Extension } from '@codemirror/state';
import { RangeSetBuilder } from '@codemirror/state';
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from '@codemirror/view';

import type { HeadingInfo } from './live-preview/heading-manager';
import { collectHeadingInfo } from './live-preview/heading-manager';
import type { LinePrimitives } from './live-preview/line-primitives';
import { parseLinePrimitives } from './live-preview/line-primitives';
import type { StructuredLines } from './live-preview/structured-lines';
import {
  collectStructuredLines,
  shouldRecomputeStructuredLines,
} from './live-preview/structured-lines';
import type { TableBlock } from './live-preview/table-parser';
import type { CompletedTaskGroupInfo } from './live-preview/task-group-manager';
import {
  collectCompletedTaskGroups,
  remapCollapsedTaskGroups,
} from './live-preview/task-group-manager';
import {
  activeLinesKey,
  getActiveLines,
  isInsideRanges,
  loadCollapsedTaskGroups,
  persistCollapsedTaskGroups,
  profile,
  setsEqual,
} from './live-preview/utilities';
import {
  addProtectedWidget,
  CompletedTaskGroupSummaryWidget,
  CompletedTaskGroupToggleWidget,
  getLineDecoration,
  HeadingToggleWidget,
  hiddenSyntaxDecoration,
  InlineTextWidget,
  TableWidget,
  TaskCheckboxWidget,
} from './live-preview/widgets';
import { collectTreeFeatures, extractMarkdownLinksFromText } from './live-preview-features';

// Re-export public API from utilities
export { setLivePreviewProfilerSink } from './live-preview/utilities';

interface LivePreviewStaticData {
  structuredLines: StructuredLines;
  treeFeatures: ReturnType<typeof collectTreeFeatures>;
}

interface LivePreviewPersistenceOptions {
  noteId?: string;
}

function buildDecorations(
  view: EditorView,
  staticData: LivePreviewStaticData,
  headingInfo: HeadingInfo,
  completedTaskGroupInfo: CompletedTaskGroupInfo,
  reason: string,
  activeLines: Set<number>,
  getLinePrimitives: (lineNumber: number, text: string) => LinePrimitives
): DecorationSet {
  return profile('build', reason, () => {
    const builder = new RangeSetBuilder<Decoration>();
    const seenLines = new Set<number>();
    const doc = view.state.doc;
    const { structuredLines, treeFeatures } = staticData;

    // Build lookup map: line number → TableBlock (only for startLine of each block)
    const tableBlockByLine = new Map<number, TableBlock>();
    const tableBlockCoveredLines = new Set<number>();
    for (const block of structuredLines.tableBlocks) {
      tableBlockByLine.set(block.startLine, block);
      for (let l = block.startLine; l <= block.endLine; l++) {
        tableBlockCoveredLines.add(l);
      }
    }

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

        // Completed task group: collapsed lines
        const taskGroup = completedTaskGroupInfo.groupByLine.get(line.number);
        if (taskGroup?.collapsed) {
          if (line.number === taskGroup.startLine) {
            // First line of collapsed group → summary widget replaces content
            builder.add(
              line.from,
              line.from,
              Decoration.widget({
                side: -1,
                widget: new CompletedTaskGroupToggleWidget(
                  taskGroup.key,
                  true,
                  taskGroup.count,
                  taskGroup.startLine
                ),
              })
            );
            builder.add(
              line.from,
              line.to,
              Decoration.replace({
                widget: new CompletedTaskGroupSummaryWidget(
                  taskGroup.key,
                  taskGroup.count,
                  taskGroup.startLine
                ),
              })
            );
          } else {
            // Remaining lines of collapsed group → hide
            builder.add(line.from, line.from, getLineDecoration('cm-live-collapsed-line'));
            builder.add(line.from, line.to, hiddenSyntaxDecoration);
          }
          continue;
        }

        // Table block handling: if this line is the start of a table block
        // and no line in the block is active, replace with a widget
        const tableBlock = tableBlockByLine.get(line.number);
        if (tableBlock) {
          const firstLine = doc.line(tableBlock.startLine);
          const lastLine = doc.line(tableBlock.endLine);
          // Check if any selection range intersects the block
          let blockIsActive = false;
          for (const range of view.state.selection.ranges) {
            if (range.from <= lastLine.to && range.to >= firstLine.from) {
              blockIsActive = true;
              break;
            }
          }
          // Also check focus: unfocused editor → never active
          if (!view.hasFocus) blockIsActive = false;

          if (!blockIsActive) {
            // First line: replace content with table widget
            builder.add(
              firstLine.from,
              firstLine.to,
              Decoration.replace({
                widget: new TableWidget(tableBlock),
              })
            );
            // Remaining lines: collapse to zero height
            for (let l = tableBlock.startLine + 1; l <= tableBlock.endLine; l++) {
              const hideLine = doc.line(l);
              seenLines.add(l);
              builder.add(
                hideLine.from,
                hideLine.from,
                getLineDecoration('cm-live-collapsed-line')
              );
              builder.add(hideLine.from, hideLine.to, hiddenSyntaxDecoration);
            }
            continue;
          }
        } else if (tableBlockCoveredLines.has(line.number)) {
          // This line is part of a block but not the start line.
          // If the block's start was already processed as a widget replacement,
          // the seenLines check above handles it. If the block is active (cursor inside),
          // fall through to normal rendering below.
        }

        const headingSection = headingInfo.headingByLine.get(line.number);
        const showHeadingToggle =
          !!headingSection && headingSection.endLine > headingSection.headingLine;

        const protectedRanges: Array<{ from: number; to: number }> = [];
        const text = line.text;
        const base = line.from;
        const isActiveLine = activeLines.has(line.number);
        const lineClasses: string[] = ['cm-live-line-metrics'];

        const primitives = getLinePrimitives(line.number, text);
        const treeTask = treeFeatures.tasksByLine.get(line.number);
        // Compute taskInfo — the replacement must cover the marker token AND
        // trailing space so no gap character remains between checkbox and text.
        // markerTokenLength already includes spacing (see getLinePrimitives).
        const taskInfo =
          treeTask && primitives.markerPrefixLength != null
            ? {
                markerLength: primitives.markerPrefixLength,
                from: treeTask.from,
                to: primitives.taskRegex
                  ? base +
                    primitives.taskRegex.markerLength +
                    primitives.taskRegex.markerTokenLength
                  : treeTask.to,
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
        // Only mark lines with actual body text as task lines — empty tasks
        // (checkbox but no text) are excluded so the draggable set stays
        // consistent with getTasksInDocument() which also skips them.
        const hasTaskBody = taskInfo && text.substring(taskInfo.to - base).trim().length > 0;
        if (hasTaskBody) {
          // Keep task-line layout stable even when active to avoid checkbox shifting.
          lineClasses.push('cm-live-task-line');
        }

        if (line.number === 1 && !isActiveLine) {
          lineClasses.push('cm-live-first-line-title');
        }
        if (primitives.heading) {
          lineClasses.push(`cm-live-heading-h${primitives.heading.level}`);
        }
        if (showHeadingToggle) {
          lineClasses.push('cm-live-heading-toggle-line');
        }
        if (!isActiveLine) {
          lineClasses.push('cm-live-preview-line');
          if (primitives.blockquote) {
            lineClasses.push('cm-live-blockquote');
          }
        }
        if (listMarkerInfo) {
          lineClasses.push('cm-live-list-item');
        }
        if (!isActiveLine) {
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

        // Expanded task group bracket decorations (always, even on active line)
        if (taskGroup && !taskGroup.collapsed) {
          let bracketClass = 'cm-live-task-group-middle';
          if (line.number === taskGroup.startLine) bracketClass = 'cm-live-task-group-first';
          else if (line.number === taskGroup.endLine) bracketClass = 'cm-live-task-group-last';
          builder.add(line.from, line.from, getLineDecoration(bracketClass));

          if (line.number === taskGroup.startLine) {
            builder.add(
              line.from,
              line.from,
              Decoration.widget({
                side: -1,
                widget: new CompletedTaskGroupToggleWidget(
                  taskGroup.key,
                  false,
                  taskGroup.count,
                  taskGroup.startLine
                ),
              })
            );
          }
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

        if (taskInfo) {
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

        if (structuredLines.codeFenceLines.has(line.number)) {
          builder.add(line.from, line.from + text.length, hiddenSyntaxDecoration);
          continue;
        }
        if (structuredLines.codeContentLines.has(line.number)) {
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
            const isOrderedMarker = /^\d+[.)]$/.test(marker);
            const markerWidgetText = isOrderedMarker ? marker : '•';
            const markerClassName = isOrderedMarker
              ? 'cm-live-list-marker cm-live-list-marker-ordered'
              : 'cm-live-list-marker cm-live-list-marker-unordered';
            addProtectedWidget(
              builder,
              protectedRanges,
              fromPos,
              toPos,
              new InlineTextWidget(markerWidgetText, markerClassName)
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

const livePreviewPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    staticData: LivePreviewStaticData;
    headingInfo: HeadingInfo;
    completedTaskGroupInfo: CompletedTaskGroupInfo;
    linePrimitivesCache = new Map<number, { text: string; parsed: LinePrimitives }>();
    activeLines: Set<number>;
    activeLinesSignature: string;
    collapsedHeadingSections = new Set<string>();
    collapsedTaskGroups = new Set<string>();
    forceRebuild = false;
    headingInfoDirty = false;
    taskGroupInfoDirty = false;
    persistenceOptions: LivePreviewPersistenceOptions;
    pendingGutterSyncFrame: number | null = null;
    gutterObserver: MutationObserver | null = null;

    constructor(view: EditorView, persistenceOptions: LivePreviewPersistenceOptions = {}) {
      this.persistenceOptions = persistenceOptions;
      this.collapsedTaskGroups = loadCollapsedTaskGroups(this.persistenceOptions.noteId);
      this.staticData = this.computeStaticData(view, 'init');
      this.headingInfo = profile('structured', 'init', () =>
        collectHeadingInfo(view, this.collapsedHeadingSections)
      );
      this.completedTaskGroupInfo = collectCompletedTaskGroups(view, this.collapsedTaskGroups);
      this.activeLines = getActiveLines(view);
      this.activeLinesSignature = activeLinesKey(this.activeLines);
      this.decorations = buildDecorations(
        view,
        this.staticData,
        this.headingInfo,
        this.completedTaskGroupInfo,
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
      const previousTaskGroupInfo = this.completedTaskGroupInfo;
      const shouldRecalcActive = update.selectionSet || update.focusChanged;
      const nextActiveLines = shouldRecalcActive ? getActiveLines(update.view) : this.activeLines;
      const nextActiveLinesSignature = shouldRecalcActive
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
        const nextTaskGroupInfo = collectCompletedTaskGroups(update.view, this.collapsedTaskGroups);
        const remappedCollapsedTaskGroups = remapCollapsedTaskGroups(
          previousTaskGroupInfo,
          nextTaskGroupInfo,
          this.collapsedTaskGroups
        );
        for (const group of nextTaskGroupInfo.groups) {
          group.collapsed = remappedCollapsedTaskGroups.has(group.key);
        }
        if (!setsEqual(this.collapsedTaskGroups, remappedCollapsedTaskGroups)) {
          this.collapsedTaskGroups = remappedCollapsedTaskGroups;
          persistCollapsedTaskGroups(this.persistenceOptions.noteId, this.collapsedTaskGroups);
        }
        this.completedTaskGroupInfo = nextTaskGroupInfo;
        this.taskGroupInfoDirty = false;
      } else if (update.viewportChanged) {
        this.staticData = {
          ...this.staticData,
          treeFeatures: profile('tree', reason, () => collectTreeFeatures(update.view)),
        };
      } else if (this.headingInfoDirty || this.taskGroupInfoDirty) {
        if (this.headingInfoDirty) {
          this.headingInfo = profile('structured', reason, () =>
            collectHeadingInfo(update.view, this.collapsedHeadingSections)
          );
          this.headingInfoDirty = false;
        }
        if (this.taskGroupInfoDirty) {
          this.completedTaskGroupInfo = collectCompletedTaskGroups(
            update.view,
            this.collapsedTaskGroups
          );
          this.taskGroupInfoDirty = false;
        }
      }

      // Auto-expand collapsed task groups when cursor moves into them
      // Only check when active lines actually changed, so that an explicit
      // user toggle (which doesn't move the cursor) isn't immediately reversed.
      if (nextActiveLinesSignature !== this.activeLinesSignature) {
        for (const group of this.completedTaskGroupInfo.groups) {
          if (!group.collapsed) continue;
          for (const activeLine of nextActiveLines) {
            if (activeLine >= group.startLine && activeLine <= group.endLine) {
              this.collapsedTaskGroups.delete(group.key);
              group.collapsed = false;
              persistCollapsedTaskGroups(this.persistenceOptions.noteId, this.collapsedTaskGroups);
              this.forceRebuild = true;
              break;
            }
          }
        }
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
          this.completedTaskGroupInfo,
          reason,
          nextActiveLines,
          this.getLinePrimitives.bind(this)
        );
        this.pruneHeadingSections(this.headingInfo.keys);
        this.pruneTaskGroups(this.completedTaskGroupInfo.keys);
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

    toggleCompletedTaskGroup(view: EditorView, key: string): boolean {
      if (this.collapsedTaskGroups.has(key)) {
        this.collapsedTaskGroups.delete(key);
      } else {
        this.collapsedTaskGroups.add(key);
      }
      persistCollapsedTaskGroups(this.persistenceOptions.noteId, this.collapsedTaskGroups);
      this.taskGroupInfoDirty = true;
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

    private pruneTaskGroups(validKeys: Set<string>) {
      let changed = false;
      for (const key of this.collapsedTaskGroups) {
        if (!validKeys.has(key)) {
          this.collapsedTaskGroups.delete(key);
          changed = true;
        }
      }
      if (changed) {
        persistCollapsedTaskGroups(this.persistenceOptions.noteId, this.collapsedTaskGroups);
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

      for (const [lineNumber, group] of this.completedTaskGroupInfo.groupByLine) {
        if (group.collapsed && lineNumber !== group.startLine) {
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

export function toggleLivePreviewCompletedTaskGroup(view: EditorView, key: string): boolean {
  const plugin = view.plugin(livePreviewPlugin);
  if (!plugin) return false;
  return plugin.toggleCompletedTaskGroup(view, key);
}

export function createLivePreviewExtension(
  persistenceOptions: LivePreviewPersistenceOptions = {}
): Extension {
  return livePreviewPlugin.of(persistenceOptions);
}
