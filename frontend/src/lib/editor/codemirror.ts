// CodeMirror 6 setup for xelanote
// This module orchestrates editor creation and re-exports the public API.

import { bracketMatching } from '@codemirror/language';
import { Compartment, EditorState, type Extension, Prec } from '@codemirror/state';
import {
  drawSelection,
  EditorView,
  highlightActiveLine,
  keymap,
  placeholder,
} from '@codemirror/view';

import { FEATURE_FLAGS } from '$lib/config';

import {
  colorTagPlugin,
  dueDatePlugin,
  firstLineTitlePlugin,
  listIndentPlugin,
  livePreviewCompartment,
  taskBracketPlugin,
  wikilinkPlugin,
} from './codemirror/decoration-plugins';
import { loadEditorExtensions } from './codemirror/extensions-loader';
import { lightTheme } from './codemirror/theme';
import {
  createSpellCheckExtension,
  emptyExtension,
  getSpellCheckLanguage,
  insertWikiLink,
  insertWikiLinkInContent,
  isSpellCheckEnabled,
  setDimInactiveLines,
  setLivePreviewMode,
  setSpellCheckLanguage,
  setTypewriterMode,
  toggleSpellCheck,
  updateEditorContent,
  updateFocusMode,
} from './codemirror/utilities';
import { createFindReplaceExtension } from './find-replace';
import { dimLinesCompartment, typewriterCompartment } from './focus-mode-extensions';
import {
  createLivePreviewExtension,
  toggleLivePreviewCompletedTaskGroup,
  toggleLivePreviewHeadingSection,
} from './live-preview';
import { spellCheckCompartment } from './spell-check';

// Re-export public API
export {
  getSpellCheckLanguage,
  insertWikiLink,
  insertWikiLinkInContent,
  isSpellCheckEnabled,
  loadEditorExtensions,
  setDimInactiveLines,
  setLivePreviewMode,
  setSpellCheckLanguage,
  setTypewriterMode,
  toggleSpellCheck,
  updateEditorContent,
  updateFocusMode,
};

export interface EditorConfig {
  doc?: string;
  onChange?: (content: string) => void;
  onSave?: () => void;
  onWikilinkClick?: (title: string) => void;
  onToggleTaskByLine?: (lineNumber: number, checked: boolean) => void;
  onColorPicker?: () => void;
  onInsertTable?: () => void;
  onBeforeNewline?: (view: EditorView) => boolean;
  onFindReplace?: (options?: { replace?: boolean }) => void;
  onExtensionsReady?: () => void;
}

export interface CanvasEditorConfig {
  doc?: string;
  readOnly?: boolean;
  onChange?: (content: string) => void;
  onSave?: () => void;
  onWikilinkClick?: (title: string) => void;
  onToggleTaskByLine?: (lineNumber: number, checked: boolean) => void;
}

// Shared decoration plugins used by both editor types
const sharedDecorationPlugins: Extension[] = [
  wikilinkPlugin,
  colorTagPlugin,
  taskBracketPlugin,
  dueDatePlugin,
  listIndentPlugin,
  firstLineTitlePlugin,
  lightTheme,
];

// Shared mousedown handler that prevents default for live preview widgets
function createMousedownHandler() {
  return (event: MouseEvent) => {
    const target = event.target as HTMLElement;
    const selectors = [
      '.cm-live-task-drag-handle',
      '.cm-live-task-checkbox',
      '.cm-live-heading-toggle',
      '.cm-live-task-group-toggle',
      '.cm-live-task-group-summary',
      '.cm-table-widget',
      '.cm-live-preview-wikilink',
      '.cm-live-preview-link',
    ];
    for (const sel of selectors) {
      if (target.closest(sel)) {
        event.preventDefault();
        return true;
      }
    }
    return false;
  };
}

