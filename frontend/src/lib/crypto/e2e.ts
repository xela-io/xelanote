import {
  bytesToString,
  decrypt,
  deriveKey,
  deriveKeyAsync,
  encrypt,
  fromBase64Standard,
  generateDEK,
  isInitialized,
  stringToBytes,
  toBase64Standard,
} from './sodium';

export type EncryptionVersion = 2 | 3;

type PayloadPurpose = 'note' | 'title';

export interface RecoveryWrappedDEKEntry {
  id: string;
  wrapped_dek_recovery: string;
}

export interface AttachmentKeyContext {
  noteID: string;
  wrappedDEK: string;
  metadataVersion?: EncryptionVersion;
  filename?: string;
}

export interface EncryptionMetadata {
  version: EncryptionVersion;
  algorithm: 'XChaCha20-Poly1305';
  kdf: 'Argon2id';
  kdf_strength: 'interactive';
  nonce_bytes: 24;
  wrapped_dek: string; // Base64 standard (with padding) - backend uses base64.StdEncoding
}

export interface EncryptedPayload {
  ciphertext: string; // Base64 standard (with padding)
  metadata: EncryptionMetadata;
}

/**
 * Custom error class for decryption failures.
 * Provides error codes for UI-level handling.
 */
export class DecryptionError extends Error {
  constructor(
    public readonly code: 'INVALID_KEY_OR_DATA' | 'NOT_INITIALIZED' | 'CORRUPTED_METADATA'
  ) {
    super(`Decryption failed: ${code}`);
    this.name = 'DecryptionError';
  }
}

/**
 * E2E Encryption implementation using libsodium.
 *
 * Security features:
 * - Argon2id KDF via Web Worker (memory-hard, GPU-resistant, non-blocking)
 * - XChaCha20-Poly1305 authenticated encryption (IETF standard)
 * - Per-note DEKs (enables easy key rotation)
 * - Unicode password normalization (NFC)
 * - 24-byte nonces (no collision risk)
 */
export class E2EEncryption {
  private kek: Uint8Array | null = null; // Key Encryption Key (raw bytes)

  private async deriveKEK(password: string, salt: Uint8Array): Promise<Uint8Array> {
    try {
      return await deriveKeyAsync(password, salt);
    } catch (err) {
      // Fallback keeps encryption functional in environments without Worker support.
      if (import.meta.env.DEV) {
        console.warn('[E2EE] Worker KDF unavailable, falling back to synchronous deriveKey:', err);
      }
      return deriveKey(password, salt);
    }
  }

  private buildAAD(
    noteID: string,
    purpose: PayloadPurpose,
    material: 'content' | 'dek' | 'dek_recovery'
  ): Uint8Array {
    return stringToBytes(`xelanote:e2ee:v3:${purpose}:${material}:${noteID}`);
  }

  private buildVersionRecoveryAAD(versionID: string): Uint8Array {
    return stringToBytes(`xelanote:e2ee:v3:version:dek_recovery:${versionID}`);
  }

  private buildAttachmentAAD(noteID: string, filename: string): Uint8Array {
    return stringToBytes(`xelanote:e2ee:attachment:v1:${noteID}:${filename.normalize('NFC')}`);
  }

  private unwrapNoteDEK(context: AttachmentKeyContext): Uint8Array {
    if (!this.kek) throw new DecryptionError('NOT_INITIALIZED');
    if (!isInitialized()) throw new DecryptionError('NOT_INITIALIZED');
    if (!context.noteID || !context.wrappedDEK) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    const metadataVersion: EncryptionVersion = context.metadataVersion ?? 3;
    const wrappedDEKBytes = fromBase64Standard(context.wrappedDEK);
    const wrapAAD =
      metadataVersion === 3 ? this.buildAAD(context.noteID, 'note', 'dek') : undefined;
    const dek = this.unwrapDEK(wrappedDEKBytes, this.kek, wrapAAD);
    if (dek === null) {
      throw new DecryptionError('INVALID_KEY_OR_DATA');
    }
    return dek;
  }

