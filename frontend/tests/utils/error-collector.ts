import type { Page } from '@playwright/test';

export type ErrorType =
  | 'console-error'
  | 'console-warning'
  | 'network-error'
  | 'js-exception'
  | 'resource-404'
  | 'http-error'
  | 'layout-shift';

export type ErrorSeverity = 'low' | 'medium' | 'high' | 'critical';

export interface CollectedError {
  type: ErrorType;
  message: string;
  url: string;
  timestamp: string;
  stackTrace?: string;
  selector?: string;
  severity: ErrorSeverity;
  statusCode?: number;
  resourceUrl?: string;
}

export interface ErrorSummary {
  errors: CollectedError[];
  totalCount: number;
  bySeverity: Record<ErrorSeverity, number>;
  byType: Partial<Record<ErrorType, number>>;
  hasBlockers: boolean;
}

const IGNORED_CONSOLE_PATTERNS = [
  /Download the Vue Devtools/,
  /Download the React Developer Tools/,
  /Vite.*HMR/,
  /\[vite\]/,
  /favicon\.ico/,
  /service-worker/i,
  /workbox/i,
];

function shouldIgnore(message: string): boolean {
  return IGNORED_CONSOLE_PATTERNS.some((pattern) => pattern.test(message));
}

function inferSeverity(type: ErrorType, statusCode?: number): ErrorSeverity {
  switch (type) {
    case 'js-exception':
      return 'critical';
    case 'console-error':
      return 'high';
    case 'network-error':
      return 'high';
    case 'http-error':
      return statusCode && statusCode >= 500 ? 'critical' : 'medium';
    case 'resource-404':
      return 'medium';
    case 'console-warning':
      return 'low';
    case 'layout-shift':
      return 'medium';
    default:
      return 'low';
  }
}

export class ErrorCollector {
  private errors: CollectedError[] = [];
  private pageUrl: string = '';

  attach(page: Page): void {
    // Track current URL
    page.on('framenavigated', (frame) => {
      if (frame === page.mainFrame()) {
        this.pageUrl = frame.url();
      }
    });

    // Console errors and warnings
    page.on('console', (msg) => {
      const type = msg.type();
      if (type !== 'error' && type !== 'warning') return;

      const text = msg.text();
      if (shouldIgnore(text)) return;

      const errorType: ErrorType = type === 'error' ? 'console-error' : 'console-warning';

      this.errors.push({
        type: errorType,
        message: text,
        url: this.pageUrl,
        timestamp: new Date().toISOString(),
        severity: inferSeverity(errorType),
      });
    });

    // Unhandled JS exceptions
    page.on('pageerror', (error) => {
      this.errors.push({
        type: 'js-exception',
        message: error.message,
        url: this.pageUrl,
        timestamp: new Date().toISOString(),
        stackTrace: error.stack,
        severity: 'critical',
      });
    });

    // Failed network requests
    page.on('requestfailed', (request) => {
      const failure = request.failure();
      if (!failure) return;

      // Ignore cancelled requests (navigation)
      if (failure.errorText === 'net::ERR_ABORTED') return;

      this.errors.push({
        type: 'network-error',
        message: `${request.method()} ${request.url()} - ${failure.errorText}`,
        url: this.pageUrl,
        timestamp: new Date().toISOString(),
        resourceUrl: request.url(),
        severity: 'high',
      });
    });

    // HTTP errors (4xx, 5xx)
    page.on('response', (response) => {
      const status = response.status();
      if (status < 400) return;

      const url = response.url();

      // Ignore expected 401s on initial auth check
      if (status === 401 && url.includes('/api/auth/me')) return;
      // Ignore 404 for optional resources
      if (status === 404 && (url.includes('favicon') || url.includes('manifest'))) return;

      const errorType: ErrorType = status === 404 ? 'resource-404' : 'http-error';

      this.errors.push({
        type: errorType,
        message: `HTTP ${status} - ${response.request().method()} ${url}`,
        url: this.pageUrl,
        timestamp: new Date().toISOString(),
        statusCode: status,
        resourceUrl: url,
        severity: inferSeverity(errorType, status),
      });
    });
  }

  getErrors(): CollectedError[] {
    return [...this.errors];
  }

  getSummary(): ErrorSummary {
    const bySeverity: Record<ErrorSeverity, number> = {
      low: 0,
      medium: 0,
      high: 0,
      critical: 0,
    };
    const byType: Partial<Record<ErrorType, number>> = {};

    for (const error of this.errors) {
      bySeverity[error.severity]++;
      byType[error.type] = (byType[error.type] ?? 0) + 1;
    }

    return {
      errors: this.getErrors(),
      totalCount: this.errors.length,
      bySeverity,
      byType,
      hasBlockers: bySeverity.critical > 0 || bySeverity.high > 0,
    };
  }

  clear(): void {
    this.errors = [];
  }

  hasErrors(minSeverity: ErrorSeverity = 'high'): boolean {
    const severityOrder: ErrorSeverity[] = ['low', 'medium', 'high', 'critical'];
    const minIndex = severityOrder.indexOf(minSeverity);
    return this.errors.some((e) => severityOrder.indexOf(e.severity) >= minIndex);
  }
}
