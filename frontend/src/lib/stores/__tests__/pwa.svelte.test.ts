import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Storage key constants (must match pwa.svelte.ts)
const STORAGE_KEY = 'xelanote-pwa-install';
const OLD_STORAGE_KEY = 'xelanote-ios-install-dismissed';

// Helper to set up navigator.userAgent for iOS Safari
function mockIOSSafariUA() {
  Object.defineProperty(navigator, 'userAgent', {
    value:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockCriOSUA(iosVersion = '17_0') {
  Object.defineProperty(navigator, 'userAgent', {
    value: `Mozilla/5.0 (iPhone; CPU iPhone OS ${iosVersion} like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0 Mobile/15E148 Safari/604.1`,
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockFxiOSUA(iosVersion = '17_0') {
  Object.defineProperty(navigator, 'userAgent', {
    value: `Mozilla/5.0 (iPhone; CPU iPhone OS ${iosVersion} like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/120.0 Mobile/15E148 Safari/604.1`,
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockEdgiOSUA(iosVersion = '17_0') {
  Object.defineProperty(navigator, 'userAgent', {
    value: `Mozilla/5.0 (iPhone; CPU iPhone OS ${iosVersion} like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) EdgiOS/120.0 Mobile/15E148 Safari/604.1`,
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockOPiOSUA(iosVersion = '16_4') {
  Object.defineProperty(navigator, 'userAgent', {
    value: `Mozilla/5.0 (iPhone; CPU iPhone OS ${iosVersion} like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) OPiOS/120.0 Mobile/15E148 Safari/604.1`,
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockCriOSUnparseableUA() {
  // iPad in desktop mode — no "CPU iPhone OS" in UA
  Object.defineProperty(navigator, 'userAgent', {
    value:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0 Safari/604.1',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'MacIntel',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockFBANUA() {
  Object.defineProperty(navigator, 'userAgent', {
    value:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FBAN Mobile/15E148 Safari/604.1',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'iPhone',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 5,
    writable: true,
    configurable: true,
  });
}

function mockDesktopUA() {
  Object.defineProperty(navigator, 'userAgent', {
    value:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'platform', {
    value: 'MacIntel',
    writable: true,
    configurable: true,
  });
  Object.defineProperty(navigator, 'maxTouchPoints', {
    value: 0,
    writable: true,
    configurable: true,
  });
}

// Mock matchMedia for standalone detection
function mockStandalone(isStandalone: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === '(display-mode: standalone)' ? isStandalone : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
    writable: true,
    configurable: true,
  });
  // Reset navigator.standalone
  Object.defineProperty(window.navigator, 'standalone', {
    value: isStandalone,
    writable: true,
    configurable: true,
  });
}

describe('pwa.svelte.ts', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    localStorage.clear();
    mockStandalone(false);
    mockDesktopUA();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  async function importPwa() {
    return await import('../pwa.svelte');
  }

  describe('state transitions', () => {
    it('starts in unknown state', async () => {
      const pwa = await importPwa();
      expect(pwa.getInstallState()).toBe('unknown');
    });

    it('transitions unknown → eligible on iOS Safari', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');
    });

    it('does NOT become eligible on desktop', async () => {
      mockDesktopUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('unknown');
    });

    it('transitions eligible → snoozed', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.snoozeInstall();
      expect(pwa.getInstallState()).toBe('snoozed');
    });

    it('transitions eligible → dismissed', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.dismissInstall();
      expect(pwa.getInstallState()).toBe('dismissed');
    });

    it('transitions any → installed', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.markInstalled();
      expect(pwa.getInstallState()).toBe('installed');
    });

    it('installed state overrides standalone on init', async () => {
      mockIOSSafariUA();
      mockStandalone(true);
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getInstallState()).toBe('installed');
    });
  });

  describe('snooze expiry', () => {
    it('stays snoozed within cooldown period', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();

      // Set up a snooze that's still active
      const futureDate = Date.now() + 3 * 86400000; // 3 days from now
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ state: 'snoozed', snoozedUntil: futureDate })
      );

      pwa.initPwaDetection();
      expect(pwa.getInstallState()).toBe('snoozed');

      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('snoozed');
    });

    it('becomes eligible again after snooze expires', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();

      // Set up an expired snooze
      const pastDate = Date.now() - 86400000; // 1 day ago
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ state: 'snoozed', snoozedUntil: pastDate })
      );

      pwa.initPwaDetection();
      // After init with expired snooze, state should be 'unknown' (ready for checkEligibility)
      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');
    });
  });

  describe('migration', () => {
    it('migrates old xelanote-ios-install-dismissed key to dismissed', async () => {
      localStorage.setItem(OLD_STORAGE_KEY, '1');
      const pwa = await importPwa();
      pwa.initPwaDetection();

      expect(pwa.getInstallState()).toBe('dismissed');
      expect(localStorage.getItem(OLD_STORAGE_KEY)).toBeNull();
      expect(localStorage.getItem(STORAGE_KEY)).not.toBeNull();

      const stored = JSON.parse(localStorage.getItem(STORAGE_KEY)!);
      expect(stored.state).toBe('dismissed');
    });
  });

  describe('localStorage errors', () => {
    it('does not throw on SecurityError in getItem', async () => {
      const pwa = await importPwa();

      // Mock localStorage to throw
      const originalGetItem = localStorage.getItem;
      localStorage.getItem = () => {
        throw new DOMException('SecurityError');
      };

      // Should not throw
      expect(() => pwa.initPwaDetection()).not.toThrow();
      expect(pwa.getInstallState()).toBe('unknown');

      localStorage.getItem = originalGetItem;
    });

    it('does not throw on QuotaExceededError in setItem', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();

      const originalSetItem = localStorage.setItem;
      localStorage.setItem = () => {
        throw new DOMException('QuotaExceededError');
      };

      pwa.initPwaDetection();
      pwa.checkEligibility();

      // Should not throw
      expect(() => pwa.snoozeInstall()).not.toThrow();
      expect(pwa.getInstallState()).toBe('snoozed');

      localStorage.setItem = originalSetItem;
    });
  });

  describe('idempotency of checkEligibility', () => {
    it('multiple calls do not change state once eligible', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();

      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('eligible');
    });

    it('does not change dismissed state', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();

      localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: 'dismissed' }));
      pwa.initPwaDetection();
      expect(pwa.getInstallState()).toBe('dismissed');

      pwa.checkEligibility();
      expect(pwa.getInstallState()).toBe('dismissed');
    });
  });

  describe('iOS PWA-capable detection', () => {
    it('detects iOS Safari correctly', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('accepts CriOS (Chrome on iOS 17.0)', async () => {
      mockCriOSUA('17_0');
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('accepts FxiOS (Firefox on iOS 17.0)', async () => {
      mockFxiOSUA('17_0');
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('rejects CriOS on iOS 16.0 (below 16.4)', async () => {
      mockCriOSUA('16_0');
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(false);
    });

    it('accepts EdgiOS on iOS 17.0', async () => {
      mockEdgiOSUA('17_0');
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('accepts OPiOS on iOS 16.4', async () => {
      mockOPiOSUA('16_4');
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('accepts CriOS with unparseable version (iPad desktop mode)', async () => {
      mockCriOSUnparseableUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(true);
    });

    it('rejects FBAN (Facebook in-app browser) regardless of version', async () => {
      mockFBANUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(false);
    });

    it('rejects desktop browser', async () => {
      mockDesktopUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSPwaCapable()).toBe(false);
    });

    it('deprecated getIsIOSSafari returns same value as getIsIOSPwaCapable', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();
      expect(pwa.getIsIOSSafari()).toBe(pwa.getIsIOSPwaCapable());
    });
  });

  describe('parseIOSVersion', () => {
    it('parses standard iPhone UA', async () => {
      const pwa = await importPwa();
      const result = pwa.parseIOSVersion(
        'Mozilla/5.0 (iPhone; CPU iPhone OS 16_4 like Mac OS X)'
      );
      expect(result).toEqual({ major: 16, minor: 4 });
    });

    it('parses iPad UA', async () => {
      const pwa = await importPwa();
      const result = pwa.parseIOSVersion('Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)');
      expect(result).toEqual({ major: 17, minor: 0 });
    });

    it('returns null for desktop UA', async () => {
      const pwa = await importPwa();
      const result = pwa.parseIOSVersion(
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0'
      );
      expect(result).toBeNull();
    });
  });

  describe('first-action trigger', () => {
    it('notifySuccessfulAction triggers checkEligibility after delay', async () => {
      vi.useFakeTimers();
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();

      pwa.notifySuccessfulAction();
      expect(pwa.getInstallState()).toBe('unknown');

      vi.advanceTimersByTime(2000);
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.cleanupTimers();
      vi.useRealTimers();
    });

    it('notifySuccessfulAction is one-shot (ignores subsequent calls)', async () => {
      vi.useFakeTimers();
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();

      pwa.notifySuccessfulAction();
      pwa.notifySuccessfulAction();
      pwa.notifySuccessfulAction();

      // Only one timer should have been created
      vi.advanceTimersByTime(2000);
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.cleanupTimers();
      vi.useRealTimers();
    });

    it('fallback timer triggers notifySuccessfulAction', async () => {
      vi.useFakeTimers();
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();

      pwa.startFallbackTimer();
      expect(pwa.getInstallState()).toBe('unknown');

      // Fallback fires at 60s
      vi.advanceTimersByTime(60000);
      // Then action delay at +2s
      vi.advanceTimersByTime(2000);
      expect(pwa.getInstallState()).toBe('eligible');

      pwa.cleanupTimers();
      vi.useRealTimers();
    });

    it('cleanupTimers prevents firing', async () => {
      vi.useFakeTimers();
      mockIOSSafariUA();
      const pwa = await importPwa();
      pwa.initPwaDetection();

      pwa.notifySuccessfulAction();
      pwa.cleanupTimers();

      vi.advanceTimersByTime(5000);
      expect(pwa.getInstallState()).toBe('unknown');

      vi.useRealTimers();
    });
  });

  describe('event logging', () => {
    it('emits ios_snoozed on snooze', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      pwa.initPwaDetection();
      pwa.checkEligibility();
      pwa.snoozeInstall();

      expect(consoleSpy).toHaveBeenCalledWith('[PWA]', 'ios_snoozed', '');
      consoleSpy.mockRestore();
    });

    it('emits ios_dismissed on dismiss', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      pwa.initPwaDetection();
      pwa.checkEligibility();
      pwa.dismissInstall();

      expect(consoleSpy).toHaveBeenCalledWith('[PWA]', 'ios_dismissed', '');
      consoleSpy.mockRestore();
    });

    it('emits ios_installed_detected on markInstalled', async () => {
      mockIOSSafariUA();
      const pwa = await importPwa();
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

      pwa.initPwaDetection();
      pwa.markInstalled();

      expect(consoleSpy).toHaveBeenCalledWith('[PWA]', 'ios_installed_detected', '');
      consoleSpy.mockRestore();
    });
  });
});
