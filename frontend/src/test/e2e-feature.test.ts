// @vitest-environment jsdom
import type { Mock } from 'vitest';
import { afterEach,beforeEach, describe, expect, it, vi } from 'vitest';

// Mock dependencies
vi.mock('$lib/api', async () => {
  const actual = await vi.importActual('$lib/api');
  return {
    ...actual,
    initApiAuth: vi.fn(),
    getCurrentUser: vi.fn(),
    refreshTokenViaCookie: vi.fn(),
    register: vi.fn(),
    login: vi.fn(),
    logoutApi: vi.fn(),
    createNote: vi.fn(),
  };
});

vi.mock('$lib/stores/encryption.svelte', async () => {
  const actual = await vi.importActual('$lib/stores/encryption.svelte');
  return {
    ...actual,
    setupEncryption: vi.fn(),
    isEncryptionUnlocked: vi.fn(),
    lockEncryption: vi.fn(),
    encryptNote: vi.fn(),
    decryptNote: vi.fn(),
  };
});

// Import stores after mocking
import * as api from '$lib/api';
import { fromBase64Standard } from '$lib/crypto/sodium';
import * as auth from '$lib/stores/auth.svelte';
import * as encryption from '$lib/stores/encryption.svelte';
import * as notes from '$lib/stores/notes.svelte';

const FAKE_USER: auth.User = { id: 1, username: 'test', email: 'test@test.com', is_admin: false };
const FAKE_ACCESS_TOKEN = 'fake_access_token';
const FAKE_REFRESH_TOKEN = 'fake_refresh_token';
const FAKE_ENCRYPTION_SALT = 'NZ/UXtGgL4+Cg2NGAi4e/w=='; // Dummy Base64 salt

