import { SvelteSet } from 'svelte/reactivity';

import type { Note } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { migrateLegacyEncryptedAttachmentLinks } from '$lib/editor/encrypted-attachment-markdown';
import { extractWikilinks } from '$lib/editor/markdown';
import * as encryption from '$lib/stores/encryption.svelte';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

/**
 * Extract wikilinks from content and deduplicate by normalized title.
 */
export function extractUniqueWikilinks(content: string) {
  const rawLinks = extractWikilinks(content);
  const seenTitles = new SvelteSet<string>();
  return rawLinks.filter((l) => {
    const norm = l.title.toLowerCase().trim();
    if (seenTitles.has(norm)) return false;
    seenTitles.add(norm);
    return true;
  });
}

/**
 * Decrypt an encrypted note response from the API in-place.
 * Returns true on success, false on failure (sets error message).
 */
export function decryptNoteFields(note: Note): boolean {
  try {
    if (!note.encrypted_content) {
      throw new Error('Missing encrypted content');
    }
    const encryptedPayload: EncryptedPayload = {
      ciphertext: note.encrypted_content,
      metadata: parseEncryptionMetadata(note.encryption_metadata),
    };
    const { title, content } = encryption.decryptNote(
      note.encrypted_title || null,
      encryptedPayload,
      note.id
    );
    const migrated = migrateLegacyEncryptedAttachmentLinks(content);
    note.title = title || note.title;
    note.content = migrated.content;
    return true;
  } catch (decryptError) {
    console.error('[NOTES] Failed to decrypt note:', decryptError);
    return false;
  }
}

/**
 * Guard for offline writes: throws if paranoid mode is active while offline.
 * If checkEncryptionLock is true, also throws if encryption is locked while offline.
 */
export function assertOnlineForParanoidMode(checkEncryptionLock = false): void {
  if (!navigator.onLine) {
    if (encryption.getSecurityLevel() === 'paranoid') {
      throw new Error('Offline-Schreiben im Paranoid-Modus nicht verfuegbar');
    }
    if (checkEncryptionLock && !encryption.isEncryptionUnlocked()) {
      throw new Error('Verschluesselung gesperrt - bitte online gehen');
    }
  }
}
