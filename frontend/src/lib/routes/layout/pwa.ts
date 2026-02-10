export interface PwaRegistrationDeps {
  registerSW: (options: {
    immediate: boolean;
    onNeedRefresh: () => void;
    onOfflineReady: () => void;
    onRegisteredSW: (swUrl: string, registration: ServiceWorkerRegistration | undefined) => void;
    onRegisterError: (error: unknown) => void;
  }) => (reloadPage?: boolean) => void;
  isDirty: () => boolean;
  getPendingCount: () => number;
  confirm: (message: string) => boolean;
  updateMessage: () => string;
  setIntervalHandle: (handle: ReturnType<typeof setInterval> | null) => void;
  getIntervalHandle: () => ReturnType<typeof setInterval> | null;
}

export function registerPwaUpdates(deps: PwaRegistrationDeps) {
  const updateSW = deps.registerSW({
    immediate: true,
    onNeedRefresh() {
      const isDirty = deps.isDirty();
      const hasPending = deps.getPendingCount() > 0;

      if (isDirty || hasPending) {
        console.log('[PWA] Update available, but waiting for save/sync...');

        const existing = deps.getIntervalHandle();
        if (existing !== null) {
          clearInterval(existing);
        }

        deps.setIntervalHandle(
          setInterval(() => {
            if (!deps.isDirty() && deps.getPendingCount() === 0) {
              const current = deps.getIntervalHandle();
              if (current !== null) {
                clearInterval(current);
              }
              deps.setIntervalHandle(null);
              promptForUpdate();
            }
          }, 1000)
        );
      } else {
        promptForUpdate();
      }

      function promptForUpdate() {
        const shouldUpdate = deps.confirm(deps.updateMessage());
        if (shouldUpdate) {
          updateSW(true);
        }
      }
    },
    onOfflineReady() {
      console.log('[PWA] App ready to work offline');
    },
    onRegisteredSW(swUrl) {
      console.log('[PWA] Service Worker registered:', swUrl);
    },
    onRegisterError(error) {
      console.error('[PWA] Service Worker registration failed:', error);
    },
  });
}
