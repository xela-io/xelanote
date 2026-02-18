// CodeMirror 6 setup for xelanote

import { bracketMatching, HighlightStyle } from '@codemirror/language';
import { Compartment, EditorState, type Extension, Prec } from '@codemirror/state';
import {
  drawSelection,
  EditorView,
  highlightActiveLine,
  keymap,
  placeholder,
} from '@codemirror/view';
import { Decoration, type DecorationSet, ViewPlugin, type ViewUpdate } from '@codemirror/view';
import { tags } from '@lezer/highlight';

import { FEATURE_FLAGS } from '$lib/config';

import { createFindReplaceExtension } from './find-replace';
import {
  dimLinesCompartment,
  emptyExtension,
  setDimInactiveLines,
  setTypewriterMode,
  typewriterCompartment,
} from './focus-mode-extensions';
import {
  createLivePreviewExtension,
  toggleLivePreviewCompletedTaskGroup,
  toggleLivePreviewHeadingSection,
} from './live-preview';
import { isValidDueDate } from './markdown';
import {
  createSpellCheckExtension,
  getSpellCheckState,
  setSpellLanguage as setSpellLanguageInternal,
  spellCheckCompartment,
  toggleSpellCheck as toggleSpellCheckInternal,
} from './spell-check';

// Wikilink decoration
const wikilinkMatcher = /\[\[([^\]|]+)(\|[^\]]+)?\]\]/g;

const wikilinkDecoration = Decoration.mark({ class: 'cm-wikilink' });

function getWikilinkDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = wikilinkMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }
  }

  return Decoration.set(
    decorations.map((d) => wikilinkDecoration.range(d.from, d.to)),
    true
  );
}

const wikilinkPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getWikilinkDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getWikilinkDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Color tag decoration
const colorOpenMatcher = /\{color:[^}]+\}/g;
const colorCloseMatcher = /\{\/color\}/g;

const colorTagDecoration = Decoration.mark({ class: 'cm-color-tag' });

function getColorTagDecorations(view: EditorView): DecorationSet {
  if (!FEATURE_FLAGS.colorSyntax) {
    return Decoration.set([]);
  }

  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);

    // Find opening tags
    let match;
    while ((match = colorOpenMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }

    // Find closing tags
    while ((match = colorCloseMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length,
      });
    }
  }

  // Sort by position for proper decoration ordering
  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => colorTagDecoration.range(d.from, d.to)),
    true
  );
}

const colorTagPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getColorTagDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getColorTagDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Task bracket decoration - highlights [ ] and [x] checkboxes with accent color
const taskBracketDecoration = Decoration.mark({ class: 'cm-task-bracket' });

function getTaskBracketDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];
  const taskBracketMatcher = /\[[ xX]\]/g;

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = taskBracketMatcher.exec(text)) !== null) {
      const absFrom = from + match.index;
      const line = view.state.doc.lineAt(absFrom);
      // Only match inside list items (- [ ], * [ ], + [ ], 1. [ ])
      if (/^\s*[-*+]\s/.test(line.text) || /^\s*\d+[.)]\s/.test(line.text)) {
        decorations.push({
          from: absFrom,
          to: absFrom + match[0].length,
        });
      }
    }
  }

  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => taskBracketDecoration.range(d.from, d.to)),
    true
  );
}

const taskBracketPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getTaskBracketDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getTaskBracketDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Due date decoration - highlights @due(YYYY-MM-DD) syntax
const dueDateDecoration = Decoration.mark({ class: 'cm-due-date' });