  private parseEncryptedPayload(raw: string): EncryptedPayload {
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    if (!parsed || typeof parsed !== 'object') {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    const payload = parsed as { ciphertext?: unknown; metadata?: unknown };
    if (
      typeof payload.ciphertext !== 'string' ||
      !payload.metadata ||
      typeof payload.metadata !== 'object'
    ) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    const metadata = payload.metadata as {
      version?: unknown;
      algorithm?: unknown;
      kdf?: unknown;
      kdf_strength?: unknown;
      nonce_bytes?: unknown;
      wrapped_dek?: unknown;
    };
    if (
      (metadata.version !== 2 && metadata.version !== 3) ||
      metadata.algorithm !== 'XChaCha20-Poly1305' ||
      metadata.kdf !== 'Argon2id' ||
      metadata.kdf_strength !== 'interactive' ||
      metadata.nonce_bytes !== 24 ||
      typeof metadata.wrapped_dek !== 'string'
    ) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    return {
      ciphertext: payload.ciphertext,
      metadata: {
        version: metadata.version as EncryptionVersion,
        algorithm: 'XChaCha20-Poly1305',
        kdf: 'Argon2id',
        kdf_strength: 'interactive',
        nonce_bytes: 24,
        wrapped_dek: metadata.wrapped_dek,
      },
    };
  }

  /**
   * Setup KEK from user password.
   * Call this on login after receiving the user's encryption salt from the server.
   *
   * Uses worker-based Argon2id derivation when available to avoid blocking the UI.
   * Falls back to synchronous derivation if the worker cannot be used.
   *
   * @param password - User's password
   * @param salt - User-specific salt (16 bytes, stored server-side)
   */
  async setupKEK(password: string, salt: Uint8Array): Promise<void> {
    this.kek = await this.deriveKEK(password, salt);
  }

  /**
   * Clear KEK from memory (on logout).
   * After calling this, all encryption operations will fail until setupKEK is called again.
   */
  clearKEK(): void {
    // Overwrite memory before releasing (defense in depth)
    if (this.kek) {
      this.kek.fill(0);
    }
    this.kek = null;
  }

  /**
   * Wrap a DEK using XChaCha20-Poly1305.
   * Nonce is automatically prepended by encrypt().
   *
   * @param dek - Data Encryption Key (32 bytes)
   * @param kek - Key Encryption Key (32 bytes)
   * @param aad - Optional additional authenticated data
   * @returns Wrapped key as Uint8Array (24-byte nonce + ciphertext + 16-byte tag)
   */
  private wrapDEK(dek: Uint8Array, kek: Uint8Array, aad?: Uint8Array): Uint8Array {
    return encrypt(dek, kek, aad);
  }

  /**
   * Unwrap a DEK using XChaCha20-Poly1305.
   *
   * @param wrappedDEK - Wrapped DEK bytes (nonce + ciphertext + tag)
   * @param kek - Key Encryption Key
   * @param aad - Optional additional authenticated data
   * @returns Unwrapped Data Encryption Key or null if decryption fails
   */
  private unwrapDEK(wrappedDEK: Uint8Array, kek: Uint8Array, aad?: Uint8Array): Uint8Array | null {
    return decrypt(wrappedDEK, kek, aad);
  }

  private encryptPayload(
    content: string,
    noteID: string | null,
    purpose: PayloadPurpose
  ): EncryptedPayload {
    if (!this.kek) throw new Error('KEK not initialized');
    if (!isInitialized()) throw new Error('libsodium not initialized');

    const metadataVersion: EncryptionVersion = noteID ? 3 : 2;

    // 1. Generate random DEK for this payload
    const dek = generateDEK();

    // 2. Encrypt content with DEK
    const plaintext = stringToBytes(content);
    const contentAAD = noteID ? this.buildAAD(noteID, purpose, 'content') : undefined;
    const ciphertext = encrypt(plaintext, dek, contentAAD);

    // 3. Wrap DEK with KEK
    const wrapAAD = noteID ? this.buildAAD(noteID, purpose, 'dek') : undefined;
    const wrappedDEK = this.wrapDEK(dek, this.kek, wrapAAD);

    // 4. Clear DEK from memory
    dek.fill(0);

    return {
      ciphertext: toBase64Standard(ciphertext),
      metadata: {
        version: metadataVersion,
        algorithm: 'XChaCha20-Poly1305',
        kdf: 'Argon2id',
        kdf_strength: 'interactive',
        nonce_bytes: 24,
        wrapped_dek: toBase64Standard(wrappedDEK),
      },
    };
  }

  private decryptPayload(
    payload: EncryptedPayload,
    noteID: string | null,
    purpose: PayloadPurpose
  ): string {
    if (!this.kek) throw new DecryptionError('NOT_INITIALIZED');
    if (!isInitialized()) throw new DecryptionError('NOT_INITIALIZED');

    const { ciphertext, metadata } = payload;

    // Validate metadata
    if (!metadata.wrapped_dek) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }
    if (metadata.version === 3 && !noteID) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    // 1. Unwrap DEK with KEK
    const wrappedDEKBytes = fromBase64Standard(metadata.wrapped_dek);
    const wrapAAD =
      metadata.version === 3 && noteID ? this.buildAAD(noteID, purpose, 'dek') : undefined;
    const dek = this.unwrapDEK(wrappedDEKBytes, this.kek, wrapAAD);

    if (dek === null) {
      throw new DecryptionError('INVALID_KEY_OR_DATA');
    }

    // 2. Decrypt content with DEK
    const ciphertextBytes = fromBase64Standard(ciphertext);
    const contentAAD =
      metadata.version === 3 && noteID ? this.buildAAD(noteID, purpose, 'content') : undefined;
    const plaintext = decrypt(ciphertextBytes, dek, contentAAD);

    // 3. Clear DEK from memory
    dek.fill(0);

    if (plaintext === null) {
      throw new DecryptionError('INVALID_KEY_OR_DATA');
    }

    return bytesToString(plaintext);
  }

