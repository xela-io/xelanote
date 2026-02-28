// UI state store using Svelte 5 runes
import {
  isValidPreviewThemeId,
  isValidThemeId,
  type PreviewThemeId,
  type ThemeId,
  THEMES,
} from '$lib/themes';

// Sidebar state
let sidebarOpen = $state(true);
let sidebarWidth = $state(240); // compact desktop default
const SIDEBAR_MIN_WIDTH = 320;
const SIDEBAR_MAX_WIDTH = 500;
const SIDEBAR_WIDTH_STORAGE_KEY = 'xelanote-sidebar-width';
const SIDEBAR_WIDTH_MIGRATION_V1_KEY = 'xelanote-sidebar-width-migrated-v1';

// Split view position (editor percentage, 20-80)
let splitPosition = $state(50);
const SPLIT_MIN = 20;
const SPLIT_MAX = 80;

// Mobile state
let isMobile = $state(false);

// Desktop editor panels (Summary/Tags area)
let editorPanelsCollapsed = $state(false);

// Keyboard state (for hiding toolbar on mobile when keyboard is open)
let isKeyboardOpen = $state(false);

// Quick switcher state
let quickSwitcherOpen = $state(false);

// Theme
let currentThemeId = $state<ThemeId>('gruvbox-dark');

// Preview Theme
let previewThemeId = $state<PreviewThemeId>('match-editor');

// Editor mode
let editorMode = $state<'edit' | 'preview' | 'split' | 'live'>('live');

// Standalone PWA state
let isStandalone = $state(false);

// iOS detection (for Safari body-scroll mode)
let isIOS = $state(false);

// Markdown guide state
let markdownGuideOpen = $state(false);
let markdownGuideTab = $state<'syntax' | 'wikilinks' | 'code'>('syntax');
let markdownGuideDropdownOpen = $state(false);

// Export functions
export function getSidebarOpen() {
  return sidebarOpen;
}

export function setSidebarOpen(open: boolean) {
  sidebarOpen = open;
}

export function toggleSidebar() {
  sidebarOpen = !sidebarOpen;
}

export function getIsMobile() {
  return isMobile;
}

export function setIsMobile(mobile: boolean) {
  isMobile = mobile;
}

export function getSidebarWidth() {
  return sidebarWidth;
}

