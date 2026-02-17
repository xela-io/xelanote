import { beforeEach, describe, expect, it, vi } from 'vitest';

const getPreferences = vi.fn();
const updatePreferences = vi.fn();
const changeEmail = vi.fn();
const changePassword = vi.fn();
const getCurrentUser = vi.fn();
const updateSecurityPreferences = vi.fn();

vi.mock('$lib/api', () => ({
  getPreferences,
  updatePreferences,
  changeEmail,
  changePassword,
  getCurrentUser,
  updateSecurityPreferences,
  addWebAuthnCredential: vi.fn(),
  deleteWebAuthnCredential: vi.fn(),
}));

const setTheme = vi.fn();
const setEditorMode = vi.fn();
const getIsMobile = vi.fn().mockReturnValue(false);
const getEditorMode = vi.fn().mockReturnValue('edit');
const getCurrentThemeId = vi.fn().mockReturnValue('gruvbox-dark');
vi.mock('$lib/stores/ui.svelte', () => ({
  setTheme,
  setEditorMode,
  getIsMobile,
  getEditorMode,
  getCurrentThemeId,
}));

const setSecurityLevel = vi.fn();
const initSettingsFromPreferences = vi.fn();
const isEncryptionUnlocked = vi.fn().mockReturnValue(false);
const getSecurityLevel = vi.fn().mockReturnValue('balanced');
const setupEncryption = vi.fn();
vi.mock('$lib/stores/encryption.svelte', () => ({
  setSecurityLevel,
  initSettingsFromPreferences,
  isEncryptionUnlocked,
  getSecurityLevel,
  setupEncryption,
}));

const initAutoLock = vi.fn();
vi.mock('$lib/stores/auto-lock.svelte', () => ({ initAutoLock }));

const error = vi.fn();
const success = vi.fn();
vi.mock('$lib/stores/toast.svelte', () => ({ error, success }));

vi.mock('$lib/themes', () => ({
  isValidThemeId: (id: string) => id === 'gruvbox-dark' || id === 'gruvbox-light',
}));

vi.mock('$lib/crypto/sodium', () => ({
  fromBase64Standard: vi.fn((s: string) => new Uint8Array([s.length])),
}));

describe('settings store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('should load preferences and apply UI + encryption settings', async () => {
    getPreferences.mockResolvedValue({
      theme: 'gruvbox-dark',
      editor_mode: 'split',
      security_level: 'balanced',
      auto_lock_timeout: 10,
      keywords_enabled: true,
      encrypt_titles: true,
    });

    const settings = await import('$lib/stores/settings.svelte');
    await settings.loadPreferences();

    expect(setTheme).toHaveBeenCalledWith('gruvbox-dark');
    expect(setEditorMode).toHaveBeenCalledWith('live');
    expect(setSecurityLevel).toHaveBeenCalledWith('balanced');
    expect(initSettingsFromPreferences).toHaveBeenCalledWith(true, true);
    expect(initAutoLock).not.toHaveBeenCalled();
  });

  it('should resolve split preference to live mode on mobile', async () => {
    getIsMobile.mockReturnValue(true);
    getPreferences.mockResolvedValue({
      theme: 'gruvbox-dark',
      editor_mode: 'split',
      security_level: 'balanced',
      auto_lock_timeout: 0,
      keywords_enabled: false,
      encrypt_titles: false,
    });

    const settings = await import('$lib/stores/settings.svelte');
    await settings.loadPreferences();

    expect(setEditorMode).toHaveBeenCalledWith('live');
  });

  it('should prefer locally stored editor mode when server returns split', async () => {
    localStorage.setItem('xelanote-editor-mode', 'preview');
    getPreferences.mockResolvedValue({
      theme: 'gruvbox-dark',
      editor_mode: 'split',
      security_level: 'balanced',
      auto_lock_timeout: 0,
      keywords_enabled: false,
      encrypt_titles: false,
    });

    const settings = await import('$lib/stores/settings.svelte');
    await settings.loadPreferences();

    expect(setEditorMode).toHaveBeenCalledWith('preview');
  });

  it('should save preferences and update UI', async () => {
    updatePreferences.mockResolvedValue(undefined);
    const settings = await import('$lib/stores/settings.svelte');

    const ok = await settings.savePreferences('gruvbox-dark', 'edit');
    expect(ok).toBe(true);
    expect(setTheme).toHaveBeenCalledWith('gruvbox-dark');
    expect(setEditorMode).toHaveBeenCalledWith('edit');
  });

  it('should persist live mode as split but keep live mode in UI', async () => {
    updatePreferences.mockResolvedValue(undefined);
    const settings = await import('$lib/stores/settings.svelte');

    const ok = await settings.savePreferences('gruvbox-dark', 'live');
    expect(ok).toBe(true);
    expect(updatePreferences).toHaveBeenCalledWith({
      theme: 'gruvbox-dark',
      editor_mode: 'split',
    });
    expect(setEditorMode).toHaveBeenCalledWith('live');
    expect(localStorage.getItem('xelanote-editor-mode')).toBe('live');
  });

  it('should map changeEmail errors', async () => {
    changeEmail.mockRejectedValueOnce(new Error('incorrect password'));
    const settings = await import('$lib/stores/settings.svelte');

    const res = await settings.changeEmail('a@b.com', 'pw');
    expect(res.success).toBe(false);
    expect(res.error).toBe('Falsches Passwort');
  });

  it('should rewrap KEK on password change when unlocked', async () => {
    isEncryptionUnlocked.mockReturnValue(true);
    getCurrentUser.mockResolvedValue({ id: 1, encryption_salt: 'abc' });
    changePassword.mockResolvedValue(undefined);

    const settings = await import('$lib/stores/settings.svelte');

    const res = await settings.changePassword('old', 'new');
    expect(res.success).toBe(true);
    expect(setupEncryption).toHaveBeenCalled();
    expect(initAutoLock).toHaveBeenCalled();
  });
});
