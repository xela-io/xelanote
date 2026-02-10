/**
 * Auto-Lock Timer Store
 *
 * Automatically locks encryption after a configured period of inactivity.
 * Features:
 * - Activity tracking (mouse, keyboard, touch events)
 * - Postponement for unsaved changes (max 3 times)
 * - Force-lock after max postponements
 * - Integration with encryption store
 *
 * Phase 2 of Hybrid KEK Persistence implementation.
 */

import { SvelteDate } from 'svelte/reactivity';
import * as encryption from './encryption.svelte';
import * as notes from './notes.svelte';
import { warning } from './toast.svelte';

// Timer state
let lockTimer = $state<ReturnType<typeof setTimeout> | null>(null);
let postponeCount = $state(0);
let timeoutMinutes = $state(0); // 0 = disabled
let lastActivity: Date = new SvelteDate();

// Constants
const MAX_POSTPONEMENTS = 3;
const POSTPONE_DURATION_MS = 5 * 60 * 1000; // 5 minutes extra time

/**
 * Initialize auto-lock with specified timeout.
 *
 * @param minutes - Lock timeout in minutes (0 = disabled, "never")
 */
export function initAutoLock(minutes: number): void {
  // Stop existing timer
  stopAutoLock();

  // Store timeout setting
  timeoutMinutes = minutes;

  // Don't start timer if disabled or encryption not unlocked
  if (minutes === 0 || !encryption.isEncryptionUnlocked()) {
    return;
  }

  // Reset state
  postponeCount = 0;
  lastActivity = new SvelteDate();

  // Start timer
  startTimer();
}

/**
 * Stop auto-lock timer.
 * Called on logout, manual lock, or security level change to paranoid.
 */
export function stopAutoLock(): void {
  if (lockTimer) {
    clearTimeout(lockTimer);
    lockTimer = null;
  }
  postponeCount = 0;
}

/**
 * Record user activity (called by global event listeners).
 * Resets the timer if running.
 */
export function recordActivity(): void {
  if (timeoutMinutes === 0 || !encryption.isEncryptionUnlocked()) {
    return;
  }

  lastActivity = new SvelteDate();

  // Restart timer
  if (lockTimer) {
    clearTimeout(lockTimer);
    startTimer();
  }
}

/**
 * Get time until lock (in seconds).
 * Used for UI display (optional).
 *
 * @returns Seconds until lock, or -1 if timer not active
 */
export function getTimeUntilLock(): number {
  if (timeoutMinutes === 0 || !lockTimer) {
    return -1;
  }

  const elapsed = Date.now() - lastActivity.getTime();
  const remaining = timeoutMinutes * 60 * 1000 - elapsed;

  return Math.max(0, Math.floor(remaining / 1000));
}

/**
 * Lock encryption immediately (bypasses postponement).
 * Called manually or after max postponements.
 */
export function lockEncryption(): void {
  stopAutoLock();
  encryption.lockEncryption();
}

/**
 * Start the lock timer.
 * Internal function - checks for unsaved changes and postpones if needed.
 */
function startTimer(): void {
  if (timeoutMinutes === 0) return;

  lockTimer = setTimeout(
    () => {
      attemptLock();
    },
    timeoutMinutes * 60 * 1000
  );
}

/**
 * Attempt to lock encryption.
 * Checks for unsaved changes and postpones up to MAX_POSTPONEMENTS times.
 */
function attemptLock(): void {
  // Check for unsaved changes
  const hasUnsavedChanges = notes.getIsDirty();

  if (hasUnsavedChanges && postponeCount < MAX_POSTPONEMENTS) {
    // Postpone lock
    postponeCount++;
    const remainingPostpones = MAX_POSTPONEMENTS - postponeCount;

    warning(
      `Auto-Lock verschoben (ungespeicherte Änderungen) - ` + `noch ${remainingPostpones}x möglich`
    );

    // Give user 5 more minutes to save
    lockTimer = setTimeout(() => {
      attemptLock();
    }, POSTPONE_DURATION_MS);

    return;
  }

  // Force lock after max postponements or no unsaved changes
  if (hasUnsavedChanges && postponeCount >= MAX_POSTPONEMENTS) {
    warning('Auto-Lock erzwungen nach 3 Verschiebungen - bitte speichern Sie Ihre Arbeit!');
  }

  lockEncryption();
}

/**
 * Check if auto-lock is active.
 *
 * @returns true if timer is running
 */
export function isAutoLockActive(): boolean {
  return lockTimer !== null && timeoutMinutes > 0;
}

/**
 * Get current timeout setting.
 *
 * @returns Timeout in minutes (0 = disabled)
 */
export function getTimeoutMinutes(): number {
  return timeoutMinutes;
}

/**
 * Get postpone count (for debugging/UI).
 *
 * @returns Number of postponements used
 */
export function getPostponeCount(): number {
  return postponeCount;
}
