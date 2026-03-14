import { afterEach, describe, expect, it, vi } from 'vitest';

const serverCollapseStateByNote = new Map<string, Record<string, boolean> | null>();

vi.mock('$lib/api/notes', () => ({
  getNoteUserState: vi.fn(async (noteId: string) => ({
    collapse_state: serverCollapseStateByNote.get(noteId) ?? null,
  })),
  updateNoteUserCollapseState: vi.fn(async (noteId: string, state: Record<string, boolean>) => {
    serverCollapseStateByNote.set(noteId, { ...state });
    return { status: 'ok' };
  }),
}));

vi.mock('$lib/api/client', () => ({
  ApiError: class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
      this.name = 'ApiError';
    }
  },
}));

import { ApiError } from '$lib/api/client';
import { getNoteUserState, updateNoteUserCollapseState } from '$lib/api/notes';

import { createEditor, setLivePreviewMode, updateEditorContent } from './codemirror';
import { setLivePreviewProfilerSink } from './live-preview';
import { _resetLivePreviewPersistenceForTest } from './live-preview/utilities';

describe('codemirror live interactions', () => {
  if (!Range.prototype.getClientRects) {
    Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
  }
  if (!Range.prototype.getBoundingClientRect) {
    Range.prototype.getBoundingClientRect = () => new DOMRect();
  }

  afterEach(() => {
    document.body.innerHTML = '';
    localStorage.clear();
    serverCollapseStateByNote.clear();
    _resetLivePreviewPersistenceForTest();
    vi.restoreAllMocks();
    vi.useRealTimers();
    setLivePreviewProfilerSink(null);
  });

  it('toggles task by source line when clicking live task checkbox', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onToggleTaskByLine = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Task',
      onToggleTaskByLine,
    });

    setLivePreviewMode(view, true);

    const checkbox = parent.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();

    checkbox?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onToggleTaskByLine).toHaveBeenCalledTimes(1);
    expect(onToggleTaskByLine).toHaveBeenCalledWith(2, true);

    view.destroy();
  });

  it('prevents editor line selection on mousedown before checkbox click', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onToggleTaskByLine = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Task',
      onToggleTaskByLine,
    });

    setLivePreviewMode(view, true);

    const checkbox = parent.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();

    checkbox?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    checkbox?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onToggleTaskByLine).toHaveBeenCalledTimes(1);
    expect(onToggleTaskByLine).toHaveBeenCalledWith(2, true);

    view.destroy();
  });

  it('triggers wikilink callback when clicking live wikilink widget', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onWikilinkClick = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n[[Target Note|Alias]]',
      onWikilinkClick,
    });

    setLivePreviewMode(view, true);

    const wikilink = parent.querySelector('.cm-live-preview-wikilink') as HTMLElement | null;
    expect(wikilink).not.toBeNull();

    wikilink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onWikilinkClick).toHaveBeenCalledTimes(1);
    expect(onWikilinkClick).toHaveBeenCalledWith('Target Note', false);

    view.destroy();
  });

  it('opens markdown links from live link widget in new tab', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    const view = createEditor(parent, {
      doc: 'Active\n[Docs](https://example.com/docs)',
    });

    setLivePreviewMode(view, true);

    const link = parent.querySelector('.cm-live-preview-link') as HTMLElement | null;
    expect(link).not.toBeNull();

    link?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(openSpy).toHaveBeenCalledTimes(1);
    expect(openSpy).toHaveBeenCalledWith(
      'https://example.com/docs',
      '_blank',
      'noopener,noreferrer'
    );

    view.destroy();
  });

  it('blocks unsafe markdown link schemes from live link widgets', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    const view = createEditor(parent, {
      doc: 'Active\n[Bad](javascript:alert(1))',
    });

    setLivePreviewMode(view, true);

    const link = parent.querySelector('.cm-live-preview-link') as HTMLElement | null;
    expect(link).not.toBeNull();

    link?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(openSpy).not.toHaveBeenCalled();

    view.destroy();
  });

  it('does not render a line-number gutter', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Open task\n- [x] Done 1\n- [x] Done 2',
    });

    setLivePreviewMode(view, true);

    expect(parent.querySelector('.cm-lineNumbers')).toBeNull();

    view.destroy();
  });

  it('renders due date once on inactive task lines in live preview', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: '- [ ] First @due(2026-02-10)\n- [ ] Second @due(2026-02-11)',
    });

    setLivePreviewMode(view, true);

    const dueWidgets = parent.querySelectorAll('.cm-live-preview-due');
    expect(dueWidgets.length).toBeGreaterThan(0);

    const lines = Array.from(parent.querySelectorAll('.cm-line'));
    const inactiveSecondLine = lines.find((line) => line.textContent?.includes('Second')) as
      | HTMLElement
      | undefined;
    expect(inactiveSecondLine).toBeTruthy();

    const lineText = inactiveSecondLine?.textContent ?? '';
    expect(lineText.includes('@due(')).toBe(false);
    const dateMatches = lineText.match(/2026-02-11/g) ?? [];
    expect(dateMatches.length).toBe(1);

    view.destroy();
  });

  it('collapses heading section when clicking live heading toggle', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: '# Heading\nline a\nline b\n## Sub\nsub line',
    });
    setLivePreviewMode(view, true);

    const toggle = parent.querySelector('.cm-live-heading-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    const hiddenLines = parent.querySelectorAll('.cm-live-collapsed-line');
    expect(hiddenLines.length).toBeGreaterThan(0);

    view.destroy();
  });

  it('moves cursor to completed-group start when clicking collapsed summary', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: 'Active\n- [x] Done 1\n- [x] Done 2\nNext',
    });
    setLivePreviewMode(view, true);

    let summary = parent.querySelector('.cm-live-task-group-summary') as HTMLElement | null;
    if (!summary) {
      const toggle = parent.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
      expect(toggle).not.toBeNull();
      toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
      toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
      summary = parent.querySelector('.cm-live-task-group-summary') as HTMLElement | null;
    }
    expect(summary).not.toBeNull();
    summary?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    summary?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    expect(view.state.doc.lineAt(view.state.selection.main.anchor).number).toBe(2);

    view.destroy();
  });

  it('preserves collapsed task-group state across note switches (A -> B -> A)', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const docA = 'Active\n- [x] Done 1\n- [x] Done 2\nNext';
    const docB = 'Other note\n- [ ] Open task';

    const view = createEditor(parent, { doc: docA });
    setLivePreviewMode(view, true, { noteId: 'note-a' });

    const toggleA = parent.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggleA).not.toBeNull();
    toggleA?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggleA?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(parent.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    // Fixed lifecycle order used in Editor.svelte:
    // 1) update document content  2) reconfigure live preview with target noteId
    updateEditorContent(view, docB);
    setLivePreviewMode(view, true, { noteId: 'note-b' });
    expect(parent.querySelectorAll('.cm-live-collapsed-line').length).toBe(0);

    updateEditorContent(view, docA);
    setLivePreviewMode(view, true, { noteId: 'note-a' });
    expect(parent.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    view.destroy();
  });

  it('preserves collapsed state across A -> B -> A and reload (server sync regression)', async () => {
    vi.useFakeTimers();

    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const docA = 'Active\n- [x] Done 1\n- [x] Done 2\nNext';
    const docB = 'Other note\n- [ ] Open task';

    const view = createEditor(parent, { doc: docA });
    setLivePreviewMode(view, true, { noteId: 'note-a' });

    const toggleA = parent.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggleA).not.toBeNull();
    toggleA?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggleA?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(parent.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    await vi.advanceTimersByTimeAsync(700); // debounce + async sync

    updateEditorContent(view, docB);
    setLivePreviewMode(view, true, { noteId: 'note-b' });
    updateEditorContent(view, docA);
    setLivePreviewMode(view, true, { noteId: 'note-a' });
    expect(parent.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    view.destroy();

    // Simulate fresh page load and force restore from server (not local cache/state)
    localStorage.clear();
    _resetLivePreviewPersistenceForTest();

    const parentReload = document.createElement('div');
    document.body.appendChild(parentReload);
    const reloaded = createEditor(parentReload, { doc: docA });
    setLivePreviewMode(reloaded, true, { noteId: 'note-a' });

    await vi.advanceTimersByTimeAsync(50); // async getNoteUserState -> view.dispatch

    expect(parentReload.querySelectorAll('.cm-live-collapsed-line').length).toBeGreaterThan(0);

    reloaded.destroy();
  });

  it('keeps server sync enabled for other notes after a 404 on one note', async () => {
    vi.useFakeTimers();

    const getStateMock = vi.mocked(getNoteUserState);
    const updateStateMock = vi.mocked(updateNoteUserCollapseState);

    getStateMock.mockImplementation(async (noteId: string) => {
      if (noteId === 'note-a') {
        throw new ApiError('note not found', 404);
      }
      return { collapse_state: serverCollapseStateByNote.get(noteId) ?? null };
    });
    updateStateMock.mockImplementation(async (noteId: string, state: Record<string, boolean>) => {
      if (noteId === 'note-a') {
        throw new ApiError('note not found', 404);
      }
      serverCollapseStateByNote.set(noteId, { ...state });
      return { status: 'ok' };
    });

    const doc = 'Active\n- [x] Done 1\n- [x] Done 2\nNext';

    const parentA = document.createElement('div');
    document.body.appendChild(parentA);
    const viewA = createEditor(parentA, { doc });
    setLivePreviewMode(viewA, true, { noteId: 'note-a' });
    const toggleA = parentA.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggleA).not.toBeNull();
    toggleA?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggleA?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(700);
    viewA.destroy();

    const parentB = document.createElement('div');
    document.body.appendChild(parentB);
    const viewB = createEditor(parentB, { doc });
    setLivePreviewMode(viewB, true, { noteId: 'note-b' });
    const toggleB = parentB.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggleB).not.toBeNull();
    toggleB?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggleB?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(700);

    expect(updateStateMock).toHaveBeenCalledWith('note-b', expect.any(Object));

    viewB.destroy();
  });

  it('caps synced collapse_state payload to backend limit while preserving non-task keys', async () => {
    vi.useFakeTimers();

    const getStateMock = vi.mocked(getNoteUserState);
    const updateStateMock = vi.mocked(updateNoteUserCollapseState);

    const baseState: Record<string, boolean> = {};
    for (let i = 0; i < 49; i++) {
      baseState[i.toString(36)] = true;
    }

    getStateMock.mockResolvedValue({ collapse_state: baseState });
    updateStateMock.mockResolvedValue({ status: 'ok' });

    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: 'Active\n- [x] Done 1\n- [x] Done 2\nNext',
    });
    setLivePreviewMode(view, true, { noteId: 'note-limit' });

    const toggle = parent.querySelector('.cm-live-task-group-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    await vi.advanceTimersByTimeAsync(700);

    const updateCall = updateStateMock.mock.calls.find((call) => call[0] === 'note-limit');
    expect(updateCall).toBeTruthy();
    const payload = updateCall?.[1] as Record<string, boolean>;
    expect(Object.keys(payload).length).toBe(50);
    expect(Object.keys(payload).filter((key) => key.startsWith('tasks:')).length).toBe(1);
    expect(payload['0']).toBe(true);

    view.destroy();
  });
});