export function setSidebarWidth(width: number) {
  sidebarWidth = Math.max(SIDEBAR_MIN_WIDTH, Math.min(SIDEBAR_MAX_WIDTH, width));
  try {
    localStorage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function getSidebarMinWidth() {
  return SIDEBAR_MIN_WIDTH;
}

export function getSidebarMaxWidth() {
  return SIDEBAR_MAX_WIDTH;
}

export function initSidebarWidth() {
  try {
    const saved = localStorage.getItem(SIDEBAR_WIDTH_STORAGE_KEY);
    if (saved) {
      const parsed = parseInt(saved, 10);
      if (!isNaN(parsed)) {
        const migrated = localStorage.getItem(SIDEBAR_WIDTH_MIGRATION_V1_KEY) === '1';
        // Only auto-normalize the legacy default width once; keep user-chosen widths intact.
        const normalized = !migrated && parsed === 256 ? 240 : parsed;
        sidebarWidth = Math.max(SIDEBAR_MIN_WIDTH, Math.min(SIDEBAR_MAX_WIDTH, normalized));
        if (!migrated) {
          localStorage.setItem(SIDEBAR_WIDTH_MIGRATION_V1_KEY, '1');
          if (normalized !== parsed) {
            localStorage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
          }
        }
      }
    }
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function getSplitPosition() {
  return splitPosition;
}

export function setSplitPosition(pos: number) {
  splitPosition = Math.max(SPLIT_MIN, Math.min(SPLIT_MAX, pos));
  try {
    localStorage.setItem('xelanote-split-position', String(splitPosition));
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function initSplitPosition() {
  try {
    const saved = localStorage.getItem('xelanote-split-position');
    if (saved) {
      const parsed = parseFloat(saved);
      if (!isNaN(parsed)) {
        splitPosition = Math.max(SPLIT_MIN, Math.min(SPLIT_MAX, parsed));
      }
    }
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function getEditorPanelsCollapsed() {
  return editorPanelsCollapsed;
}

export function setEditorPanelsCollapsed(collapsed: boolean) {
  editorPanelsCollapsed = collapsed;
  try {
    localStorage.setItem('xelanote-editor-panels-collapsed', String(editorPanelsCollapsed));
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function toggleEditorPanelsCollapsed() {
  setEditorPanelsCollapsed(!editorPanelsCollapsed);
}

export function initEditorPanelsCollapsed() {
  try {
    const saved = localStorage.getItem('xelanote-editor-panels-collapsed');
    if (saved === 'true' || saved === 'false') {
      editorPanelsCollapsed = saved === 'true';
    }
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function closeSidebarOnMobile() {
  if (isMobile) {
    sidebarOpen = false;
  }
}

export function getIsStandalone(): boolean {
  return isStandalone;
}

export function initStandaloneDetection(): void {
  if (typeof window === 'undefined') return;
  isStandalone =
    (window.navigator as { standalone?: boolean }).standalone === true ||
    window.matchMedia('(display-mode: standalone)').matches;
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('is-standalone', isStandalone);
  }
}

export function getIsIOS(): boolean {
  return isIOS;
}

export function initIOSDetection(): void {
  if (typeof navigator === 'undefined') return;
  const ua = navigator.userAgent;
  isIOS =
    /iPad|iPhone|iPod/.test(ua) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('is-ios', isIOS);
  }
}

export function getIsKeyboardOpen() {
  return isKeyboardOpen;
}

export function setIsKeyboardOpen(open: boolean) {
  isKeyboardOpen = open;
}

export function getQuickSwitcherOpen() {
  return quickSwitcherOpen;
}

export function setQuickSwitcherOpen(open: boolean) {
  quickSwitcherOpen = open;
}

export function toggleQuickSwitcher() {
  quickSwitcherOpen = !quickSwitcherOpen;
}

export function getCurrentTheme() {
  return THEMES[currentThemeId];
}

export function getCurrentThemeId() {
  return currentThemeId;
}

export function getTheme() {
  return getCurrentTheme().variant;
}

export function setTheme(themeId: ThemeId) {
  currentThemeId = themeId;
  if (typeof document !== 'undefined') {
    const theme = THEMES[themeId];

    // Alle Theme-Klassen entfernen
    document.documentElement.classList.remove(
      ...Object.values(THEMES)
        .map((t) => t.className)
        .filter(Boolean)
    );

    // Neue Theme-Klasse hinzufügen
    if (theme.className) {
      document.documentElement.classList.add(theme.className);
    }

    try {
      localStorage.setItem('xelanote-theme', themeId);
    } catch {
      // localStorage may throw SecurityError in Firefox private browsing
    }
  }
}

export function toggleTheme() {
  const current = getCurrentTheme();
  setTheme(current.variant === 'dark' ? 'gruvbox-light' : 'gruvbox-dark');
}

export function initTheme() {
  try {
    const saved = localStorage.getItem('xelanote-theme');

    // Validierung
    if (saved && isValidThemeId(saved)) {
      setTheme(saved);
      return;
    }
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }

  // Fallback: System-Präferenz
  if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    setTheme('gruvbox-dark');
  } else {
    setTheme('gruvbox-light');
  }
}

// Preview Theme Functions
export function getPreviewThemeId(): PreviewThemeId {
  return previewThemeId;
}

export function setPreviewTheme(themeId: PreviewThemeId) {
  previewThemeId = themeId;
  try {
    localStorage.setItem('xelanote-preview-theme', themeId);
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function getEffectivePreviewTheme(): ThemeId {
  if (previewThemeId === 'match-editor') {
    return currentThemeId;
  }
  return previewThemeId as ThemeId;
}

export function getEffectivePreviewThemeClass(): string {
  const themeId = getEffectivePreviewTheme();
  return THEMES[themeId].className;
}

export function initPreviewTheme() {
  try {
    const saved = localStorage.getItem('xelanote-preview-theme');
    if (saved && isValidPreviewThemeId(saved)) {
      previewThemeId = saved;
    }
  } catch {
    // localStorage may throw SecurityError in Firefox private browsing
  }
}

export function getEditorMode() {
  return editorMode;
}

export function setEditorMode(mode: 'edit' | 'preview' | 'split' | 'live') {
  editorMode = mode;
}

/**
 * Reset UI state to defaults (called on logout to prevent user data leak).
 */
export function resetToDefaults() {
  // Reset theme to system preference (with fallback for test environments)
  let prefersDark = false;
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    try {
      prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    } catch {
      // Ignore matchMedia errors in test environments
    }
  }
  setTheme(prefersDark ? 'gruvbox-dark' : 'gruvbox-light');

  // Reset editor mode to default
  editorMode = 'split';

  // Reset preview theme to default
  previewThemeId = 'match-editor';

  // Reset desktop editor panel visibility to default (visible)
  editorPanelsCollapsed = false;

  console.log('[UI] Reset to defaults on logout');
}

export function getMarkdownGuideOpen() {
  return markdownGuideOpen;
}

export function setMarkdownGuideOpen(open: boolean) {
  markdownGuideOpen = open;
}

export function toggleMarkdownGuide() {
  markdownGuideOpen = !markdownGuideOpen;
}

export function getMarkdownGuideTab() {
  return markdownGuideTab;
}

export function setMarkdownGuideTab(tab: 'syntax' | 'wikilinks' | 'code') {
  markdownGuideTab = tab;
}

export function getMarkdownGuideDropdownOpen() {
  return markdownGuideDropdownOpen;
}

export function setMarkdownGuideDropdownOpen(open: boolean) {
  markdownGuideDropdownOpen = open;
}

export function toggleMarkdownGuideDropdown() {
  markdownGuideDropdownOpen = !markdownGuideDropdownOpen;
}
