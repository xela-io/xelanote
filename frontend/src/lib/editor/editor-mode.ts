import type { EditorView } from '@codemirror/view';

interface EnsureEditorReadyParams {
  getEditorView: () => EditorView | undefined;
  setEditorMode: (mode: 'edit' | 'preview' | 'split') => void;
  tick: () => Promise<void>;
}

export async function ensureEditorReady(
  params: EnsureEditorReadyParams
): Promise<EditorView | undefined> {
  const current = params.getEditorView();
  if (current) return current;

  params.setEditorMode('edit');
  await params.tick();
  return params.getEditorView();
}
