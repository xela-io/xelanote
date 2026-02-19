import type { EditorView } from '@codemirror/view';

import { createEditor, type EditorConfig } from '$lib/editor/codemirror';

export interface InitEditorDeps {
  getDoc: () => string;
  onChange: (content: string) => void;
  onSave: () => void;
  onWikilinkClick: (title: string) => void;
  onToggleTaskByLine: (lineNumber: number, checked: boolean) => void;
  onColorPicker: () => void;
  onInsertTable?: () => void;
  onBeforeNewline: (view: EditorView) => boolean;
  onFindReplace: (options?: { replace?: boolean }) => void;
  onExtensionsReady: () => void;
  setEditorView: (view: EditorView | undefined) => void;
  setExtensionsReady: (ready: boolean) => void;
}

export function initEditorAction(deps: InitEditorDeps) {
  return (node: HTMLElement) => {
    const config: EditorConfig = {
      doc: deps.getDoc(),
      onChange: deps.onChange,
      onSave: deps.onSave,
      onWikilinkClick: deps.onWikilinkClick,
      onToggleTaskByLine: deps.onToggleTaskByLine,
      onColorPicker: deps.onColorPicker,
      onInsertTable: deps.onInsertTable,
      onBeforeNewline: deps.onBeforeNewline,
      onFindReplace: deps.onFindReplace,
      onExtensionsReady: deps.onExtensionsReady,
    };

    const editorView = createEditor(node, config);
    deps.setEditorView(editorView);

    return {
      destroy() {
        editorView.destroy();
        deps.setEditorView(undefined);
        deps.setExtensionsReady(false);
      },
    };
  };
}
