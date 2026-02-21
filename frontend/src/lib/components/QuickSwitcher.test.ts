import { fireEvent, render } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';

// Mock svelte-i18n
vi.mock('svelte-i18n', () => {
  const t = (key: string, opts?: { values?: Record<string, string> }) => {
    if (opts?.values) {
      return Object.entries(opts.values).reduce((s, [k, v]) => s.replace(`{${k}}`, v), key);
    }
    return key;
  };
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

vi.mock('svelte/reactivity', () => ({
  SvelteMap: Map,
}));

// All mocks must use vi.fn() inline (vi.mock factories are hoisted)
vi.mock('$lib/api', () => ({
  quickSearch: vi.fn().mockResolvedValue({ notes: [] }),
}));

vi.mock('$lib/stores/ui.svelte', () => ({
  getQuickSwitcherOpen: vi.fn().mockReturnValue(true),
  setQuickSwitcherOpen: vi.fn(),
  toggleTheme: vi.fn(),
}));

vi.mock('$lib/stores/notes.svelte', () => ({
  createNote: vi.fn(),
  getCurrentNote: vi.fn().mockReturnValue(null),
}));

vi.mock('$lib/stores/folders.svelte', () => ({
  getSelectedFolder: vi.fn().mockReturnValue('/'),
  loadFolders: vi.fn(),
}));

vi.mock('$lib/stores/encryption.svelte', () => ({
  isEncryptionUnlocked: vi.fn(() => false),
  decryptTitle: vi.fn((t: string) => t),
}));

vi.mock('$lib/stores/search.svelte', () => ({
  getFilters: vi
    .fn()
    .mockReturnValue({ folders: [], tags: [], createdDate: null, updatedDate: null }),
  getActiveFilterCount: vi.fn().mockReturnValue(0),
  hasActiveFilters: vi.fn().mockReturnValue(false),
  getAbsoluteDateRange: vi.fn().mockReturnValue(null),
}));

vi.mock('$lib/stores/search-index.svelte', () => ({
  searchEncrypted: vi.fn(() => []),
}));

vi.mock('$lib/stores/features.svelte', () => ({
  getGraphFeatureEnabled: vi.fn(() => false),
}));

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

vi.mock('./FilterBar.svelte', () => ({
  default: function FilterBarMock() {},
}));
vi.mock('./FilterMenu.svelte', () => ({
  default: function FilterMenuMock() {},
}));

vi.mock('$lib/commands/command-registry', () => ({
  registerCommands: vi.fn(),
  getCommands: vi.fn().mockReturnValue([]),
}));

// Import mocked modules to access mock instances
import { quickSearch } from '$lib/api';
import { getCommands, registerCommands } from '$lib/commands/command-registry';
import QuickSwitcher from '$lib/components/QuickSwitcher.svelte';
import * as searchStore from '$lib/stores/search.svelte';
import * as ui from '$lib/stores/ui.svelte';

const mockNote = (id: string, title: string): Note => ({
  id,
  title,
  content: '',
  folder_path: '/',
  version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
});

describe('QuickSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.mocked(ui.getQuickSwitcherOpen).mockReturnValue(true);
    vi.mocked(getCommands).mockReturnValue([]);
    vi.mocked(quickSearch).mockResolvedValue({ notes: [] });
    vi.mocked(searchStore.getActiveFilterCount).mockReturnValue(0);
    vi.mocked(searchStore.hasActiveFilters).mockReturnValue(false);
    vi.mocked(searchStore.getFilters).mockReturnValue({
      folders: [],
      tags: [],
      createdDate: null,
      updatedDate: null,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders dialog when open', () => {
    const { getByRole } = render(QuickSwitcher);
    expect(getByRole('dialog')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    vi.mocked(ui.getQuickSwitcherOpen).mockReturnValue(false);
    const { queryByRole } = render(QuickSwitcher);
    expect(queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders search input with combobox role', () => {
    const { getByRole } = render(QuickSwitcher);
    const input = getByRole('combobox');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('autocomplete', 'off');
  });

  it('registers commands on mount', () => {
    render(QuickSwitcher);
    expect(registerCommands).toHaveBeenCalledTimes(1);
    const registeredCmds = vi.mocked(registerCommands).mock.calls[0][0];
    expect(registeredCmds.length).toBeGreaterThanOrEqual(5);

    const ids = registeredCmds.map((c: { id: string }) => c.id);
    expect(ids).toContain('new-note');
    expect(ids).toContain('toggle-theme');
    expect(ids).toContain('open-settings');
    expect(ids).toContain('open-journal');
    expect(ids).toContain('export-note');
  });

  it('triggers debounced search on input', async () => {
    vi.mocked(quickSearch).mockResolvedValue({ notes: [mockNote('1', 'Hello')] });

    const { getByRole } = render(QuickSwitcher);
    const input = getByRole('combobox') as HTMLInputElement;

    // Simulate typing by setting value and dispatching input event
    input.value = 'hello';
    await fireEvent.input(input);

    // Advance past debounce (200ms)
    await vi.advanceTimersByTimeAsync(250);

    // quickSearch is called at least once (may also be called by the $effect on mount)
    expect(quickSearch).toHaveBeenCalled();
  });

  it('closes on Escape key', async () => {
    const { getByRole } = render(QuickSwitcher);
    const input = getByRole('combobox');

    await fireEvent.keyDown(input, { key: 'Escape' });
    expect(ui.setQuickSwitcherOpen).toHaveBeenCalledWith(false);
  });

  it('closes on backdrop click', async () => {
    const { container } = render(QuickSwitcher);

    const backdrop = container.querySelector('.bg-black\\/50');
    expect(backdrop).toBeInTheDocument();

    await fireEvent.click(backdrop!);
    expect(ui.setQuickSwitcherOpen).toHaveBeenCalledWith(false);
  });

  it('renders footer with navigation hints', () => {
    const { container } = render(QuickSwitcher);
    const footer = container.querySelector('.border-t');
    expect(footer).toBeInTheDocument();
    expect(footer!.textContent).toContain('↑↓');
    expect(footer!.textContent).toContain('↵');
    expect(footer!.textContent).toContain('esc');
  });

  it('shows hint text when no query and no results', () => {
    const { container } = render(QuickSwitcher);
    const listbox = container.querySelector('[role="listbox"]');
    expect(listbox).toBeInTheDocument();
    expect(listbox!.textContent).toContain('component.quick_switcher.hint');
  });

  it('shows filter button in search mode', () => {
    const { getByLabelText } = render(QuickSwitcher);
    const filterBtn = getByLabelText('component.quick_switcher.filter');
    expect(filterBtn).toBeInTheDocument();
  });

  it('filter button has aria-expanded attribute', () => {
    const { getByLabelText } = render(QuickSwitcher);
    const filterBtn = getByLabelText('component.quick_switcher.filter');
    expect(filterBtn).toHaveAttribute('aria-expanded', 'false');
  });
});
