import { describe, it, expect } from 'vitest';
import { THEMES, getThemesByVariant, isValidThemeId } from './index';

describe('Theme System', () => {
  it('sollte genau 2 Themes haben (Gruvbox Light & Dark)', () => {
    expect(Object.keys(THEMES).length).toBe(2);
  });

  it('sollte gültige Theme-IDs haben', () => {
    Object.keys(THEMES).forEach((id) => {
      expect(isValidThemeId(id)).toBe(true);
    });
  });

  it('sollte Themes nach Variante filtern', () => {
    const lightThemes = getThemesByVariant('light');
    const darkThemes = getThemesByVariant('dark');

    expect(lightThemes.length).toBe(1);
    expect(darkThemes.length).toBe(1);
    expect(lightThemes.every((t) => t.variant === 'light')).toBe(true);
    expect(darkThemes.every((t) => t.variant === 'dark')).toBe(true);
  });

  it('sollte eindeutige Klassennamen haben', () => {
    const classNames = Object.values(THEMES)
      .map((t) => t.className)
      .filter(Boolean);
    const uniqueClassNames = new Set(classNames);

    expect(classNames.length).toBe(uniqueClassNames.size);
  });

  it('sollte alle erforderlichen Themes enthalten', () => {
    const requiredThemes = ['gruvbox-light', 'gruvbox-dark'];

    requiredThemes.forEach((themeId) => {
      expect(isValidThemeId(themeId)).toBe(true);
      expect(THEMES[themeId as keyof typeof THEMES]).toBeDefined();
    });
  });

  it('sollte für jedes Theme die erforderlichen Eigenschaften haben', () => {
    Object.values(THEMES).forEach((theme) => {
      expect(theme).toHaveProperty('id');
      expect(theme).toHaveProperty('name');
      expect(theme).toHaveProperty('variant');
      expect(theme).toHaveProperty('className');
      expect(['light', 'dark']).toContain(theme.variant);
    });
  });
});
