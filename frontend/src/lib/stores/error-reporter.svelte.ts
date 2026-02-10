// Error reporter store using Svelte 5 runes
// Handles automatic error capture and manual feedback submission

import { SvelteSet } from 'svelte/reactivity';

import { getApiBaseUrl } from '$lib/config';

// === State ===
let enabled = $state(true);
let serviceAvailable = $state(false);

// Client-side rate limiting: max 3 reports per 5 minutes
const MAX_REPORTS = 3;
const RATE_WINDOW_MS = 5 * 60 * 1000;
let reportTimestamps: number[] = [];

// Session dedup: track sent fingerprints
const sentFingerprints = new SvelteSet<string>();

// === Public API ===

export function isEnabled(): boolean {
  return enabled;
}

export function setEnabled(value: boolean): void {
  enabled = value;
  try {
    localStorage.setItem('error-reporting-enabled', value ? 'true' : 'false');
  } catch {
    // localStorage may be unavailable
  }
}

export function setServiceAvailable(available: boolean): void {
  serviceAvailable = available;
}

export function getServiceAvailable(): boolean {
  return serviceAvailable;
}

/**
 * Initialize error handlers. Returns cleanup function.
 * Should be called early — before config is loaded.
 * Reports are only sent when both `enabled` and `serviceAvailable` are true.
 */
export function initErrorHandler(): () => void {
  // Load opt-out preference from localStorage
  try {
    const stored = localStorage.getItem('error-reporting-enabled');
    if (stored === 'false') {
      enabled = false;
    }
  } catch {
    // localStorage may be unavailable
  }

  const handleError = (event: ErrorEvent) => {
    if (!isEnabled() || !serviceAvailable) return;

    reportError({
      errorType: event.error?.name || 'Error',
      message: event.message || 'Unknown error',
      stack: event.error?.stack || '',
      url: window.location.pathname,
      component: event.filename || '',
    });
  };

  const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
    if (!isEnabled() || !serviceAvailable) return;

    const reason = event.reason;
    const message =
      reason instanceof Error
        ? reason.message
        : typeof reason === 'string'
          ? reason
          : 'Unhandled promise rejection';
    const stack = reason instanceof Error ? reason.stack || '' : '';
    const errorType = reason instanceof Error ? reason.name : 'UnhandledRejection';

    reportError({
      errorType,
      message,
      stack,
      url: window.location.pathname,
      component: '',
    });
  };

  window.addEventListener('error', handleError);
  window.addEventListener('unhandledrejection', handleUnhandledRejection);

  return () => {
    window.removeEventListener('error', handleError);
    window.removeEventListener('unhandledrejection', handleUnhandledRejection);
  };
}

interface ErrorContext {
  errorType: string;
  message: string;
  stack: string;
  url: string;
  component: string;
}

/**
 * Report an automatic error. Applies rate limiting and dedup.
 */
async function reportError(ctx: ErrorContext): Promise<void> {
  try {
    const fingerprint = await computeFingerprint(ctx.errorType, ctx.message);

    // Session dedup
    if (sentFingerprints.has(fingerprint)) return;

    // Rate limiting
    if (!checkRateLimit()) return;

    sentFingerprints.add(fingerprint);

    // Truncate fields to match backend limits
    const message = ctx.message.slice(0, 500);
    const stack = ctx.stack.slice(0, 4000);

    await submitReport({
      type: 'automatic',
      error_type: ctx.errorType,
      message,
      stack,
      fingerprint,
      url: ctx.url,
      component: ctx.component,
      app_version: '',
      description: '',
      steps_to_reproduce: '',
    });
  } catch {
    // Silently ignore — don't report errors about error reporting
  }
}

/**
 * Submit manual user feedback. Returns result from backend.
 */
export async function reportManualFeedback(
  description: string,
  steps?: string
): Promise<{ accepted: boolean }> {
  const fingerprint = await computeFingerprint('UserFeedback', description);

  return submitReport({
    type: 'manual',
    error_type: 'UserFeedback',
    message: description,
    stack: '',
    fingerprint,
    url: window.location.pathname,
    component: '',
    app_version: '',
    description,
    steps_to_reproduce: steps || '',
  });
}

// === Internal helpers ===

interface ErrorReportPayload {
  type: string;
  error_type: string;
  message: string;
  stack: string;
  fingerprint: string;
  url: string;
  component: string;
  app_version: string;
  description: string;
  steps_to_reproduce: string;
}

async function submitReport(payload: ErrorReportPayload): Promise<{ accepted: boolean }> {
  // Use fetch() directly — not the api.ts request() function
  // to avoid circular error reporting if api.ts itself throws
  const baseUrl = getApiBaseUrl();
  const resp = await fetch(`${baseUrl}/error-reports`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  if (!resp.ok) {
    return { accepted: false };
  }

  return resp.json();
}

function checkRateLimit(): boolean {
  const now = Date.now();
  // Remove timestamps outside the window
  reportTimestamps = reportTimestamps.filter((t) => now - t < RATE_WINDOW_MS);
  if (reportTimestamps.length >= MAX_REPORTS) return false;
  reportTimestamps.push(now);
  return true;
}

/**
 * Compute a fingerprint for dedup: SHA-256 of normalized errorType:message,
 * truncated to first 16 hex characters.
 */
export async function computeFingerprint(errorType: string, message: string): Promise<string> {
  const normalized = normalizeMessage(`${errorType}:${message}`);
  const encoder = new TextEncoder();
  const data = encoder.encode(normalized);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = new Uint8Array(hashBuffer);
  const hex = Array.from(hashArray)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
  return hex.slice(0, 16);
}

/**
 * Normalize a message for fingerprinting:
 * - Replace numbers with N
 * - Replace UUIDs with UUID
 * - Replace ISO dates with DATE
 */
export function normalizeMessage(msg: string): string {
  return (
    msg
      // UUIDs first (before numbers, since UUIDs contain hex digits)
      .replace(/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi, 'UUID')
      // ISO dates
      .replace(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[.\dZ+-]*/g, 'DATE')
      // Numbers (standalone or in paths)
      .replace(/\b\d+\b/g, 'N')
  );
}

// Export for testing
export function _resetForTesting(): void {
  reportTimestamps = [];
  sentFingerprints.clear();
  enabled = true;
  serviceAvailable = false;
}
