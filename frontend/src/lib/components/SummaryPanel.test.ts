import { fireEvent, render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';
import { summarizeNoteStream } from '$lib/api';
import SummaryPanel from '$lib/components/SummaryPanel.svelte';

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

vi.mock('$lib/api', () => ({
  summarizeNoteStream: vi.fn(async () => {}),
}));

vi.mock('$lib/stores/encryption.svelte', () => ({
  isEncryptionUnlocked: vi.fn(() => true),
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

const baseNote = (overrides: Partial<Note> = {}): Note => ({
  id: 'note-1',
  title: 'Title',
  content: 'Body',
  folder_path: '/',
  version: 1,
  created_at: '2026-02-28T00:00:00Z',
  updated_at: '2026-02-28T00:00:00Z',
  ...overrides,
});

describe('SummaryPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('disables summary generation for encrypted notes', async () => {
    const note = baseNote({
      content_encrypted: true,
      encrypted_content: 'cipher',
      encryption_metadata: '{"version":3}',
    });

    const { getByRole, getByText } = render(SummaryPanel, {
      props: { note, decryptedContent: 'plaintext-from-client' },
    });

    expect(getByText('summary.encrypted_note')).toBeInTheDocument();

    const regenerateBtn = getByRole('button', { name: 'summary.regenerate' });
    expect(regenerateBtn).toBeDisabled();

    await fireEvent.click(regenerateBtn);
    expect(summarizeNoteStream).not.toHaveBeenCalled();
  });

  it('calls streaming summarize API for plaintext notes', async () => {
    const note = baseNote({ content_encrypted: false });

    const { getByRole } = render(SummaryPanel, {
      props: { note },
    });

    const regenerateBtn = getByRole('button', { name: 'summary.regenerate' });
    expect(regenerateBtn).not.toBeDisabled();

    await fireEvent.click(regenerateBtn);

    expect(summarizeNoteStream).toHaveBeenCalledTimes(1);
    const call = vi.mocked(summarizeNoteStream).mock.calls[0];
    expect(call[0]).toBe('note-1');
    expect(typeof call[1]).toBe('function');
    expect(typeof call[2]).toBe('function');
    expect(typeof call[3]).toBe('function');
    expect(call[4]).toBeUndefined();
  });
});
