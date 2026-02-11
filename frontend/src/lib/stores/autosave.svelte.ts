// Auto-Save Settings Store
// Manages user preferences for auto-save feature with localStorage persistence

const STORAGE_KEY = 'xelanote-autosave-settings';
const DEFAULT_DELAY_MS = 2000; // 2 seconds (hardcoded)

interface AutoSaveSettings {
  enabled: boolean;
  delayMs: number;
}

// Reactive state
let autoSaveSettings = $state<AutoSaveSettings>({
  enabled: true, // Default: enabled (user preference!)
  delayMs: DEFAULT_DELAY_MS,
});

/**
 * Get whether auto-save is enabled
 */
export function getAutoSaveEnabled(): boolean {
  return autoSaveSettings.enabled;
}

/**
 * Get the auto-save delay in milliseconds
 */
export function getAutoSaveDelay(): number {
  return autoSaveSettings.delayMs;
}

/**
 * Set auto-save enabled state
 */
export function setAutoSaveEnabled(enabled: boolean): void {
  autoSaveSettings.enabled = enabled;
  persistSettings();
}

/**
 * Initialize auto-save settings from localStorage
 * Call this on app startup
 */
export function initAutoSaveSettings(): void {
  if (typeof window === 'undefined') return;

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = parseAutoSaveSettings(stored);
      if (!parsed) return;

      // Merge with defaults
      autoSaveSettings = {
        enabled: parsed.enabled ?? true,
        delayMs: DEFAULT_DELAY_MS, // Always use default delay (not configurable)
      };
    }
  } catch (error) {
    console.error('Failed to load auto-save settings from localStorage:', error);
    // Keep defaults
  }
}

function parseAutoSaveSettings(raw: string): Partial<AutoSaveSettings> | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!parsed || typeof parsed !== 'object') return null;
  const candidate = parsed as { enabled?: unknown; delayMs?: unknown };
  if (candidate.enabled !== undefined && typeof candidate.enabled !== 'boolean') {
    return null;
  }
  if (candidate.delayMs !== undefined && typeof candidate.delayMs !== 'number') {
    return null;
  }
  return {
    enabled: candidate.enabled,
    delayMs: candidate.delayMs,
  };
}

/**
 * Persist settings to localStorage
 */
function persistSettings(): void {
  if (typeof window === 'undefined') return;

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(autoSaveSettings));
  } catch (error) {
    console.error('Failed to persist auto-save settings to localStorage:', error);
  }
}
