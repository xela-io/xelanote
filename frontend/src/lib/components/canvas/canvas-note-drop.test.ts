import { describe, expect, it } from 'vitest';

import {
  CANVAS_LEGACY_NOTE_MIME,
  CANVAS_TEXT_PLAIN_MIME,
  CANVAS_TREE_ITEM_MIME,
  parseDroppedSidebarNote,
} from '$lib/components/canvas/canvas-note-drop';

function createDataTransfer(payloadByMime: Record<string, string>): DataTransfer {
  return {
    getData: (format: string) => payloadByMime[format] ?? '',
  } as DataTransfer;
}

describe('parseDroppedSidebarNote', () => {
  it('parses note payload from unified tree mime', () => {
    const transfer = createDataTransfer({
      [CANVAS_TREE_ITEM_MIME]: JSON.stringify({
        type: 'note',
        id: 'note-1',
        title: 'My Note',
      }),
    });

    expect(parseDroppedSidebarNote(transfer)).toEqual({
      id: 'note-1',
      title: 'My Note',
    });
  });

  it('ignores unified tree folder payloads', () => {
    const transfer = createDataTransfer({
      [CANVAS_TREE_ITEM_MIME]: JSON.stringify({
        type: 'folder',
        id: 42,
        path: '/work',
      }),
    });

    expect(parseDroppedSidebarNote(transfer)).toBeNull();
  });

  it('parses legacy note payload from sidebar mime', () => {
    const transfer = createDataTransfer({
      [CANVAS_LEGACY_NOTE_MIME]: JSON.stringify({
        id: 'note-2',
        title: 'Legacy Note',
      }),
    });

    expect(parseDroppedSidebarNote(transfer)).toEqual({
      id: 'note-2',
      title: 'Legacy Note',
    });
  });

  it('returns null for invalid json payload', () => {
    const transfer = createDataTransfer({
      [CANVAS_TREE_ITEM_MIME]: '{invalid-json',
    });

    expect(parseDroppedSidebarNote(transfer)).toBeNull();
  });

  it('returns null when no known mime payload is present', () => {
    const transfer = createDataTransfer({
      'text/plain': 'anything',
    });

    expect(parseDroppedSidebarNote(transfer)).toBeNull();
  });

  it('parses unified tree note payload from text/plain fallback', () => {
    const transfer = createDataTransfer({
      [CANVAS_TEXT_PLAIN_MIME]: JSON.stringify({
        type: 'note',
        id: 'note-3',
        title: 'Text Fallback',
      }),
    });

    expect(parseDroppedSidebarNote(transfer)).toEqual({
      id: 'note-3',
      title: 'Text Fallback',
    });
  });

  it('parses legacy payload without folder_path', () => {
    const transfer = createDataTransfer({
      [CANVAS_TEXT_PLAIN_MIME]: JSON.stringify({
        id: 'note-4',
        title: 'Legacy Without Folder',
      }),
    });

    expect(parseDroppedSidebarNote(transfer)).toEqual({
      id: 'note-4',
      title: 'Legacy Without Folder',
    });
  });
});
