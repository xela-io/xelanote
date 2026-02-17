import { beforeEach, describe, expect, it, vi } from 'vitest';

const initApiAuth = vi.fn();
const logoutApi = vi.fn();
const getCurrentUser = vi.fn();

vi.mock('$lib/api', () => ({
  initApiAuth,
  refreshTokenViaCookie: vi.fn().mockResolvedValue({ success: false, reason: 'auth_error' }),
  getCurrentUser,
  register: vi.fn(),
  login: vi.fn(),
  logoutApi,
}));

vi.mock('$lib/config', () => ({
  getApiBaseUrl: () => 'http://localhost:8080/api',
  getServerUrl: () => 'http://localhost',
  isDesktop: () => false,
}));

const lockEncryption = vi.fn();
vi.mock('$lib/stores/encryption.svelte', () => ({
  lockEncryption,
  setupEncryption: vi.fn(),
}));

const resetSettings = vi.fn();
vi.mock('$lib/stores/settings.svelte', () => ({ resetSettings }));

const resetToDefaults = vi.fn();
vi.mock('$lib/stores/ui.svelte', () => ({ resetToDefaults }));

const resetJournalFeature = vi.fn();
const resetRecipeFeature = vi.fn();
vi.mock('$lib/stores/features.svelte', () => ({
  resetJournalFeature,
  resetRecipeFeature,
}));

const resetJournalState = vi.fn();
vi.mock('$lib/stores/journal.svelte', () => ({ resetJournalState }));

const resetRecipeState = vi.fn();
vi.mock('$lib/stores/recipes.svelte', () => ({ resetRecipeState }));

function makeJwt(exp: number, iat: number): string {
  const header = { alg: 'none', typ: 'JWT' };
  const payload = { exp, iat };
  const encode = (obj: object) =>
    Buffer.from(JSON.stringify(obj))
      .toString('base64')
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
  return `${encode(header)}.${encode(payload)}.sig`;
}

describe('auth store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('should set auth state and notify token listeners', async () => {
    const auth = await import('$lib/stores/auth.svelte');

    const token = makeJwt(1234, 1000);
    const listener = vi.fn();
    auth.addTokenUpdateListener(listener);

    await auth.setAuth(token, 'refresh', {
      id: 1,
      username: 'user',
      email: 'user@example.com',
      is_admin: false,
    });

    expect(auth.getAuthState().isAuthenticated).toBe(true);
    expect(auth.getAccessToken()).toBe(token);
    expect(auth.getTokenExpiry()).toBe(1234);
    expect(auth.getTokenIssuedAt()).toBe(1000);
    expect(listener).toHaveBeenCalledWith(1234, 1000);
    expect(sessionStorage.getItem('xelanote_token_exp')).toBe('1234');
    expect(sessionStorage.getItem('xelanote_token_iat')).toBe('1000');
  });

  it('should ignore updateTokens after logout', async () => {
    const auth = await import('$lib/stores/auth.svelte');

    auth.logout();
    auth.updateTokens(makeJwt(2000, 1500), 'refresh');

    expect(auth.getAccessToken()).toBe(null);
    expect(auth.getRefreshToken()).toBe(null);
    expect(auth.getTokenExpiry()).toBe(0);
  });

  it('should clear state and reset dependent stores on logout', async () => {
    const auth = await import('$lib/stores/auth.svelte');

    await auth.setAuth(makeJwt(1234, 1000), 'refresh', {
      id: 1,
      username: 'user',
      email: 'user@example.com',
      is_admin: false,
    });

    auth.logout();

    expect(auth.getAuthState().user).toBe(null);
    expect(lockEncryption).toHaveBeenCalled();
    expect(resetSettings).toHaveBeenCalled();
    expect(resetToDefaults).toHaveBeenCalled();
    expect(resetJournalFeature).toHaveBeenCalled();
    expect(resetJournalState).toHaveBeenCalled();
    expect(resetRecipeFeature).toHaveBeenCalled();
    expect(resetRecipeState).toHaveBeenCalled();
  });
});