// Shared click handler for live preview widgets
function createClickHandler(config: {
  onWikilinkClick?: (title: string) => void;
  onToggleTaskByLine?: (lineNumber: number, checked: boolean) => void;
}) {
  return (event: MouseEvent, view: EditorView) => {
    const target = event.target as HTMLElement;

    if (target.closest('.cm-live-task-drag-handle')) {
      event.preventDefault();
      return true;
    }

    const liveTableWidget = target.closest('.cm-table-widget') as HTMLElement | null;
    if (liveTableWidget) {
      const startLine = parseInt(liveTableWidget.dataset.startLine ?? '', 10);
      if (Number.isInteger(startLine) && startLine > 0 && startLine <= view.state.doc.lines) {
        const line = view.state.doc.line(startLine);
        view.dispatch({ selection: { anchor: line.from } });
        view.focus();
        event.preventDefault();
        return true;
      }
    }

    const liveHeadingToggle = target.closest('.cm-live-heading-toggle') as HTMLElement | null;
    if (liveHeadingToggle?.dataset.section) {
      if (toggleLivePreviewHeadingSection(view, liveHeadingToggle.dataset.section)) {
        event.preventDefault();
        return true;
      }
    }

    const liveTaskGroupToggle = target.closest('.cm-live-task-group-toggle') as HTMLElement | null;
    if (liveTaskGroupToggle?.dataset.taskGroup) {
      if (toggleLivePreviewCompletedTaskGroup(view, liveTaskGroupToggle.dataset.taskGroup)) {
        event.preventDefault();
        return true;
      }
    }

    const liveTaskGroupSummary = target.closest(
      '.cm-live-task-group-summary'
    ) as HTMLElement | null;
    if (liveTaskGroupSummary?.dataset.taskGroup) {
      // Move cursor to task group start line
      const lineNumber = parseInt(liveTaskGroupSummary.dataset.line ?? '', 10);
      if (Number.isInteger(lineNumber) && lineNumber > 0 && lineNumber <= view.state.doc.lines) {
        const line = view.state.doc.line(lineNumber);
        view.dispatch({ selection: { anchor: line.from } });
        view.focus();
      }
      if (toggleLivePreviewCompletedTaskGroup(view, liveTaskGroupSummary.dataset.taskGroup)) {
        event.preventDefault();
        return true;
      }
    }

    const liveTaskCheckbox = target.closest('.cm-live-task-checkbox') as HTMLElement | null;
    if (liveTaskCheckbox) {
      const lineNumber = parseInt(liveTaskCheckbox.dataset.line ?? '', 10);
      const checked = liveTaskCheckbox.dataset.checked === 'true';
      if (Number.isInteger(lineNumber) && lineNumber > 0) {
        config.onToggleTaskByLine?.(lineNumber, !checked);
        event.preventDefault();
        return true;
      }
    }

    const liveWikilink = target.closest('.cm-live-preview-wikilink') as HTMLElement | null;
    if (liveWikilink?.dataset.title) {
      config.onWikilinkClick?.(liveWikilink.dataset.title);
      event.preventDefault();
      return true;
    }

    const liveLink = target.closest('.cm-live-preview-link') as HTMLElement | null;
    if (liveLink?.dataset.href) {
      const href = liveLink.dataset.href;
      if (href) {
        window.open(href, '_blank', 'noopener,noreferrer');
        event.preventDefault();
        return true;
      }
    }

    if (target.classList.contains('cm-wikilink')) {
      const pos = view.posAtDOM(target);
      const line = view.state.doc.lineAt(pos);
      const text = line.text;
      const matches = [...text.matchAll(/\[\[([^\]|]+)(\|[^\]]+)?\]\]/g)];
      for (const match of matches) {
        const start = line.from + (match.index ?? 0);
        const end = start + match[0].length;
        if (pos >= start && pos <= end) {
          config.onWikilinkClick?.(match[1]);
          event.preventDefault();
          return true;
        }
      }
    }
    return false;
  };
}

