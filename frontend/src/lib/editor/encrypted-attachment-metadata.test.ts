import { describe, expect, it } from 'vitest';

import {
  decodeEncryptedAttachmentMetadata,
  encodeEncryptedAttachmentMetadata,
  inferMimeTypeFromFilename,
  isImageMimeType,
  stripFragment,
} from './encrypted-attachment-metadata';

describe('encrypted attachment metadata', () => {
  it('encodes and decodes metadata roundtrip', () => {
    const encoded = encodeEncryptedAttachmentMetadata({
      name: 'photo.png',
      type: 'image/png',
    });

    const decoded = decodeEncryptedAttachmentMetadata(encoded);
    expect(decoded).toEqual({
      version: 1,
      name: 'photo.png',
      type: 'image/png',
    });
  });

  it('returns null for invalid metadata', () => {
    expect(decodeEncryptedAttachmentMetadata('xela-enc-v1:not-valid-base64')).toBeNull();
    expect(decodeEncryptedAttachmentMetadata('xenc:image/png')).toBeNull();
    expect(decodeEncryptedAttachmentMetadata('')).toBeNull();
  });

  it('infers mime types and image flag correctly', () => {
    expect(inferMimeTypeFromFilename('a.webp')).toBe('image/webp');
    expect(inferMimeTypeFromFilename('doc.pdf')).toBe('application/pdf');
    expect(isImageMimeType('image/png')).toBe(true);
    expect(isImageMimeType('application/pdf')).toBe(false);
  });

  it('strips hash fragments from URLs', () => {
    expect(stripFragment('/api/uploads/1/file.xenc#xela-xenc=1')).toBe('/api/uploads/1/file.xenc');
    expect(stripFragment('/api/uploads/1/file.xenc')).toBe('/api/uploads/1/file.xenc');
  });
});