  /**
   * Encrypt note content with per-note DEK.
   *
   * This generates a new random DEK for each note, encrypts the content with
   * XChaCha20-Poly1305, then wraps the DEK with the user's KEK.
   *
   * @param content - Plaintext note content
   * @returns Encrypted payload with metadata
   * @throws Error if KEK is not initialized
   */
  encryptNote(content: string, noteID?: string): EncryptedPayload {
    return this.encryptPayload(content, noteID ?? null, 'note');
  }

  /**
   * Decrypt note content.
   *
   * This unwraps the DEK using the user's KEK, then decrypts the content.
   *
   * @param payload - Encrypted payload with metadata
   * @returns Decrypted plaintext content
   * @throws DecryptionError if decryption fails
   */
  decryptNote(payload: EncryptedPayload, noteID?: string): string {
    return this.decryptPayload(payload, noteID ?? null, 'note');
  }

  /**
   * Encrypt title (optional).
   * Returns a JSON string containing the encrypted payload.
   *
   * @param title - Plaintext title
   * @returns JSON string of encrypted payload
   */
  encryptTitle(title: string, noteID?: string): string {
    const payload = this.encryptPayload(title, noteID ?? null, 'title');
    return JSON.stringify(payload);
  }

  /**
   * Decrypt title.
   *
   * @param encryptedTitle - JSON string of encrypted payload
   * @returns Decrypted plaintext title
   */
  decryptTitle(encryptedTitle: string, noteID?: string): string {
    const payload = this.parseEncryptedPayload(encryptedTitle);
    return this.decryptPayload(payload, noteID ?? null, 'title');
  }

  /**
   * Encrypt binary attachment data with the note DEK.
   *
   * The note DEK is unwrapped with the current KEK and then used to encrypt
   * attachment bytes via XChaCha20-Poly1305. The ciphertext includes nonce+tag.
   *
   * @param data - Binary attachment bytes
   * @param context - Note key context (note ID + wrapped DEK)
   * @returns Encrypted bytes (nonce + ciphertext + tag)
   */
  encryptAttachment(data: Uint8Array, context: AttachmentKeyContext): Uint8Array {
    const dek = this.unwrapNoteDEK(context);

    try {
      const attachmentAAD = this.buildAttachmentAAD(
        context.noteID,
        context.filename || 'attachment'
      );
      return encrypt(data, dek, attachmentAAD);
    } finally {
      dek.fill(0);
    }
  }

  /**
   * Decrypt binary attachment data with the note DEK.
   *
   * @param encrypted - Encrypted bytes (nonce + ciphertext + tag)
   * @param context - Note key context (note ID + wrapped DEK)
   * @returns Decrypted plaintext bytes
   */
  decryptAttachment(encrypted: Uint8Array, context: AttachmentKeyContext): Uint8Array {
    const dek = this.unwrapNoteDEK(context);

    try {
      const attachmentAAD = this.buildAttachmentAAD(
        context.noteID,
        context.filename || 'attachment'
      );
      const plaintext = decrypt(encrypted, dek, attachmentAAD);
      if (plaintext === null) {
        throw new DecryptionError('INVALID_KEY_OR_DATA');
      }
      return plaintext;
    } finally {
      dek.fill(0);
    }
  }

