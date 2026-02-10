import type { EditorView } from '@codemirror/view';

export function indentSelection(editorView: EditorView) {
  const state = editorView.state;
  const selection = state.selection.main;
  const doc = state.doc;

  // Finde alle Zeilen in der Selektion
  const startLine = doc.lineAt(selection.from);
  const endLine = doc.lineAt(selection.to);

  const changes: { from: number; to: number; insert: string }[] = [];

  for (let lineNum = startLine.number; lineNum <= endLine.number; lineNum++) {
    const line = doc.line(lineNum);
    // Tab am Zeilenanfang einfügen
    changes.push({ from: line.from, to: line.from, insert: '\t' });
  }

  editorView.dispatch({ changes });
  editorView.focus();
}

export function outdentSelection(editorView: EditorView) {
  const state = editorView.state;
  const selection = state.selection.main;
  const doc = state.doc;

  // Finde alle Zeilen in der Selektion
  const startLine = doc.lineAt(selection.from);
  const endLine = doc.lineAt(selection.to);

  const changes: { from: number; to: number; insert: string }[] = [];

  for (let lineNum = startLine.number; lineNum <= endLine.number; lineNum++) {
    const line = doc.line(lineNum);
    const text = line.text;

    // Entferne führenden Tab oder bis zu 4 Spaces
    if (text.startsWith('\t')) {
      changes.push({ from: line.from, to: line.from + 1, insert: '' });
    } else if (text.startsWith('    ')) {
      changes.push({ from: line.from, to: line.from + 4, insert: '' });
    } else if (text.startsWith('   ')) {
      changes.push({ from: line.from, to: line.from + 3, insert: '' });
    } else if (text.startsWith('  ')) {
      changes.push({ from: line.from, to: line.from + 2, insert: '' });
    } else if (text.startsWith(' ')) {
      changes.push({ from: line.from, to: line.from + 1, insert: '' });
    }
  }

  if (changes.length > 0) {
    editorView.dispatch({ changes });
  }
  editorView.focus();
}
