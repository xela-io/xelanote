import { fireEvent, render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';

// jsdom lacks ResizeObserver — provide a stub
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}

// Mock svelte-i18n: $_ returns the key as-is, $derived readable
vi.mock('svelte-i18n', () => {
  const t = (key: string) => key;
  return {
    _: {
      subscribe: (fn: (v: typeof t) => void) => {
        fn(t);
        return () => {};
      },
    },
    $_: t,
  };
});

vi.mock('$lib/config', () => ({
  FEATURE_FLAGS: {
    livePreview: true,
    taskLists: true,
    spellCheck: true,
    tagSuggestions: true,
    linkSuggestions: true,
  },
}));

vi.mock('$lib/utils/time', () => ({
  formatRelativeTime: vi.fn(() => 'just now'),
}));

vi.mock('../SpellCheckToggle.svelte', () => ({
  default: { render: () => ({ html: '' }) },
}));

import EditorToolbar from './EditorToolbar.svelte';

const baseNote: Note = {
  id: 'note-1',
  title: 'Test Note',
  content: 'Body',
  folder_path: '/',
  version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

function mockCallbacks() {
  return {
    onSetEditorMode: vi.fn(),
    onSave: vi.fn(),
    onUndo: vi.fn(),
    onRedo: vi.fn(),
    onShowHistory: vi.fn(),
    onToggleFocus: vi.fn(),
    onAIActions: vi.fn(),
    onOpenInsertMenu: vi.fn(),
    onOpenMoreMenu: vi.fn(),
  };
}

describe('EditorToolbar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('does not render title input for regular notes (inline title in editor)', () => {
    const cbs = mockCallbacks();
    const { container } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    const titleInput = container.querySelector('input[type="text"]');
    expect(titleInput).not.toBeInTheDocument();
  });

  it('shows read-only title span for journal notes', () => {
    const cbs = mockCallbacks();
    const journalNote = { ...baseNote, note_type: 'journal', title: '2026-01-15' };
    const { container } = render(EditorToolbar, {
      props: { note: journalNote, ...cbs },
    });

    const titleSpan = container.querySelector('span.truncate');
    expect(titleSpan).toBeInTheDocument();
    expect(titleSpan?.textContent?.trim()).toBe('2026-01-15');
  });

  it('save button is disabled when not dirty', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isDirty: false, isSaving: false, ...cbs },
    });

    const saveBtn = getByLabelText('component.editor.toolbar.save');
    expect(saveBtn).toBeDisabled();
  });

  it('save button is enabled when dirty', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isDirty: true, isSaving: false, ...cbs },
    });

    const saveBtn = getByLabelText('component.editor.toolbar.save');
    expect(saveBtn).not.toBeDisabled();
  });

  it('save button is disabled while saving', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isDirty: true, isSaving: true, ...cbs },
    });

    const saveBtn = getByLabelText('component.editor.toolbar.save');
    expect(saveBtn).toBeDisabled();
  });

  it('calls onSave when save button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isDirty: true, isSaving: false, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.save'));
    expect(cbs.onSave).toHaveBeenCalledTimes(1);
  });

  it('keeps insert actions out of the main toolbar', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    expect(queryByLabelText('component.editor.toolbar.task')).not.toBeInTheDocument();
    expect(queryByLabelText('component.editor.toolbar.table')).not.toBeInTheDocument();
    expect(queryByLabelText('component.editor.toolbar.upload')).not.toBeInTheDocument();
  });

  it('opens insert menu when insert button is clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.table_insert.insert'));
    expect(cbs.onOpenInsertMenu).toHaveBeenCalledTimes(1);
  });

  it('calls onShowHistory when history button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.history'));
    expect(cbs.onShowHistory).toHaveBeenCalledTimes(1);
  });

  it('allows direct mode selection on desktop', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: false, editorMode: 'edit', ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.mode_preview'));
    expect(cbs.onSetEditorMode).toHaveBeenCalledWith('preview');
  });

  it('uses compact mode menu on mobile without split option', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText, queryByText, getAllByText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, editorMode: 'live', ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.mode_group'));
    expect(getAllByText('component.editor.toolbar.mode_live').length).toBeGreaterThan(0);
    expect(getAllByText('component.editor.toolbar.mode_edit').length).toBeGreaterThan(0);
    expect(getAllByText('component.editor.toolbar.mode_preview').length).toBeGreaterThan(0);
    expect(queryByText('component.editor.toolbar.mode_split')).not.toBeInTheDocument();
  });

  it('shows focus mode toggle on desktop, not on mobile', () => {
    const cbs = mockCallbacks();

    // Desktop: focus button visible
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: false, ...cbs },
    });
    expect(getByLabelText('component.editor.toolbar.focus_mode')).toBeInTheDocument();
  });

  it('hides focus mode toggle on mobile', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, ...cbs },
    });
    expect(queryByLabelText('component.editor.toolbar.focus_mode')).not.toBeInTheDocument();
  });

  it('shows undo button on mobile and calls onUndo', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, canUndo: true, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.undo'));
    expect(cbs.onUndo).toHaveBeenCalledTimes(1);
  });

  it('disables undo button on mobile when no undo is available', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, canUndo: false, ...cbs },
    });

    expect(getByLabelText('component.editor.toolbar.undo')).toBeDisabled();
  });

  it('shows redo button on mobile and calls onRedo', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, canRedo: true, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.redo'));
    expect(cbs.onRedo).toHaveBeenCalledTimes(1);
  });

  it('disables redo button on mobile when no redo is available', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, canRedo: false, ...cbs },
    });

    expect(getByLabelText('component.editor.toolbar.redo')).toBeDisabled();
  });

  it('shows AI actions button only when aiEnabled and in edit-compatible mode', () => {
    const cbs = mockCallbacks();

    // AI enabled + edit mode → visible
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, aiEnabled: true, editorMode: 'edit', ...cbs },
    });
    expect(getByLabelText('component.editor.ai_actions')).toBeInTheDocument();
  });

  it('hides AI actions button when aiEnabled=false', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, aiEnabled: false, editorMode: 'edit', ...cbs },
    });
    expect(queryByLabelText('component.editor.ai_actions')).not.toBeInTheDocument();
  });

  it('hides AI actions button in preview mode', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, aiEnabled: true, editorMode: 'preview', ...cbs },
    });
    expect(queryByLabelText('component.editor.ai_actions')).not.toBeInTheDocument();
  });

  it('does not show autosave toggle in the main toolbar', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    expect(queryByLabelText('component.editor.toolbar.autosave')).not.toBeInTheDocument();
  });

  it('shows more menu button with correct aria-expanded', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, showMoreMenu: false, ...cbs },
    });

    const moreBtn = getByLabelText('component.editor.toolbar.more_options');
    expect(moreBtn).toHaveAttribute('aria-expanded', 'false');
  });
});