  /**
   * Extract keywords from content (opt-in, with warning).
   * Unicode-safe implementation using \p{L} for letter matching.
   *
   * WARNING: Keywords leak semantic information about note content!
   * Only use if user explicitly opts in and understands the privacy trade-off.
   *
   * @param content - Plaintext content
   * @param maxKeywords - Maximum number of keywords to extract (default: 30)
   * @returns Array of keywords sorted by frequency
   */
  extractKeywords(content: string, maxKeywords = 30): string[] {
    // Stopword filter (common words to ignore)
    const stopWords = new Set([
      // German
      'der',
      'die',
      'das',
      'und',
      'oder',
      'aber',
      'ist',
      'sind',
      'war',
      'ein',
      'eine',
      'als',
      'für',
      'mit',
      'auf',
      'nicht',
      'von',
      'dem',
      'den',
      'des',
      'im',
      'zu',
      'sich',
      'auch',
      'dass',
      'bei',
      'aus',
      'nach',
      'werden',
      'kann',
      'hat',
      // English
      'the',
      'a',
      'an',
      'and',
      'or',
      'but',
      'is',
      'are',
      'was',
      'were',
      'in',
      'on',
      'at',
      'to',
      'for',
      'of',
      'with',
      'by',
      'from',
      'not',
      'that',
      'this',
      'have',
      'has',
      'been',
      'can',
      'will',
      'would',
    ]);

    // Use Unicode-aware regex
    // \p{L} matches any Unicode letter (includes äöüß, é, ñ, etc.)
    const words = content.toLowerCase().match(/\p{L}+/gu) || []; // Unicode word boundary

    // Filter and count
    const freq = new Map<string, number>();
    words.forEach((word) => {
      if (word.length > 3 && !stopWords.has(word)) {
        freq.set(word, (freq.get(word) || 0) + 1);
      }
    });

    // Return top keywords by frequency
    return Array.from(freq.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, maxKeywords)
      .map(([word]) => word);
  }

