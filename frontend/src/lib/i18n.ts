import { browser } from '$app/environment';
import { init, register } from 'svelte-i18n';
import de from './locales/de.json';
import en from './locales/en.json';

const defaultLocale = 'de';
const supportedLocales = ['de', 'en'];

register('de', () => Promise.resolve(de));
register('en', () => Promise.resolve(en));

/**
 * Normalize browser locale (e.g., "de-DE" -> "de", "en-US" -> "en")
 * Falls back to defaultLocale if not supported
 */
export function normalizeLocale(locale: string): string {
  // First try exact match
  if (supportedLocales.includes(locale)) {
    return locale;
  }
  // Try language code only (e.g., "de-DE" -> "de")
  const languageCode = locale.split('-')[0];
  if (supportedLocales.includes(languageCode)) {
    return languageCode;
  }
  return defaultLocale;
}

function getInitialLocale(): string {
  if (!browser) return defaultLocale;

  // Check localStorage first
  const savedLocale = window.localStorage.getItem('locale');
  if (savedLocale && supportedLocales.includes(savedLocale)) {
    return savedLocale;
  }

  // Normalize browser language
  return normalizeLocale(window.navigator.language);
}

init({
  fallbackLocale: defaultLocale,
  initialLocale: getInitialLocale(),
});
