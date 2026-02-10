/**
 * Focus Mode Store for Desktop App
 *
 * Manages distraction-free writing mode.
 * Uses Svelte 5 runes for reactive state.
 */

interface FocusModeState {
  isActive: boolean;
  hideSidebar: boolean;
  hideTabBar: boolean;
  typewriterMode: boolean; // Keep cursor centered
  dimInactiveLines: boolean;
}

// Default settings
const defaults: FocusModeState = {
  isActive: false,
  hideSidebar: true,
  hideTabBar: true,
  typewriterMode: false,
  dimInactiveLines: true,
};

// State
let state = $state<FocusModeState>({ ...defaults });

// Track previous fullscreen state
let wasFullscreen = false;

// Getters
export function isActive(): boolean {
  return state.isActive;
}

export function getSettings(): FocusModeState {
  return { ...state };
}

export function shouldHideSidebar(): boolean {
  return state.isActive && state.hideSidebar;
}

export function shouldHideTabBar(): boolean {
  return state.isActive && state.hideTabBar;
}

export function isTypewriterMode(): boolean {
  return state.isActive && state.typewriterMode;
}

export function shouldDimInactiveLines(): boolean {
  return state.isActive && state.dimInactiveLines;
}

// Actions
export async function enter(): Promise<void> {
  if (state.isActive) return;

  // Remember current fullscreen state
  if (typeof document !== 'undefined') {
    wasFullscreen = !!document.fullscreenElement;

    // Enter fullscreen
    if (!wasFullscreen) {
      try {
        await document.documentElement.requestFullscreen();
      } catch (err) {
        console.warn('Failed to enter fullscreen:', err);
      }
    }
  }

  state.isActive = true;
}

export async function exit(): Promise<void> {
  if (!state.isActive) return;

  // Exit fullscreen if we entered it
  if (typeof document !== 'undefined' && !wasFullscreen && document.fullscreenElement) {
    try {
      await document.exitFullscreen();
    } catch (err) {
      console.warn('Failed to exit fullscreen:', err);
    }
  }

  state.isActive = false;
}

export async function toggle(): Promise<void> {
  if (state.isActive) {
    await exit();
  } else {
    await enter();
  }
}

// Update settings
export function updateSettings(newSettings: Partial<FocusModeState>): void {
  state = { ...state, ...newSettings };
}

export function setHideSidebar(hide: boolean): void {
  state.hideSidebar = hide;
}

export function setHideTabBar(hide: boolean): void {
  state.hideTabBar = hide;
}

export function setTypewriterMode(enabled: boolean): void {
  state.typewriterMode = enabled;
}

export function setDimInactiveLines(dim: boolean): void {
  state.dimInactiveLines = dim;
}

// Reset to defaults
export function reset(): void {
  if (state.isActive) {
    exit();
  }
  state = { ...defaults };
}

// Keyboard shortcut handler
export function handleKeydown(event: KeyboardEvent): boolean {
  // F11 or Ctrl+Shift+F to toggle
  if (event.key === 'F11' || (event.ctrlKey && event.shiftKey && event.key === 'F')) {
    event.preventDefault();
    toggle();
    return true;
  }

  // Escape to exit
  if (event.key === 'Escape' && state.isActive) {
    event.preventDefault();
    exit();
    return true;
  }

  return false;
}

// Initialize keyboard listener
export function initKeyboardListener(): () => void {
  if (typeof window === 'undefined') return () => {};

  const handler = (event: KeyboardEvent) => handleKeydown(event);
  window.addEventListener('keydown', handler);

  return () => {
    window.removeEventListener('keydown', handler);
  };
}