  /**
   * Re-wrap all DEKs when user changes password.
   * This is critical for maintaining access to encrypted notes after password change.
   *
   * Flow:
   * 1. Derive old KEK from old password
   * 2. Derive new KEK from new password
   * 3. For each note/version: unwrap DEK with old KEK, re-wrap with new KEK
   * 4. Perform deterministic full validation for all re-wrapped entries
   * 5. Return maps of noteID/versionID -> new wrapped_dek
   *
   * @param notes - Array of notes with wrapped_dek field
   * @param versions - Array of note versions with wrapped_dek field
   * @param oldPassword - User's current password
   * @param newPassword - User's new password
   * @param salt - User's encryption salt (16 bytes)
   * @returns Object with notes and versions maps (ID -> new wrapped_dek in base64)
   * @throws Error if re-wrapping fails or test decryption fails
   */
  async reWrapAllDEKs(
    notes: Array<{ id: string; wrapped_dek?: string; encrypted_content?: string }>,
    versions: Array<{ id: number; wrapped_dek?: string; encrypted_content?: string }>,
    oldPassword: string,
    newPassword: string,
    salt: Uint8Array
  ): Promise<{
    notes: Map<string, string>;
    versions: Map<string, string>;
  }> {
    if (!isInitialized()) throw new Error('libsodium not initialized');

    // Step 1: Derive old KEK from old password
    const oldKEK = await this.deriveKEK(oldPassword, salt);

    // Step 2: Derive new KEK from new password
    const newKEK = await this.deriveKEK(newPassword, salt);

    const reWrappedNotes = new Map<string, string>();
    const reWrappedVersions = new Map<string, string>();

    // Step 3: Re-wrap all note DEKs
    for (const note of notes) {
      if (!note.wrapped_dek) continue; // Skip notes without encryption

      try {
        // Parse wrapped DEK from base64
        const oldWrappedDEK = fromBase64Standard(note.wrapped_dek);

        // Unwrap DEK with old KEK
        const dek = this.unwrapDEK(oldWrappedDEK, oldKEK);
        if (dek === null) {
          throw new Error(
            `Failed to unwrap DEK for note ${note.id} - incorrect old password or corrupted data`
          );
        }

        // Re-wrap DEK with new KEK
        const newWrappedDEK = this.wrapDEK(dek, newKEK);

        // Store as base64
        reWrappedNotes.set(note.id, toBase64Standard(newWrappedDEK));

        // Clear DEK from memory
        dek.fill(0);
      } catch (err) {
        // Clean up KEKs before throwing
        oldKEK.fill(0);
        newKEK.fill(0);
        throw new Error(`Failed to re-wrap note ${note.id}: ${err}`);
      }
    }

    // Step 4: Re-wrap all version DEKs
    for (const version of versions) {
      if (!version.wrapped_dek) continue; // Skip versions without encryption

      try {
        // Parse wrapped DEK from base64
        const oldWrappedDEK = fromBase64Standard(version.wrapped_dek);

        // Unwrap DEK with old KEK
        const dek = this.unwrapDEK(oldWrappedDEK, oldKEK);
        if (dek === null) {
          throw new Error(
            `Failed to unwrap DEK for version ${version.id} - incorrect old password or corrupted data`
          );
        }

        // Re-wrap DEK with new KEK
        const newWrappedDEK = this.wrapDEK(dek, newKEK);

        // Store as base64 (using string key since Map keys are strings in JSON)
        reWrappedVersions.set(version.id.toString(), toBase64Standard(newWrappedDEK));

        // Clear DEK from memory
        dek.fill(0);
      } catch (err) {
        // Clean up KEKs before throwing
        oldKEK.fill(0);
        newKEK.fill(0);
        throw new Error(`Failed to re-wrap version ${version.id}: ${err}`);
      }
    }

    // Step 5: Deterministic full validation of ALL re-wrapped entries
    // Validate each note DEK (and decrypt content where available)
    for (const note of notes) {
      if (!note.wrapped_dek) continue;
      try {
        const newWrappedDEK = reWrappedNotes.get(note.id);
        if (!newWrappedDEK) {
          throw new Error(`Missing re-wrapped DEK for note ${note.id}`);
        }

        // Unwrap DEK with NEW KEK
        const newWrappedDEKBytes = fromBase64Standard(newWrappedDEK);
        const dek = this.unwrapDEK(newWrappedDEKBytes, newKEK);

        if (dek === null) {
          throw new Error(`Validation failed for note ${note.id} - unwrap with new KEK failed`);
        }

        // If encrypted content is available, validate full decrypt path too.
        if (note.encrypted_content) {
          const ciphertext = fromBase64Standard(note.encrypted_content);
          const plaintext = decrypt(ciphertext, dek);
          if (plaintext === null) {
            throw new Error(`Validation failed for note ${note.id} - content decryption failed`);
          }
        }

        // Clear DEK from memory
        dek.fill(0);
      } catch (err) {
        // Clean up KEKs before throwing
        oldKEK.fill(0);
        newKEK.fill(0);
        throw new Error(`Re-wrapping validation failed for note ${note.id}: ${err}`);
      }
    }

    // Validate each version DEK (and decrypt content where available)
    for (const version of versions) {
      if (!version.wrapped_dek) continue;
      const versionID = version.id.toString();
      try {
        const newWrappedDEK = reWrappedVersions.get(versionID);
        if (!newWrappedDEK) {
          throw new Error(`Missing re-wrapped DEK for version ${versionID}`);
        }

        // Unwrap DEK with NEW KEK
        const newWrappedDEKBytes = fromBase64Standard(newWrappedDEK);
        const dek = this.unwrapDEK(newWrappedDEKBytes, newKEK);
        if (dek === null) {
          throw new Error(
            `Validation failed for version ${versionID} - unwrap with new KEK failed`
          );
        }

        // If encrypted content is available, validate full decrypt path too.
        if (version.encrypted_content) {
          const ciphertext = fromBase64Standard(version.encrypted_content);
          const plaintext = decrypt(ciphertext, dek);
          if (plaintext === null) {
            throw new Error(
              `Validation failed for version ${versionID} - content decryption failed`
            );
          }
        }

        // Clear DEK from memory
        dek.fill(0);
      } catch (err) {
        // Clean up KEKs before throwing
        oldKEK.fill(0);
        newKEK.fill(0);
        throw new Error(`Re-wrapping validation failed for version ${versionID}: ${err}`);
      }
    }

    // Clean up KEKs from memory
    oldKEK.fill(0);
    newKEK.fill(0);

    return {
      notes: reWrappedNotes,
      versions: reWrappedVersions,
    };
  }