const dueDateMatcher = /@due\((\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\)/g;

function getDueDateDecorations(view: EditorView): DecorationSet {
  if (!FEATURE_FLAGS.dueDateSyntax) {
    return Decoration.set([]);
  }

  const livePreviewExt = livePreviewCompartment.get(view.state);
  const livePreviewEnabled = livePreviewExt !== undefined && livePreviewExt !== emptyExtension;
  const activeLines = new Set<number>();
  if (livePreviewEnabled) {
    for (const range of view.state.selection.ranges) {
      const fromLine = view.state.doc.lineAt(range.from).number;
      const toLine = view.state.doc.lineAt(range.to).number;
      for (let line = fromLine; line <= toLine; line += 1) {
        activeLines.add(line);
      }
    }
  }

  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = dueDateMatcher.exec(text)) !== null) {
      const matchFrom = from + match.index;
      const lineNumber = view.state.doc.lineAt(matchFrom).number;
      if (livePreviewEnabled && !activeLines.has(lineNumber)) {
        continue;
      }
      const dateStr = match[1];
      if (isValidDueDate(dateStr)) {
        decorations.push({
          from: matchFrom,
          to: matchFrom + match[0].length,
        });
      }
    }
  }

  decorations.sort((a, b) => a.from - b.from);

  return Decoration.set(
    decorations.map((d) => dueDateDecoration.range(d.from, d.to)),
    true
  );
}

const dueDatePlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = getDueDateDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = getDueDateDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// List hanging indent via inline styles.
const listIndentPattern = /^(\s*)([-*+]|\d+[.)])\s/;
const taskListPattern = /^(\s*)([-*+]|\d+[.)])\s\[[ xX]\]\s/;
const blankLinePattern = /^\s*$/;

// Task continuation lines must match the task line's text start position.
// Uses the same CSS variables as .cm-live-task-line in app.css.
const taskContinuationStyle =
  'padding-left: calc(var(--live-preview-marker-column-width) + var(--live-preview-marker-gap));';

function buildListIndentDecorations(view: EditorView): DecorationSet {
  const items: Array<{ pos: number; style: string }> = [];

  for (const { from, to } of view.visibleRanges) {
    let pos = from;
    let inTaskItem = false;
    while (pos <= to) {
      const line = view.state.doc.lineAt(pos);
      const text = line.text;

      if (blankLinePattern.test(text)) {
        inTaskItem = false;
        pos = line.to + 1;
        continue;
      }

      if (taskListPattern.test(text)) {
        // Task lines need inline style for hanging indent (same as regular lists).
        // CSS class alone doesn't reliably control wrapped-line indent in CodeMirror.
        items.push({
          pos: line.from,
          style:
            'padding-left: calc(var(--live-preview-marker-column-width) + var(--live-preview-marker-gap)); text-indent: calc(-1 * (var(--live-preview-marker-column-width) + var(--live-preview-marker-gap)));',
        });
        inTaskItem = true;
        pos = line.to + 1;
        continue;
      }

      if (inTaskItem && !listIndentPattern.test(text)) {
        items.push({ pos: line.from, style: taskContinuationStyle });
        pos = line.to + 1;
        continue;
      }

      inTaskItem = false;
      const match = listIndentPattern.exec(text);
      if (match) {
        items.push({
          pos: line.from,
          style: `padding-left: ${match[0].length}ch; text-indent: -${match[0].length}ch;`,
        });
      }
      pos = line.to + 1;
    }
  }

  return Decoration.set(
    items.map((item) =>
      Decoration.line({
        attributes: {
          style: item.style,
        },
      }).range(item.pos)
    ),
    true
  );
}

const listIndentPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = buildListIndentDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = buildListIndentDecorations(update.view);
      }
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Theme extension for dark/light mode
const lightTheme = EditorView.theme({
  '&': {
    backgroundColor: 'var(--color-background)',
    color: 'var(--color-foreground)',
  },
  '.cm-content': {
    caretColor: 'var(--color-foreground)',
  },
  '.cm-cursor': {
    borderLeftColor: 'var(--color-foreground)',
  },
  // CodeMirror 6 renders selection in a separate layer BEHIND the content.
  // Solid backgrounds on .cm-activeLine block selection visibility.
  // Solution: Use semi-transparent backgrounds so selection shows through.
  // See: https://discuss.codemirror.net/t/cm6-selection-not-visible-on-active-line
  '.cm-activeLine': {
    backgroundColor: 'color-mix(in srgb, var(--color-muted) 15%, transparent)',
  },
  // Selection background - must match CodeMirror's base theme selector specificity
  // Base theme uses: &.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground (5 classes)
  // Simple &.cm-focused .cm-selectionBackground (3 classes) loses the specificity battle
  '.cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--color-selection)',
  },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
    backgroundColor: 'var(--color-selection)',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--color-sidebar-background)',
    color: 'var(--color-muted-foreground)',
    borderRight: '1px solid var(--color-border)',
  },
  // Bracket matching - theme-aware colors for both light and dark
  '.cm-matchingBracket': {
    backgroundColor: 'color-mix(in srgb, var(--color-primary) 25%, transparent)',
    color: 'inherit',
    outline: '1px solid color-mix(in srgb, var(--color-primary) 40%, transparent)',
  },
  '.cm-nonmatchingBracket': {
    backgroundColor: 'color-mix(in srgb, var(--color-destructive) 25%, transparent)',
    color: 'inherit',
  },
});

const markdownSyntaxStyle = HighlightStyle.define([
  {
    tag: [tags.meta, tags.punctuation],
    color: 'var(--color-muted-foreground)',
    fontWeight: '600',
  },
  // Task markers [ ]/[x] - defaultHighlightStyle colors atoms in blue (#221199),
  // which is invisible on dark backgrounds
  {
    tag: tags.atom,
    color: 'var(--color-muted-foreground)',
  },
  // Links/URLs - use theme-aware primary color instead of default blue
  {
    tag: [tags.link, tags.url],
    color: 'var(--color-primary)',
  },
]);

export interface EditorConfig {
  doc?: string;
  onChange?: (content: string) => void;
  onSave?: () => void;
  onWikilinkClick?: (title: string) => void;
  onToggleTaskByLine?: (lineNumber: number, checked: boolean) => void;
  onColorPicker?: () => void;
  onBeforeNewline?: (view: EditorView) => boolean; // Return true if handled (prevents default)
  onFindReplace?: (options?: { replace?: boolean }) => void;
  onExtensionsReady?: () => void;
}

// Lazy load editor extensions
let lazyExtensionsPromise: Promise<Extension[]> | null = null;
const livePreviewCompartment = new Compartment();

export async function loadEditorExtensions(_config: EditorConfig): Promise<Extension[]> {
  // Return cached promise if already loading
  if (lazyExtensionsPromise) {
    return lazyExtensionsPromise;
  }

  lazyExtensionsPromise = (async () => {
    const [
      { defaultKeymap, history, historyKeymap, indentWithTab },
      { markdown, markdownLanguage },
      { syntaxHighlighting, defaultHighlightStyle },
      { createWikilinkAutocomplete },
    ] = await Promise.all([
      import('@codemirror/commands'),
      import('@codemirror/lang-markdown'),
      import('@codemirror/language'),
      import('./wikilink-autocomplete'),
    ]);

    const wikilinkExt = createWikilinkAutocomplete();

    return [
      history(),
      markdown({ base: markdownLanguage }),
      syntaxHighlighting(markdownSyntaxStyle),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      wikilinkExt,
      // Mod-s is already in the base extensions with Prec.highest
      keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
    ];
  })();

  return lazyExtensionsPromise;
}

