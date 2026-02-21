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
    onTitleInput: vi.fn(),
    onOpenSidebar: vi.fn(),
    onSetEditorMode: vi.fn(),
    onInsertTask: vi.fn(),
    onInsertTable: vi.fn(),
    onSave: vi.fn(),
    onUpload: vi.fn(),
    onShowHistory: vi.fn(),
    onToggleFocus: vi.fn(),
    onToggleAutosave: vi.fn(),
    onAIActions: vi.fn(),
    onOpenMoreMenu: vi.fn(),
  };
}

describe('EditorToolbar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders title input with note title', () => {
    const cbs = mockCallbacks();
    const { container } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    const titleInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    expect(titleInput).toBeInTheDocument();
    expect(titleInput.value).toBe('Test Note');
  });

  it('sets title input readonly for journal notes', () => {
    const cbs = mockCallbacks();
    const journalNote = { ...baseNote, note_type: 'journal' };
    const { container } = render(EditorToolbar, {
      props: { note: journalNote, ...cbs },
    });

    const titleInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    expect(titleInput.readOnly).toBe(true);
  });

  it('title input is editable for regular notes', () => {
    const cbs = mockCallbacks();
    const { container } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    const titleInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    expect(titleInput.readOnly).toBe(false);
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

  it('calls onInsertTable when table button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.table'));
    expect(cbs.onInsertTable).toHaveBeenCalledTimes(1);
  });

  it('calls onInsertTask when task button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.task'));
    expect(cbs.onInsertTask).toHaveBeenCalledTimes(1);
  });

  it('calls onUpload when upload button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.upload'));
    expect(cbs.onUpload).toHaveBeenCalledTimes(1);
  });

  it('calls onShowHistory when history button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.history'));
    expect(cbs.onShowHistory).toHaveBeenCalledTimes(1);
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

  it('shows sidebar menu button on mobile', () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: true, ...cbs },
    });
    expect(getByLabelText('component.editor.toolbar.open_menu')).toBeInTheDocument();
  });

  it('hides sidebar menu button on desktop', () => {
    const cbs = mockCallbacks();
    const { queryByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, isMobile: false, ...cbs },
    });
    expect(queryByLabelText('component.editor.toolbar.open_menu')).not.toBeInTheDocument();
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

  it('calls onToggleAutosave when autosave button clicked', async () => {
    const cbs = mockCallbacks();
    const { getByLabelText } = render(EditorToolbar, {
      props: { note: baseNote, ...cbs },
    });

    await fireEvent.click(getByLabelText('component.editor.toolbar.autosave'));
    expect(cbs.onToggleAutosave).toHaveBeenCalledTimes(1);
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
