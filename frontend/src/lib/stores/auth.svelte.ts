// Auth store for managing user authentication state
import * as api from '$lib/api';
import { getServerUrl, isDesktop } from '$lib/config';
import { fromBase64Standard } from '$lib/crypto/sodium';
import { type DesktopBridge, getDesktopBridge } from '$lib/desktop';

import * as encryption from './encryption.svelte';
import * as features from './features.svelte';
import * as journal from './journal.svelte';
import * as recipes from './recipes.svelte';
import * as settings from './settings.svelte';
import * as ui from './ui.svelte';

export interface User {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  encryption_salt?: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
}

// SEC-006: Tokens are NO LONGER stored in sessionStorage (XSS vulnerability)
// Authentication now relies exclusively on HttpOnly cookies set by the backend
// Desktop apps still use OS keyring for persistence across app restarts

// SessionStorage keys - DEPRECATED (kept for cleanup migration only)
const ACCESS_TOKEN_KEY = 'xelanote_access_token';
const REFRESH_TOKEN_KEY = 'xelanote_refresh_token';

// Token expiry keys (NOT sensitive - only timestamps, not tokens)
// Used to restore proactive token refresh after page reload
const TOKEN_EXPIRY_KEY = 'xelanote_token_exp';
const TOKEN_ISSUED_KEY = 'xelanote_token_iat';

// Desktop bridge instance (lazy initialized)
let desktopBridge: DesktopBridge | null = null;

// State
const authState = $state<AuthState>({
  user: null,
  accessToken: null,
  refreshToken: null,
  isAuthenticated: false,
});

// Token expiry tracking for proactive refresh
let tokenExpiresAt = $state<number>(0); // Unix timestamp in SECONDS (from exp)
let tokenIssuedAt = $state<number>(0); // Unix timestamp in SECONDS (from iat)

// Event-based subscriber pattern for token updates
type TokenUpdateListener = (exp: number, iat: number) => void;
const tokenUpdateListeners: TokenUpdateListener[] = [];

// JWT payload interface
interface JWTPayload {
  exp: number; // SECONDS (Unix timestamp)
  iat: number; // SECONDS (Unix timestamp)
}

/**
 * Parse exp and iat claims from a JWT token.
 * Returns {exp: 0, iat: 0} on SSR or parse failure.
 */
function parseJWTClaims(token: string): JWTPayload {
  // Guard against SSR (atob not available)
  if (typeof window === 'undefined') {
    return { exp: 0, iat: 0 };
  }

  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      console.warn('[Auth] Invalid JWT format (not 3 parts)');
      return { exp: 0, iat: 0 };
    }

    // Convert URL-safe Base64 to standard Base64
    let base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    while (base64.length % 4) {
      base64 += '=';
    }

    const payload = JSON.parse(atob(base64));

    // Strict type validation
    const exp = Number(payload.exp);
    const iat = Number(payload.iat);

    return {
      exp: Number.isFinite(exp) && exp > 0 ? exp : 0,
      iat: Number.isFinite(iat) && iat > 0 ? iat : 0,
    };
  } catch (err) {
    console.error('[Auth] Failed to parse JWT:', err);
    return { exp: 0, iat: 0 };
  }
}

/**
 * Subscribe to token updates.
 * Listener is called with (exp, iat) whenever tokens are updated.
 * @returns Unsubscribe function
 */
export function addTokenUpdateListener(listener: TokenUpdateListener): () => void {
  tokenUpdateListeners.push(listener);
  // Return unsubscribe function
  return () => {
    const idx = tokenUpdateListeners.indexOf(listener);
    if (idx !== -1) {
      tokenUpdateListeners.splice(idx, 1);
    }
  };
}

/**
 * Notify all listeners with exp=0, iat=0 then clear the array.
 * Called on logout to signal token invalidation.
 */
export function clearTokenUpdateListeners(): void {
  // Notify all listeners with invalid values to signal logout
  notifyTokenUpdate(0, 0);
  // Then clear the array
  tokenUpdateListeners.length = 0;
}

/**
 * Notify all registered listeners of token update.
 */
function notifyTokenUpdate(exp: number, iat: number): void {
  // Shallow copy to prevent mutation issues if listener unsubscribes during iteration
  const listenersCopy = [...tokenUpdateListeners];
  for (const listener of listenersCopy) {
    try {
      listener(exp, iat);
    } catch (err) {
      console.error('[Auth] Token update listener error:', err);
    }
  }
}

/**
 * Get token expiry timestamp in SECONDS (Unix timestamp).
 */
