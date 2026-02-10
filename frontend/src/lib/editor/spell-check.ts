/**
 * CodeMirror extension for LLM-based spell checking.
 * Displays underlines for spelling/grammar issues with hover tooltips for suggestions.
 */

import { StateField, StateEffect, type Extension, Compartment } from '@codemirror/state';
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
  showTooltip,
  type Tooltip,
} from '@codemirror/view';
import type { SpellIssue } from '$lib/api';
import * as api from '$lib/api';

// Effects for managing spell check state
export const setSpellCheckEnabled = StateEffect.define<boolean>();
export const setSpellCheckLanguage = StateEffect.define<'de' | 'en'>();
export const setSpellCheckIssues = StateEffect.define<SpellIssue[]>();
export const clearSpellCheckIssues = StateEffect.define<null>();

// Configuration interface
export interface SpellCheckConfig {
  enabled: boolean;
  language: 'de' | 'en';
  debounceMs: number;
}

// Default configuration
const defaultConfig: SpellCheckConfig = {
  enabled: false,
  language: 'en',
  debounceMs: 2000, // 2 seconds after typing stops
};

// State field for spell check issues
interface SpellCheckState {
  issues: SpellIssue[];
  enabled: boolean;
  language: 'de' | 'en';
}

const spellCheckState = StateField.define<SpellCheckState>({
  create() {
    return { issues: [], enabled: false, language: 'en' };
  },
  update(state, tr) {
    let newState = state;

    for (const effect of tr.effects) {
      if (effect.is(setSpellCheckEnabled)) {
        newState = { ...newState, enabled: effect.value };
        if (!effect.value) {
          newState = { ...newState, issues: [] }; // Clear issues when disabled
        }
      } else if (effect.is(setSpellCheckLanguage)) {
        newState = { ...newState, language: effect.value };
      } else if (effect.is(setSpellCheckIssues)) {
        newState = { ...newState, issues: effect.value };
      } else if (effect.is(clearSpellCheckIssues)) {
        newState = { ...newState, issues: [] };
      }
    }

    return newState;
  },
});

// Decoration for spelling errors (red wavy underline)
const spellingDecoration = Decoration.mark({
  class: 'cm-spell-error',
  attributes: { 'data-spell-type': 'spelling' },
});

// Decoration for grammar errors (blue wavy underline)
const grammarDecoration = Decoration.mark({
  class: 'cm-grammar-error',
  attributes: { 'data-spell-type': 'grammar' },
});

/**
 * Convert byte offset to character position in the document.
 * JavaScript strings are UTF-16, but our byte offsets are UTF-8.
 */
function byteOffsetToCharPos(text: string, byteOffset: number): number {
  const encoder = new TextEncoder();
  let charPos = 0;
  let bytePos = 0;

  for (const char of text) {
    const charBytes = encoder.encode(char).length;
    if (bytePos + charBytes > byteOffset) break;
    bytePos += charBytes;
    charPos += char.length; // UTF-16 code units
  }

  return charPos;
}

/**
 * Convert byte length to character length starting from a byte offset.
 */
function byteLengthToCharLength(text: string, byteOffset: number, byteLength: number): number {
  const encoder = new TextEncoder();
  let bytePos = 0;
  let charPos = 0;
  let startCharPos = 0;
  let foundStart = false;

  for (const char of text) {
    const charBytes = encoder.encode(char).length;

    if (!foundStart && bytePos >= byteOffset) {
      startCharPos = charPos;
      foundStart = true;
    }

    if (foundStart && bytePos >= byteOffset + byteLength) {
      return charPos - startCharPos;
    }

    bytePos += charBytes;
    charPos += char.length;
  }

  return charPos - startCharPos;
}