  /**
   * Create recovery-wrapped DEKs from the currently unlocked password KEK.
   * Used when setting up a recovery key for accounts with encrypted content.
   */
  async createRecoveryWrappedDEKs(
    notes: Array<{ id: string; wrapped_dek?: string | null }>,
    versions: Array<{ id: number | string; wrapped_dek?: string | null }>,
    recoveryKey: string,
    recoverySalt: Uint8Array
  ): Promise<{
    notes: Map<string, string>;
    versions: Map<string, string>;
  }> {
    if (!isInitialized()) throw new Error('libsodium not initialized');
    if (!this.kek) throw new Error('KEK not initialized');
    if (!recoveryKey) throw new Error('recovery key is required');

    const recoveryKEK = await this.deriveKEK(recoveryKey, recoverySalt);
    const recoveryWrappedNotes = new Map<string, string>();
    const recoveryWrappedVersions = new Map<string, string>();

    try {
      for (const note of notes) {
        if (!note.id || !note.wrapped_dek) continue;
        const wrappedDEK = fromBase64Standard(note.wrapped_dek);

        const noteAAD = this.buildAAD(note.id, 'note', 'dek');
        // Compatibility fallback for legacy wrappers without AAD.
        let dek = this.unwrapDEK(wrappedDEK, this.kek, noteAAD);
        if (dek === null) {
          dek = this.unwrapDEK(wrappedDEK, this.kek);
        }
        if (dek === null) {
          throw new Error(`failed to unwrap note DEK for ${note.id}`);
        }

        const wrappedRecoveryDEK = this.wrapDEK(
          dek,
          recoveryKEK,
          this.buildAAD(note.id, 'note', 'dek_recovery')
        );
        recoveryWrappedNotes.set(note.id, toBase64Standard(wrappedRecoveryDEK));
        dek.fill(0);
      }

      for (const version of versions) {
        if (!version.id || !version.wrapped_dek) continue;
        const versionID = String(version.id);
        const wrappedDEK = fromBase64Standard(version.wrapped_dek);

        // Version wrapped_dek values are historically unauthenticated.
        const dek = this.unwrapDEK(wrappedDEK, this.kek);
        if (dek === null) {
          throw new Error(`failed to unwrap version DEK for ${versionID}`);
        }

        const wrappedRecoveryDEK = this.wrapDEK(
          dek,
          recoveryKEK,
          this.buildVersionRecoveryAAD(versionID)
        );
        recoveryWrappedVersions.set(versionID, toBase64Standard(wrappedRecoveryDEK));
        dek.fill(0);
      }
    } finally {
      recoveryKEK.fill(0);
    }

    return {
      notes: recoveryWrappedNotes,
      versions: recoveryWrappedVersions,
    };
  }

  /**
   * Re-wrap DEKs from recovery wrappers to password wrappers.
   * Used during password recovery reset for encrypted accounts.
   */
  async reWrapRecoveryDEKs(
    notes: RecoveryWrappedDEKEntry[],
    versions: RecoveryWrappedDEKEntry[],
    recoveryKey: string,
    newPassword: string,
    recoverySalt: Uint8Array,
    encryptionSalt: Uint8Array
  ): Promise<{
    notes: Map<string, string>;
    versions: Map<string, string>;
  }> {
    if (!isInitialized()) throw new Error('libsodium not initialized');

    const recoveryKEK = await this.deriveKEK(recoveryKey, recoverySalt);
    const newKEK = await this.deriveKEK(newPassword, encryptionSalt);

    const reWrappedNotes = new Map<string, string>();
    const reWrappedVersions = new Map<string, string>();

    try {
      for (const note of notes) {
        if (!note.id || !note.wrapped_dek_recovery) {
          throw new Error('Invalid note recovery wrapper payload');
        }

        const wrappedDEK = fromBase64Standard(note.wrapped_dek_recovery);
        const aad = this.buildAAD(note.id, 'note', 'dek_recovery');
        // Compatibility fallback for legacy wrappers without AAD.
        let dek = this.unwrapDEK(wrappedDEK, recoveryKEK, aad);
        if (dek === null) {
          dek = this.unwrapDEK(wrappedDEK, recoveryKEK);
        }
        if (dek === null) {
          throw new Error(`Failed to unwrap recovery DEK for note ${note.id}`);
        }

        const newWrappedDEK = this.wrapDEK(dek, newKEK, this.buildAAD(note.id, 'note', 'dek'));
        reWrappedNotes.set(note.id, toBase64Standard(newWrappedDEK));
        dek.fill(0);
      }

      for (const version of versions) {
        if (!version.id || !version.wrapped_dek_recovery) {
          throw new Error('Invalid version recovery wrapper payload');
        }

        const wrappedDEK = fromBase64Standard(version.wrapped_dek_recovery);
        const aad = this.buildVersionRecoveryAAD(version.id);
        // Compatibility fallback for legacy wrappers without AAD.
        let dek = this.unwrapDEK(wrappedDEK, recoveryKEK, aad);
        if (dek === null) {
          dek = this.unwrapDEK(wrappedDEK, recoveryKEK);
        }
        if (dek === null) {
          throw new Error(`Failed to unwrap recovery DEK for version ${version.id}`);
        }

        const newWrappedDEK = this.wrapDEK(dek, newKEK);
        reWrappedVersions.set(version.id, toBase64Standard(newWrappedDEK));
        dek.fill(0);
      }
    } finally {
      recoveryKEK.fill(0);
      newKEK.fill(0);
    }

    return {
      notes: reWrappedNotes,
      versions: reWrappedVersions,
    };
  }

