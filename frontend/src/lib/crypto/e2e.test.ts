import { beforeEach, describe, expect, it, vi } from 'vitest';

const sodiumMocks = vi.hoisted(() => ({
  decrypt: vi.fn(),
  deriveKey: vi.fn(),
  deriveKeyAsync: vi.fn(),
  encrypt: vi.fn(),
  fromBase64Standard: vi.fn(),
  generateDEK: vi.fn(),
  isInitialized: vi.fn(() => true),
  toBase64Standard: vi.fn(),
}));

vi.mock('./sodium', () => ({
  bytesToString: (value: Uint8Array) => new TextDecoder().decode(value),
  decrypt: sodiumMocks.decrypt,
  deriveKey: sodiumMocks.deriveKey,
  deriveKeyAsync: sodiumMocks.deriveKeyAsync,
  encrypt: sodiumMocks.encrypt,
  fromBase64Standard: sodiumMocks.fromBase64Standard,
  generateDEK: sodiumMocks.generateDEK,
  isInitialized: sodiumMocks.isInitialized,
  stringToBytes: (value: string) => new TextEncoder().encode(value),
  toBase64Standard: sodiumMocks.toBase64Standard,
}));

import { DecryptionError, E2EEncryption } from './e2e';

describe('E2EEncryption setupKEK', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('uses async KDF when Worker path is available', async () => {
    const salt = new Uint8Array([1, 2, 3, 4]);
    const asyncKEK = new Uint8Array([9, 8, 7, 6]);
    sodiumMocks.deriveKeyAsync.mockResolvedValue(asyncKEK);

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', salt);

    expect(sodiumMocks.deriveKeyAsync).toHaveBeenCalledWith('password', salt);
    expect(sodiumMocks.deriveKey).not.toHaveBeenCalled();
    expect(encryption.exportKEK()).toEqual(asyncKEK);
  });

  it('falls back to sync KDF when async Worker derivation fails', async () => {
    const salt = new Uint8Array([4, 3, 2, 1]);
    const fallbackKEK = new Uint8Array([1, 1, 1, 1]);
    sodiumMocks.deriveKeyAsync.mockRejectedValue(new Error('worker unavailable'));
    sodiumMocks.deriveKey.mockReturnValue(fallbackKEK);
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    try {
      const encryption = new E2EEncryption();
      await encryption.setupKEK('password', salt);

      expect(sodiumMocks.deriveKeyAsync).toHaveBeenCalledWith('password', salt);
      expect(sodiumMocks.deriveKey).toHaveBeenCalledWith('password', salt);
      expect(encryption.exportKEK()).toEqual(fallbackKEK);
    } finally {
      warnSpy.mockRestore();
    }
  });
});

describe('E2EEncryption decrypt error handling', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('throws INVALID_KEY_OR_DATA when wrapped DEK cannot be unwrapped', async () => {
    const kek = new Uint8Array([1, 2, 3, 4]);
    sodiumMocks.deriveKeyAsync.mockResolvedValue(kek);
    sodiumMocks.fromBase64Standard.mockReturnValueOnce(new Uint8Array([9, 9, 9]));
    sodiumMocks.decrypt.mockReturnValueOnce(null);

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', new Uint8Array([1, 1, 1, 1]));

    const payload = {
      ciphertext: 'ciphertext-b64',
      metadata: {
        version: 3 as const,
        algorithm: 'XChaCha20-Poly1305' as const,
        kdf: 'Argon2id' as const,
        kdf_strength: 'interactive' as const,
        nonce_bytes: 24 as const,
        wrapped_dek: 'wrapped-b64',
      },
    };

    try {
      encryption.decryptNote(payload, 'note-1');
      expect.fail('expected DecryptionError');
    } catch (err) {
      expect(err).toBeInstanceOf(DecryptionError);
      expect((err as DecryptionError).code).toBe('INVALID_KEY_OR_DATA');
    }
  });

  it('throws INVALID_KEY_OR_DATA when ciphertext decryption fails', async () => {
    const kek = new Uint8Array([2, 2, 2, 2]);
    const dek = new Uint8Array(32).fill(7);
    sodiumMocks.deriveKeyAsync.mockResolvedValue(kek);
    sodiumMocks.fromBase64Standard
      .mockReturnValueOnce(new Uint8Array([1, 2, 3])) // wrapped_dek
      .mockReturnValueOnce(new Uint8Array([4, 5, 6])); // ciphertext
    sodiumMocks.decrypt.mockReturnValueOnce(dek).mockReturnValueOnce(null);

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', new Uint8Array([2, 2, 2, 2]));

    const payload = {
      ciphertext: 'ciphertext-b64',
      metadata: {
        version: 3 as const,
        algorithm: 'XChaCha20-Poly1305' as const,
        kdf: 'Argon2id' as const,
        kdf_strength: 'interactive' as const,
        nonce_bytes: 24 as const,
        wrapped_dek: 'wrapped-b64',
      },
    };

    try {
      encryption.decryptNote(payload, 'note-2');
      expect.fail('expected DecryptionError');
    } catch (err) {
      expect(err).toBeInstanceOf(DecryptionError);
      expect((err as DecryptionError).code).toBe('INVALID_KEY_OR_DATA');
    }
  });

  it('throws CORRUPTED_METADATA for invalid encrypted title payload', async () => {
    sodiumMocks.deriveKeyAsync.mockResolvedValue(new Uint8Array([3, 3, 3, 3]));

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', new Uint8Array([3, 3, 3, 3]));

    try {
      encryption.decryptTitle('not-json', 'note-3');
      expect.fail('expected DecryptionError');
    } catch (err) {
      expect(err).toBeInstanceOf(DecryptionError);
      expect((err as DecryptionError).code).toBe('CORRUPTED_METADATA');
    }
  });
});