// Plugin to create decorations from spell check issues
const spellCheckDecorations = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = this.buildDecorations(view);
    }

    update(update: ViewUpdate) {
      // Rebuild decorations if document changed or state changed
      if (
        update.docChanged ||
        update.state.field(spellCheckState) !== update.startState.field(spellCheckState)
      ) {
        this.decorations = this.buildDecorations(update.view);
      }
    }

    buildDecorations(view: EditorView): DecorationSet {
      const state = view.state.field(spellCheckState);
      if (!state.enabled || state.issues.length === 0) {
        return Decoration.none;
      }

      const doc = view.state.doc.toString();
      const decorations: { from: number; to: number; decoration: Decoration }[] = [];

      for (const issue of state.issues) {
        // Convert byte offset to character position
        const from = byteOffsetToCharPos(doc, issue.byte_offset);
        const length = byteLengthToCharLength(doc, issue.byte_offset, issue.byte_length);
        const to = from + length;

        // Skip if out of bounds
        if (from < 0 || to > doc.length || from >= to) continue;

        const decoration = issue.type === 'grammar' ? grammarDecoration : spellingDecoration;
        decorations.push({ from, to, decoration });
      }

      // Sort by position
      decorations.sort((a, b) => a.from - b.from);

      return Decoration.set(decorations.map((d) => d.decoration.range(d.from, d.to)));
    }
  },
  {
    decorations: (v) => v.decorations,
  }
);

// Plugin to handle debounced spell checking
function createSpellCheckPlugin(config: SpellCheckConfig): Extension {
  return ViewPlugin.fromClass(
    class {
      private debounceTimer: ReturnType<typeof setTimeout> | null = null;
      private lastContent: string = '';

      constructor(private view: EditorView) {
        // Initial check if enabled
        const state = view.state.field(spellCheckState);
        if (state.enabled) {
          this.scheduleCheck();
        }
      }

      update(update: ViewUpdate) {
        const state = update.state.field(spellCheckState);
        const prevState = update.startState.field(spellCheckState);

        // Check if enabled status changed
        if (state.enabled !== prevState.enabled) {
          if (state.enabled) {
            this.scheduleCheck();
          } else {
            this.cancelCheck();
          }
        }

        // Check if language changed (re-check)
        if (state.language !== prevState.language && state.enabled) {
          this.scheduleCheck();
        }

        // Check if document changed
        if (update.docChanged && state.enabled) {
          this.scheduleCheck();
        }
      }

      destroy() {
        this.cancelCheck();
      }

      private cancelCheck() {
        if (this.debounceTimer) {
          clearTimeout(this.debounceTimer);
          this.debounceTimer = null;
        }
      }

      private scheduleCheck() {
        this.cancelCheck();
        this.debounceTimer = setTimeout(() => {
          this.performCheck();
        }, config.debounceMs);
      }

      private async performCheck() {
        const state = this.view.state.field(spellCheckState);
        if (!state.enabled) return;

        const content = this.view.state.doc.toString();

        // Skip if content hasn't changed
        if (content === this.lastContent) return;
        this.lastContent = content;

        // Skip empty or very short content
        if (content.length < 10) {
          this.view.dispatch({
            effects: clearSpellCheckIssues.of(null),
          });
          return;
        }

        try {
          const issues = await api.spellCheck(content, state.language);

          // Only update if view is still valid and content hasn't changed
          if (this.view.state.doc.toString() === content) {
            this.view.dispatch({
              effects: setSpellCheckIssues.of(issues),
            });
          }
        } catch (error) {
          console.error('Spell check failed:', error);
          // Clear issues on error
          this.view.dispatch({
            effects: clearSpellCheckIssues.of(null),
          });
        }
      }
    }
  );
}

