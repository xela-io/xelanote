import { EditorSelection, EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import {
  createLivePreviewExtension,
  toggleLivePreviewCompletedTaskGroup,
  toggleLivePreviewHeadingSection,
} from './live-preview';

function createView(doc: string, selectionAnchor = 0, noteId?: string): EditorView {
  const parent = document.createElement('div');
  document.body.appendChild(parent);

  const state = EditorState.create({
    doc,
    selection: EditorSelection.cursor(selectionAnchor),
    extensions: [createLivePreviewExtension({ noteId })],
  });

  const view = new EditorView({ state, parent });
  // Simulate focus: JSDOM doesn't fire focus events properly, so manually
  // focus + re-dispatch the selection to trigger active-line recalculation.
  view.focus();
  view.dispatch({ selection: EditorSelection.cursor(selectionAnchor) });
  return view;
}

describe('live-preview', () => {
  const originalHasFocus = document.hasFocus.bind(document);

  beforeEach(() => {
    // JSDOM doesn't support document.hasFocus(), but CodeMirror's view.hasFocus
    // relies on it. Stub it so active-line detection works in tests.
    document.hasFocus = () => true;
    localStorage.clear();
  });

  afterEach(() => {
    document.hasFocus = originalHasFocus;
    document.body.innerHTML = '';
    localStorage.clear();
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
    const handle = view.dom.querySelector('.cm-live-task-drag-handle') as HTMLElement | null;
    expect(checkbox).not.toBeNull();
    expect(handle).not.toBeNull();
    expect(checkbox?.dataset.line).toBe('2');
    expect(handle?.dataset.line).toBe('2');
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
    const taskLineText = view.dom.querySelectorAll('.cm-line')[1]?.textContent ?? '';
    expect(taskLineText).not.toContain('-');
    const taskLine = view.dom.querySelectorAll('.cm-line')[1] as HTMLElement | undefined;
    expect(taskLine?.classList.contains('cm-live-task-line')).toBe(true);

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

  it('renders due date widget inside task lines in live preview', () => {
    const doc = 'Active\n- [ ] Task @due(2027-02-10)';
    const view = createView(doc, 0);

    const taskLine = view.dom.querySelector('.cm-line.cm-live-task-line') as HTMLElement | null;
    const due = view.dom.querySelector('.cm-live-preview-due') as HTMLElement | null;

    expect(taskLine).not.toBeNull();
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

  it('keeps heading styling for active heading lines with visible markdown syntax', () => {
    const doc = '# Heading\nSecond';
    const view = createView(doc, 0);

    const activeLine = view.dom.querySelectorAll('.cm-line')[0] as HTMLElement | undefined;
    expect(activeLine?.classList.contains('cm-live-heading-h1')).toBe(true);
    expect(activeLine?.textContent).toContain('# Heading');

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
    expect(marker?.textContent).toBe('•');

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

  it('persists collapsed completed task groups by noteId', () => {
    const noteId = 'note-persist-1';
    const doc = 'Active\n- [x] Done A\n- [x] Done B\nNext';
    const view = createView(doc, 0, noteId);

    const toggle = view.dom.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    const groupKey = toggle?.dataset.taskGroup;
    expect(groupKey).toBeTruthy();

    toggleLivePreviewCompletedTaskGroup(view, groupKey!);
    expect(view.dom.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);
    view.destroy();

    const recreated = createView(doc, 0, noteId);
    expect(recreated.dom.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);
    recreated.destroy();
  });

  it('keeps completed task group collapsed after task updates', () => {
    const doc = 'Active\n- [ ] Todo\n- [x] Done A\n- [x] Done B';
    const view = createView(doc, 0);

    const initialToggle = view.dom.querySelector(
      '.cm-live-task-group-toggle'
    ) as HTMLElement | null;
    expect(initialToggle).not.toBeNull();
    const groupKey = initialToggle?.dataset.taskGroup;
    expect(groupKey).toBeTruthy();

    toggleLivePreviewCompletedTaskGroup(view, groupKey!);
    expect(view.dom.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    const todoLine = view.state.doc.line(2);
    const uncheckedTokenStart = todoLine.text.indexOf('[ ]');
    expect(uncheckedTokenStart).toBeGreaterThanOrEqual(0);
    view.dispatch({
      changes: {
        from: todoLine.from + uncheckedTokenStart + 1,
        to: todoLine.from + uncheckedTokenStart + 2,
        insert: 'x',
      },
    });
    expect(view.dom.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    const groupEnd = view.state.doc.line(4).to;
    view.dispatch({
      changes: {
        from: groupEnd,
        to: groupEnd,
        insert: '\n- [x] Done C',
      },
    });
    expect(view.dom.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    view.destroy();
  });
});
