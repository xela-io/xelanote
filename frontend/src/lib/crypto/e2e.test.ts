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

import { E2EEncryption } from './e2e';

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
