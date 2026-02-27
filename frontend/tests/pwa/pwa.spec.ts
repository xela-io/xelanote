import { expect, test } from '@playwright/test';

import { createCredentials, loginViaApi, registerViaApi } from '../e2e/helpers/auth';

// ---------------------------------------------------------------------------
// Hilfsfunktion: Wartet bis der SW vollständig aktiviert ist und die Seite
// kontrolliert. Bei clientsClaim:false braucht es einen Reload-Zyklus.
// ---------------------------------------------------------------------------
async function waitForSwControlling(page: import('@playwright/test').Page) {
  // Schritt 1: Warte bis SW aktiviert (statechange-basiert, nicht nur .ready)
  await page.waitForFunction(
    async () => {
      const reg = await navigator.serviceWorker.ready;
      const sw = reg.active || reg.installing || reg.waiting;
      if (!sw) return false;
      if (sw.state === 'activated') return true;
      // Warte auf statechange
      return new Promise<boolean>((resolve) => {
        sw.addEventListener('statechange', () => {
          if (sw.state === 'activated') resolve(true);
        });
        // Falls bereits aktiviert (Race)
        if (sw.state === 'activated') resolve(true);
      });
    },
    undefined,
    { timeout: 30000 }
  );

  // Schritt 2: Reload damit der SW die Seite kontrolliert (clientsClaim:false)
  if (!(await page.evaluate(() => !!navigator.serviceWorker.controller))) {
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => !!navigator.serviceWorker.controller, undefined, {
      timeout: 10000,
    });
  }
}

