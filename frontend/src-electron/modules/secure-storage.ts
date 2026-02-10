/**
 * Secure Token Storage
 *
 * Uses electron.safeStorage for OS-native encryption of auth tokens.
 * Tokens are stored as encrypted files in the app's userData directory.
 */

import { app,safeStorage } from 'electron';
import { existsSync, mkdirSync,readFileSync, unlinkSync, writeFileSync } from 'fs';
import { join } from 'path';

// Type for auth tokens
interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user_id: number | null;
}

/**
 * Get the path for storing tokens for a specific server.
 */
function getTokenPath(serverUrl: string): string {
  // Sanitize server URL for use as filename
  const key = serverUrl.replace(/[/:]/g, '_').replace(/_{2,}/g, '_');
  return join(app.getPath('userData'), 'tokens', `${key}.tokens.enc`);
}

/**
 * Ensure the tokens directory exists.
 */
function ensureTokenDir(): void {
  const dir = join(app.getPath('userData'), 'tokens');
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
}

/**
 * Store auth tokens securely using OS-native encryption.
 *
 * @param serverUrl - Server URL (used as identifier)
 * @param tokens - Auth tokens to store
 * @throws Error if encryption is not available
 */
export async function storeTokens(serverUrl: string, tokens: AuthTokens): Promise<void> {
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error('Secure storage encryption not available on this system');
  }

  ensureTokenDir();

  const json = JSON.stringify(tokens);
  const encrypted = safeStorage.encryptString(json);

  writeFileSync(getTokenPath(serverUrl), encrypted);
  console.log(`[SecureStorage] Tokens stored for ${serverUrl}`);
}

/**
 * Load auth tokens from secure storage.
 *
 * @param serverUrl - Server URL (used as identifier)
 * @returns Auth tokens or null if not found
 */
export async function loadTokens(serverUrl: string): Promise<AuthTokens | null> {
  const path = getTokenPath(serverUrl);

  if (!existsSync(path)) {
    console.log(`[SecureStorage] No tokens found for ${serverUrl}`);
    return null;
  }

  if (!safeStorage.isEncryptionAvailable()) {
    console.error('[SecureStorage] Encryption not available, cannot decrypt tokens');
    return null;
  }

  try {
    const encrypted = readFileSync(path);
    const json = safeStorage.decryptString(encrypted);
    const tokens = JSON.parse(json) as AuthTokens;
    console.log(`[SecureStorage] Tokens loaded for ${serverUrl}`);
    return tokens;
  } catch (err) {
    console.error(`[SecureStorage] Failed to load tokens for ${serverUrl}:`, err);
    // Delete corrupted file
    try {
      unlinkSync(path);
    } catch {
      // Ignore deletion errors
    }
    return null;
  }
}

/**
 * Delete stored tokens for a server.
 *
 * @param serverUrl - Server URL (used as identifier)
 */
export async function deleteTokens(serverUrl: string): Promise<void> {
  const path = getTokenPath(serverUrl);

  if (existsSync(path)) {
    try {
      unlinkSync(path);
      console.log(`[SecureStorage] Tokens deleted for ${serverUrl}`);
    } catch (err) {
      console.error(`[SecureStorage] Failed to delete tokens for ${serverUrl}:`, err);
    }
  }
}

/**
 * Check if secure storage is available.
 */
export function isSecureStorageAvailable(): boolean {
  return safeStorage.isEncryptionAvailable();
}
