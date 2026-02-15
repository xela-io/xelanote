// Web Vitals performance metrics reporting
// Follows data governance rules from docs/conventions.md:
// - 10% client-side sampling
// - URL sanitizing (no UUIDs, no query params)
// - DNT respected as hard floor
// - Max 1KB payloads

import type { Metric } from 'web-vitals';
import { onCLS, onFCP, onINP, onLCP, onTTFB } from 'web-vitals';

import { getApiBaseUrl } from '$lib/config';

// Sampling: only 10% of sessions report metrics
let samplingEnabled = false;
let initialized = false;

// Rate limiting: max 5 reports per session
const MAX_REPORTS_PER_SESSION = 5;
let reportCount = 0;

/**
 * Sanitize a URL for telemetry: strip query params, replace UUIDs and numeric IDs.
 */
export function sanitizeUrl(url: string): string {
  // Remove query string and hash
  let sanitized = url.split('?')[0].split('#')[0];
  // Replace UUIDs
  sanitized = sanitized.replace(
    /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi,
    ':id'
  );
  // Replace numeric path segments
  sanitized = sanitized.replace(/\/\d+/g, '/:id');
  return sanitized;
}

/**
 * Check if Do-Not-Track is enabled.
 */
function isDNTEnabled(): boolean {
  if (typeof navigator === 'undefined') return false;
  return navigator.doNotTrack === '1';
}

/**
 * Report a single Web Vital metric to the backend.
 */
async function reportMetric(metric: Metric): Promise<void> {
  if (!samplingEnabled || reportCount >= MAX_REPORTS_PER_SESSION) return;
  reportCount++;

  const payload = {
    metric_name: metric.name,
    value: metric.value,
    rating: metric.rating,
    sanitized_url: sanitizeUrl(window.location.pathname),
  };

  try {
    const baseUrl = getApiBaseUrl();
    await fetch(`${baseUrl}/perf-metrics`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(payload),
    });
  } catch {
    // Fire-and-forget: silently ignore network errors
  }
}

/**
 * Initialize Web Vitals reporting.
 * Should be called once during app initialization.
 * Respects DNT and applies 10% sampling.
 */
export function initPerfMetrics(): void {
  if (initialized) return;
  initialized = true;

  // DNT is a hard floor — always respect it
  if (isDNTEnabled()) return;

  // 10% client-side sampling (decided once per session)
  if (Math.random() >= 0.1) return;

  samplingEnabled = true;

  onLCP(reportMetric);
  onINP(reportMetric);
  onCLS(reportMetric);
  onFCP(reportMetric);
  onTTFB(reportMetric);
}

// Export for testing
export function _resetForTesting(): void {
  samplingEnabled = false;
  initialized = false;
  reportCount = 0;
}