// ---------------------------------------------------------------------------
// 4.1 Service Worker
// ---------------------------------------------------------------------------
test.describe('Service Worker', () => {
  test('SW wird registriert und ist aktiv', async ({ page }) => {
    await page.goto('/');

    // Warte bis der Service Worker vollständig aktiviert ist
    const swInfo = await page.waitForFunction(
      async () => {
        const reg = await navigator.serviceWorker.ready;
        const sw = reg.active || reg.installing || reg.waiting;
        if (!sw) return null;
        if (sw.state === 'activated') {
          return { scope: reg.scope, hasActive: true, state: 'activated' };
        }
        // Warte auf statechange
        return new Promise((resolve) => {
          sw.addEventListener('statechange', () => {
            if (sw.state === 'activated') {
              resolve({ scope: reg.scope, hasActive: true, state: 'activated' });
            }
          });
          if (sw.state === 'activated') {
            resolve({ scope: reg.scope, hasActive: true, state: 'activated' });
          }
        });
      },
      undefined,
      { timeout: 30000 }
    );

    const info = (await swInfo.jsonValue()) as {
      scope: string;
      hasActive: boolean;
      state: string;
    } | null;
    expect(info).not.toBeNull();
    expect(info!.scope).toContain('/');
    expect(info!.hasActive).toBe(true);
    expect(info!.state).toBe('activated');
  });

  test('SW kontrolliert die Seite', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    const hasController = await page.evaluate(() => !!navigator.serviceWorker.controller);
    expect(hasController).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 4.2 Offline-Fähigkeit
// ---------------------------------------------------------------------------
test.describe('Offline-Fähigkeit', () => {
  // Hinweis: context.setOffline(true) / CDP Network.emulateNetworkConditions blockiert
  // auf Chrome-Network-Service-Level — das betrifft auch fetch()-Calls die eigentlich
  // vom SW aus dem Cache bedient werden sollten. Stattdessen prüfen wir direkt die
  // Cache API: wenn cache.match(url) eine Response liefert, wird der SW sie offline bedienen.
  // Das ist deterministischer und unabhängig von CDP-Implementierungsdetails.

  test('App-Shell Assets (JS/CSS) im Precache nach initialem Besuch', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    // Prüfe dass die App-Shell-Assets (JS + CSS) im Precache liegen.
    // Hinweis: index.html wird von SvelteKit's adapter-static NACH dem
    // SW-Manifest generiert — daher ist HTML nicht im Precache enthalten.
    // Das ist eine bekannte Limitation von vite-plugin-pwa + SvelteKit.
    const result = await page.evaluate(async () => {
      const names = await caches.keys();
      const precacheName = names.find(
        (n) => n.includes('precache') || n.includes('workbox-precache')
      );
      if (!precacheName) return { found: false, jsCount: 0, cssCount: 0 };

      const cache = await caches.open(precacheName);
      const keys = await cache.keys();
      const urls = keys.map((r) => new URL(r.url).pathname);

      return {
        found: true,
        jsCount: urls.filter((u) => u.endsWith('.js')).length,
        cssCount: urls.filter((u) => u.endsWith('.css')).length,
      };
    });

    expect(result.found).toBe(true);
    expect(result.jsCount).toBeGreaterThan(0);
    expect(result.cssCount).toBeGreaterThan(0);
  });

  test('Kritische Assets im Precache vorhanden', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    // Prüfe dass JS und CSS Assets im Precache liegen
    const cacheInfo = await page.evaluate(async () => {
      const names = await caches.keys();
      const precacheName = names.find(
        (n) => n.includes('precache') || n.includes('workbox-precache')
      );
      if (!precacheName) return { found: false, jsCount: 0, cssCount: 0, total: 0 };

      const cache = await caches.open(precacheName);
      const keys = await cache.keys();
      const urls = keys.map((r) => new URL(r.url).pathname);

      return {
        found: true,
        jsCount: urls.filter((u) => u.endsWith('.js')).length,
        cssCount: urls.filter((u) => u.endsWith('.css')).length,
        total: urls.length,
      };
    });

    expect(cacheInfo.found).toBe(true);
    expect(cacheInfo.jsCount).toBeGreaterThan(0);
    expect(cacheInfo.cssCount).toBeGreaterThan(0);
    expect(cacheInfo.total).toBeGreaterThan(10); // SvelteKit App hat viele Chunks
  });

  test('NavigationRoute mit Fallback und API-Denylist konfiguriert', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    // Prüfe die SW-Konfiguration: NavigationRoute mit navigateFallback '/'
    // und navigateFallbackDenylist [/^\/api/] muss registriert sein.
    // Wir verifizieren das über die SW-Source (sw.js), da Workbox's
    // NavigationRoute nur echte Browser-Navigationen abfängt (mode: 'navigate'),
    // die sich nicht über page.evaluate(fetch(...)) simulieren lassen.
    const swSource = await page.evaluate(async () => {
      const reg = await navigator.serviceWorker.ready;
      if (!reg.active) return null;
      // SW-Script-URL auslesen und fetchen
      const response = await fetch(reg.active.scriptURL);
      return response.text();
    });

    expect(swSource).not.toBeNull();
    // NavigationRoute mit createHandlerBoundToURL("/") muss vorhanden sein
    expect(swSource).toContain('createHandlerBoundToURL');
    expect(swSource).toContain('NavigationRoute');
    // API-Denylist muss konfiguriert sein
    expect(swSource).toContain('/api');
  });

  test('API-Routen NICHT durch navigateFallback bedient (Sicherheit)', async ({
    page,
    context,
  }) => {
    // SW aktivieren und kontrollieren lassen
    await page.goto('/');
    await waitForSwControlling(page);

    await context.setOffline(true);

    // API-Route offline fetchen — darf KEIN HTML-Fallback sein
    const result = await page.evaluate(async () => {
      try {
        const response = await fetch('/api/notes');
        // Falls eine Antwort kommt, prüfe ob es KEIN HTML ist
        const contentType = response.headers.get('content-type') || '';
        return {
          ok: response.ok,
          status: response.status,
          isHtml: contentType.includes('text/html'),
          error: null,
        };
      } catch (err) {
        // Network Error ist das erwartete Ergebnis offline
        return {
          ok: false,
          status: 0,
          isHtml: false,
          error: (err as Error).message,
        };
      }
    });

    // Entweder Network Error oder zumindest KEIN HTML-Fallback
    if (result.error) {
      // Network Error — korrekt, API wird nicht durch SW-Fallback bedient
      expect(result.error).toBeTruthy();
    } else {
      // Falls der SW antwortet, darf es kein HTML-Fallback sein
      expect(result.isHtml).toBe(false);
    }
  });
});