export function createEditor(parent: HTMLElement, config: EditorConfig = {}): EditorView {
  // Compartment for lazy-loaded extensions
  const lazyCompartment = new Compartment();
  const moveCursorToTaskGroupStart = (view: EditorView, element: HTMLElement | null) => {
    const lineNumber = parseInt(element?.dataset.line ?? '', 10);
    if (!Number.isInteger(lineNumber) || lineNumber <= 0 || lineNumber > view.state.doc.lines) {
      return;
    }
    const line = view.state.doc.line(lineNumber);
    view.dispatch({ selection: { anchor: line.from } });
    view.focus();
  };

  // Base extensions (loaded immediately)
  const baseExtensions: Extension[] = [
    EditorView.lineWrapping,
    highlightActiveLine(),
    drawSelection(),
    bracketMatching(),
    wikilinkPlugin,
    colorTagPlugin,
    taskBracketPlugin,
    dueDatePlugin,
    listIndentPlugin,
    lightTheme,
    // iOS Autokorrektur aktivieren
    // (Diese Attribute überschreiben CodeMirror 6 Defaults: autocorrect="off", autocapitalize="off", spellcheck="false")
    EditorView.contentAttributes.of({
      autocorrect: 'on',
      autocapitalize: 'sentences',
      spellcheck: 'true',
    }),
    // High priority keymaps (must run before defaultKeymap)
    Prec.highest(
      keymap.of([
        {
          key: 'Enter',
          run: (view) => {
            // Check if we need to reorder tasks before inserting newline
            if (config.onBeforeNewline?.(view)) {
              return true; // Handler took care of everything
            }
            return false; // Let default Enter behavior happen
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
      ])
    ),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        config.onChange?.(update.state.doc.toString());
      }
    }),
    // Handle wikilink clicks
    EditorView.domEventHandlers({
      mousedown: (event) => {
        const target = event.target as HTMLElement;
        const liveTaskCheckbox = target.closest('.cm-live-task-checkbox') as HTMLElement | null;
        const liveHeadingToggle = target.closest('.cm-live-heading-toggle') as HTMLElement | null;
        const liveTaskGroupToggle = target.closest(
          '.cm-live-task-group-toggle'
        ) as HTMLElement | null;
        const liveTaskGroupSummary = target.closest(
          '.cm-live-task-group-summary'
        ) as HTMLElement | null;
        if (liveTaskCheckbox) {
          event.preventDefault();
          return true;
        }
        if (liveHeadingToggle) {
          event.preventDefault();
          return true;
        }
        if (liveTaskGroupToggle || liveTaskGroupSummary) {
          event.preventDefault();
          return true;
        }
        const liveWikilink = target.closest('.cm-live-preview-wikilink') as HTMLElement | null;
        if (liveWikilink) {
          event.preventDefault();
          return true;
        }
        const liveLink = target.closest('.cm-live-preview-link') as HTMLElement | null;
        if (liveLink) {
          event.preventDefault();
          return true;
        }
        return false;
      },
      click: (event, view) => {
        const target = event.target as HTMLElement;

        const liveHeadingToggle = target.closest('.cm-live-heading-toggle') as HTMLElement | null;
        if (liveHeadingToggle?.dataset.section) {
          if (toggleLivePreviewHeadingSection(view, liveHeadingToggle.dataset.section)) {
            event.preventDefault();
            return true;
          }
        }

        const liveTaskGroupToggle = target.closest(
          '.cm-live-task-group-toggle'
        ) as HTMLElement | null;
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
          moveCursorToTaskGroupStart(view, liveTaskGroupSummary);
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
          // Extract wikilink title from the clicked element
          const pos = view.posAtDOM(target);
          const line = view.state.doc.lineAt(pos);
          const text = line.text;

          // Find wikilink at this position
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
      },
    }),
    // Compartment for lazy extensions (initially empty)
    lazyCompartment.of([]),
    // Focus mode compartments (initially empty)
    typewriterCompartment.of(emptyExtension),
    dimLinesCompartment.of(emptyExtension),
    // Spell check compartment (disabled by default)
    spellCheckCompartment.of(createSpellCheckExtension({ enabled: false })),
    // Find & Replace (search extension with hidden panel for custom UI)
    createFindReplaceExtension(),
    // Live preview compartment (disabled by default)
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

  // Lazy load additional extensions
  loadEditorExtensions(config).then((lazyExtensions) => {
    view.dispatch({
      effects: lazyCompartment.reconfigure(lazyExtensions),
    });
    config.onExtensionsReady?.();
  });

  return view;
}

