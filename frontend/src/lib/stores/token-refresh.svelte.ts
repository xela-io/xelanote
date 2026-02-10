/**
 * Token Refresh Store
 *
 * Proaktiver Token-Refresh-Mechanismus, der Access Tokens bereits VOR Ablauf erneuert.
 * Pattern nach auto-lock.svelte.ts mit Activity- und Visibility-Tracking.
 *
 * Features:
 * - Proaktiver Refresh bei 80% der Token-Lebensdauer (default: 12 Min bei 15 Min Token)
 * - Idle-Detection: Pausiert bei >10 Min Inaktivität
 * - Visibility-Detection: Pausiert bei hidden Tab
 * - Sofortiger Refresh bei visibility wenn Token abgelaufen
 * - Hybrid-Ansatz: Proaktiv + reaktiver 401-Handler als Fallback
 *
 * HARD REQUIREMENT: Access Token MUSS JWT mit gültigem `exp` Claim sein.
 * Optional: `iat` Claim für präzisere Berechnung (Fallback: 3 Min Buffer)
 */

import { SvelteDate } from 'svelte/reactivity';
import * as api from '$lib/api';
import * as auth from './auth.svelte';

// Timer state
let refreshTimer = $state<ReturnType<typeof setTimeout> | null>(null);
let tokenExpiresAt = $state<number>(0); // MS (converted from JWT seconds)
let refreshAt = $state<number>(0); // MS
let lastActivity = $state<number>(0); // MS
let isIdlePaused = $state(false);
let isHiddenPaused = $state(false);
let isRefreshing = $state(false);
let isVisibilityListenerRegistered = false;

// Constants
const REFRESH_THRESHOLD_PERCENT = 0.8; // 80% of lifetime = refresh bei 20% Restzeit
const FALLBACK_BUFFER_MS = 3 * 60 * 1000; // 3 Min vor Expiry (wenn iat fehlt)
const IDLE_THRESHOLD_MS = 10 * 60 * 1000; // 10 Min idle threshold
const MIN_REFRESH_DELAY_MS = 30 * 1000; // 30s minimum (nur wenn delay > 0)
const REFRESH_RETRY_MS = 1000; // 1s retry wenn isRefreshing short-circuit

// Phase 3: Retry-Logik mit Exponential Backoff
const MAX_NETWORK_RETRIES = 5;
const NETWORK_RETRY_DELAYS = [5000, 15000, 30000, 60000, 120000]; // 5s, 15s, 30s, 1min, 2min
let networkRetryCount = $state(0);

/**
 * Schedule a timer for refresh attempt.
 * IMPORTANT: Always clears previous timer to prevent multiple pending timeouts.
 */
function scheduleTimer(delay: number): void {
  // Always clear previous timer to prevent multiple pending timeouts
  if (refreshTimer) {
    clearTimeout(refreshTimer);
  }
  refreshTimer = setTimeout(() => {
    attemptRefresh();
  }, delay);
  console.log(`[TokenRefresh] Timer scheduled in ${Math.floor(delay / 1000)}s`);
}

/**
 * Initialize token refresh with given token timestamps.
 *
 * @param tokenExpiry - Token expiry timestamp in SECONDS (from JWT exp)
 * @param tokenIssuedAt - Optional token issued-at timestamp in SECONDS (from JWT iat)
 */