  /**
   * Encrypt folder path using the note's existing DEK.
   * Returns base64 ciphertext (nonce + ciphertext + tag).
   */
  encryptFolderPath(folderPath: string, noteID: string, wrappedDEK: string): string {
    if (!this.kek) throw new Error('KEK not initialized');
    if (!isInitialized()) throw new Error('libsodium not initialized');

    // Unwrap the note's DEK
    const wrappedDEKBytes = fromBase64Standard(wrappedDEK);
    const wrapAAD = this.buildAAD(noteID, 'note', 'dek');
    let dek = this.unwrapDEK(wrappedDEKBytes, this.kek, wrapAAD);
    if (dek === null) {
      // Fallback for legacy (v2) wrapped DEKs without AAD
      dek = this.unwrapDEK(wrappedDEKBytes, this.kek);
    }
    if (dek === null) throw new DecryptionError('INVALID_KEY_OR_DATA');

    // Encrypt folder path with DEK using folder_path-specific AAD
    const plaintext = stringToBytes(folderPath);
    const folderAAD = stringToBytes(`xelanote:e2ee:v3:note:folder_path:${noteID}`);
    const ciphertext = encrypt(plaintext, dek, folderAAD);

    dek.fill(0);
    return toBase64Standard(ciphertext);
  }

  /**
   * Decrypt folder path using the note's existing DEK.
   * Returns plaintext folder path.
   */
  decryptFolderPath(encryptedFolderPath: string, noteID: string, wrappedDEK: string): string {
    if (!this.kek) throw new DecryptionError('NOT_INITIALIZED');
    if (!isInitialized()) throw new DecryptionError('NOT_INITIALIZED');

    // Unwrap the note's DEK
    const wrappedDEKBytes = fromBase64Standard(wrappedDEK);
    const wrapAAD = this.buildAAD(noteID, 'note', 'dek');
    let dek = this.unwrapDEK(wrappedDEKBytes, this.kek, wrapAAD);
    if (dek === null) {
      dek = this.unwrapDEK(wrappedDEKBytes, this.kek);
    }
    if (dek === null) throw new DecryptionError('INVALID_KEY_OR_DATA');

    // Decrypt folder path with DEK
    const ciphertextBytes = fromBase64Standard(encryptedFolderPath);
    const folderAAD = stringToBytes(`xelanote:e2ee:v3:note:folder_path:${noteID}`);
    const plaintext = decrypt(ciphertextBytes, dek, folderAAD);

    dek.fill(0);

    if (plaintext === null) throw new DecryptionError('INVALID_KEY_OR_DATA');
    return bytesToString(plaintext);
  }

  /**
   * Export KEK for persistence (internal use only).
   * Returns a COPY to prevent external mutation.
   *
   * @throws Error if KEK not initialized
   */
  exportKEK(): Uint8Array {
    if (!this.kek) {
      throw new Error('KEK not initialized');
    }
    return new Uint8Array(this.kek); // Copy
  }

  /**
   * Import KEK from persistence (internal use only).
   * Stores a COPY to prevent external mutation.
   *
   * @param kek The KEK to import (will be copied)
   */
  importKEK(kek: Uint8Array): void {
    this.kek = new Uint8Array(kek); // Copy
  }
}

// Singleton instance
export const e2eEncryption = new E2EEncryption();
