import { describe, expect, it } from 'vitest';

import { migrateLegacyEncryptedAttachmentLinks } from './encrypted-attachment-markdown';
import {
  decodeEncryptedAttachmentMetadata,
  encodeEncryptedAttachmentMetadata,
} from './encrypted-attachment-metadata';

function extractQuotedTitle(markdown: string): string {
  const match = markdown.match(/"([^"]+)"/);
  if (!match) {
    throw new Error(`No quoted title found in: ${markdown}`);
  }
  return match[1];
}

describe('migrateLegacyEncryptedAttachmentLinks', () => {
  it('migrates legacy encrypted image links to markdown image syntax', () => {
    const input =
      '[Encrypted attachment: photo.png](/api/uploads/1/file.xenc?signature=s&expires=1)';

    const result = migrateLegacyEncryptedAttachmentLinks(input);
    const metadata = decodeEncryptedAttachmentMetadata(extractQuotedTitle(result.content));

    expect(result.migratedCount).toBe(1);
    expect(result.content).toContain('![photo.png](');
    expect(result.content).toContain('/api/uploads/1/file.xenc?signature=s&expires=1');
    expect(result.content).not.toContain('#xela-xenc=');
    expect(metadata).toMatchObject({ version: 1, name: 'photo.png', type: 'image/png' });
  });

  it('keeps non-image legacy attachments as links but adds structured metadata', () => {
    const input = '[Encrypted attachment: report.pdf](/api/uploads/1/file.xenc)';
    const result = migrateLegacyEncryptedAttachmentLinks(input);
    const metadata = decodeEncryptedAttachmentMetadata(extractQuotedTitle(result.content));

    expect(result.migratedCount).toBe(1);
    expect(result.content).toContain('[Encrypted attachment: report.pdf]');
    expect(result.content).not.toContain('#xela-xenc=');
    expect(metadata).toMatchObject({ version: 1, name: 'report.pdf', type: 'application/pdf' });
  });

  it('migrates hash-marker image tokens to structured metadata title', () => {
    const input =
      '![photo](/api/uploads/1/file.xenc#xela-xenc=1&name=photo.png&type=image%2Fpng "xenc:image/png")';
    const result = migrateLegacyEncryptedAttachmentLinks(input);
    const metadata = decodeEncryptedAttachmentMetadata(extractQuotedTitle(result.content));

    expect(result.migratedCount).toBe(1);
    expect(result.content).toContain('![photo](/api/uploads/1/file.xenc ');
    expect(result.content).not.toContain('#xela-xenc=');
    expect(metadata).toMatchObject({ version: 1, name: 'photo.png', type: 'image/png' });
  });

  it('is idempotent for already structured metadata', () => {
    const title = encodeEncryptedAttachmentMetadata({ name: 'photo.webp', type: 'image/webp' });
    const input = `![photo.webp](/api/uploads/1/file.xenc "${title}")`;
    const result = migrateLegacyEncryptedAttachmentLinks(input);

    expect(result.migratedCount).toBe(0);
    expect(result.content).toBe(input);
  });
});
