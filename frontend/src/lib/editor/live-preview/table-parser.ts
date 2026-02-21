// Table parsing: detection, cell extraction, alignment parsing, block collection

export interface TableBlock {
  startLine: number;
  endLine: number;
  headerCells: string[];
  alignments: ('left' | 'center' | 'right' | null)[];
  rows: string[][];
}

export function isCodeFence(text: string): boolean {
  // CommonMark spec: code fences may be indented by 0-3 spaces only.
  // 4+ spaces = indented code block (literal text, not a fence).
  // Both backtick (```) and tilde (~~~) fences are supported by markdown-it.
  return /^ {0,3}(?:```|~~~)/.test(text);
}

export function isTableSeparatorLine(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.includes('|') || !trimmed.includes('-')) return false;
  return /^[\s|:-]+$/.test(trimmed);
}

export function isTableCandidateLine(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed.includes('|')) return false;
  const pipeCount = (trimmed.match(/\|/g) ?? []).length;
  return pipeCount >= 1;
}

export function parseTableCells(line: string): string[] {
  const trimmed = line.trim();
  // Remove leading/trailing pipes
  const inner = trimmed.startsWith('|') ? trimmed.slice(1) : trimmed;
  const withoutTrailing = inner.endsWith('|') ? inner.slice(0, -1) : inner;

  // Split on unescaped pipes
  const cells: string[] = [];
  let current = '';
  for (let i = 0; i < withoutTrailing.length; i++) {
    if (
      withoutTrailing[i] === '\\' &&
      i + 1 < withoutTrailing.length &&
      withoutTrailing[i + 1] === '|'
    ) {
      current += '|';
      i++; // skip the pipe
    } else if (withoutTrailing[i] === '|') {
      cells.push(current.trim());
      current = '';
    } else {
      current += withoutTrailing[i];
    }
  }
  cells.push(current.trim());
  return cells;
}

export function parseAlignments(separatorLine: string): ('left' | 'center' | 'right' | null)[] {
  const cells = parseTableCells(separatorLine);
  return cells.map((cell) => {
    const trimmed = cell.trim();
    const left = trimmed.startsWith(':');
    const right = trimmed.endsWith(':');
    if (left && right) return 'center';
    if (right) return 'right';
    if (left) return 'left';
    return null;
  });
}

export function collectTableBlocks(tableLines: Set<number>, lines: string[]): TableBlock[] {
  const blocks: TableBlock[] = [];

  // Group consecutive table lines into ranges
  const sortedLines = [...tableLines].sort((a, b) => a - b);
  if (sortedLines.length === 0) return blocks;

  const ranges: Array<{ start: number; end: number }> = [];
  let rangeStart = sortedLines[0];
  let rangeEnd = sortedLines[0];

  for (let i = 1; i < sortedLines.length; i++) {
    if (sortedLines[i] === rangeEnd + 1) {
      rangeEnd = sortedLines[i];
    } else {
      ranges.push({ start: rangeStart, end: rangeEnd });
      rangeStart = sortedLines[i];
      rangeEnd = sortedLines[i];
    }
  }
  ranges.push({ start: rangeStart, end: rangeEnd });

  for (const range of ranges) {
    // Need at least 2 lines (header + separator)
    if (range.end - range.start < 1) continue;

    // Line numbers are 1-based, lines array is 0-based
    const headerLineIdx = range.start - 1;
    const separatorLineIdx = range.start; // second line = index start

    // Validate: second line must be a separator
    if (separatorLineIdx >= lines.length) continue;
    if (!isTableSeparatorLine(lines[separatorLineIdx])) continue;

    const headerCells = parseTableCells(lines[headerLineIdx]);
    const alignments = parseAlignments(lines[separatorLineIdx]);

    // Pad alignments if fewer than header columns
    while (alignments.length < headerCells.length) {
      alignments.push(null);
    }

    // Parse data rows (lines after separator)
    const rows: string[][] = [];
    for (let lineNo = range.start + 2; lineNo <= range.end; lineNo++) {
      const lineIdx = lineNo - 1;
      if (lineIdx < lines.length) {
        rows.push(parseTableCells(lines[lineIdx]));
      }
    }

    blocks.push({
      startLine: range.start,
      endLine: range.end,
      headerCells,
      alignments,
      rows,
    });
  }

  return blocks;
}