describe('E2E Encryption Feature Flow', () => {
  beforeEach(() => {
    // Reset mocks and state before each test
    vi.clearAllMocks();
    sessionStorage.clear();
    localStorage.clear(); // Clear local storage for setup too
    // Directly reset auth state to avoid calling logout() and its side effects
    Object.assign(auth.getAuthState(), {
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
    });

    // Default mock implementations
    (api.getCurrentUser as Mock).mockResolvedValue(FAKE_USER);
    (encryption.isEncryptionUnlocked as Mock).mockReturnValue(true);
    (encryption.encryptNote as Mock).mockReturnValue({
      encryptedTitle: 'encrypted-title',
      encryptedContent: {
        ciphertext: 'base64-ciphertext-string',
        metadata: { wrapped_dek: 'wrapped-dek', version: 2, algorithm: 'XChaCha20-Poly1305' },
      },
      keywords: [],
    });
    (api.createNote as Mock).mockResolvedValue({
      id: 'note-1',
      title: '',
      content: '',
      encrypted_content: 'blob',
      version: 1,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  });

  afterEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it('Scenario 1: Full login, encryption setup, and note creation', async () => {
    // --- 1. LOGIN ---
    (api.login as Mock).mockResolvedValue({
      access_token: FAKE_ACCESS_TOKEN,
      refresh_token: FAKE_REFRESH_TOKEN,
      user: FAKE_USER,
      encryption_salt: FAKE_ENCRYPTION_SALT,
    });

    await auth.login('test', 'password');

    // --- 2. VERIFY AUTH STATE ---
    expect(auth.isAuthenticated()).toBe(true);
    expect(auth.getAccessToken()).toBe(FAKE_ACCESS_TOKEN);
    // SEC-006: Tokens are NO LONGER stored in sessionStorage
    expect(sessionStorage.getItem('xelanote_access_token')).toBeNull();

    // --- 3. VERIFY ENCRYPTION SETUP ---
    expect(encryption.setupEncryption).toHaveBeenCalledOnce();
    const salt = fromBase64Standard(FAKE_ENCRYPTION_SALT);
    expect(encryption.setupEncryption).toHaveBeenCalledWith('password', FAKE_USER.id, salt);

    // --- 4. CREATE NOTE ---
    await notes.createNote('My Encrypted Note', 'This is secret');

    // --- 5. VERIFY ENCRYPTION AND API CALL ---
    expect(encryption.isEncryptionUnlocked).toHaveBeenCalled();
    expect(encryption.encryptNote).toHaveBeenCalledOnce();
    expect(encryption.encryptNote).toHaveBeenCalledWith('My Encrypted Note', 'This is secret');

    expect(api.createNote).toHaveBeenCalledOnce();
    const createNotePayload = (api.createNote as Mock).mock.calls[0][0];

    // IMPORTANT: Verify that plain text is NOT sent to the backend
    expect(createNotePayload.title).toBe(''); // Title should be blank because it's encrypted
    expect(createNotePayload).not.toHaveProperty('content');
    expect(createNotePayload.encrypted_title).toBe('encrypted-title');
    expect(createNotePayload.encrypted_content).toBe('base64-ciphertext-string');
    expect(createNotePayload.wrapped_dek).toBe('wrapped-dek');
  });

  it('Scenario 2: Session is persisted on app re-initialization (F5 simulation)', async () => {
    // SEC-006: Session persistence now works via HttpOnly cookies, not sessionStorage
    // This test verifies that old sessionStorage tokens are cleaned up

    // --- 1. SETUP OLD SESSION STORAGE (deprecated) ---
    sessionStorage.setItem('xelanote_access_token', FAKE_ACCESS_TOKEN);
    sessionStorage.setItem('xelanote_refresh_token', FAKE_REFRESH_TOKEN);

    // --- 2. INITIALIZE AUTH ---
    (api.refreshTokenViaCookie as Mock).mockResolvedValue({
      success: true,
      tokens: { access_token: FAKE_ACCESS_TOKEN, refresh_token: FAKE_REFRESH_TOKEN },
    });
    await auth.initAuth();

    // --- 3. VERIFY OLD TOKENS WERE CLEANED UP ---
    // SEC-006: sessionStorage tokens should be removed during migration
    expect(sessionStorage.getItem('xelanote_access_token')).toBeNull();
    expect(sessionStorage.getItem('xelanote_refresh_token')).toBeNull();

    // --- 4. VERIFY AUTH RESTORATION FROM COOKIES ---
    // getCurrentUser is called to restore session from HttpOnly cookies
    expect(api.getCurrentUser).toHaveBeenCalledOnce();
    expect(auth.isAuthenticated()).toBe(true);
    expect(auth.getCurrentUser()).toEqual(FAKE_USER);
    // Token is set in memory from the refresh result
    expect(auth.getAccessToken()).toBe(FAKE_ACCESS_TOKEN);

    // NOTE: Encryption is NOT automatically unlocked. User must re-enter password.
    // This is the correct and secure behavior.
    expect(encryption.setupEncryption).not.toHaveBeenCalled();
  });

  it('Scenario 3: Logout clears session, tokens, and encryption keys', async () => {
    // --- 1. LOGIN FIRST ---
    (api.login as Mock).mockResolvedValue({
      access_token: FAKE_ACCESS_TOKEN,
      refresh_token: FAKE_REFRESH_TOKEN,
      user: FAKE_USER,
    });
    await auth.login('test', 'password');
    expect(auth.isAuthenticated()).toBe(true);

    // --- 2. LOGOUT ---
    await auth.logoutAsync();

    // --- 3. VERIFY STATE IS CLEARED ---
    expect(auth.isAuthenticated()).toBe(false);
    expect(auth.getCurrentUser()).toBeNull();
    expect(auth.getAccessToken()).toBeNull();
    expect(sessionStorage.getItem('xelanote_access_token')).toBeNull();

    // --- 4. VERIFY SIDE-EFFECTS ---
    expect(encryption.lockEncryption).toHaveBeenCalledOnce(); // Should be called once now
    expect(api.logoutApi).toHaveBeenCalledOnce();
    expect(api.logoutApi).toHaveBeenCalledWith(FAKE_REFRESH_TOKEN);
  });

  it('should not create a note if encryption is locked', async () => {
    // --- 1. LOGIN ---
    (api.login as Mock).mockResolvedValue({
      access_token: FAKE_ACCESS_TOKEN,
      refresh_token: FAKE_REFRESH_TOKEN,
      user: FAKE_USER,
    });
    await auth.login('test', 'password');
    expect(auth.isAuthenticated()).toBe(true);

    // --- 2. LOCK ENCRYPTION ---
    (encryption.isEncryptionUnlocked as Mock).mockReturnValue(false);

    // --- 3. ATTEMPT TO CREATE NOTE ---
    await expect(notes.createNote('Title', 'Content')).rejects.toThrow('ENCRYPTION_LOCKED');

    // --- 4. VERIFY NO API CALL WAS MADE ---
    expect(api.createNote).not.toHaveBeenCalled();
  });
});