describe('E2EEncryption AAD behavior', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('uses AAD for v3 note encryption when noteID is provided', async () => {
    sodiumMocks.deriveKeyAsync.mockResolvedValue(new Uint8Array([4, 4, 4, 4]));
    sodiumMocks.generateDEK.mockReturnValue(new Uint8Array(32).fill(9));
    sodiumMocks.encrypt.mockReturnValue(new Uint8Array([1, 2, 3]));
    sodiumMocks.toBase64Standard.mockReturnValue('b64');

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', new Uint8Array([4, 4, 4, 4]));
    encryption.encryptNote('hello', 'note-v3');

    expect(sodiumMocks.encrypt).toHaveBeenCalledTimes(2);
    expect(ArrayBuffer.isView(sodiumMocks.encrypt.mock.calls[0][2])).toBe(true);
    expect(ArrayBuffer.isView(sodiumMocks.encrypt.mock.calls[1][2])).toBe(true);
  });

  it('does not use AAD for v2 note encryption when noteID is omitted', async () => {
    sodiumMocks.deriveKeyAsync.mockResolvedValue(new Uint8Array([5, 5, 5, 5]));
    sodiumMocks.generateDEK.mockReturnValue(new Uint8Array(32).fill(8));
    sodiumMocks.encrypt.mockReturnValue(new Uint8Array([1, 2, 3]));
    sodiumMocks.toBase64Standard.mockReturnValue('b64');

    const encryption = new E2EEncryption();
    await encryption.setupKEK('password', new Uint8Array([5, 5, 5, 5]));
    encryption.encryptNote('hello');

    expect(sodiumMocks.encrypt).toHaveBeenCalledTimes(2);
    expect(sodiumMocks.encrypt.mock.calls[0][2]).toBeUndefined();
    expect(sodiumMocks.encrypt.mock.calls[1][2]).toBeUndefined();
  });
});

describe('E2EEncryption recovery rewrap', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('re-wraps recovery DEKs into password DEKs', async () => {
    const recoveryKEK = new Uint8Array([7, 7, 7, 7]);
    const newKEK = new Uint8Array([8, 8, 8, 8]);
    const unwrappedDEK = new Uint8Array(32).fill(5);

    sodiumMocks.deriveKeyAsync.mockResolvedValueOnce(recoveryKEK).mockResolvedValueOnce(newKEK);
    sodiumMocks.fromBase64Standard.mockReturnValue(new Uint8Array([1, 2, 3]));
    sodiumMocks.decrypt.mockReturnValue(new Uint8Array(unwrappedDEK));
    sodiumMocks.encrypt.mockReturnValue(new Uint8Array([9, 9, 9]));
    sodiumMocks.toBase64Standard
      .mockReturnValueOnce('wrapped-note')
      .mockReturnValueOnce('wrapped-version');

    const encryption = new E2EEncryption();
    const result = await encryption.reWrapRecoveryDEKs(
      [{ id: 'note-1', wrapped_dek_recovery: 'note-recovery-b64' }],
      [{ id: '101', wrapped_dek_recovery: 'version-recovery-b64' }],
      'recovery-key',
      'new-password',
      new Uint8Array([1, 1, 1, 1]),
      new Uint8Array([2, 2, 2, 2])
    );

    expect(result.notes.get('note-1')).toBe('wrapped-note');
    expect(result.versions.get('101')).toBe('wrapped-version');
    expect(sodiumMocks.decrypt).toHaveBeenCalledTimes(2);
    expect(sodiumMocks.encrypt).toHaveBeenCalledTimes(2);
  });
});
