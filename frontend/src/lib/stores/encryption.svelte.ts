import * as api from '$lib/api';
import { isDesktop } from '$lib/config';
import { DecryptionError, e2eEncryption, type EncryptedPayload } from '$lib/crypto/e2e';
import {
  clearPersistedKEK,
  loadPersistedKEK,
  persistKEK,
  type SecurityLevel,
} from '$lib/crypto/kek-persistence';
import { type DesktopBridge, getDesktopBridge } from '$lib/desktop';
import { error as showError, warning } from '$lib/stores/toast.svelte';

// Re-export DecryptionError for UI use
export { DecryptionError };
export type { SecurityLevel };

// Desktop bridge instance (lazy initialized)
let desktopBridge: DesktopBridge | null = null;

// Encryption state
let isUnlocked = $state(false);
let userID = $state<number | null>(null);
let userSalt = $state<Uint8Array | null>(null);

// Security preferences
let securityLevel = $state<SecurityLevel>('balanced');

// Settings
export interface EncryptionSettings {
  encryptTitles: boolean; // Optional title encryption
  extractKeywords: boolean; // Opt-in for keyword search
}

let settings = $state<EncryptionSettings>({
  encryptTitles: false,
  extractKeywords: false,
});

/**
 * Setup encryption on login.
 * Fetches salt from server and derives KEK using libsodium (in Web Worker).
 *
 * This function should be called immediately after successful login,
 * before the user can interact with any encrypted notes.
 *
 * @param password - User's password (cleared from memory after use)
 * @param userId - User's ID
 * @param salt - Encryption salt (16 bytes, stored server-side)
 * @param secLevel - Security level (default: balanced)
 * @param skipPersistence - Skip KEK persistence (for paranoid mode or errors)
 */
export async function setupEncryption(
  password: string,
  userId: number,
  salt: Uint8Array,
  secLevel: SecurityLevel = 'balanced',
  skipPersistence: boolean = false
): Promise<void> {
  userID = userId;
  userSalt = salt;

  // Derive KEK from password (Argon2id in Web Worker - non-blocking)
  await e2eEncryption.setupKEK(password, salt);

  isUnlocked = true;

  // Store KEK in desktop secure memory for defense-in-depth
  if (isDesktop()) {
    try {
      if (!desktopBridge) {
        desktopBridge = await getDesktopBridge();
      }
      const kek = e2eEncryption.exportKEK();
      await desktopBridge.storeKek(kek);
      // KEK stored in desktop secure memory
    } catch (err) {
      console.error('[ENCRYPTION] Failed to store KEK in desktop memory:', err);
      // Non-fatal: JavaScript memory still has KEK
    }
  }

  // Persist KEK if enabled (for session restoration)
  if (!skipPersistence && secLevel !== 'paranoid') {
    try {
      const kek = e2eEncryption.exportKEK();
      await persistKEK(userId, kek, secLevel);
      // KEK persisted successfully
    } catch (err) {
      console.error('Failed to persist KEK:', err);
      warning('KEK konnte nicht gespeichert werden - Paranoid-Modus aktiv');
    }
  }

  // Build client-side search index for encrypted notes (fire-and-forget, dynamic import to avoid circular dep)
  import('$lib/stores/search-index.svelte').then((m) => m.buildIndex());
}

/**
 * Lock encryption (on logout or timeout).
 * Clears KEK from memory, requiring re-authentication to access encrypted content.
 *
 * IMPORTANT: IndexedDB is NOT cleared (only memory).
 * To clear IndexedDB, call clearPersistedKEK() separately (logout or paranoid switch).
 */