export function init(tokenExpiry: number, tokenIssuedAt?: number): void {
  stop();

  if (tokenExpiry === 0) {
    console.error('[TokenRefresh] Invalid token expiry, skipping init');
    return;
  }

  // Convert seconds to MS (JWT exp/iat are in seconds!)
  // IMPORTANT: tokenExpiry and tokenIssuedAt come in SECONDS, internally all MS
  tokenExpiresAt = tokenExpiry * 1000; // Seconds -> Milliseconds

  // Calculate refreshAt (in MS)
  if (tokenIssuedAt && tokenIssuedAt > 0) {
    // lifetimeMs = (expiry_sec - issued_sec) * 1000 = lifetime in MS
    const lifetimeMs = (tokenExpiry - tokenIssuedAt) * 1000;

    // Guard against invalid/skewed timestamps (iat >= exp)
    if (lifetimeMs <= 0) {
      console.warn('[TokenRefresh] Invalid JWT timestamps (iat >= exp), using fallback');
      refreshAt = tokenExpiresAt - FALLBACK_BUFFER_MS;
    } else {
      // refreshAt = tokenExpiresAt_ms - (lifetime_ms * (1 - 0.8)) = tokenExpiresAt - 20% lifetime
      refreshAt = tokenExpiresAt - lifetimeMs * (1 - REFRESH_THRESHOLD_PERCENT);
    }
  } else {
    // Fallback: 3 Min vor Expiry (wenn iat fehlt)
    refreshAt = tokenExpiresAt - FALLBACK_BUFFER_MS;
  }

  isIdlePaused = false;
  isHiddenPaused = false;
  lastActivity = Date.now();

  let delay = refreshAt - Date.now();

  // ALWAYS schedule timer, never direct attemptRefresh() call
  // This prevents race condition when init() is called from attemptRefresh()
  // while isRefreshing is still true
  if (delay <= 0) {
    console.log('[TokenRefresh] Refresh overdue, scheduling immediate timer');
    delay = 0; // setTimeout(..., 0) executes attemptRefresh async
  } else if (delay < MIN_REFRESH_DELAY_MS) {
    // Only clamp if positive but small (prevents refresh storms on clock skew)
    console.warn(
      `[TokenRefresh] Delay (${delay}ms) below minimum, using ${MIN_REFRESH_DELAY_MS}ms`
    );
    delay = MIN_REFRESH_DELAY_MS;
  }

  scheduleTimer(delay);

  if (typeof document !== 'undefined' && !isVisibilityListenerRegistered) {
    document.addEventListener('visibilitychange', handleVisibilityChange);
    isVisibilityListenerRegistered = true;
  }

  console.log(
    `[TokenRefresh] Initialized. Token expires at ${new SvelteDate(tokenExpiresAt).toLocaleTimeString()}, ` +
      `refresh at ${new SvelteDate(refreshAt).toLocaleTimeString()}`
  );
}

/**
 * Attempt to refresh the token proactively.
 * Uses central mutex from api.ts to prevent race conditions with 401 handlers.
 */
async function attemptRefresh(): Promise<void> {
  if (isRefreshing) {
    console.log('[TokenRefresh] Refresh already in progress, scheduling retry');
    // Prevent lost timer state by scheduling retry
    // This handles the case where visibility handler calls attemptRefresh while refresh is running
    if (!refreshTimer) {
      scheduleTimer(REFRESH_RETRY_MS);
    }
    return;
  }

  // Guard: Pause flags
  if (isIdlePaused || isHiddenPaused) {
    return;
  }

  // Guard: Idle check
  if (Date.now() - lastActivity > IDLE_THRESHOLD_MS) {
    isIdlePaused = true;
    console.log('[TokenRefresh] User idle, pausing');
    return;
  }

  isRefreshing = true;
  try {
    // Use central mutex to prevent race conditions with 401 handlers
    // This ensures only ONE refresh happens even if proactive + reactive trigger simultaneously
    const result = await api.refreshWithMutex();

    if (result.success) {
      networkRetryCount = 0; // Reset on success

      // Re-init timer with new token timestamps
      const newExp = auth.getTokenExpiry();
      const newIat = auth.getTokenIssuedAt();

      try {
        init(newExp, newIat > 0 ? newIat : undefined);
      } catch (initError) {
        console.error('[TokenRefresh] Re-init failed:', initError);
      }

      console.log('[TokenRefresh] Token refreshed proactively');
    } else if (result.reason === 'auth_error') {
      // Token definitiv ungültig - stoppen
      console.warn('[TokenRefresh] Auth error, stopping');
      stop();
    } else if (result.reason === 'server_error') {
      // Server-Problem - kein Retry (würde Server überlasten)
      console.error('[TokenRefresh] Server error, stopping (manual reload required)');
      stop();
    } else {
      // network_error oder timeout - Retry mit Exponential Backoff
      if (networkRetryCount >= MAX_NETWORK_RETRIES) {
        console.error('[TokenRefresh] Max retries reached, stopping');
        stop();
        return;
      }

      const baseDelay =
        NETWORK_RETRY_DELAYS[Math.min(networkRetryCount, NETWORK_RETRY_DELAYS.length - 1)];
      // Jitter: ±20% zur Vermeidung von Thundering Herd
      const jitter = baseDelay * 0.2 * (Math.random() * 2 - 1);
      const delay = Math.round(baseDelay + jitter);

      networkRetryCount++;
      console.log(
        `[TokenRefresh] ${result.reason}, retry ${networkRetryCount}/${MAX_NETWORK_RETRIES} in ${delay / 1000}s`
      );
      scheduleTimer(delay);
    }
  } catch (error) {
    console.error('[TokenRefresh] Unexpected error:', error);
  } finally {
    isRefreshing = false;
  }
}

