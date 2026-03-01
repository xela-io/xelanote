// Background Sync handler for xelanote offline queue.
// Injected into the Workbox-generated SW via importScripts.
// When the browser fires a 'sync' event (Chromium only), we notify
// all open clients to replay the IndexedDB queue. If no clients are
// open, the sync will happen next time the user opens the app.

self.addEventListener('sync', (event) => {
  if (event.tag === 'offline-queue') {
    event.waitUntil(
      self.clients.matchAll({ type: 'window' }).then((clients) => {
        for (const client of clients) {
          client.postMessage({ type: 'REPLAY_OFFLINE_QUEUE' });
        }
      })
    );
  }
});
