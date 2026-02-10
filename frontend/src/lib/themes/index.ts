export type ThemeVariant = 'light' | 'dark';

export type ThemeId = 'gruvbox-light' | 'gruvbox-dark';

export interface Theme {
  id: ThemeId;
  name: string;
  variant: ThemeVariant;
  description?: string;
  className: string;
}

export const THEMES: Record<ThemeId, Theme> = {
  'gruvbox-light': {
    id: 'gruvbox-light',
    name: 'Gruvbox Hell',
    variant: 'light',
    description: 'Warme Retro-Farben mit hohem Kontrast',
    className: 'theme-gruvbox-light',
  },
  'gruvbox-dark': {
    id: 'gruvbox-dark',
    name: 'Gruvbox Dunkel',
    variant: 'dark',
    description: 'Warme Retro-Farben mit hohem Kontrast',
    className: 'theme-gruvbox-dark',
  },
};

export function getThemesByVariant(variant: ThemeVariant): Theme[] {
  return Object.values(THEMES).filter((t) => t.variant === variant);
}

export function isValidThemeId(id: string): id is ThemeId {
  return id in THEMES;
}

// Preview Theme Types
export type PreviewThemeId = ThemeId | 'match-editor';

export interface PreviewThemeOption {
  id: PreviewThemeId;
  name: string;
}

export const PREVIEW_THEME_OPTIONS: PreviewThemeOption[] = [
  { id: 'match-editor', name: 'Wie Editor' },
  ...Object.values(THEMES).map((t) => ({ id: t.id, name: t.name })),
];

export function isValidPreviewThemeId(id: string): id is PreviewThemeId {
  return id === 'match-editor' || id in THEMES;
}
