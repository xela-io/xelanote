// Focus Mode extensions for CodeMirror
import { Compartment } from '@codemirror/state';
import { EditorView, ViewPlugin, type ViewUpdate } from '@codemirror/view';

// Compartments for dynamic reconfiguration
export const typewriterCompartment = new Compartment();
export const dimLinesCompartment = new Compartment();

// Typewriter Mode: Keeps the cursor line centered in the viewport
export const typewriterPlugin = ViewPlugin.fromClass(
  class {
    constructor(_view: EditorView) {}

    update(update: ViewUpdate) {
      if (update.selectionSet || update.docChanged) {
        const { view } = update;
        const head = view.state.selection.main.head;
        // Scroll cursor to center with smooth behavior
        view.dispatch({
          effects: EditorView.scrollIntoView(head, { y: 'center' }),
        });
      }
    }
  }
);

// Dim Inactive Lines: Reduces opacity of non-active lines
export const dimInactiveLinesTheme = EditorView.theme({
  '.cm-line:not(.cm-activeLine)': {
    opacity: '0.4',
    transition: 'opacity 0.15s ease',
  },
  '.cm-line:not(.cm-activeLine):hover': {
    opacity: '0.7',
  },
});

// Empty extension placeholder
export const emptyExtension: [] = [];

// Helper functions to configure the compartments
export function setTypewriterMode(view: EditorView, enabled: boolean) {
  view.dispatch({
    effects: typewriterCompartment.reconfigure(enabled ? [typewriterPlugin] : emptyExtension),
  });
}

export function setDimInactiveLines(view: EditorView, enabled: boolean) {
  view.dispatch({
    effects: dimLinesCompartment.reconfigure(enabled ? [dimInactiveLinesTheme] : emptyExtension),
  });
}
