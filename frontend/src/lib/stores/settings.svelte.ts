/**
 * Settings Store - Manages user preferences with server sync
 *
 * Handles theme, editor mode, and account settings.
 * Includes localStorage migration for first-time server sync.
 */

import * as api from '$lib/api';
import { FEATURE_FLAGS } from '$lib/config';
import { fromBase64Standard } from '$lib/crypto/sodium';
import * as autoLock from '$lib/stores/auto-lock.svelte';
import * as encryption from '$lib/stores/encryption.svelte';
import * as toast from '$lib/stores/toast.svelte';
import * as ui from '$lib/stores/ui.svelte';
import { isValidThemeId, type ThemeId } from '$lib/themes';

// State
let isLoading = $state(false);
let preferencesLoaded = $state(false);
let error = $state<string | null>(null);

// Action-specific loading states
let isSavingPreferences = $state(false);
let isChangingEmail = $state(false);
let isChangingPassword = $state(false);

// Virtual Tree Preference (local only, no server sync)
const VIRTUAL_TREE_KEY = 'xelanote_virtual_tree_enabled';
const EDITOR_MODE_KEY = 'xelanote-editor-mode';
let virtualTreeEnabled = $state<boolean>(false);

type EditorMode = 'edit' | 'preview' | 'split' | 'live';

// Getters
export function getIsLoading() {
  return isLoading;
}

export function getPreferencesLoaded() {
  return preferencesLoaded;
}

export function getError() {
  return error;
}

export function getIsSavingPreferences() {
  return isSavingPreferences;
}

export function getIsChangingEmail() {
  return isChangingEmail;
}

export function getIsChangingPassword() {
  return isChangingPassword;
}

export function getVirtualTreeEnabled(): boolean {
  return virtualTreeEnabled;
}

export function setVirtualTreeEnabled(enabled: boolean): void {
  virtualTreeEnabled = enabled;
  try {
    localStorage.setItem(VIRTUAL_TREE_KEY, JSON.stringify(enabled));
  } catch (e) {
    console.error('[SETTINGS] Failed to save virtual tree preference:', e);
  }
}

/**
 * Load virtual tree preference from localStorage (local only, no server sync).
 * Should be called on app initialization.
 */
export function loadVirtualTreePreference(): void {
  try {
    const stored = localStorage.getItem(VIRTUAL_TREE_KEY);
    if (stored !== null) {
      const parsed = parseBooleanPreference(stored);
      if (parsed !== null) {
        virtualTreeEnabled = parsed;
      }
    }
  } catch (e) {
    console.error('[SETTINGS] Failed to load virtual tree preference:', e);
    virtualTreeEnabled = false; // Default: disabled
  }
}

function parseBooleanPreference(raw: string): boolean | null {
  try {
    const parsed = JSON.parse(raw);
    return typeof parsed === 'boolean' ? parsed : null;
  } catch {
    return null;
  }
}

function parseEditorModePreference(raw: string): EditorMode | null {
  if (raw === 'edit' || raw === 'preview' || raw === 'split' || raw === 'live') return raw;
  return null;
}

function readLocalEditorMode(): EditorMode | null {
  try {
    const stored = localStorage.getItem(EDITOR_MODE_KEY);
    return stored ? parseEditorModePreference(stored) : null;
  } catch {
    return null;
  }
}

