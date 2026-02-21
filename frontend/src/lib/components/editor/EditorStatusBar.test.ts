import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';

// Mock svelte-i18n
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
    tagSuggestions: true,
    linkSuggestions: true,
  },
}));

// Mock the EditorPanels sub-component to avoid pulling in its dependencies
vi.mock('./EditorPanels.svelte', () => {
  return {
    default: function EditorPanelsMock() {},
  };
});

import EditorStatusBar from './EditorStatusBar.svelte';

const baseNote: Note = {
  id: 'note-1',
  title: 'Test',
  content: '',
  folder_path: '/',
  version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const baseProps = {
  note: baseNote,
  backlinks: [],
  editorView: undefined,
  onInsertLink: vi.fn(),
  onSummaryUpdated: vi.fn(),
};

describe('EditorStatusBar', () => {
  it('shows toggle button on desktop when panels are collapsed', () => {
    const onToggle = vi.fn();
    const { container } = render(EditorStatusBar, {
      props: {
        ...baseProps,
        isMobile: false,
        editorPanelsCollapsed: true,
        onTogglePanelsCollapsed: onToggle,
      },
    });

    const btn = container.querySelector('button');
    expect(btn).toBeInTheDocument();
    expect(btn!.getAttribute('aria-expanded')).toBe('false');
  });

  it('shows toggle button on desktop when panels are expanded', () => {
    const onToggle = vi.fn();
    const { container } = render(EditorStatusBar, {
      props: {
        ...baseProps,
        isMobile: false,
        editorPanelsCollapsed: false,
        onTogglePanelsCollapsed: onToggle,
      },
    });

    const btn = container.querySelector('button');
    expect(btn).toBeInTheDocument();
    expect(btn!.getAttribute('aria-expanded')).toBe('true');
  });

  it('calls onTogglePanelsCollapsed when toggle button clicked', async () => {
    const onToggle = vi.fn();
    const { container } = render(EditorStatusBar, {
      props: {
        ...baseProps,
        isMobile: false,
        editorPanelsCollapsed: true,
        onTogglePanelsCollapsed: onToggle,
      },
    });

    const btn = container.querySelector('button')!;
    await fireEvent.click(btn);
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it('hides toggle button on mobile', () => {
    const onToggle = vi.fn();
    const { container } = render(EditorStatusBar, {
      props: {
        ...baseProps,
        isMobile: true,
        editorPanelsCollapsed: false,
        onTogglePanelsCollapsed: onToggle,
      },
    });

    // On mobile, no toggle button is rendered
    const btn = container.querySelector('button');
    expect(btn).not.toBeInTheDocument();
  });
});
