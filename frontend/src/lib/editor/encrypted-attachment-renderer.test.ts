import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  isEncryptionUnlocked: vi.fn(() => true),
  decryptAttachment: vi.fn(() => new Uint8Array([9, 8, 7])),
}));

vi.mock('$lib/stores/encryption.svelte', () => ({
  isEncryptionUnlocked: mocks.isEncryptionUnlocked,
  decryptAttachment: mocks.decryptAttachment,
}));

import { encodeEncryptedAttachmentMetadata } from './encrypted-attachment-metadata';
import { encryptedAttachmentRenderer } from './encrypted-attachment-renderer';

async function waitUntil(assertion: () => void, timeoutMs = 500): Promise<void> {
  const start = Date.now();
  let lastError: unknown;
  while (Date.now() - start < timeoutMs) {
    try {
      assertion();
      return;
    } catch (err) {
      lastError = err;
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }
  throw lastError instanceof Error ? lastError : new Error('waitUntil timeout');
}

describe('encryptedAttachmentRenderer', () => {
  const createObjectURL = vi.fn(() => 'blob:decrypted');
  const revokeObjectURL = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(new Uint8Array([1, 2, 3])))
    );
    URL.createObjectURL = createObjectURL;
    URL.revokeObjectURL = revokeObjectURL;
    mocks.isEncryptionUnlocked.mockReturnValue(true);
    mocks.decryptAttachment.mockReturnValue(new Uint8Array([9, 8, 7]));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('decrypts encrypted images and replaces src with blob URL', async () => {
    const metadata = encodeEncryptedAttachmentMetadata({ name: 'photo.jpg', type: 'image/jpeg' });
    const node = document.createElement('div');
    node.innerHTML = `<img src="/api/uploads/1/file.xenc" title="${metadata}" alt="photo.jpg">`;

    const action = encryptedAttachmentRenderer(node, {
      revision: 'r1',
      context: {
        noteID: 'note-1',
        wrappedDEK: 'wrapped',
        metadataVersion: 3,
      },
    });

    const img = node.querySelector('img');
    expect(img).not.toBeNull();
    await waitUntil(() => {
      expect(img!.getAttribute('src')).toBe('blob:decrypted');
    });
    expect(mocks.decryptAttachment).toHaveBeenCalledWith(
      expect.any(Uint8Array),
      expect.objectContaining({
        noteID: 'note-1',
        wrappedDEK: 'wrapped',
        metadataVersion: 3,
        filename: 'photo.jpg',
      })
    );

    action.destroy?.();
  });

  it('decrypts encrypted links and adds download attribute', async () => {
    const node = document.createElement('div');
    node.innerHTML = '<a href="/api/uploads/1/file.xenc">Encrypted attachment: manual.pdf</a>';

    const action = encryptedAttachmentRenderer(node, {
      revision: 'r1',
      context: {
        noteID: 'note-2',
        wrappedDEK: 'wrapped-2',
        metadataVersion: 3,
      },
    });

    const link = node.querySelector('a');
    expect(link).not.toBeNull();
    await waitUntil(() => {
      expect(link!.getAttribute('href')).toBe('blob:decrypted');
      expect(link!.getAttribute('download')).toBe('manual.pdf');
    });
    expect(mocks.decryptAttachment).toHaveBeenCalledWith(
      expect.any(Uint8Array),
      expect.objectContaining({
        noteID: 'note-2',
        filename: 'manual.pdf',
      })
    );

    action.destroy?.();
  });

  it('skips processing when context is missing', async () => {
    const node = document.createElement('div');
    node.innerHTML = '<a href="/api/uploads/1/file.xenc">Encrypted attachment: skip.txt</a>';

    const action = encryptedAttachmentRenderer(node, {
      revision: 'r1',
      context: null,
    });

    expect(fetch).not.toHaveBeenCalled();
    expect(mocks.decryptAttachment).not.toHaveBeenCalled();

    action.destroy?.();
  });
});