export function updateEditorContent(view: EditorView, content: string) {
  // Only update if content actually changed to prevent focus loss
  const currentContent = view.state.doc.toString();
  if (currentContent === content) {
    return;
  }

  view.dispatch({
    changes: {
      from: 0,
      to: view.state.doc.length,
      insert: content,
    },
  });
}

export function setLivePreviewMode(
  view: EditorView,
  enabled: boolean,
  options: { noteId?: string } = {}
) {
  view.dispatch({
    effects: livePreviewCompartment.reconfigure(
      enabled ? createLivePreviewExtension({ noteId: options.noteId }) : emptyExtension
    ),
  });
}

// Re-export focus mode functions
export { setDimInactiveLines, setTypewriterMode };

// Combined function to update all focus mode settings
export function updateFocusMode(
  view: EditorView,
  options: { typewriter?: boolean; dimLines?: boolean }
) {
  if (options.typewriter !== undefined) {
    setTypewriterMode(view, options.typewriter);
  }
  if (options.dimLines !== undefined) {
    setDimInactiveLines(view, options.dimLines);
  }
}

/**
 * Insert a wikilink at the first occurrence of a term in the document.
 * Returns true if the term was found and replaced, false otherwise.
 *
 * @param view The CodeMirror EditorView
 * @param term The text to find and replace
 * @param targetTitle The note title to link to
 */
export function insertWikiLink(view: EditorView, term: string, targetTitle: string): boolean {
  const doc = view.state.doc.toString();

  // Find the first occurrence of the term (case-insensitive)
  const lowerDoc = doc.toLowerCase();
  const lowerTerm = term.toLowerCase();
  const idx = lowerDoc.indexOf(lowerTerm);

  if (idx === -1) {
    return false;
  }

  // Get the actual text at this position (preserve original casing)
  const actualTerm = doc.substring(idx, idx + term.length);

  // Build the wikilink
  // If term matches title exactly (case-insensitive), use [[title]]
  // Otherwise, use [[title|term]] as alias
  let wikilink: string;
  if (actualTerm.toLowerCase() === targetTitle.toLowerCase()) {
    wikilink = `[[${targetTitle}]]`;
  } else {
    wikilink = `[[${targetTitle}|${actualTerm}]]`;
  }

  // Replace the term with the wikilink
  view.dispatch({
    changes: {
      from: idx,
      to: idx + term.length,
      insert: wikilink,
    },
  });

  return true;
}

/**
 * Insert a wikilink for content that's not in CodeMirror (preview mode).
 * Returns the modified content string.
 *
 * @param content The content string to modify
 * @param term The text to find and replace
 * @param targetTitle The note title to link to
 */
// ============================================================================
// Canvas Editor
// ============================================================================

export interface CanvasEditorConfig {
  doc?: string;
  readOnly?: boolean;
  onChange?: (content: string) => void;
  onSave?: () => void;
  onWikilinkClick?: (title: string) => void;
  onToggleTaskByLine?: (lineNumber: number, checked: boolean) => void;
}

