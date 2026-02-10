// Network status store for offline detection
// Using Svelte 5 runes

import { startSync, refreshPendingCount, getPendingCount } from '$lib/offline/sync-manager.svelte';

let isOnline = $state(typeof navigator !== 'undefined' ? navigator.onLine : true);
let showOfflineBanner = $state(false);

// Getters
export function getIsOnline() {
  return isOnline;
}

export function getShowOfflineBanner() {
  return showOfflineBanner;
}

// Setup online/offline event listeners
if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    isOnline = true;
    showOfflineBanner = false;
    console.log('Network: Online');

    // Trigger sync after 1s delay (let network stabilize)
    setTimeout(async () => {
      try {
        await refreshPendingCount();
        if (getPendingCount() > 0) {
          console.log('[Network] Online - starting sync for pending operations');
          await startSync();
        }
      } catch (err) {
        console.error('[Network] Sync trigger failed:', err);
      }
    }, 1000);
  });

  window.addEventListener('offline', () => {
    isOnline = false;
    showOfflineBanner = true;
    console.log('Network: Offline');
  });

  // Multi-tab consistency: refresh pending count when tab becomes visible
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible' && navigator.onLine) {
      refreshPendingCount().catch(() => {});
    }
  });
}

// Manual setters (for testing)
export function setIsOnline(online: boolean) {
  isOnline = online;
  showOfflineBanner = !online;
}