export function createEditor(parent: HTMLElement, config: EditorConfig = {}): EditorView {
  const lazyCompartment = new Compartment();

  const baseExtensions: Extension[] = [
    EditorView.lineWrapping,
    highlightActiveLine(),
    drawSelection(),
    bracketMatching(),
    ...sharedDecorationPlugins,
    EditorView.contentAttributes.of({
      autocorrect: 'on',
      autocapitalize: 'sentences',
      spellcheck: 'true',
    }),
    Prec.highest(
      keymap.of([
        {
          key: 'Enter',
          run: (view) => {
            if (config.onBeforeNewline?.(view)) {
              return true;
            }
            return false;
          },
        },
        {
          key: 'Mod-s',
          run: () => {
            config.onSave?.();
            return true;
          },
        },
        {
          key: 'Mod-Shift-c',
          run: () => {
            if (FEATURE_FLAGS.colorSyntax && config.onColorPicker) {
              config.onColorPicker();
              return true;
            }
            return false;
          },
        },
        {
          key: 'Mod-f',
          run: () => {
            config.onFindReplace?.();
            return true;
          },
        },
        {
          key: 'Mod-h',
          run: () => {
            config.onFindReplace?.({ replace: true });
            return true;
          },
        },
        {
          key: 'Mod-Shift-t',
          run: () => {
            if (config.onInsertTable) {
              config.onInsertTable();
              return true;
            }
            return false;
          },
        },
      ])
    ),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        config.onChange?.(update.state.doc.toString());
      }
    }),
    EditorView.domEventHandlers({
      mousedown: createMousedownHandler(),
      click: createClickHandler(config),
    }),
    lazyCompartment.of([]),
    typewriterCompartment.of(emptyExtension),
    dimLinesCompartment.of(emptyExtension),
    spellCheckCompartment.of(createSpellCheckExtension({ enabled: false })),
    createFindReplaceExtension(),
    livePreviewCompartment.of(emptyExtension),
  ];

  const state = EditorState.create({
    doc: config.doc ?? '',
    extensions: baseExtensions,
  });

  const view = new EditorView({
    state,
    parent,
  });

  loadEditorExtensions(config).then((lazyExtensions) => {
    view.dispatch({
      effects: lazyCompartment.reconfigure(lazyExtensions),
    });
    config.onExtensionsReady?.();
  });

  return view;
}

export function createCanvasEditor(
  parent: HTMLElement,
  config: CanvasEditorConfig = {}
): EditorView {
  const canvasLazyCompartment = new Compartment();
  const canvasLivePreviewCompartment = new Compartment();

  const baseExtensions: Extension[] = [
    EditorView.lineWrapping,
    ...(config.readOnly ? [EditorView.editable.of(false), EditorState.readOnly.of(true)] : []),
    drawSelection(),
    bracketMatching(),
    ...sharedDecorationPlugins,
    ...(config.readOnly ? [] : [placeholder('Type markdown here...')]),
    Prec.highest(
      keymap.of([
        {
          key: 'Escape',
          run: (view) => {
            view.contentDOM.blur();
            return true;
          },
        },
        {
          key: 'Mod-s',
          run: () => {
            config.onSave?.();
            return true;
          },
        },
      ])
    ),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        config.onChange?.(update.state.doc.toString());
      }
    }),
    EditorView.domEventHandlers({
      mousedown: createMousedownHandler(),
      click: createClickHandler(config),
    }),
    canvasLazyCompartment.of([]),
    canvasLivePreviewCompartment.of(createLivePreviewExtension({})),
  ];

  const state = EditorState.create({
    doc: config.doc ?? '',
    extensions: baseExtensions,
  });

  const view = new EditorView({
    state,
    parent,
  });

  loadEditorExtensions(config as EditorConfig).then((lazyExtensions) => {
    view.dispatch({
      effects: canvasLazyCompartment.reconfigure(lazyExtensions),
    });
  });

  return view;
}
