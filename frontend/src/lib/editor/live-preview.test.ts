import { EditorSelection, EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { afterEach, describe, expect, it } from 'vitest';

import { createLivePreviewExtension, toggleLivePreviewHeadingSection } from './live-preview';

function createView(doc: string, selectionAnchor = 0): EditorView {
  const parent = document.createElement('div');
  document.body.appendChild(parent);

  const state = EditorState.create({
    doc,
    selection: EditorSelection.cursor(selectionAnchor),
    extensions: [createLivePreviewExtension()],
  });

  return new EditorView({ state, parent });
}

describe('live-preview', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('marks non-active lines as live preview lines', () => {
    const doc = 'Line 1\nLine 2\nLine 3';
    const line2Pos = doc.indexOf('Line 2');
    const view = createView(doc, line2Pos);

    const inactiveLines = view.dom.querySelectorAll('.cm-live-preview-line');
    expect(inactiveLines.length).toBe(2);

    view.destroy();
  });

  it('renders task marker as a clickable live checkbox on inactive lines', () => {
    const doc = 'Active\n- [ ] Task';
    const view = createView(doc, 0);

    const checkbox = view.dom.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();
    expect(checkbox?.dataset.line).toBe('2');
    expect(checkbox?.dataset.checked).toBe('false');
    const input = checkbox?.querySelector(
      '.cm-live-task-checkbox-input'
    ) as HTMLInputElement | null;
    expect(input).not.toBeNull();
    expect(input?.checked).toBe(false);

    view.destroy();
  });

  it('renders task marker as a clickable live checkbox on active lines', () => {
    const doc = 'Line 1\n- [ ] Task';
    const taskLinePos = doc.indexOf('- [ ] Task');
    const view = createView(doc, taskLinePos);

    const checkbox = view.dom.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();
    expect(checkbox?.dataset.line).toBe('2');
    expect(checkbox?.dataset.checked).toBe('false');

    view.destroy();
  });

  it('renders wikilinks as live preview widgets with target metadata', () => {
    const doc = 'Active\n[[Target Note|Alias]]';
    const view = createView(doc, 0);

    const wikilink = view.dom.querySelector('.cm-live-preview-wikilink') as HTMLElement | null;
    expect(wikilink).not.toBeNull();
    expect(wikilink?.textContent).toBe('Alias');
    expect(wikilink?.dataset.title).toBe('Target Note');

    view.destroy();
  });

  it('renders markdown links with href metadata', () => {
    const doc = 'Active\n[Docs](https://example.com/docs)';
    const view = createView(doc, 0);

    const link = view.dom.querySelector('.cm-live-preview-link') as HTMLElement | null;
    expect(link).not.toBeNull();
    expect(link?.textContent).toBe('Docs');
    expect(link?.dataset.href).toBe('https://example.com/docs');

    view.destroy();
  });

  it('renders markdown links with nested parentheses href correctly', () => {
    const doc = 'Active\n[Spec](https://example.com/a_(b))';
    const view = createView(doc, 0);

    const link = view.dom.querySelector('.cm-live-preview-link') as HTMLElement | null;
    expect(link).not.toBeNull();
    expect(link?.textContent).toBe('Spec');
    expect(link?.dataset.href).toBe('https://example.com/a_(b)');

    view.destroy();
  });

  it('renders inline code as preview code widget on inactive lines', () => {
    const doc = 'Active\nUse `inline` code';
    const view = createView(doc, 0);

    const code = view.dom.querySelector('.cm-live-preview-code') as HTMLElement | null;
    expect(code).not.toBeNull();
    expect(code?.textContent).toBe('inline');

    view.destroy();
  });

  it('renders due date syntax as preview due widget', () => {
    const doc = 'Active\nTask @due(2027-02-10)';
    const view = createView(doc, 0);

    const due = view.dom.querySelector('.cm-live-preview-due') as HTMLElement | null;
    expect(due).not.toBeNull();
    expect(due?.textContent).toBe('2027-02-10');

    view.destroy();
  });

  it('renders bold and italic markdown as styled preview widgets', () => {
    const doc = 'Active\n**Bold** and *Italic*';
    const view = createView(doc, 0);

    const strong = view.dom.querySelector('.cm-live-preview-strong') as HTMLElement | null;
    const em = view.dom.querySelector('.cm-live-preview-em') as HTMLElement | null;

    expect(strong).not.toBeNull();
    expect(strong?.textContent).toBe('Bold');
    expect(em).not.toBeNull();
    expect(em?.textContent).toBe('Italic');

    view.destroy();
  });

  it('adds heading class for inactive heading lines', () => {
    const doc = 'Active\n# Heading';
    const view = createView(doc, 0);

    const headingLine = view.dom.querySelector('.cm-live-heading-h1') as HTMLElement | null;
    expect(headingLine).not.toBeNull();

    view.destroy();
  });

  it('keeps active line in raw markdown mode', () => {
    const doc = '**Bold**\nSecond';
    const view = createView(doc, 0);

    const strong = view.dom.querySelector('.cm-live-preview-strong');
    expect(strong).toBeNull();

    view.destroy();
  });

  it('renders unordered list marker as preview bullet widget', () => {
    const doc = 'Active\n- Item';
    const view = createView(doc, 0);

    const marker = view.dom.querySelector('.cm-live-list-marker') as HTMLElement | null;
    expect(marker).not.toBeNull();
    expect(marker?.textContent).toBe('• ');

    view.destroy();
  });

  it('adds code line class inside fenced code blocks', () => {
    const doc = 'Active\n```\nconst x = 1;\n```';
    const view = createView(doc, 0);

    const codeLine = view.dom.querySelector('.cm-live-code-line') as HTMLElement | null;
    expect(codeLine).not.toBeNull();

    view.destroy();
  });

  it('adds table line class for markdown table rows', () => {
    const doc = 'Active\n| A | B |\n| --- | --- |\n| 1 | 2 |';
    const view = createView(doc, 0);

    const tableLine = view.dom.querySelector('.cm-live-table-line') as HTMLElement | null;
    expect(tableLine).not.toBeNull();

    view.destroy();
  });

  it('collapses and expands heading sections via heading toggle', () => {
    const doc = '# Heading\nline a\nline b\n## Sub\nsub line';
    const view = createView(doc, 0);

    const headingToggle = view.dom.querySelector('.cm-live-heading-toggle') as HTMLElement | null;
    expect(headingToggle).not.toBeNull();
    const sectionKey = headingToggle?.dataset.section;
    expect(sectionKey).toBeTruthy();

    const beforeHidden = view.dom.querySelectorAll('.cm-live-collapsed-line');
    expect(beforeHidden.length).toBe(0);

    toggleLivePreviewHeadingSection(view, sectionKey!);
    const afterHidden = view.dom.querySelectorAll('.cm-live-collapsed-line');
    expect(afterHidden.length).toBeGreaterThan(0);

    toggleLivePreviewHeadingSection(view, sectionKey!);
    const afterExpand = view.dom.querySelectorAll('.cm-live-collapsed-line');
    expect(afterExpand.length).toBe(0);

    view.destroy();
  });
});
