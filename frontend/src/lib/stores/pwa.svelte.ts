// PWA Install Coach state management for iOS
// State machine: unknown → eligible → snoozed/dismissed/installed

// --- Constants ---
const INSTALL_SNOOZE_DAYS = 7;
const INSTALL_ACTION_DELAY_MS = 2000;
const INSTALL_FALLBACK_TIMEOUT_MS = 60000;
const STORAGE_KEY = 'xelanote-pwa-install';
const OLD_STORAGE_KEY = 'xelanote-ios-install-dismissed';

// --- Types ---
export type InstallState = 'unknown' | 'eligible' | 'snoozed' | 'dismissed' | 'installed';

export type PwaEventName =
  | 'ios_coach_shown'
  | 'ios_step_changed'
  | 'ios_snoozed'
  | 'ios_dismissed'
  | 'ios_installed_detected';

interface StoredData {
  state: 'snoozed' | 'dismissed' | 'installed';
  snoozedUntil?: number;
}

// --- Safe localStorage helpers ---
function safeGetItem(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSetItem(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* silent - private browsing, quota exceeded, SecurityError */
  }
}

function safeRemoveItem(key: string): void {
  try {
    localStorage.removeItem(key);
  } catch {
    /* silent */
  }
}

// --- Event logging ---
export function emitPwaEvent(name: PwaEventName, data?: Record<string, unknown>): void {
  console.log('[PWA]', name, data ?? '');
}

// --- iOS version parsing ---
export function parseIOSVersion(ua: string): { major: number; minor: number } | null {
  const match = ua.match(/CPU (?:iPhone )?OS (\d+)[_.](\d+)/);
  if (!match) return null;
  return { major: parseInt(match[1], 10), minor: parseInt(match[2], 10) };
}

// --- iOS PWA-capable detection ---
// Supports Safari (always), Chrome/Firefox/Edge/Opera on iOS 16.4+
// Excludes in-app WebViews (Facebook, Instagram, LinkedIn, Twitter)
function detectIOSPwaCapable(): boolean {
  if (typeof navigator === 'undefined') return false;
  const ua = navigator.userAgent;

  // 1. iOS device?
  const isIOS =
    /iPad|iPhone|iPod/.test(ua) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
  if (!isIOS) return false;

  // 2. Hard-block in-app WebViews (these never support PWA install)
  if (/FBAN|FBAV|Instagram|LinkedIn|Twitter/.test(ua)) return false;

  // 3. Safari → always capable
  if (/Safari/.test(ua) && !/CriOS|FxiOS|EdgiOS|OPiOS/.test(ua)) return true;

  // 4. Third-party browsers (CriOS/FxiOS/EdgiOS/OPiOS) → check iOS version ≥ 16.4
  if (/CriOS|FxiOS|EdgiOS|OPiOS/.test(ua)) {
    const version = parseIOSVersion(ua);
    if (!version) {
      // Defensive fallback: if version unparseable (e.g. iPad desktop mode),
      // assume capable — better to show coach unnecessarily than miss opportunity
      return true;
    }
    return version.major > 16 || (version.major === 16 && version.minor >= 4);
  }

  return false;
}

// --- Standalone detection ---
function isStandalone(): boolean {
  if (typeof window === 'undefined') return false;
  return (
    (window.navigator as { standalone?: boolean }).standalone === true ||
    window.matchMedia('(display-mode: standalone)').matches
  );
}

// --- Reactive state ---
let installState = $state<InstallState>('unknown');
let isIOSPwaCapable = $state(false);

// --- One-shot action trigger state ---
let actionFired = false;
let actionTimer: ReturnType<typeof setTimeout> | null = null;
let fallbackTimer: ReturnType<typeof setTimeout> | null = null;

// --- Getters ---
export function getInstallState(): InstallState {
  return installState;
}

export function getIsIOSPwaCapable(): boolean {
  return isIOSPwaCapable;
}

/** @deprecated Use getIsIOSPwaCapable() instead */
export function getIsIOSSafari(): boolean {
  return isIOSPwaCapable;
}

export function getActionDelayMs(): number {
  return INSTALL_ACTION_DELAY_MS;
}