export function lockEncryption(): void {
  // Destroy search index synchronously via cached module (dynamic import to avoid circular dep)
  // Module is already loaded at this point since buildIndex was called earlier
  import('$lib/stores/search-index.svelte').then((m) => m.destroyIndex());
  e2eEncryption.clearKEK();
  isUnlocked = false;

  // Clear KEK from desktop memory (async, fire-and-forget)
  if (isDesktop() && desktopBridge) {
    desktopBridge.lockKek().catch((err) => {
      console.error('[ENCRYPTION] Failed to clear KEK from desktop memory:', err);
    });
  }

  // IMPORTANT: userID and userSalt remain for KEK restoration
  // Only clear on full logout
}

/**
 * Check if encryption is unlocked.
 *
 * @returns true if KEK is available in memory
 */
export function isEncryptionUnlocked(): boolean {
  return isUnlocked;
}

/**
 * Encrypt note with current settings.
 *
 * This function applies user preferences for title encryption and keyword extraction.
 * The encrypted content and wrapped DEK are returned for storage in the backend.
 *
 * @param title - Plaintext title
 * @param content - Plaintext content
 * @returns Encrypted data including optional title and keywords
 * @throws Error if encryption is locked
 */
export function encryptNote(
  title: string,
  content: string
): {
  encryptedTitle: string | null;
  encryptedContent: EncryptedPayload;
  keywords: string[];
} {
  if (!isUnlocked) throw new Error('Encryption locked - please re-login');

  // Encrypt content (always)
  const encryptedContent = e2eEncryption.encryptNote(content);

  // Encrypt title (optional)
  const encryptedTitle = settings.encryptTitles ? e2eEncryption.encryptTitle(title) : null;

  // Extract keywords (opt-in with warning)
  const keywords = settings.extractKeywords ? e2eEncryption.extractKeywords(content) : [];

  return { encryptedTitle, encryptedContent, keywords };
}

/**
 * Encrypt a short text (task description) with its own DEK.
 * Uses the same primitive as encryptNote but without title encryption or keyword extraction.
 */
export function encryptTaskText(text: string): EncryptedPayload {
  if (!isUnlocked) throw new Error('Encryption locked - please re-login');
  return e2eEncryption.encryptNote(text);
}

/**
 * Decrypt note.
 *
 * @param encryptedTitle - Encrypted title (JSON string) or null
 * @param encryptedContent - Encrypted payload with metadata
 * @returns Decrypted title and content
 * @throws DecryptionError if encryption is locked or decryption fails
 */
export function decryptNote(
  encryptedTitle: string | null,
  encryptedContent: EncryptedPayload
): { title: string | null; content: string } {
  if (!isUnlocked) throw new DecryptionError('NOT_INITIALIZED');

  const content = e2eEncryption.decryptNote(encryptedContent);
  const title = encryptedTitle ? e2eEncryption.decryptTitle(encryptedTitle) : null;

  return { title, content };
}

/**
 * Decrypt only the title of an encrypted note.
 *
 * @param encryptedTitle - Encrypted title JSON string
 * @returns Decrypted plaintext title, or null if encryption is locked
 */
export function decryptTitle(encryptedTitle: string): string | null {
  if (!isUnlocked) return null;
  try {
    return e2eEncryption.decryptTitle(encryptedTitle);
  } catch {
    return null;
  }
}

/**
 * Update encryption settings.
 * Changes are applied immediately to new encryptions and persisted to backend.
 *
 * @param newSettings - Partial settings to update
 */
export async function updateSettings(newSettings: Partial<EncryptionSettings>): Promise<void> {
  settings = { ...settings, ...newSettings };

  try {
    await api.updateEncryptionPreferences({
      keywords_enabled: settings.extractKeywords,
      encrypt_titles: settings.encryptTitles,
    });
  } catch (err) {
    console.error('[ENCRYPTION] Failed to persist settings to backend:', err);
    warning('Failed to save encryption settings to server');
  }
}

/**
 * Initialize encryption settings from server preferences (no API call).
 * Called by settings store after loading preferences.
 */
export function initSettingsFromPreferences(
  keywordsEnabled: boolean,
  encryptTitles: boolean
): void {
  settings = { ...settings, extractKeywords: keywordsEnabled, encryptTitles };
}

