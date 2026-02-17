// @vitest-environment jsdom
// SEC-001: Tests for token-exposure elimination in web auth flow
import type { Mock } from 'vitest';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Mock dependencies
vi.mock('$lib/api', async () => {
  const actual = await vi.importActual('$lib/api');
  return {
    ...actual,
    initApiAuth: vi.fn(),
    getCurrentUser: vi.fn(),
    refreshTokenViaCookie: vi.fn(),
    login: vi.fn(),
    logoutApi: vi.fn(),
  };
});

vi.mock('$lib/stores/encryption.svelte', async () => {
  const actual = await vi.importActual('$lib/stores/encryption.svelte');
  return {
    ...actual,
    setupEncryption: vi.fn(),
    isEncryptionUnlocked: vi.fn(),
    lockEncryption: vi.fn(),
  };
});

import * as api from '$lib/api';
import * as auth from '$lib/stores/auth.svelte';

const FAKE_USER: auth.User = { id: 1, username: 'test', email: 'test@test.com', is_admin: false };
const fetchMock = vi.fn();

function resetAuthState() {
  Object.assign(auth.getAuthState(), {
    user: null,
    accessToken: null,
    refreshToken: null,
    isAuthenticated: false,
  });
}

describe('SEC-001: Token exposure elimination', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', fetchMock);
    fetchMock.mockResolvedValue(new Response(null, { status: 401 }));
    sessionStorage.clear();
    localStorage.clear();
    resetAuthState();
    (api.getCurrentUser as Mock).mockResolvedValue(FAKE_USER);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    sessionStorage.clear();
    localStorage.clear();
  });

  describe('updateTokens', () => {
    it('should be a no-op when accessToken is empty', () => {
      // Set up some existing state
      Object.assign(auth.getAuthState(), {
        accessToken: 'old-token',
        refreshToken: 'old-refresh',
        isAuthenticated: true,
        user: FAKE_USER,
      });

      auth.updateTokens('', 'new-refresh');

      // State must NOT change
      expect(auth.getAccessToken()).toBe('old-token');
    });

    it('should be a no-op when refreshToken is empty', () => {
      Object.assign(auth.getAuthState(), {
        accessToken: 'old-token',
        refreshToken: 'old-refresh',
        isAuthenticated: true,
        user: FAKE_USER,
      });

      auth.updateTokens('new-access', '');

      expect(auth.getAccessToken()).toBe('old-token');
    });

    it('should update tokens when both are provided', () => {
      Object.assign(auth.getAuthState(), {
        accessToken: 'old-token',
        refreshToken: 'old-refresh',
        isAuthenticated: true,
        user: FAKE_USER,
      });

      auth.updateTokens('new-access', 'new-refresh');

      expect(auth.getAccessToken()).toBe('new-access');
    });
  });

  describe('setAuthCookieOnly', () => {
    it('should set user and isAuthenticated without tokens', () => {
      auth.setAuthCookieOnly(FAKE_USER);

      expect(auth.isAuthenticated()).toBe(true);
      expect(auth.getCurrentUser()).toEqual(FAKE_USER);
      expect(auth.getAccessToken()).toBeNull();
    });
  });

  describe('login (web client path)', () => {
    it('should authenticate when response has user but no tokens', async () => {
      // Web client: server returns user + encryption_salt but NO tokens
      (api.login as Mock).mockResolvedValue({
        user: FAKE_USER,
        // No access_token, no refresh_token (web client, cookies only)
      });

      await auth.login('test', 'password');

      expect(auth.isAuthenticated()).toBe(true);
      expect(auth.getCurrentUser()).toEqual(FAKE_USER);
      // No tokens in memory for web clients
      expect(auth.getAccessToken()).toBeNull();
    });

    it('should store tokens when response includes them (desktop path)', async () => {
      (api.login as Mock).mockResolvedValue({
        access_token: 'desktop-access',
        refresh_token: 'desktop-refresh',
        user: FAKE_USER,
      });

      await auth.login('test', 'password');

      expect(auth.isAuthenticated()).toBe(true);
      expect(auth.getAccessToken()).toBe('desktop-access');
    });
  });

  describe('initAuth (web client path)', () => {
    it('should restore session when refresh returns no tokens in body', async () => {
      // Web client: refresh succeeds via cookies, but body has no tokens
      (api.refreshTokenViaCookie as Mock).mockResolvedValue({
        success: true,
        tokens: {}, // Empty — web client gets tokens only in cookies
      });

      await auth.initAuth();

      expect(auth.isAuthenticated()).toBe(true);
      expect(auth.getCurrentUser()).toEqual(FAKE_USER);
      // No tokens in memory
      expect(auth.getAccessToken()).toBeNull();
    });

    it('should set tokens in memory when refresh returns them (desktop path)', async () => {
      (api.refreshTokenViaCookie as Mock).mockResolvedValue({
        success: true,
        tokens: {
          access_token: 'refreshed-access',
          refresh_token: 'refreshed-refresh',
        },
      });

      await auth.initAuth();

      expect(auth.isAuthenticated()).toBe(true);
      expect(auth.getAccessToken()).toBe('refreshed-access');
    });
  });
});
