/**
 * Shared relative time formatting utility.
 * Uses i18n translation keys from page.time.* namespace.
 *
 * @param dateString - ISO 8601 date string
 * @param t - svelte-i18n translation function ($_)
 * @returns Localized relative time string
 */
import type { FormatXMLElementFn } from 'intl-messageformat';

type InterpolationValues =
  | Record<
      string,
      string | number | boolean | Date | FormatXMLElementFn<unknown> | null | undefined
    >
  | undefined;

type MessageObject = {
  id: string;
  locale?: string;
  format?: string;
  default?: string;
  values?: InterpolationValues;
};

type TranslateFn = (id: string | MessageObject, options?: Omit<MessageObject, 'id'>) => string;

export function formatRelativeTime(dateString: string, t: TranslateFn): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return t('page.time.just_now');
  if (diffMins < 60) return t('page.time.minutes_ago', { values: { count: diffMins } });
  if (diffHours < 24) return t('page.time.hours_ago', { values: { count: diffHours } });
  if (diffDays === 1) return t('page.time.yesterday');
  if (diffDays < 7) return t('page.time.days_ago', { values: { count: diffDays } });

  return date.toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
}