// Tooltip for spell check suggestions
const spellCheckTooltip = StateField.define<readonly Tooltip[]>({
  create() {
    return [];
  },

  update(tooltips, tr) {
    // Handle selection changes
    if (!tr.docChanged && !tr.selection) return tooltips;

    const state = tr.state.field(spellCheckState);
    if (!state.enabled || state.issues.length === 0) return [];

    // Get cursor position
    const pos = tr.state.selection.main.head;
    const doc = tr.state.doc.toString();

    // Find issue at cursor position
    for (const issue of state.issues) {
      const from = byteOffsetToCharPos(doc, issue.byte_offset);
      const length = byteLengthToCharLength(doc, issue.byte_offset, issue.byte_length);
      const to = from + length;

      if (pos >= from && pos <= to) {
        return [
          {
            pos: from,
            above: true,
            strictSide: true,
            arrow: true,
            create: () => {
              const dom = document.createElement('div');
              dom.className = 'cm-spell-tooltip';

              // Message
              const msgDiv = document.createElement('div');
              msgDiv.className = 'cm-spell-tooltip-message';
              msgDiv.textContent = issue.message;
              dom.appendChild(msgDiv);

              // Suggestions
              if (issue.suggestions && issue.suggestions.length > 0) {
                const suggestionsDiv = document.createElement('div');
                suggestionsDiv.className = 'cm-spell-tooltip-suggestions';

                for (const suggestion of issue.suggestions) {
                  const btn = document.createElement('button');
                  btn.className = 'cm-spell-suggestion-btn';
                  btn.textContent = suggestion;
                  btn.onclick = () => {
                    // Dispatch custom event - Editor.svelte handles the replacement
                    const event = new CustomEvent('spell-check-replace', {
                      detail: { from, to, replacement: suggestion },
                    });
                    document.dispatchEvent(event);
                  };
                  suggestionsDiv.appendChild(btn);
                }

                dom.appendChild(suggestionsDiv);
              }

              return { dom };
            },
          },
        ];
      }
    }

    return [];
  },

  provide: (f) => showTooltip.computeN([f], (state) => state.field(f)),
});

// Compartment for spell check extension
export const spellCheckCompartment = new Compartment();

// Create the spell check extension
export function createSpellCheckExtension(config: Partial<SpellCheckConfig> = {}): Extension {
  const fullConfig: SpellCheckConfig = { ...defaultConfig, ...config };

  return [
    spellCheckState,
    spellCheckDecorations,
    createSpellCheckPlugin(fullConfig),
    spellCheckTooltip,
    // CSS styles for spell check decorations
    EditorView.baseTheme({
      '.cm-spell-error': {
        textDecoration: 'underline wavy red',
        textDecorationSkipInk: 'none',
      },
      '.cm-grammar-error': {
        textDecoration: 'underline wavy blue',
        textDecorationSkipInk: 'none',
      },
      '.cm-spell-tooltip': {
        backgroundColor: 'var(--color-popover)',
        border: '1px solid var(--color-border)',
        borderRadius: '4px',
        padding: '8px',
        maxWidth: '300px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
        fontSize: '13px',
      },
      '.cm-spell-tooltip-message': {
        marginBottom: '6px',
        color: 'var(--color-popover-foreground)',
      },
      '.cm-spell-tooltip-suggestions': {
        display: 'flex',
        flexWrap: 'wrap',
        gap: '4px',
      },
      '.cm-spell-suggestion-btn': {
        padding: '2px 8px',
        backgroundColor: 'var(--color-accent)',
        border: '1px solid var(--color-border)',
        borderRadius: '4px',
        cursor: 'pointer',
        fontSize: '12px',
        '&:hover': {
          backgroundColor: 'var(--color-muted)',
        },
      },
    }),
  ];
}

// Toggle spell check on/off
export function toggleSpellCheck(view: EditorView, enabled: boolean) {
  view.dispatch({
    effects: setSpellCheckEnabled.of(enabled),
  });
}

// Change spell check language
export function setSpellLanguage(view: EditorView, language: 'de' | 'en') {
  view.dispatch({
    effects: setSpellCheckLanguage.of(language),
  });
}

// Trigger a manual spell check
export function triggerSpellCheck(view: EditorView) {
  const content = view.state.doc.toString();
  const state = view.state.field(spellCheckState);

  if (!state.enabled || content.length < 10) return;

  api
    .spellCheck(content, state.language)
    .then((issues) => {
      view.dispatch({
        effects: setSpellCheckIssues.of(issues),
      });
    })
    .catch((error) => {
      console.error('Manual spell check failed:', error);
    });
}

// Get current spell check state
export function getSpellCheckState(view: EditorView): SpellCheckState {
  return view.state.field(spellCheckState);
}
