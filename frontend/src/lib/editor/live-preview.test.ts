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

  it('renders table widget when cursor is not on table', () => {
    const doc = 'Active\n| A | B |\n| --- | --- |\n| 1 | 2 |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).not.toBeNull();
    // Widget should contain a rendered table
    const table = tableWidget?.querySelector('table');
    expect(table).not.toBeNull();
    // Check header cells
    const ths = tableWidget?.querySelectorAll('th');
    expect(ths?.length).toBe(2);
    expect(ths?.[0].textContent).toBe('A');
    expect(ths?.[1].textContent).toBe('B');
    // Check data cells
    const tds = tableWidget?.querySelectorAll('td');
    expect(tds?.length).toBe(2);
    expect(tds?.[0].textContent).toBe('1');
    expect(tds?.[1].textContent).toBe('2');

    view.destroy();
  });

  it('shows raw markdown when cursor is on a table line', () => {
    const doc = 'Active\n| A | B |\n| --- | --- |\n| 1 | 2 |';
    // Put cursor on the header line
    const headerPos = doc.indexOf('| A | B |');
    const view = createView(doc, headerPos);

    // Widget should NOT be rendered since cursor is on table
    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).toBeNull();
    // Table line class should be present
    const tableLine = view.dom.querySelector('.cm-live-table-line') as HTMLElement | null;
    expect(tableLine).not.toBeNull();

    view.destroy();
  });

  it('renders correct cell count and alignment in table widget', () => {
    const doc = 'Active\n| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).not.toBeNull();

    const ths = tableWidget?.querySelectorAll('th');
    expect(ths?.length).toBe(3);
    expect(ths?.[0].style.textAlign).toBe('left');
    expect(ths?.[1].style.textAlign).toBe('center');
    expect(ths?.[2].style.textAlign).toBe('right');

    const tds = tableWidget?.querySelectorAll('td');
    expect(tds?.length).toBe(3);
    expect(tds?.[0].style.textAlign).toBe('left');
    expect(tds?.[1].style.textAlign).toBe('center');
    expect(tds?.[2].style.textAlign).toBe('right');

    view.destroy();
  });

  it('does not render table inside a code block', () => {
    const doc = 'Active\n```\n| A | B |\n| --- | --- |\n| 1 | 2 |\n```';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).toBeNull();

    view.destroy();
  });

  it('renders table with only header and separator (no data rows)', () => {
    const doc = 'Active\n| A | B |\n| --- | --- |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).not.toBeNull();
    const ths = tableWidget?.querySelectorAll('th');
    expect(ths?.length).toBe(2);
    const tds = tableWidget?.querySelectorAll('td');
    expect(tds?.length).toBe(0);

    view.destroy();
  });

  it('handles escaped pipes in cells correctly', () => {
    const doc = 'Active\n| a \\| b | c |\n| --- | --- |\n| d | e |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).not.toBeNull();
    const ths = tableWidget?.querySelectorAll('th');
    expect(ths?.length).toBe(2);
    expect(ths?.[0].textContent).toBe('a | b');
    expect(ths?.[1].textContent).toBe('c');

    view.destroy();
  });

  it('handles alignment mismatch (more header cols than separator)', () => {
    const doc = 'Active\n| A | B | C |\n| --- | --- |\n| 1 | 2 | 3 |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).not.toBeNull();
    // Should still render 3 header cells even though separator has only 2
    const ths = tableWidget?.querySelectorAll('th');
    expect(ths?.length).toBe(3);

    view.destroy();
  });

  it('does not render block without separator as table widget', () => {
    const doc = 'Active\n| A | B |\n| 1 | 2 |';
    const view = createView(doc, 0);

    const tableWidget = view.dom.querySelector('.cm-table-widget') as HTMLElement | null;
    expect(tableWidget).toBeNull();

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

  it('uses distinct keys for duplicate completed task groups with identical content', () => {
    const doc = 'Active\n- [x] Done A\n- [x] Done B\n\nActive\n- [x] Done A\n- [x] Done B\nNext';
    const view = createView(doc, 0);

    const toggles = [...view.dom.querySelectorAll('.cm-live-task-group-toggle')] as HTMLElement[];
    expect(toggles.length).toBe(2);
    const firstKey = toggles[0]?.dataset.taskGroup;
    const secondKey = toggles[1]?.dataset.taskGroup;
    expect(firstKey).toBeTruthy();
    expect(secondKey).toBeTruthy();
    expect(firstKey).not.toBe(secondKey);

    toggleLivePreviewCompletedTaskGroup(view, firstKey!);
    expect(view.dom.querySelectorAll('.cm-live-collapsed-line').length).toBe(1);

    view.destroy();
  });

  it('keeps collapse on the same visual duplicate group after inserting another duplicate above', () => {
    const doc =
      'Top\n\nActive\n- [x] Done A\n- [x] Done B\n\nActive\n- [x] Done A\n- [x] Done B\nTail';
    const view = createView(doc, 0);

    const togglesBefore = [
      ...view.dom.querySelectorAll('.cm-live-task-group-toggle'),
    ] as HTMLElement[];
    expect(togglesBefore.length).toBe(2);
    const secondKeyBefore = togglesBefore[1]?.dataset.taskGroup;
    expect(secondKeyBefore).toBeTruthy();

    toggleLivePreviewCompletedTaskGroup(view, secondKeyBefore!);
    let summaries = [...view.dom.querySelectorAll('.cm-live-task-group-summary')] as HTMLElement[];
    expect(summaries.length).toBe(1);
    const collapsedLineBefore = Number.parseInt(summaries[0]?.dataset.line ?? '', 10);
    expect(Number.isInteger(collapsedLineBefore)).toBe(true);

    const insertPos = view.state.doc.line(3).from;
    view.dispatch({
      changes: {
        from: insertPos,
        to: insertPos,
        insert: 'Active\n- [x] Done A\n- [x] Done B\n\n',
      },
    });

    summaries = [...view.dom.querySelectorAll('.cm-live-task-group-summary')] as HTMLElement[];
    expect(summaries.length).toBe(1);
    const collapsedLineAfter = Number.parseInt(summaries[0]?.dataset.line ?? '', 10);
    expect(collapsedLineAfter).toBe(collapsedLineBefore + 4);

    view.destroy();
  });

  it('adds cm-live-nest-1 class for indented task', () => {
    const doc = 'Active\n- [ ] Parent\n  - [ ] Child';
    const view = createView(doc, 0);

    const nestedLines = view.dom.querySelectorAll('.cm-live-nest-1');
    expect(nestedLines.length).toBe(1);

    // Top-level task should NOT have nest class
    const taskLines = view.dom.querySelectorAll('.cm-live-task-line');
    const topLevelTask = Array.from(taskLines).find(
      (el) => !el.classList.contains('cm-live-nest-1')
    );
    expect(topLevelTask).toBeDefined();

    view.destroy();
  });

  it('does not add nest class for top-level task', () => {
    const doc = 'Active\n- [ ] Task A\n- [ ] Task B';
    const view = createView(doc, 0);

    const nestedLines = view.dom.querySelectorAll('[class*="cm-live-nest-"]');
    expect(nestedLines.length).toBe(0);

    view.destroy();
  });

  it('adds nest class to indented list items', () => {
    const doc = 'Active\n- Item\n  - Sub item';
    const view = createView(doc, 0);

    const nestedLines = view.dom.querySelectorAll('.cm-live-nest-1');
    expect(nestedLines.length).toBe(1);

    view.destroy();
  });

  it('supports multiple nesting levels', () => {
    const doc = 'Active\n- [ ] L0\n  - [ ] L1\n    - [ ] L2';
    const view = createView(doc, 0);

    expect(view.dom.querySelectorAll('.cm-live-nest-1').length).toBe(1);
    expect(view.dom.querySelectorAll('.cm-live-nest-2').length).toBe(1);

    view.destroy();
  });

  it('combines nest class with task-group class on checked nested tasks', () => {
    const doc = 'Active\n- [x] Parent\n  - [x] Child A\n  - [x] Child B';
    const view = createView(doc, 0);

    // The nested checked items should have both nest and task-group classes
    const nestedGroupLines = view.dom.querySelectorAll(
      '.cm-live-nest-1.cm-live-task-group-middle, .cm-live-nest-1.cm-live-task-group-last'
    );
    expect(nestedGroupLines.length).toBeGreaterThan(0);

    view.destroy();
  });

  it('does not group checked children within an unchecked parent as completed', () => {
    // After unchecking a child, parent auto-unchecks but other children stay checked.
    // Those checked children must NOT form their own completed task group.
    const doc =
      'Active\n- [ ] Parent\n  - [x] Child A\n  - [ ] Child B\n  - [x] Child C\n- [x] Other';
    const view = createView(doc, 0);

    // Child A and Child C are checked but parent is unchecked → no task-group class on them
    const nestedGroupLines = view.dom.querySelectorAll(
      '.cm-live-nest-1.cm-live-task-group-first, .cm-live-nest-1.cm-live-task-group-middle, .cm-live-nest-1.cm-live-task-group-last'
    );
    expect(nestedGroupLines.length).toBe(0);

    view.destroy();
  });
});
