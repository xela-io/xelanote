import { beforeEach, describe, expect, it, vi } from 'vitest';

const encryptNote = vi.fn((content: string) => ({ ciphertext: content, metadata: {} }));
const decryptNote = vi.fn((payload: { ciphertext: string }) => payload.ciphertext);
const encryptTitle = vi.fn((title: string) => `enc:${title}`);
const decryptTitle = vi.fn((title: string) => title.replace('enc:', ''));
const setupKEK = vi.fn();
const exportKEK = vi.fn(() => new Uint8Array([1, 2, 3]));
const importKEK = vi.fn();
const clearKEK = vi.fn();

vi.mock('$lib/crypto/e2e', () => ({
  DecryptionError: class DecryptionError extends Error {},
  e2eEncryption: {
    encryptNote,
    decryptNote,
    encryptTitle,
    decryptTitle,
    setupKEK,
    exportKEK,
    importKEK,
    clearKEK,
  },
}));

const persistKEK = vi.fn();
const loadPersistedKEK = vi.fn();
const clearPersistedKEK = vi.fn();
vi.mock('$lib/crypto/kek-persistence', () => ({
  persistKEK,
  loadPersistedKEK,
  clearPersistedKEK,
}));

const updateEncryptionPreferences = vi.fn();
const updateSecurityPreferences = vi.fn();
vi.mock('$lib/api', () => ({
  updateEncryptionPreferences,
  updateSecurityPreferences,
}));

const warning = vi.fn();
const error = vi.fn();
vi.mock('$lib/stores/toast.svelte', () => ({
  warning,
  error,
}));

vi.mock('$lib/config', () => ({
  isDesktop: () => false,
}));

vi.mock('$lib/desktop', () => ({
  getDesktopBridge: vi.fn(),
}));

describe('encryption store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('should encrypt and decrypt notes when unlocked', async () => {
    const encryption = await import('$lib/stores/encryption.svelte');

    await encryption.setupEncryption('pw', 1, new Uint8Array([1, 2, 3]));

    const encrypted = encryption.encryptNote('title', 'content', 'note-1');
    expect(encryptNote).toHaveBeenCalledWith('content', 'note-1');
    expect(encrypted.encryptedTitle).toBe(null);

    const decrypted = encryption.decryptNote(null, encrypted.encryptedContent, 'note-1');
    expect(decryptNote).toHaveBeenCalled();
    expect(decrypted.content).toBe('content');
  });

  it('should encrypt title but never emit plaintext keywords for encrypted notes', async () => {
    const encryption = await import('$lib/stores/encryption.svelte');

    await encryption.setupEncryption('pw', 1, new Uint8Array([1, 2, 3]));
    await encryption.updateSettings({ encryptTitles: true });
    expect(updateEncryptionPreferences).toHaveBeenCalledWith({
      keywords_enabled: false,
      encrypt_titles: true,
    });

    const encrypted = encryption.encryptNote('title', 'content', 'note-1');
    expect(encryptTitle).toHaveBeenCalledWith('title', 'note-1');
    expect(encrypted.encryptedTitle).toBe('enc:title');
    expect(encrypted.keywords).toEqual([]);
  });

  it('should restore KEK and unlock state when persisted', async () => {
    loadPersistedKEK.mockResolvedValue(new Uint8Array([9, 9]));
    const encryption = await import('$lib/stores/encryption.svelte');

    const ok = await encryption.tryRestoreKEK(1);
    expect(ok).toBe(true);
    expect(importKEK).toHaveBeenCalled();
    expect(encryption.isEncryptionUnlocked()).toBe(true);
    expect(encryption.getUserID()).toBe(1);
  });

  it('should clear persisted KEK on restore error', async () => {
    loadPersistedKEK.mockRejectedValueOnce(new Error('boom'));
    const encryption = await import('$lib/stores/encryption.svelte');

    const ok = await encryption.tryRestoreKEK(1);
    expect(ok).toBe(false);
    expect(clearPersistedKEK).toHaveBeenCalledWith(1);
    expect(warning).toHaveBeenCalled();
  });

  it('should update security level and persist KEK', async () => {
    const encryption = await import('$lib/stores/encryption.svelte');
    await encryption.setupEncryption('pw', 1, new Uint8Array([1, 2, 3]));

    await encryption.updateSecurityLevel('balanced');
    expect(updateSecurityPreferences).toHaveBeenCalledWith({ security_level: 'balanced' });
  });
});