export function getFallbackTimeoutMs(): number {
  return INSTALL_FALLBACK_TIMEOUT_MS;
}

// --- Persistence ---
function loadStoredState(): StoredData | null {
  const raw = safeGetItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as StoredData;
    if (parsed && typeof parsed.state === 'string') {
      return parsed;
    }
  } catch {
    // corrupted data, ignore
  }
  return null;
}

function saveState(data: StoredData): void {
  safeSetItem(STORAGE_KEY, JSON.stringify(data));
}

// --- Migration from old key ---
function migrateOldKey(): boolean {
  const old = safeGetItem(OLD_STORAGE_KEY);
  if (old) {
    saveState({ state: 'dismissed' });
    safeRemoveItem(OLD_STORAGE_KEY);
    return true;
  }
  return false;
}

// --- Init ---
export function initPwaDetection(): void {
  isIOSPwaCapable = detectIOSPwaCapable();

  // Standalone overrides everything
  if (isStandalone()) {
    installState = 'installed';
    saveState({ state: 'installed' });
    return;
  }

  // Migrate old key
  if (migrateOldKey()) {
    installState = 'dismissed';
    return;
  }

  // Load stored state
  const stored = loadStoredState();
  if (stored) {
    if (stored.state === 'installed' || stored.state === 'dismissed') {
      installState = stored.state;
      return;
    }
    if (stored.state === 'snoozed' && stored.snoozedUntil) {
      if (Date.now() < stored.snoozedUntil) {
        installState = 'snoozed';
        return;
      }
      // Snooze expired, will be re-evaluated by checkEligibility
    }
  }

  // State stays 'unknown' until checkEligibility is called
}

// --- State machine ---
export function checkEligibility(): void {
  // Already in a terminal or active state? Don't change.
  if (installState === 'installed' || installState === 'dismissed' || installState === 'eligible') {
    return;
  }

  // If snoozed, check if snooze has expired
  if (installState === 'snoozed') {
    const stored = loadStoredState();
    if (stored?.snoozedUntil && Date.now() < stored.snoozedUntil) {
      return; // Still snoozed
    }
    // Snooze expired, fall through to eligibility check
  }

  // Only show on iOS PWA-capable browsers
  if (!isIOSPwaCapable) return;

  // Not standalone (already checked in init, but double-check)
  if (isStandalone()) {
    markInstalled();
    return;
  }

  installState = 'eligible';
}

// --- First-action trigger ---
// One-shot: after a successful user action, trigger eligibility check after delay.
// Multiple calls are ignored after the first.
export function notifySuccessfulAction(): void {
  if (actionFired) return;
  actionFired = true;

  actionTimer = setTimeout(() => {
    checkEligibility();
    actionTimer = null;
  }, INSTALL_ACTION_DELAY_MS);
}

// Start the fallback timer (called once after auth)
export function startFallbackTimer(): void {
  if (fallbackTimer) return;
  fallbackTimer = setTimeout(() => {
    notifySuccessfulAction();
    fallbackTimer = null;
  }, INSTALL_FALLBACK_TIMEOUT_MS);
}

// Cleanup all timers (called from layout destroy)
export function cleanupTimers(): void {
  if (actionTimer) {
    clearTimeout(actionTimer);
    actionTimer = null;
  }
  if (fallbackTimer) {
    clearTimeout(fallbackTimer);
    fallbackTimer = null;
  }
}

export function snoozeInstall(): void {
  const snoozedUntil = Date.now() + INSTALL_SNOOZE_DAYS * 86400000;
  installState = 'snoozed';
  saveState({ state: 'snoozed', snoozedUntil });
  emitPwaEvent('ios_snoozed');
}

export function dismissInstall(): void {
  installState = 'dismissed';
  saveState({ state: 'dismissed' });
  emitPwaEvent('ios_dismissed');
}

export function markInstalled(): void {
  installState = 'installed';
  saveState({ state: 'installed' });
  emitPwaEvent('ios_installed_detected');
}

// --- Reset (for testing / logout) ---
export function _resetForTesting(): void {
  installState = 'unknown';
  isIOSPwaCapable = false;
  actionFired = false;
  cleanupTimers();
}
