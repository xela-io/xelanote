/**
 * Coordinator module for logout cleanup.
 * Centralizes cross-store reset logic that was previously embedded in auth.svelte.ts.
 * This prevents the auth store from having direct dependencies on every feature store.
 */
import * as encryption from './encryption.svelte';
import * as features from './features.svelte';
import * as journal from './journal.svelte';
import * as recipes from './recipes.svelte';
import * as settings from './settings.svelte';
import * as ui from './ui.svelte';

/**
 * Reset all user-specific stores to prevent data leakage between different users.
 * Each reset is isolated in its own try-catch to ensure one failure
 * doesn't prevent the remaining stores from being cleaned up.
 */
export function resetAllStores(): void {
  // Clear encryption KEK from memory
  encryption.lockEncryption();

  // CRITICAL: Reset stores to prevent User A's settings from leaking to User B
  try {
    settings.resetSettings();
  } catch (err) {
    console.error('[Logout] Failed to reset settings:', err);
  }

  try {
    ui.resetToDefaults();
  } catch (err) {
    console.error('[Logout] Failed to reset UI:', err);
  }

  try {
    features.resetJournalFeature();
    journal.resetJournalState();
    features.resetRecipeFeature();
    recipes.resetRecipeState();
    features.resetCanvasFeature();
  } catch (err) {
    console.error('[Logout] Failed to reset feature state:', err);
  }
}
