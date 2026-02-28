import { fireEvent, render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getNoteTitles, suggestLinks } from '$lib/api';
import LinkSuggestionsPanel from '$lib/components/LinkSuggestionsPanel.svelte';

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
  getNoteTitles: vi.fn(async () => []),
  suggestLinks: vi.fn(async () => []),
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

describe('LinkSuggestionsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('does not call API for encrypted notes', async () => {
    const { getByRole, getAllByText } = render(LinkSuggestionsPanel, {
      props: {
        noteId: 'note-1',
        isEncrypted: true,
        plaintextContent: 'decrypted-content-should-not-be-sent',
        onInsertLink: () => {},
      },
    });

    await fireEvent.click(getByRole('button', { name: 'linkSuggestions.title' }));

    expect(getNoteTitles).not.toHaveBeenCalled();
    expect(suggestLinks).not.toHaveBeenCalled();
    expect(getAllByText('ai.encrypted_processing_disabled').length).toBeGreaterThan(0);
  });
});
