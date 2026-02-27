// Heading section collapse management

import { EditorView } from '@codemirror/view';

export interface HeadingSection {
  key: string;
  headingLine: number;
  endLine: number;
  collapsed: boolean;
}

export interface HeadingInfo {
  headingByLine: Map<number, HeadingSection>;
  sectionByLine: Map<number, HeadingSection>;
  keys: Set<string>;
}

export function buildHeadingSectionByLineForViewport(
  headingByLine: Map<number, HeadingSection>,
  visibleFrom: number,
  visibleTo: number
): Map<number, HeadingSection> {
  const sectionByLine = new Map<number, HeadingSection>();
  for (const section of headingByLine.values()) {
    if (section.endLine <= section.headingLine) continue;
    const populateStart = Math.max(section.headingLine + 1, visibleFrom);
    const populateEnd = Math.min(section.endLine, visibleTo);
    for (let line = populateStart; line <= populateEnd; line++) {
      sectionByLine.set(line, section);
    }
  }
  return sectionByLine;
}

/**
 * Collect heading info with viewport-optimized sectionByLine population.
 * All headings are still scanned (cheap regex), but sectionByLine is only
 * populated for lines within the visible range to avoid O(n) map entries
 * for large collapsed sections.
 */
export function collectHeadingInfo(
  view: EditorView,
  collapsedSections: Set<string>,
  visibleFrom?: number,
  visibleTo?: number
): HeadingInfo {
  const headingByLine = new Map<number, HeadingSection>();
  const keys = new Set<string>();
  const headings: Array<{ line: number; level: number }> = [];

  // Pass 1: Find all headings (cheap regex scan over all lines)
  for (let lineNo = 1; lineNo <= view.state.doc.lines; lineNo++) {
    const text = view.state.doc.line(lineNo).text;
    const match = /^(\s{0,3})(#{1,6})(\s+)/.exec(text);
    if (!match) continue;
    headings.push({ line: lineNo, level: match[2].length });
  }

  // Determine visible line range with buffer
  const viewportStartLine = visibleFrom ?? 1;
  const viewportEndLine = visibleTo ?? view.state.doc.lines;

  // Pass 2: Compute section boundaries and populate maps
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

    // sectionByLine is built in one pass below for the active viewport.
  }

  return {
    headingByLine,
    sectionByLine: buildHeadingSectionByLineForViewport(
      headingByLine,
      viewportStartLine,
      viewportEndLine
    ),
    keys,
  };
}
