import '$lib/i18n'; // Import to initialize svelte-i18n
import { waitLocale } from 'svelte-i18n';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
  // i18n.ts handles locale initialization (localStorage + browser detection)
  await waitLocale();
};