/**
 * Record user activity. Called by global event listeners.
 * Resumes refresh if paused due to idle.
 */
export function recordActivity(): void {
  lastActivity = Date.now();

  if (isIdlePaused) {
    isIdlePaused = false;
    console.log('[TokenRefresh] User activity detected, resuming');
    checkAndRefresh();
    return;
  }

  // Prevent reschedule churn
  if (!refreshTimer) {
    checkAndRefresh();
  } else if (refreshAt > 0 && refreshAt - Date.now() < 5000) {
    // Refresh very soon, may need immediate action
    checkAndRefresh();
  }
}

/**
 * Handle visibility change events.
 */
function handleVisibilityChange(): void {
  if (document.visibilityState === 'visible') {
    if (isHiddenPaused) {
      isHiddenPaused = false;
      console.log('[TokenRefresh] Tab visible');

      const now = Date.now();

      // Immediately refresh if overdue, REGARDLESS of idle
      // This prevents 401s before recordActivity() runs
      if (refreshAt > 0 && now >= refreshAt) {
        console.log('[TokenRefresh] Refresh overdue on visibility, refreshing immediately');
        // Temporarily ignore idle for this one refresh
        isIdlePaused = false;
        attemptRefresh();
        // Then set idle status based on actual inactivity
        if (now - lastActivity > IDLE_THRESHOLD_MS) {
          isIdlePaused = true;
        }
        return;
      }

      // Token not overdue: normal idle check
      if (now - lastActivity > IDLE_THRESHOLD_MS) {
        isIdlePaused = true;
        console.log('[TokenRefresh] User was idle during hidden, waiting for activity');
      } else {
        checkAndRefresh();
      }
    }
  } else {
    if (!isHiddenPaused) {
      isHiddenPaused = true;
      console.log('[TokenRefresh] Tab hidden, pausing');

      if (refreshTimer) {
        clearTimeout(refreshTimer);
        refreshTimer = null;
      }
    }
  }
}

/**
 * Check if refresh is needed and schedule or trigger as appropriate.
 */
function checkAndRefresh(): void {
  // Guard against invalid refreshAt (parse error)
  if (refreshAt <= 0) {
    console.warn('[TokenRefresh] Invalid refreshAt, skipping check');
    return;
  }

  if (isIdlePaused || isHiddenPaused || isRefreshing) {
    return;
  }

  const now = Date.now();

  if (now >= refreshAt) {
    console.log('[TokenRefresh] Refresh overdue, refreshing immediately');
    attemptRefresh();
  } else if (!refreshTimer) {
    // Timer missing but refreshAt in future -> schedule
    const delay = refreshAt - now;
    console.log(`[TokenRefresh] Scheduling in ${Math.floor(delay / 60000)} min`);
    scheduleTimer(delay);
  }
  // Else: Timer already running, nothing to do
}

/**
 * Stop the token refresh mechanism.
 * Called on logout or when tokens are invalidated.
 */
export function stop(): void {
  if (refreshTimer) {
    clearTimeout(refreshTimer);
    refreshTimer = null;
  }

  isIdlePaused = false;
  isHiddenPaused = false;
  tokenExpiresAt = 0;
  refreshAt = 0;
  lastActivity = 0;
  networkRetryCount = 0; // Reset retry counter

  if (typeof document !== 'undefined' && isVisibilityListenerRegistered) {
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    isVisibilityListenerRegistered = false;
  }

  console.log('[TokenRefresh] Stopped');
}

/**
 * Check if token refresh is active.
 */
export function isActive(): boolean {
  return refreshTimer !== null && refreshAt > 0;
}

/**
 * Get current refresh state for debugging.
 */
export function getState(): {
  isActive: boolean;
  isIdlePaused: boolean;
  isHiddenPaused: boolean;
  isRefreshing: boolean;
  refreshAt: number;
  tokenExpiresAt: number;
} {
  return {
    isActive: isActive(),
    isIdlePaused,
    isHiddenPaused,
    isRefreshing,
    refreshAt,
    tokenExpiresAt,
  };
}