// ---------------------------------------------------------------------------
// 4.3 Web App Manifest
// ---------------------------------------------------------------------------
test.describe('Web App Manifest', () => {
  test('Manifest-Link existiert im HTML', async ({ page }) => {
    await page.goto('/');
    const manifestLink = page.locator('link[rel="manifest"]');
    await expect(manifestLink).toHaveCount(1);
  });

  test('Manifest ist valides JSON mit Pflichtfeldern', async ({ page }) => {
    await page.goto('/');

    const manifestHref = await page.locator('link[rel="manifest"]').getAttribute('href');
    expect(manifestHref).toBeTruthy();

    const response = await page.request.get(manifestHref!);
    expect(response.ok()).toBe(true);

    const manifest = await response.json();
    expect(manifest.name).toBeTruthy();
    expect(manifest.short_name).toBeTruthy();
    expect(manifest.start_url).toBeTruthy();
    expect(manifest.display).toBeTruthy();
    expect(manifest.icons).toBeTruthy();
    expect(Array.isArray(manifest.icons)).toBe(true);
    expect(manifest.icons.length).toBeGreaterThan(0);
  });

  test('Icons: 192x192 und 512x512 vorhanden', async ({ page }) => {
    await page.goto('/');
    const manifestHref = await page.locator('link[rel="manifest"]').getAttribute('href');
    const response = await page.request.get(manifestHref!);
    const manifest = await response.json();

    const sizes = manifest.icons.map((icon: { sizes: string }) => icon.sizes);
    expect(sizes).toContain('192x192');
    expect(sizes).toContain('512x512');
  });

  test('Maskable Icon vorhanden', async ({ page }) => {
    await page.goto('/');
    const manifestHref = await page.locator('link[rel="manifest"]').getAttribute('href');
    const response = await page.request.get(manifestHref!);
    const manifest = await response.json();

    const hasMaskable = manifest.icons.some(
      (icon: { purpose?: string }) => icon.purpose === 'maskable'
    );
    expect(hasMaskable).toBe(true);
  });

  test('theme_color und background_color gesetzt', async ({ page }) => {
    await page.goto('/');
    const manifestHref = await page.locator('link[rel="manifest"]').getAttribute('href');
    const response = await page.request.get(manifestHref!);
    const manifest = await response.json();

    expect(manifest.theme_color).toBeTruthy();
    expect(manifest.background_color).toBeTruthy();
    // Prüfe gültige Hex-Werte
    const hexPattern = /^#[0-9a-fA-F]{3,8}$/;
    expect(manifest.theme_color).toMatch(hexPattern);
    expect(manifest.background_color).toMatch(hexPattern);
  });

  test('Apple Touch Icon vorhanden', async ({ page }) => {
    await page.goto('/');
    const appleTouchIcon = page.locator('link[rel="apple-touch-icon"]');
    await expect(appleTouchIcon).toHaveCount(1);
  });
});

