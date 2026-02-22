// CodeMirror theme and syntax highlighting styles

import { HighlightStyle } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import { tags } from '@lezer/highlight';

export const lightTheme = EditorView.theme({
  '&': {
    backgroundColor: 'var(--color-background)',
    color: 'var(--color-foreground)',
    fontFamily: 'var(--font-sans)',
  },
  '.cm-content': {
    caretColor: 'var(--color-foreground)',
    fontFamily: 'var(--font-sans)',
  },
  '.cm-cursor': {
    borderLeftColor: 'var(--color-foreground)',
  },
  '.cm-activeLine': {
    backgroundColor: 'color-mix(in srgb, var(--color-muted) 15%, transparent)',
  },
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

export const markdownSyntaxStyle = HighlightStyle.define([
  {
    tag: [tags.meta, tags.punctuation],
    color: 'var(--color-muted-foreground)',
    fontWeight: '600',
  },
  {
    tag: tags.atom,
    color: 'var(--color-muted-foreground)',
  },
  {
    tag: [tags.link, tags.url],
    color: 'var(--color-primary)',
  },
  {
    tag: tags.strong,
    fontWeight: '700',
  },
  {
    tag: tags.emphasis,
    fontStyle: 'italic',
  },
]);
