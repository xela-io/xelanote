import { fireEvent, render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { suggestTags } from '$lib/api';
import TagSuggestionsPanel from '$lib/components/TagSuggestionsPanel.svelte';

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
  suggestTags: vi.fn(async () => []),
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  error: vi.fn(),
}));

describe('TagSuggestionsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('does not call API for encrypted notes', async () => {
    const { getByRole, getAllByText } = render(TagSuggestionsPanel, {
      props: {
        noteId: 'note-1',
        isEncrypted: true,
        plaintextContent: 'decrypted-content-should-not-be-sent',
        existingTagNames: [],
        onAddTag: async () => {},
      },
    });

    await fireEvent.click(getByRole('button', { name: 'tagSuggestions.title' }));

    expect(suggestTags).not.toHaveBeenCalled();
    expect(getAllByText('ai.encrypted_processing_disabled').length).toBeGreaterThan(0);
  });
});
