import { fireEvent,render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import NoteItem from '$lib/components/NoteItem.svelte';

const baseNote = {
  id: 'note-1',
  title: 'Test Note',
  content: '',
  folder_path: '/',
  version: 1,
  created_at: '2026-01-17T00:00:00Z',
  updated_at: '2026-01-17T00:00:00Z',
};

describe('NoteItem', () => {
  it('renders the title and selected state', () => {
    const handleClick = vi.fn();
    const { getByRole } = render(NoteItem, {
      props: { note: baseNote, isSelected: true, onclick: handleClick },
    });

    const button = getByRole('button', { name: 'Test Note' });
    expect(button).toHaveClass('selected');
  });

  it('invokes click handler when clicked', async () => {
    const handleClick = vi.fn();
    const { getByRole } = render(NoteItem, {
      props: { note: baseNote, isSelected: false, onclick: handleClick },
    });

    const button = getByRole('button', { name: 'Test Note' });
    await fireEvent.click(button);

    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
