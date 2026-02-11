import type { EditorView } from '@codemirror/view';

export function openMoreMenuAction(
  setState: {
    setShowColorPicker: (value: boolean) => void;
    setMarkdownGuideDropdownOpen: (value: boolean) => void;
    setMoreMenuTriggerRect: (rect: DOMRect) => void;
    setShowMoreMenu: (value: boolean) => void;
  },
  triggerRect: DOMRect
): void {
  setState.setShowColorPicker(false);
  setState.setMarkdownGuideDropdownOpen(false);
  setState.setMoreMenuTriggerRect(triggerRect);
  setState.setShowMoreMenu(true);
}

export function openColorPickerAction(setState: {
  setShowMoreMenu: (value: boolean) => void;
  setMarkdownGuideDropdownOpen: (value: boolean) => void;
  setShowColorPicker: (value: boolean) => void;
}): void {
  setState.setShowMoreMenu(false);
  setState.setMarkdownGuideDropdownOpen(false);
  setState.setShowColorPicker(true);
}

export function openMarkdownHelpAction(setState: {
  setShowMoreMenu: (value: boolean) => void;
  setShowColorPicker: (value: boolean) => void;
  toggleMarkdownGuideDropdown: () => void;
}): void {
  setState.setShowMoreMenu(false);
  setState.setShowColorPicker(false);
  setState.toggleMarkdownGuideDropdown();
}

export function handleColorSelectAction(editorView: EditorView, color: string): void {
  const selection = editorView.state.selection.main;
  const selectedText = editorView.state.doc.sliceString(selection.from, selection.to);

  const text = selectedText || 'Text';
  const openTag = `{color:${color}}`;
  const closeTag = '{/color}';
  const fullText = `${openTag}${text}${closeTag}`;

  editorView.dispatch({
    changes: {
      from: selection.from,
      to: selection.to,
      insert: fullText,
    },
    selection: selectedText
      ? { anchor: selection.from + fullText.length }
      : {
          anchor: selection.from + openTag.length,
          head: selection.from + openTag.length + text.length,
        },
  });
  editorView.focus();
}

export function extractFilesFromInputChangeEvent(e: Event): File[] {
  return Array.from((e.target as HTMLInputElement).files || []);
}

export function resetInputEventValue(e: Event): void {
  (e.target as HTMLInputElement).value = '';
}