function writeLocalEditorMode(mode: EditorMode): void {
  try {
    localStorage.setItem(EDITOR_MODE_KEY, mode);
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

/**
 * Load preferences from server (theme + editor mode + security level).
 * This function loads ALL preferences in one API call.
 */
export async function loadPreferences(): Promise<void> {
  if (isLoading || preferencesLoaded) return;

  isLoading = true;
  error = null;

  try {
    const prefs = await api.getPreferences();

    // Apply server preferences to UI (theme + editor mode)
    if (isValidThemeId(prefs.theme)) {
      ui.setTheme(prefs.theme as ThemeId);
    }
    // Backend currently persists `live` as `split`; restore the intended UI mode.
    const localEditorMode = readLocalEditorMode();
    const serverMode = parseEditorModePreference(prefs.editor_mode) ?? 'edit';
    let resolvedMode: EditorMode;
    if (!FEATURE_FLAGS.livePreview && serverMode === 'live') {
      resolvedMode = 'edit';
    } else if (FEATURE_FLAGS.livePreview && serverMode === 'split') {
      resolvedMode = localEditorMode ?? 'live';
    } else {
      resolvedMode = serverMode;
    }
    // On mobile, split mode is not supported - fall back to edit
    const effectiveMode = ui.getIsMobile() && resolvedMode === 'split' ? 'edit' : resolvedMode;
    ui.setEditorMode(effectiveMode);
    writeLocalEditorMode(effectiveMode);

    // CRITICAL FIX: Apply security preferences (same API call, no duplication)
    encryption.setSecurityLevel(prefs.security_level);

    // Apply encryption settings (keywords + title encryption)
    encryption.initSettingsFromPreferences(prefs.keywords_enabled, prefs.encrypt_titles);

    // Initialize auto-lock timer if encryption is unlocked and timeout > 0
    if (encryption.isEncryptionUnlocked() && prefs.auto_lock_timeout > 0) {
      autoLock.initAutoLock(prefs.auto_lock_timeout);
    }

    preferencesLoaded = true;
    console.log('[SETTINGS] Preferences loaded from server:', {
      theme: prefs.theme,
      editorMode: prefs.editor_mode,
      securityLevel: prefs.security_level,
      autoLockTimeout: prefs.auto_lock_timeout,
    });
  } catch (err: unknown) {
    console.error('[SETTINGS] Failed to load preferences:', err);
    error = err instanceof Error ? err.message : 'Failed to load preferences';

    // CRITICAL: Do NOT apply fallback security settings
    // Failing to load is safer than downgrading security
    // Show user notification
    toast.error('Einstellungen konnten nicht geladen werden');
  } finally {
    isLoading = false;
  }
}

/**
 * Save preferences to server and update UI.
 */
export async function savePreferences(theme: ThemeId, editorMode: EditorMode): Promise<boolean> {
  isSavingPreferences = true;
  error = null;

  try {
    const uiMode = !FEATURE_FLAGS.livePreview && editorMode === 'live' ? 'edit' : editorMode;
    // Backend compatibility: until server-side validation allows `live`,
    // persist it as `split` while keeping the client in `live` mode.
    const persistedMode = uiMode === 'live' ? 'split' : uiMode;
    await api.updatePreferences({ theme, editor_mode: persistedMode });

    // Apply to UI
    ui.setTheme(theme);
    ui.setEditorMode(uiMode);
    writeLocalEditorMode(uiMode);

    return true;
  } catch (err: unknown) {
    console.error('Failed to save preferences:', err);
    error = err instanceof Error ? err.message : 'Failed to save preferences';
    toast.error('Einstellungen konnten nicht gespeichert werden');
    return false;
  } finally {
    isSavingPreferences = false;
  }
}

/**
 * Update theme preference (convenience wrapper).
 */
export async function setThemePreference(theme: ThemeId): Promise<boolean> {
  const currentMode = ui.getEditorMode();
  return savePreferences(theme, currentMode);
}

/**
 * Update editor mode preference (convenience wrapper).
 */
export async function setEditorModePreference(mode: EditorMode): Promise<boolean> {
  const currentTheme = ui.getCurrentThemeId();
  return savePreferences(currentTheme, mode);
}

/**
 * Change user email with password verification.
 * Other sessions will be invalidated.
 */
export async function changeEmail(
  newEmail: string,
  currentPassword: string
): Promise<{ success: boolean; error?: string }> {
  isChangingEmail = true;
  error = null;

  try {
    await api.changeEmail(newEmail, currentPassword);
    toast.success('E-Mail-Adresse wurde geändert');
    return { success: true };
  } catch (err: unknown) {
    const errorMsg = err instanceof Error ? err.message : 'Failed to change email';
    error = errorMsg;

    // Map API errors to user-friendly messages
    if (errorMsg.includes('incorrect password')) {
      return { success: false, error: 'Falsches Passwort' };
    }
    if (errorMsg.includes('already in use')) {
      return { success: false, error: 'Diese E-Mail-Adresse wird bereits verwendet' };
    }
    if (errorMsg.includes('invalid email')) {
      return { success: false, error: 'Ungültiges E-Mail-Format' };
    }

    return { success: false, error: 'E-Mail konnte nicht geändert werden' };
  } finally {
    isChangingEmail = false;
  }
}

/**
 * Change user password with verification.
 * Other sessions will be invalidated.
 *
 * ✅ CRITICAL: Re-wraps KEK with new password to prevent lockout!
 */
export async function changePassword(
  currentPassword: string,
  newPassword: string
): Promise<{ success: boolean; error?: string }> {
  isChangingPassword = true;
  error = null;

  try {
    // Step 1: Update password hash on backend
    await api.changePassword(currentPassword, newPassword);

    // Step 2: Re-wrap KEK with NEW password (CRITICAL!)
    // Without this, user cannot unlock encryption after password change
    if (encryption.isEncryptionUnlocked() && encryption.getSecurityLevel() !== 'paranoid') {
      try {
        const currentUser = await api.getCurrentUser();
        if (!currentUser.encryption_salt) {
          throw new Error('No encryption salt found');
        }

        const salt = fromBase64Standard(currentUser.encryption_salt);

        // ⚠️ CRITICAL: Re-derive KEK from NEW password and persist
        // This generates a fresh IV and re-wraps the KEK in IndexedDB
        await encryption.setupEncryption(
          newPassword,
          currentUser.id,
          salt,
          encryption.getSecurityLevel(),
          false // persist = true (re-wrap with new password)
        );

        // Restart auto-lock timer after KEK re-wrap
        const autoLockTimeout = 15; // minutes
        autoLock.initAutoLock(autoLockTimeout);

        // KEK successfully re-wrapped
      } catch (kekErr) {
        console.error('Failed to re-wrap KEK after password change:', kekErr);
        toast.error(
          'Passwort geändert, aber Verschlüsselung konnte nicht aktualisiert werden. ' +
            'Bitte melden Sie sich ab und erneut an.'
        );
        // Return success because password WAS changed on backend
        // User just needs to re-login to fix KEK
        return { success: true };
      }
    }

    toast.success('Passwort wurde geändert');
    return { success: true };
  } catch (err: unknown) {
    const errorMsg = err instanceof Error ? err.message : 'Failed to change password';
    error = errorMsg;

    // Map API errors to user-friendly messages
    if (errorMsg.includes('incorrect password')) {
      return { success: false, error: 'Falsches aktuelles Passwort' };
    }
    if (errorMsg.includes('at least 8 characters')) {
      return { success: false, error: 'Passwort muss mindestens 8 Zeichen haben' };
    }

    return { success: false, error: 'Passwort konnte nicht geändert werden' };
  } finally {
    isChangingPassword = false;
  }
}

/**
 * Reset the store state (e.g., on logout).
 * Note: virtualTreeEnabled is NOT reset (persists across logins).
 */
export function resetSettings(): void {
  isLoading = false;
  preferencesLoaded = false;
  error = null;
  isSavingPreferences = false;
  isChangingEmail = false;
  isChangingPassword = false;
  // virtualTreeEnabled is NOT reset - it's a device-local preference
}

/**
 * Update security preferences (security_level and/or auto_lock_timeout).
 *
 * @param data - Partial security preferences
 * @returns true on success, false on error
 */
export async function updateSecurityPreferences(data: {
  security_level?: string;
  auto_lock_timeout?: number;
}): Promise<boolean> {
  try {
    await api.updateSecurityPreferences(data);
    return true;
  } catch (err) {
    console.error('Failed to update security preferences:', err);
    return false;
  }
}

/**
 * Add a WebAuthn credential.
 *
 * @param credentialId - Base64-encoded credential ID
 * @param deviceName - User-friendly device name
 * @returns Credential info on success, null on error
 */
export async function addWebAuthnCredential(
  credentialId: string,
  deviceName: string
): Promise<api.WebAuthnCredentialInfo | null> {
  try {
    return await api.addWebAuthnCredential(credentialId, deviceName);
  } catch (err) {
    console.error('Failed to add WebAuthn credential:', err);
    return null;
  }
}

/**
 * Delete a WebAuthn credential.
 *
 * @param credentialId - Credential ID to delete
 * @returns true on success, false on error
 */
export async function deleteWebAuthnCredential(credentialId: string): Promise<boolean> {
  try {
    await api.deleteWebAuthnCredential(credentialId);
    return true;
  } catch (err) {
    console.error('Failed to delete WebAuthn credential:', err);
    return false;
  }
}