export function getTokenExpiry(): number {
  return tokenExpiresAt;
}

/**
 * Get token issued-at timestamp in SECONDS (Unix timestamp).
 */
export function getTokenIssuedAt(): number {
  return tokenIssuedAt;
}

// Initialize API auth connection (call immediately)
if (typeof window !== 'undefined') {
  api.initApiAuth(getAccessToken, getRefreshToken, updateTokens, logout);
}

// Getters
export function getAuthState() {
  return authState;
}

export function getAccessToken() {
  return authState.accessToken;
}

export function getRefreshToken() {
  return authState.refreshToken;
}

export function isAuthenticated() {
  // User is only authenticated if we have tokens AND a valid user loaded
  return authState.isAuthenticated && authState.user !== null;
}

export function getCurrentUser() {
  return authState.user;
}

export function isAdmin() {
  return authState.user?.is_admin ?? false;
}

// Initialize from storage (call this on app startup)
// In Desktop (Tauri/Electron): loads from OS keyring (persisted across sessions)
// In Web: loads from sessionStorage (cleared on browser close for E2E security)
export async function initAuth() {
  // Desktop: try to load from secure storage first
  if (isDesktop()) {
    try {
      desktopBridge = await getDesktopBridge();
      const tokens = await desktopBridge.loadAuthTokens(getServerUrl());

      if (tokens) {
        authState.accessToken = tokens.access_token;
        authState.refreshToken = tokens.refresh_token;

        // Parse JWT Claims for token-refresh store
        const claims = parseJWTClaims(tokens.access_token);
        if (claims.exp > 0) {
          tokenExpiresAt = claims.exp;
          tokenIssuedAt = claims.iat;
          // Notify listeners for initial state
          notifyTokenUpdate(claims.exp, claims.iat);
        }

        // Try to load user info
        try {
          await loadCurrentUser();
          authState.isAuthenticated = true;
          console.log(`[AUTH] Restored session from ${desktopBridge.platform} secure storage`);
          return;
        } catch (err) {
          console.warn('Failed to load user from desktop tokens, clearing:', err);
          await clearDesktopTokens();
        }
      }
    } catch (err) {
      console.error('Failed to load tokens from desktop storage:', err);
    }
  }

  // SEC-006: Clean up deprecated sessionStorage tokens (one-time migration)
  if (typeof window !== 'undefined') {
    // Clear old localStorage tokens if present
    if (localStorage) {
      const hasOldTokens =
        localStorage.getItem(ACCESS_TOKEN_KEY) || localStorage.getItem(REFRESH_TOKEN_KEY);
      if (hasOldTokens) {
        console.log('[AUTH] Cleaning up old localStorage tokens...');
        localStorage.removeItem(ACCESS_TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
      }
    }

    // Clear sessionStorage tokens (deprecated as of SEC-006 fix)
    if (sessionStorage) {
      const hasSessionTokens =
        sessionStorage.getItem(ACCESS_TOKEN_KEY) || sessionStorage.getItem(REFRESH_TOKEN_KEY);
      if (hasSessionTokens) {
        console.log('[AUTH] Cleaning up deprecated sessionStorage tokens...');
        sessionStorage.removeItem(ACCESS_TOKEN_KEY);
        sessionStorage.removeItem(REFRESH_TOKEN_KEY);
      }
    }

    // SEC-006: Authentication now relies on HttpOnly cookies
    // FIX: First try token refresh (Refresh Token Cookie valid 30 days),
    // then load user. This handles the case where Access Token Cookie (15 min)
    // is expired but Refresh Token Cookie is still valid after browser restart.
    try {
      console.log('[AUTH] Attempting token refresh to restore session...');
      const result = await api.refreshTokenViaCookie();

      if (result.success) {
        authState.accessToken = result.tokens.access_token;
        authState.refreshToken = result.tokens.refresh_token;

        // Parse JWT Claims for token-refresh store
        const claims = parseJWTClaims(result.tokens.access_token);
        if (claims.exp > 0) {
          tokenExpiresAt = claims.exp;
          tokenIssuedAt = claims.iat;

          // Persist expiry timestamps to sessionStorage
          sessionStorage.setItem(TOKEN_EXPIRY_KEY, String(claims.exp));
          if (claims.iat > 0) {
            sessionStorage.setItem(TOKEN_ISSUED_KEY, String(claims.iat));
          }

          // Notify listeners for initial state
          notifyTokenUpdate(claims.exp, claims.iat);
        }

        // Now load user info with the fresh tokens
        await loadCurrentUser();
        authState.isAuthenticated = true;
        console.log('[AUTH] Session restored via token refresh');
      } else if (result.reason === 'auth_error') {
        // Token definitiv abgelaufen oder ungültig
        console.log('[AUTH] No valid session (token expired or revoked)');
      } else {
        // network_error, server_error, timeout - temporäres Problem
        console.warn(`[AUTH] Session restore failed: ${result.reason}`);
        // Bei Netzwerkfehlern: Retry wenn wieder online
        if (typeof window !== 'undefined' && result.reason === 'network_error') {
          window.addEventListener('online', () => initAuth(), { once: true });
          console.log('[AUTH] Will retry session restore when back online');
        }
      }
    } catch (err) {
      // Unexpected error - should not happen with RefreshResult
      console.error('[AUTH] Unexpected error during session restore:', err);
    }
  }
}

// Helper to clear desktop tokens
async function clearDesktopTokens(): Promise<void> {
  if (!isDesktop() || !desktopBridge) return;
  try {
    await desktopBridge.deleteAuthTokens(getServerUrl());
  } catch (err) {
    console.error('Failed to clear desktop tokens:', err);
  }
}

// Set auth tokens and user (called after login/register)
export async function setAuth(accessToken: string, refreshToken: string, user: User) {
  authState.user = user;
  authState.accessToken = accessToken;
  authState.refreshToken = refreshToken;
  authState.isAuthenticated = true;

  // Parse exp + iat from Access Token for proactive refresh
  const claims = parseJWTClaims(accessToken);
  // Only update timestamps if parsing succeeded
  if (claims.exp > 0) {
    tokenExpiresAt = claims.exp;
    tokenIssuedAt = claims.iat;

    // Persist expiry timestamps to sessionStorage (NOT sensitive - just timestamps)
    // This allows token-refresh to restore after page reload
    if (typeof sessionStorage !== 'undefined') {
      sessionStorage.setItem(TOKEN_EXPIRY_KEY, String(claims.exp));
      if (claims.iat > 0) {
        sessionStorage.setItem(TOKEN_ISSUED_KEY, String(claims.iat));
      }
    }

    // Notify listeners (important for Login after +layout mount!)
    notifyTokenUpdate(claims.exp, claims.iat);
  }

  // SEC-006: Persist tokens only in desktop secure storage (not sessionStorage)
  if (isDesktop()) {
    // Desktop: persist to secure storage (OS keyring)
    try {
      if (!desktopBridge) {
        desktopBridge = await getDesktopBridge();
      }
      await desktopBridge.storeAuthTokens(getServerUrl(), {
        access_token: accessToken,
        refresh_token: refreshToken,
        user_id: user.id,
      });
    } catch (err) {
      console.error('Failed to persist tokens to desktop storage:', err);
      // Desktop apps will need to re-login on next start if this fails
    }
  }
  // Web: Tokens are ONLY kept in memory (not persisted to sessionStorage)
  // Authentication relies on HttpOnly cookies set by the backend
}

// Update access and refresh tokens (called after token refresh)
// SEC-006: Only updates in-memory state, not sessionStorage
export function updateTokens(accessToken: string, refreshToken: string) {
  // SECURITY: Don't update tokens if user explicitly logged out.
  // Check both !isAuthenticated AND no existing tokens to distinguish
  // "logged out" from "initializing" (during initAuth, isAuthenticated
  // is false but tokens are being set up).
  if (!authState.isAuthenticated && !authState.accessToken && !authState.refreshToken) {
    console.warn('[Auth] Ignoring token update after logout');
    return;
  }

  authState.accessToken = accessToken;
  authState.refreshToken = refreshToken;

  // Parse JWT claims and notify listeners for proactive refresh
  const claims = parseJWTClaims(accessToken);
  if (claims.exp > 0) {
    tokenExpiresAt = claims.exp;
    tokenIssuedAt = claims.iat;
    console.log('[Auth] Token timestamps updated');

    // Persist expiry timestamps to sessionStorage
    if (typeof sessionStorage !== 'undefined') {
      sessionStorage.setItem(TOKEN_EXPIRY_KEY, String(claims.exp));
      if (claims.iat > 0) {
        sessionStorage.setItem(TOKEN_ISSUED_KEY, String(claims.iat));
      }
    }

    // Notify all listeners
    notifyTokenUpdate(claims.exp, claims.iat);
  } else {
    console.error('[Auth] Failed to parse JWT in updateTokens, keeping previous values');
  }

  // Tokens are NOT persisted to sessionStorage (SEC-006)
  // HttpOnly cookies are the source of truth for authentication
}

// Logout - clear all auth state
// SEC-006: Clears in-memory state only (cookies are cleared by backend)
export function logout() {
  authState.user = null;
  authState.accessToken = null;
  authState.refreshToken = null;
  authState.isAuthenticated = false;

  // Clear token expiry timestamps
  tokenExpiresAt = 0;
  tokenIssuedAt = 0;

  // Clear persisted expiry from sessionStorage
  if (typeof sessionStorage !== 'undefined') {
    sessionStorage.removeItem(TOKEN_EXPIRY_KEY);
    sessionStorage.removeItem(TOKEN_ISSUED_KEY);
  }

  // Notify listeners with exp=0, iat=0 then clear
  // Listener in +layout.svelte detects exp=0 and calls tokenRefresh.stop()
  clearTokenUpdateListeners();

  // Clear encryption KEK from memory
  encryption.lockEncryption();

  // CRITICAL FIX: Reset stores to prevent User A's settings from leaking to User B
  try {
    settings.resetSettings();
  } catch (err) {
    console.error('[AUTH] Failed to reset settings:', err);
  }

  try {
    ui.resetToDefaults();
  } catch (err) {
    console.error('[AUTH] Failed to reset UI:', err);
  }

  try {
    features.resetJournalFeature();
    journal.resetJournalState();
    features.resetRecipeFeature();
    recipes.resetRecipeState();
  } catch (err) {
    console.error('[AUTH] Failed to reset feature state:', err);
  }

  // Clear desktop tokens (async, fire-and-forget)
  if (isDesktop()) {
    clearDesktopTokens().catch(console.error);
  }

  // SEC-006: No sessionStorage cleanup needed
}

// Load current user info from backend (private helper)
async function loadCurrentUser() {
  const user = await api.getCurrentUser();
  authState.user = user;
}

// Register a new user
export async function register(
  username: string,
  email: string,
  password: string,
  captchaToken?: string
): Promise<void> {
  await api.register(username, email, password, captchaToken);
  // User is not automatically logged in after registration
  // They need to log in manually at /login
}

// Login with username/email and password
export async function login(
  usernameOrEmail: string,
  password: string,
  captchaToken?: string,
  totpCode?: string,
  backupCode?: string
): Promise<{
  requiresTwoFactor: boolean;
  twoFactorMethods?: string[];
  pendingLoginToken?: string;
}> {
  const response = await api.login(usernameOrEmail, password, captchaToken, totpCode, backupCode);

  if (response.requires_two_factor) {
    return {
      requiresTwoFactor: true,
      twoFactorMethods: response.two_factor_methods,
      pendingLoginToken: response.pending_login_token,
    };
  }

  if (!response.access_token || !response.refresh_token || !response.user) {
    throw new Error('Login fehlgeschlagen: Keine gültige Antwort vom Server');
  }

  await setAuth(response.access_token, response.refresh_token, response.user);

  // Setup E2E encryption with password + salt
  if (response.encryption_salt) {
    try {
      const salt = fromBase64Standard(response.encryption_salt);
      await encryption.setupEncryption(password, response.user.id, salt);
    } catch (_error) {
      console.error('[AUTH] Failed to setup encryption');
      // Don't fail login if encryption setup fails
    }
  }

  return { requiresTwoFactor: false };
}

// Logout - revoke refresh token and clear state
export async function logoutAsync(): Promise<void> {
  // IMPORTANT: Capture refresh token BEFORE clearing state
  const refreshToken = authState.refreshToken;

  // ✅ NEW: Clear Service Worker caches to prevent data leakage
  // CRITICAL: Browser-only guard prevents SSR errors
  if (typeof window !== 'undefined' && 'caches' in window) {
    try {
      const cacheNames = await caches.keys();
      // Clear API caches AND uploads cache, keep workbox-precache for offline login screen
      await Promise.all(
        cacheNames
          .filter((name) => name.startsWith('api-') || name === 'uploads')
          .map((name) => {
            console.log('[Logout] Clearing cache:', name);
            return caches.delete(name);
          })
      );

      console.log('[Logout] API + uploads caches cleared (login screen still available offline)');
    } catch (err) {
      console.warn('[Logout] Failed to clear caches:', err);
      // Non-critical, continue with logout
    }
  }

  // Clear local state (after cache purge, before API call)
  logout();

  // Try to revoke token on backend (best effort)
  // Uses the token captured BEFORE logout() cleared the state
  if (refreshToken) {
    try {
      await api.logoutApi(refreshToken);
    } catch (err) {
      // Ignore errors - we're already logged out locally
      console.error('Failed to revoke refresh token:', err);
    }
  }
}