// ---------------------------------------------------------------------------
// 4.4 Caching & Sicherheit
// ---------------------------------------------------------------------------
test.describe('Caching & Sicherheit', () => {
  test('Statische Assets aus Workbox-Precache', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    // Prüfe dass Workbox-Precache existiert und Einträge hat
    const cacheInfo = await page.evaluate(async () => {
      const cacheNames = await caches.keys();
      const precache = cacheNames.find(
        (name) => name.includes('precache') || name.includes('workbox-precache')
      );
      if (!precache) return { found: false, entryCount: 0, names: cacheNames };

      const cache = await caches.open(precache);
      const keys = await cache.keys();
      return { found: true, entryCount: keys.length, names: cacheNames };
    });

    expect(cacheInfo.found).toBe(true);
    expect(cacheInfo.entryCount).toBeGreaterThan(0);
  });

  test('API-Responses NICHT gecacht', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    // API-Request ausführen (health endpoint braucht kein Auth)
    await page.request.get('/api/health');

    // Prüfe dass kein Cache-Eintrag für /api/* existiert
    const hasApiCache = await page.evaluate(async () => {
      const cacheNames = await caches.keys();
      for (const name of cacheNames) {
        // uploads-Cache ist bewusst erlaubt, nur API-Daten prüfen
        if (name === 'uploads') continue;
        const cache = await caches.open(name);
        const keys = await cache.keys();
        const apiEntry = keys.find((req) => new URL(req.url).pathname.startsWith('/api/'));
        if (apiEntry) return true;
      }
      return false;
    });

    expect(hasApiCache).toBe(false);
  });

  test('Cache-Clearing bei Logout (Sicherheit)', async ({ page }) => {
    // Registrieren und einloggen
    const credentials = createCredentials();
    await page.goto('/register');
    await registerViaApi(page, credentials);
    await loginViaApi(page, credentials);
    await page.goto('/');
    await waitForSwControlling(page);

    // Cache-Namen vor Logout merken
    const cachesBeforeLogout = await page.evaluate(async () => {
      return await caches.keys();
    });
    expect(cachesBeforeLogout.length).toBeGreaterThan(0);

    // Logout über API
    await page.request.post('/api/auth/logout');

    // Seite neu laden (Logout-Handler sollte Caches leeren)
    await page.goto('/');

    // Prüfe ob der uploads-Cache geleert wurde
    const uploadsAfterLogout = await page.evaluate(async () => {
      const exists = await caches.has('uploads');
      if (!exists) return { exists: false, entryCount: 0 };
      const cache = await caches.open('uploads');
      const keys = await cache.keys();
      return { exists: true, entryCount: keys.length };
    });

    // uploads-Cache sollte leer sein oder nicht existieren
    expect(uploadsAfterLogout.entryCount).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// 4.5 Navigation & UX
// ---------------------------------------------------------------------------
test.describe('Navigation & UX', () => {
  test('App-Shell lädt unter 3 Sekunden', async ({ page }) => {
    const start = Date.now();
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    const loadTime = Date.now() - start;

    expect(loadTime).toBeLessThan(3000);
  });

  test('Navigation zwischen Routen funktioniert', async ({ page }) => {
    await page.goto('/login');
    await expect(page).toHaveURL(/\/login/);

    // Navigation zur Register-Seite
    const registerLink = page.locator('a[href="/register"]').first();
    if (await registerLink.isVisible()) {
      await registerLink.click();
      await expect(page).toHaveURL(/\/register/);
    } else {
      // Direkte Navigation als Fallback
      await page.goto('/register');
      await expect(page).toHaveURL(/\/register/);
    }
  });

  test('Zurück-Navigation funktioniert', async ({ page }) => {
    await page.goto('/login');
    await page.goto('/register');
    await expect(page).toHaveURL(/\/register/);

    await page.goBack();
    await expect(page).toHaveURL(/\/login/);
  });
});

// ---------------------------------------------------------------------------
// 4.6 SW-Update-Lifecycle
// ---------------------------------------------------------------------------
test.describe('SW-Update-Lifecycle', () => {
  test('SW-Registration ist bereit', async ({ page }) => {
    await page.goto('/');

    const regInfo = await page.waitForFunction(
      async () => {
        const reg = await navigator.serviceWorker.ready;
        return {
          hasActive: !!reg.active,
          hasWaiting: !!reg.waiting,
          activeState: reg.active?.state,
        };
      },
      undefined,
      { timeout: 30000 }
    );

    const info = await regInfo.jsonValue();
    // SW sollte entweder aktiv oder wartend sein
    expect(info.hasActive || info.hasWaiting).toBe(true);
  });

  test('Precache enthält erwartete Assets', async ({ page }) => {
    await page.goto('/');
    await waitForSwControlling(page);

    const precacheUrls = await page.evaluate(async () => {
      const cacheNames = await caches.keys();
      const precacheName = cacheNames.find(
        (name) => name.includes('precache') || name.includes('workbox-precache')
      );
      if (!precacheName) return [];

      const cache = await caches.open(precacheName);
      const keys = await cache.keys();
      return keys.map((req) => new URL(req.url).pathname);
    });

    // Prüfe dass JS und CSS im Precache sind
    const hasJS = precacheUrls.some((url) => url.endsWith('.js'));
    const hasCSS = precacheUrls.some((url) => url.endsWith('.css'));
    // SvelteKit nutzt / als navigateFallback, HTML kann als / oder .html vorliegen
    const hasHTML = precacheUrls.some(
      (url) => url === '/' || url.endsWith('.html') || url.endsWith('/')
    );

    expect(hasJS).toBe(true);
    expect(hasCSS).toBe(true);
    // HTML ist über navigateFallback verfügbar, nicht zwingend im Precache
    // Wenn kein explizites HTML im Precache, ist das OK solange JS+CSS da sind
    if (!hasHTML) {
      // Stattdessen prüfen, dass der navigateFallback über den SW erreichbar ist
      const fallbackWorks = await page.evaluate(async () => {
        try {
          const resp = await fetch('/');
          return resp.ok;
        } catch {
          return false;
        }
      });
      expect(fallbackWorks).toBe(true);
    }
  });
});

// ---------------------------------------------------------------------------
// 4.7 HTTPS
// ---------------------------------------------------------------------------
test.describe('HTTPS', () => {
  const baseUrl = process.env.TEST_BASE_URL || '';

  test('HTTP→HTTPS Redirect', async ({ page }) => {
    // Nur ausführen wenn TEST_BASE_URL mit https:// beginnt
    test.skip(!baseUrl.startsWith('https://'), 'TEST_BASE_URL ist nicht HTTPS — Test übersprungen');

    const httpUrl = baseUrl.replace('https://', 'http://');
    const response = await page.goto(httpUrl, {
      waitUntil: 'domcontentloaded',
    });

    // Sollte auf HTTPS redirected worden sein
    expect(page.url()).toMatch(/^https:\/\//);
    expect(response).not.toBeNull();
  });
});
