import type { EditorView } from '@codemirror/view';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { handleLiveTocClick, handlePreviewClick } from './preview-interactions';

function createTaskPreviewItem(attrs: { index: number; line?: number; checked?: boolean }) {
  const preview = document.createElement('div');
  preview.className = 'markdown-preview';

  const li = document.createElement('li');
  li.className = 'task-list-item';
  li.setAttribute('data-task-index', String(attrs.index));
  if (attrs.line) li.setAttribute('data-task-line', String(attrs.line));

  const label = document.createElement('label');
  const checkbox = document.createElement('input');
  checkbox.type = 'checkbox';
  checkbox.className = 'task-list-item-checkbox';
  if (attrs.checked) checkbox.setAttribute('checked', '');

  label.appendChild(checkbox);
  li.appendChild(label);
  preview.appendChild(li);
  document.body.appendChild(preview);

  return { preview, li, label, checkbox };
}

describe('handlePreviewClick', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('toggles task with source line when clicking checkbox input', () => {
    const { checkbox } = createTaskPreviewItem({ index: 0, line: 12, checked: false });
    const onToggleTask = vi.fn();
    let lastClick = 0;

    const event = new MouseEvent('click', { bubbles: true, cancelable: true });
    checkbox.dispatchEvent(event);

    handlePreviewClick(event, {
      featureTaskLists: true,
      getLastTaskClickTime: () => lastClick,
      setLastTaskClickTime: (value) => {
        lastClick = value;
      },
      onWikilink: vi.fn(),
      onToggleTask,
    });

    expect(onToggleTask).toHaveBeenCalledTimes(1);
    expect(onToggleTask).toHaveBeenCalledWith(0, true, 12);
    expect(lastClick).toBeGreaterThan(0);
  });

  it('falls back to index-only toggle when source line is missing', () => {
    const { label } = createTaskPreviewItem({ index: 2, checked: true });
    const onToggleTask = vi.fn();

    const event = new MouseEvent('click', { bubbles: true, cancelable: true });
    label.dispatchEvent(event);

    handlePreviewClick(event, {
      featureTaskLists: true,
      getLastTaskClickTime: () => 0,
      setLastTaskClickTime: vi.fn(),
      onWikilink: vi.fn(),
      onToggleTask,
    });

    expect(onToggleTask).toHaveBeenCalledTimes(1);
    expect(onToggleTask).toHaveBeenCalledWith(2, false, undefined);
  });

  it('does not toggle during debounce window', () => {
    const { checkbox } = createTaskPreviewItem({ index: 1, line: 7, checked: false });
    const onToggleTask = vi.fn();
    const now = Date.now();

    const event = new MouseEvent('click', { bubbles: true, cancelable: true });
    checkbox.dispatchEvent(event);

    handlePreviewClick(event, {
      featureTaskLists: true,
      getLastTaskClickTime: () => now - 100,
      setLastTaskClickTime: vi.fn(),
      onWikilink: vi.fn(),
      onToggleTask,
    });

    expect(onToggleTask).not.toHaveBeenCalled();
    expect(event.defaultPrevented).toBe(true);
  });
});

describe('handleLiveTocClick', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('scrolls CodeMirror editor to the matching heading slug', () => {
    const dispatch = vi.fn();
    const focus = vi.fn();
    const liveEditorView = {
      state: {
        doc: {
          lines: 4,
          line: (lineNumber: number) => ({ from: (lineNumber - 1) * 10 }),
        },
      },
      dispatch,
      focus,
    } as unknown as EditorView;

    const handled = handleLiveTocClick(
      'intro-1',
      ['# Intro', '## Intro', '### Details', 'Text'].join('\n'),
      liveEditorView
    );

    expect(handled).toBe(true);
    expect(dispatch).toHaveBeenCalledWith({
      selection: { anchor: 10 },
      scrollIntoView: true,
    });
    expect(focus).toHaveBeenCalledTimes(1);
  });

  it('returns false when no matching heading slug exists', () => {
    const dispatch = vi.fn();
    const focus = vi.fn();
    const liveEditorView = {
      state: {
        doc: {
          lines: 2,
          line: (lineNumber: number) => ({ from: (lineNumber - 1) * 10 }),
        },
      },
      dispatch,
      focus,
    } as unknown as EditorView;

    const handled = handleLiveTocClick('missing-heading', '# Intro\n## Details', liveEditorView);

    expect(handled).toBe(false);
    expect(dispatch).not.toHaveBeenCalled();
    expect(focus).not.toHaveBeenCalled();
  });
});