export function createCanvasEditor(
  parent: HTMLElement,
  config: CanvasEditorConfig = {}
): EditorView {
  // Per-instance compartments (not shared singletons like the main editor)
  const canvasLazyCompartment = new Compartment();
  const canvasLivePreviewCompartment = new Compartment();

  const baseExtensions: Extension[] = [
    EditorView.lineWrapping,
    ...(config.readOnly ? [EditorView.editable.of(false), EditorState.readOnly.of(true)] : []),
    drawSelection(),
    bracketMatching(),
    wikilinkPlugin,
    colorTagPlugin,
    taskBracketPlugin,
    dueDatePlugin,
    listIndentPlugin,
    lightTheme,
    ...(config.readOnly ? [] : [placeholder('Type markdown here...')]),
    // High priority keymaps
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
    // Handle live preview widget clicks (checkboxes, wikilinks, links, headings, task groups)
    EditorView.domEventHandlers({
      mousedown: (event) => {
        const target = event.target as HTMLElement;
        const liveTaskCheckbox = target.closest('.cm-live-task-checkbox') as HTMLElement | null;
        const liveHeadingToggle = target.closest('.cm-live-heading-toggle') as HTMLElement | null;
        const liveTaskGroupToggle = target.closest(
          '.cm-live-task-group-toggle'
        ) as HTMLElement | null;
        const liveTaskGroupSummary = target.closest(
          '.cm-live-task-group-summary'
        ) as HTMLElement | null;
        if (liveTaskCheckbox) {
          event.preventDefault();
          return true;
        }
        if (liveHeadingToggle) {
          event.preventDefault();
          return true;
        }
        if (liveTaskGroupToggle || liveTaskGroupSummary) {
          event.preventDefault();
          return true;
        }
        const liveWikilink = target.closest('.cm-live-preview-wikilink') as HTMLElement | null;
        if (liveWikilink) {
          event.preventDefault();
          return true;
        }
        const liveLink = target.closest('.cm-live-preview-link') as HTMLElement | null;
        if (liveLink) {
          event.preventDefault();
          return true;
        }
        return false;
      },
      click: (event, view) => {
        const target = event.target as HTMLElement;

        const liveHeadingToggle = target.closest('.cm-live-heading-toggle') as HTMLElement | null;
        if (liveHeadingToggle?.dataset.section) {
          if (toggleLivePreviewHeadingSection(view, liveHeadingToggle.dataset.section)) {
            event.preventDefault();
            return true;
          }
        }

        const liveTaskGroupToggle = target.closest(
          '.cm-live-task-group-toggle'
        ) as HTMLElement | null;
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
      },
    }),
    // Compartment for lazy extensions (initially empty)
    canvasLazyCompartment.of([]),
    // Live preview compartment (enabled immediately with empty persistence)
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

  // Lazy load additional extensions (markdown language, history, keymaps, autocomplete)
  loadEditorExtensions(config as EditorConfig).then((lazyExtensions) => {
    view.dispatch({
      effects: canvasLazyCompartment.reconfigure(lazyExtensions),
    });
  });

  return view;
}

// ============================================================================
// Spell Check Functions
// ============================================================================

/**
 * Toggle spell check on/off for the editor.
 */
export function toggleSpellCheck(view: EditorView, enabled: boolean) {
  toggleSpellCheckInternal(view, enabled);
}

/**
 * Set the language for spell checking.
 */
export function setSpellCheckLanguage(view: EditorView, language: 'de' | 'en') {
  setSpellLanguageInternal(view, language);
}

/**
 * Check if spell check is currently enabled.
 */
export function isSpellCheckEnabled(view: EditorView): boolean {
  return getSpellCheckState(view).enabled;
}

/**
 * Get the current spell check language.
 */
export function getSpellCheckLanguage(view: EditorView): 'de' | 'en' {
  return getSpellCheckState(view).language;
}

// ============================================================================
// Wikilink Insertion Functions
// ============================================================================

export function insertWikiLinkInContent(
  content: string,
  term: string,
  targetTitle: string
): { newContent: string; found: boolean } {
  // Find the first occurrence of the term (case-insensitive)
  const lowerContent = content.toLowerCase();
  const lowerTerm = term.toLowerCase();
  const idx = lowerContent.indexOf(lowerTerm);

  if (idx === -1) {
    return { newContent: content, found: false };
  }

  // Get the actual text at this position (preserve original casing)
  const actualTerm = content.substring(idx, idx + term.length);

  // Build the wikilink
  let wikilink: string;
  if (actualTerm.toLowerCase() === targetTitle.toLowerCase()) {
    wikilink = `[[${targetTitle}]]`;
  } else {
    wikilink = `[[${targetTitle}|${actualTerm}]]`;
  }

  // Replace the term with the wikilink
  const newContent = content.substring(0, idx) + wikilink + content.substring(idx + term.length);

  return { newContent, found: true };
}
