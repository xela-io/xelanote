import { describe, expect, it, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  getFolders: vi.fn(),
}));

import * as api from '$lib/api';
import * as foldersStore from '$lib/stores/folders.svelte';

describe('folders store', () => {
  it('memoizes folder tree when data is unchanged', async () => {
    const getFoldersMock = vi.mocked(api.getFolders);
    getFoldersMock.mockResolvedValue({
      folders: [
        {
          id: 1,
          path: '/',
          name: '',
          note_count: 1,
          created_at: '2023-01-01T00:00:00Z',
          updated_at: '2023-01-01T00:00:00Z',
        },
        {
          id: 2,
          path: '/projects',
          name: 'projects',
          note_count: 2,
          created_at: '2023-01-01T00:00:00Z',
          updated_at: '2023-01-01T00:00:00Z',
        },
      ],
    });

    await foldersStore.loadFolders();

    const firstTree = foldersStore.getFolderTree();
    const secondTree = foldersStore.getFolderTree();

    expect(secondTree).toBe(firstTree);
  });
});
