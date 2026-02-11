import type { EditorView } from '@codemirror/view';

interface SpellCheckDeps {
  getEditorView: () => EditorView | undefined;
  scheduleAutoSave: () => void;
}

export function registerSpellCheckReplaceListener(deps: SpellCheckDeps): () => void {
  function handleSpellCheckReplace(e: Event): void {
    const event = e as CustomEvent<{ from: number; to: number; replacement: string }>;
    const editorView = deps.getEditorView();
    if (!editorView) return;

    const { from, to, replacement } = event.detail;
    editorView.dispatch({
      changes: { from, to, insert: replacement },
    });
    deps.scheduleAutoSave();
  }

  document.addEventListener('spell-check-replace', handleSpellCheckReplace as EventListener);
  return () => {
    document.removeEventListener('spell-check-replace', handleSpellCheckReplace as EventListener);
  };
}
