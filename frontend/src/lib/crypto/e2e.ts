import {
  bytesToString,
  decrypt,
  deriveKey,
  encrypt,
  fromBase64Standard,
  generateDEK,
  isInitialized,
  stringToBytes,
  toBase64Standard,
} from './sodium';

export interface EncryptionMetadata {
  version: 2;
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
      metadata.version !== 2 ||
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
        version: 2,
        algorithm: 'XChaCha20-Poly1305',
        kdf: 'Argon2id',
        kdf_strength: 'interactive',
        nonce_bytes: 24,
        wrapped_dek: metadata.wrapped_dek,
      },
    };
  }

  /**
   * Setup KEK from user password (synchronous for Phase 1).
   * Call this on login after receiving the user's encryption salt from the server.
   *
   * TEMPORARY: Using synchronous derivation until worker bundling is fixed.
   * This will block UI briefly (~100-500ms) during login.
   *
   * @param password - User's password
   * @param salt - User-specific salt (16 bytes, stored server-side)
   */
  async setupKEK(password: string, salt: Uint8Array): Promise<void> {
    // Synchronous key derivation (blocks UI briefly)
    this.kek = deriveKey(password, salt);
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
   * @returns Wrapped key as Uint8Array (24-byte nonce + ciphertext + 16-byte tag)
   */
  private wrapDEK(dek: Uint8Array, kek: Uint8Array): Uint8Array {
    return encrypt(dek, kek);
  }

  /**
   * Unwrap a DEK using XChaCha20-Poly1305.
   *
   * @param wrappedDEK - Wrapped DEK bytes (nonce + ciphertext + tag)
   * @param kek - Key Encryption Key
   * @returns Unwrapped Data Encryption Key or null if decryption fails
   */
  private unwrapDEK(wrappedDEK: Uint8Array, kek: Uint8Array): Uint8Array | null {
    return decrypt(wrappedDEK, kek);
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
  encryptNote(content: string): EncryptedPayload {
    if (!this.kek) throw new Error('KEK not initialized');
    if (!isInitialized()) throw new Error('libsodium not initialized');

    // 1. Generate random DEK for this note
    const dek = generateDEK();

    // 2. Encrypt content with DEK
    const plaintext = stringToBytes(content);
    const ciphertext = encrypt(plaintext, dek);

    // 3. Wrap DEK with KEK
    const wrappedDEK = this.wrapDEK(dek, this.kek);

    // 4. Clear DEK from memory
    dek.fill(0);

    // 5. Return payload
    return {
      ciphertext: toBase64Standard(ciphertext),
      metadata: {
        version: 2,
        algorithm: 'XChaCha20-Poly1305',
        kdf: 'Argon2id',
        kdf_strength: 'interactive',
        nonce_bytes: 24,
        wrapped_dek: toBase64Standard(wrappedDEK),
      },
    };
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
  decryptNote(payload: EncryptedPayload): string {
    if (!this.kek) throw new DecryptionError('NOT_INITIALIZED');
    if (!isInitialized()) throw new DecryptionError('NOT_INITIALIZED');

    const { ciphertext, metadata } = payload;

    // Validate metadata
    if (!metadata.wrapped_dek) {
      throw new DecryptionError('CORRUPTED_METADATA');
    }

    // 1. Unwrap DEK with KEK
    const wrappedDEKBytes = fromBase64Standard(metadata.wrapped_dek);
    const dek = this.unwrapDEK(wrappedDEKBytes, this.kek);

    if (dek === null) {
      throw new DecryptionError('INVALID_KEY_OR_DATA');
    }

    // 2. Decrypt content with DEK
    const ciphertextBytes = fromBase64Standard(ciphertext);
    const plaintext = decrypt(ciphertextBytes, dek);

    // 3. Clear DEK from memory
    dek.fill(0);

    if (plaintext === null) {
      throw new DecryptionError('INVALID_KEY_OR_DATA');
    }

    return bytesToString(plaintext);
  }

  /**
   * Encrypt title (optional).
   * Returns a JSON string containing the encrypted payload.
   *
   * @param title - Plaintext title
   * @returns JSON string of encrypted payload
   */
  encryptTitle(title: string): string {
    const payload = this.encryptNote(title);
    return JSON.stringify(payload);
  }

  /**
   * Decrypt title.
   *
   * @param encryptedTitle - JSON string of encrypted payload
   * @returns Decrypted plaintext title
   */
  decryptTitle(encryptedTitle: string): string {
    const payload = this.parseEncryptedPayload(encryptedTitle);
    return this.decryptNote(payload);
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
   * 4. Perform test decryption on random samples to validate
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
    const oldKEK = deriveKey(oldPassword, salt);

    // Step 2: Derive new KEK from new password
    const newKEK = deriveKey(newPassword, salt);

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

    // Step 5: Validate re-wrapping with test decryption
    // Sample 3 random notes (or all if < 3)
    const notesToTest = notes.filter((n) => n.wrapped_dek && n.encrypted_content);
    const sampleSize = Math.min(3, notesToTest.length);
    const testSamples = [];

    // Randomly select samples
    const shuffled = [...notesToTest].sort(() => Math.random() - 0.5);
    for (let i = 0; i < sampleSize; i++) {
      testSamples.push(shuffled[i]);
    }

    // Test decryption with new KEK
    for (const note of testSamples) {
      try {
        const newWrappedDEK = reWrappedNotes.get(note.id);
        if (!newWrappedDEK || !note.encrypted_content) continue;

        // Unwrap DEK with NEW KEK
        const newWrappedDEKBytes = fromBase64Standard(newWrappedDEK);
        const dek = this.unwrapDEK(newWrappedDEKBytes, newKEK);

        if (dek === null) {
          throw new Error(`Test decryption failed for note ${note.id} - re-wrapping corrupted`);
        }

        // Try to decrypt content with unwrapped DEK
        const ciphertext = fromBase64Standard(note.encrypted_content);
        const plaintext = decrypt(ciphertext, dek);

        if (plaintext === null) {
          throw new Error(`Test decryption failed for note ${note.id} - content decryption failed`);
        }

        // Clear DEK from memory
        dek.fill(0);
      } catch (err) {
        // Clean up KEKs before throwing
        oldKEK.fill(0);
        newKEK.fill(0);
        throw new Error(`Re-wrapping validation failed: ${err}`);
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
