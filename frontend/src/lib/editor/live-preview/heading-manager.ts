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

export function collectHeadingInfo(view: EditorView, collapsedSections: Set<string>): HeadingInfo {
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
