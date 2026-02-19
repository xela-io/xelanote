import type { EditorView } from '@codemirror/view';

export function buildMarkdownTable(rows: number, cols: number): string {
  const headerCells = Array.from({ length: cols }, (_, i) => ` Spalte ${i + 1} `);
  const header = `|${headerCells.join('|')}|`;
  const separator = `|${Array.from({ length: cols }, () => ' --- ').join('|')}|`;
  const dataRows = Array.from(
    { length: rows },
    () => `|${Array.from({ length: cols }, () => '  ').join('|')}|`
  );
  return [header, separator, ...dataRows].join('\n');
}

export function insertTable(editorView: EditorView, rows: number, cols: number): void {
  const doc = editorView.state.doc;
  const selection = editorView.state.selection.main;
  const cursorLine = doc.lineAt(selection.from);

  const isEmptyLine = cursorLine.text.trim() === '';
  const prefix = isEmptyLine ? '' : '\n';
  const table = buildMarkdownTable(rows, cols);
  const text = `${prefix}${table}\n`;

  const insertPos = isEmptyLine ? cursorLine.from : cursorLine.to;

  // Position cursor in the first data cell (third line, after first pipe + space)
  const tableStart = insertPos + prefix.length;
  const tableLines = table.split('\n');
  // First data row is at index 2 (after header and separator)
  const firstDataRowOffset = tableLines[0].length + 1 + tableLines[1].length + 1;
  // Cursor after the first pipe and space in the data row
  const cursorPos = tableStart + firstDataRowOffset + 2;

  editorView.dispatch({
    changes: { from: insertPos, to: insertPos, insert: text },
    selection: { anchor: cursorPos },
  });

  editorView.focus();
}
