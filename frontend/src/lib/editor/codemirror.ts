// CodeMirror 6 setup for xelanote

import { EditorState, Compartment, Prec, type Extension } from '@codemirror/state';
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
  drawSelection,
} from '@codemirror/view';
import { bracketMatching, HighlightStyle } from '@codemirror/language';
import { tags } from '@lezer/highlight';
import { Decoration, type DecorationSet, ViewPlugin, type ViewUpdate } from '@codemirror/view';
import { FEATURE_FLAGS } from '$lib/config';
import { isValidDueDate } from './markdown';
import {
  typewriterCompartment,
  dimLinesCompartment,
  emptyExtension,
  setTypewriterMode,
  setDimInactiveLines,
} from './focus-mode-extensions';
import {
  spellCheckCompartment,
  createSpellCheckExtension,
  toggleSpellCheck as toggleSpellCheckInternal,
  setSpellLanguage as setSpellLanguageInternal,
  getSpellCheckState,
} from './spell-check';
import { createFindReplaceExtension } from './find-replace';

// Wikilink decoration
const wikilinkMatcher = /\[\[([^\]|]+)(\|[^\]]+)?\]\]/g;

const wikilinkDecoration = Decoration.mark({ class: 'cm-wikilink' });

function getWikilinkDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];
  const doc = view.state.doc.toString();

  let match;
  while ((match = wikilinkMatcher.exec(doc)) !== null) {
    decorations.push({
      from: match.index,
      to: match.index + match[0].length,
    });
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
  const doc = view.state.doc.toString();

  // Find opening tags
  let match;
  while ((match = colorOpenMatcher.exec(doc)) !== null) {
    decorations.push({
      from: match.index,
      to: match.index + match[0].length,
    });
  }

  // Find closing tags
  while ((match = colorCloseMatcher.exec(doc)) !== null) {
    decorations.push({
      from: match.index,
      to: match.index + match[0].length,
    });
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
      if (/^\s*[-*+]\s/.test(line.text) || /^\s*\d+\.\s/.test(line.text)) {
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

  const decorations: { from: number; to: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = dueDateMatcher.exec(text)) !== null) {
      const dateStr = match[1];
      if (isValidDueDate(dateStr)) {
        decorations.push({
          from: from + match.index,
          to: from + match.index + match[0].length,
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

// List hanging indent - makes wrapped text in list items align with text after the marker
const listIndentPattern = /^(\s*)([-*+]|\d+\.)\s(\[[ xX]\]\s)?/;

function buildListIndentDecorations(view: EditorView): DecorationSet {
  const items: { pos: number; indent: number }[] = [];

  for (const { from, to } of view.visibleRanges) {
    let pos = from;
    while (pos <= to) {
      const line = view.state.doc.lineAt(pos);
      const match = listIndentPattern.exec(line.text);
      if (match) {
        items.push({ pos: line.from, indent: match[0].length });
      }
      pos = line.to + 1;
    }
  }

  return Decoration.set(
    items.map((item) =>
      Decoration.line({
        attributes: {
          style: `padding-left: ${item.indent}ch; text-indent: -${item.indent}ch;`,
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
  onColorPicker?: () => void;
  onBeforeNewline?: (view: EditorView) => boolean; // Return true if handled (prevents default)
  onFindReplace?: (options?: { replace?: boolean }) => void;
  onExtensionsReady?: () => void;
}

// Lazy load editor extensions
let lazyExtensionsPromise: Promise<Extension[]> | null = null;

export async function loadEditorExtensions(config: EditorConfig): Promise<Extension[]> {
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

    console.log('[CodeMirror] Loading extensions with wikilink autocomplete');
    const wikilinkExt = createWikilinkAutocomplete();
    console.log('[CodeMirror] Wikilink extension created:', wikilinkExt);

    return [
      history(),
      markdown({ base: markdownLanguage }),
      syntaxHighlighting(markdownSyntaxStyle),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      wikilinkExt,
      keymap.of([
        ...defaultKeymap,
        ...historyKeymap,
        indentWithTab,
        {
          key: 'Mod-s',
          run: () => {
            config.onSave?.();
            return true;
          },
        },
      ]),
    ];
  })();

  return lazyExtensionsPromise;
}

export function createEditor(parent: HTMLElement, config: EditorConfig = {}): EditorView {
  // Compartment for lazy-loaded extensions
  const lazyCompartment = new Compartment();

  // Base extensions (loaded immediately)
  const baseExtensions: Extension[] = [
    EditorView.lineWrapping,
    lineNumbers(),
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
      click: (event, view) => {
        const target = event.target as HTMLElement;
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

// Re-export focus mode functions
export { setTypewriterMode, setDimInactiveLines };

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