/**
 * Check if keyword warning should be shown.
 *
 * @returns true if keywords are enabled (data leakage risk)
 */
export function showKeywordWarning(): boolean {
  return settings.extractKeywords;
}

/**
 * Get current settings (read-only).
 */
export function getSettings(): EncryptionSettings {
  return { ...settings };
}

/**
 * Try to restore KEK from IndexedDB on page load.
 * FIX: Explicit cleanup on corruption.
 *
 * @param userId - User ID
 * @returns true if KEK was restored, false otherwise
 */
export async function tryRestoreKEK(userId: number): Promise<boolean> {
  try {
    const kek = await loadPersistedKEK(userId);
    if (!kek) return false;

    // Restore to memory
    e2eEncryption.importKEK(kek);
    isUnlocked = true;
    userID = userId;

    // Build client-side search index for encrypted notes (fire-and-forget, dynamic import to avoid circular dep)
    import('$lib/stores/search-index.svelte').then((m) => m.buildIndex());

    return true;
  } catch (err) {
    console.error('Failed to restore KEK (corrupt?), clearing storage:', err);

    // FIX: Explicit cleanup prevents stuck states
    await clearPersistedKEK(userId);

    warning('Verschlüsselungsschlüssel beschädigt - bitte Passwort erneut eingeben');
    return false;
  }
}

/**
 * Update security level preference.
 * FIX: Clears timers and credentials when switching to paranoid.
 *
 * @param level - New security level
 */
export async function updateSecurityLevel(level: SecurityLevel): Promise<void> {
  const oldLevel = securityLevel;
  securityLevel = level;

  if (level === 'paranoid') {
    // Clear IndexedDB
    if (userID) {
      await clearPersistedKEK(userID);
    }

    // FIX: Clear WebAuthn state (handled in settings component)
  } else if (isUnlocked && userID) {
    // Re-persist with new level
    try {
      const kek = e2eEncryption.exportKEK();
      await persistKEK(userID, kek, level);
    } catch (err) {
      console.error('Failed to re-persist KEK:', err);
      showError('Fehler beim Aktualisieren der Sicherheitsstufe');
      securityLevel = oldLevel; // Rollback
      throw err;
    }
  }

  // Save to backend
  try {
    await api.updateSecurityPreferences({ security_level: level });
  } catch (err) {
    console.error('Failed to update security level on backend:', err);
    // Rollback on failure
    securityLevel = oldLevel;
    throw err;
  }
}

/**
 * Set security level (for internal state updates).
 * Used by kek-persistence failure handler.
 */
export function setSecurityLevel(level: SecurityLevel): void {
  securityLevel = level;
}

/**
 * Get current security level.
 */
export function getSecurityLevel(): SecurityLevel {
  return securityLevel;
}

/**
 * Get current user ID.
 */
export function getUserID(): number | null {
  return userID;
}

/**
 * Get current user salt.
 */
export function getUserSalt(): Uint8Array | null {
  return userSalt;
}

/**
 * Set encryption unlocked state (for internal use).
 */
export function setIsUnlocked(unlocked: boolean): void {
  isUnlocked = unlocked;
}

/**
 * Full logout - clears all encryption state including IndexedDB.
 */
export async function logout(): Promise<void> {
  // Destroy search index (dynamic import to avoid circular dep)
  import('$lib/stores/search-index.svelte').then((m) => m.destroyIndex());

  // Clear IndexedDB
  if (userID) {
    await clearPersistedKEK(userID);
  }

  // Clear memory
  e2eEncryption.clearKEK();
  isUnlocked = false;
  userID = null;
  userSalt = null;

  // Clear desktop KEK (async, fire-and-forget)
  if (isDesktop() && desktopBridge) {
    desktopBridge.lockKek().catch((err) => {
      console.error('[ENCRYPTION] Failed to clear KEK from desktop memory:', err);
    });
  }

  // TODO: Stop auto-lock timer (Phase 2)
}
