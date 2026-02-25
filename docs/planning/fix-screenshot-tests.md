# Fix: Playwright Screenshot-Tests — richtige Seiten + Inhalte

> Status: Plan
> Zuletzt aktualisiert: 2026-02-24

## Problem

Die Screenshot-Tests (`frontend/tests/e2e/ui-consistency-screenshots.spec.ts`) haben
mehrere Probleme, die dazu führen, dass Screenshots falsche oder leere Seiten zeigen:

| Seite | IST | SOLL |
|-------|-----|------|
| `/recipes` | Home-Screen (Redirect) | Rezeptliste mit 3+ Rezepten |
| `/journal` | "Encryption locked" Meldung | Journal mit Kalender + Einträgen |
| `/graph` | 2 Nodes (E2E Test Note + Wikilink) | 6+ Nodes mit Kanten |
| `/note/{id}` (Editor) | Minimaler Content | Reichhaltiger Markdown-Content |
| `/note/{id}` (Rezept-Tabs) | Nicht vorhanden | 3 Tabs: Zutaten, Anleitung, Vorschau |
| Home/Settings/Encryption/Migration | OK | OK (bleiben unverändert) |

### Ursachen

1. **`/recipes` Race Condition (Production-Bug):** Feature-Guard in `onMount` prüft
   `featureEnabled` synchron, bevor `loadRecipeFeature()` (Sidebar, async) abgeschlossen
   ist. Default = `false` → sofortiger Redirect zu `/`.
2. **Journal Encryption:** Test-User hat Encryption nie unlocked → `isEncryptionLocked = true`
   → Lock-Screen statt Inhalte.
3. **Fehlende Testdaten:** Auth-Fixture erstellt nur eine Note mit einem Wikilink. Keine
   Rezepte, keine Journal-Einträge.

---

## Implementierung in 3 Phasen

### Phase 1: Recipe Feature Guard + Data Seeding + Waits (niedrigstes Risiko)

#### 1a. Production-Fix: Recipe-Page Feature Guard

**Datei:** `frontend/src/routes/recipes/+page.svelte`

Aktuell (Zeilen 63, 71–75):
```typescript
const featureEnabled = $derived(features.getRecipeFeatureEnabled());
onMount(async () => {
  if (!featureEnabled) { goto('/'); return; }     // BUG
  await Promise.all([recipes.loadRecipes(), ...]);
});
```

Fix — Pattern der Journal-Seite (`journal/+page.svelte:121–165`) übernehmen:
```typescript
const featureEnabled = $derived(features.getRecipeFeatureEnabled());
const featureLoaded = $derived(features.getRecipeFeatureLoaded()); // existiert bereits

let dataLoaded = false;

$effect(() => {
  if (featureLoaded && !featureEnabled) {
    goto('/');
  }
});

$effect(() => {
  if (featureLoaded && featureEnabled && !dataLoaded) {
    dataLoaded = true;
    Promise.all([
      recipes.loadRecipes(),
      recipes.loadCollections(),
      recipes.loadSharedRecipes(),
    ]);
  }
});
```

Template-Änderung: `{#if loading}` → `{#if !featureLoaded || loading}`.

**Edge Case:** Wenn der API-Call für `loadRecipeFeature` fehlschlägt, setzt der Store
`recipeFeatureEnabled = false` + `recipeFeatureLoaded = true` → Redirect zu `/`. Das ist
das gleiche Verhalten wie bei Journal und akzeptabel (Feature tatsächlich nicht verfügbar).

#### 1b. Auth-Fixture erweitern

**Datei:** `frontend/tests/fixtures/auth.fixture.ts`

Interface erweitern:
```typescript
interface AuthContext {
  page: Page;
  testNoteId: string;
  baseURL: string;
  credentials: TestCredentials;  // für Encryption-Setup in Phase 2
}
```

Test-Note mit reichhaltigerem Content erstellen:
```typescript
data: {
  title: 'E2E Test Note',
  content: [
    '# Project Notes',
    '',
    '## Architecture',
    'The application uses a **Go backend** with an **SQLite database**.',
    '',
    '### Key Components',
    '- REST API with Chi router',
    '- SvelteKit frontend with Svelte 5 Runes',
    '- End-to-end encryption',
    '',
    '## Links',
    '[[Architecture Overview]] | [[API Design]]',
    '',
    '```go',
    'func main() {',
    '    r := chi.NewRouter()',
    '    r.Get("/api/notes", handlers.ListNotes)',
    '}',
    '```',
    '',
    '> This is a test note for visual regression screenshots.',
  ].join('\n'),
  folder_path: '/',
}
```

#### 1c. Test-Daten Seeding

**Datei:** `frontend/tests/e2e/ui-consistency-screenshots.spec.ts`

Neue Helper-Funktion `seedTestData()`, aufgerufen am Anfang beider Tests. Nutzt
`page.request` (wie Auth-Fixture) mit CSRF-Token aus Cookies.

**CSRF-Token Handling** (wie in `auth.fixture.ts:41–49`):
```typescript
const cookies = await page.context().cookies(baseURL);
const csrfToken = cookies.find((c) => c.name === 'csrf_token')?.value;
const headers: Record<string, string> = {
  'Content-Type': 'application/json',
  'X-Forwarded-For': '203.0.113.42',
};
if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
```

**Graph-Notizen (4 Stück mit gegenseitigen Wikilinks):**

Wikilinks verweisen auf Titel, nicht IDs. Alle 4 Notes in einem Durchgang erstellen — die
`[[Titel]]`-Referenzen müssen nicht auf existierende Notes zeigen, da der Graph auch
unresolved Links als Nodes darstellt. Wenn alle 4 Notes existieren, werden die Links
automatisch aufgelöst.

```
POST /api/notes × 4:
- "Project Planning"      → content: "... [[Architecture Overview]] [[API Design]] ..."
- "Architecture Overview"  → content: "... [[Project Planning]] [[Database Schema]] ..."
- "API Design"             → content: "... [[Project Planning]] [[Architecture Overview]] ..."
- "Database Schema"        → content: "... [[Architecture Overview]] [[API Design]] ..."
```

**Rezepte (3 Stück):**

```
POST /api/notes mit note_type: "recipe" × 3, dann:
PUT /api/recipes/{id}/metadata (expected_updated_at aus Note-Response)
PUT /api/recipes/{id}/ingredients (expected_updated_at aus Metadata-Response)
```

| Rezept | Portionen | Prep | Cook | Difficulty |
|--------|-----------|------|------|------------|
| Spaghetti Carbonara | 4 | 15 | 20 | easy |
| Chicken Tikka Masala | 6 | 30 | 45 | medium |
| Chocolate Lava Cake | 2 | 20 | 12 | hard |

Jeweils 3–5 Zutaten mit `amount`, `unit`, `name`, `group_name`.

**Hinweis:** Die erste erstellte Recipe-ID als `testRecipeId` speichern für
Rezept-Tab-Screenshots (Phase 3).

#### 1d. Route-spezifische Waits verbessern

**Datei:** `frontend/tests/e2e/ui-consistency-screenshots.spec.ts`

In `captureOnNewPage()` nach `waitForAppReady()` route-spezifische Waits hinzufügen.
Fehler werden geloggt (nicht geschluckt), damit Debugging möglich bleibt:

```typescript
const ROUTE_READY_HINTS: Record<string, { selector: string; label: string }> = {
  '/recipes':             { selector: '[role="tab"]', label: 'recipe tabs' },
  '/journal':             { selector: '.journal-heatmap, [role="button"]', label: 'journal UI' },
  '/graph':               { selector: 'canvas', label: 'graph canvas' },
};

// In captureOnNewPage, nach waitForAppReady:
const hint = Object.entries(ROUTE_READY_HINTS).find(([r]) => route.startsWith(r));
if (hint) {
  await page.waitForSelector(hint[1].selector, { timeout: 10000 })
    .catch((err) => console.warn(`[Screenshot] ${route}: ${hint[1].label} not found: ${err.message}`));
}
```

Bestehende Screenshots (Home, Settings, etc.) sind nicht betroffen, da sie keinen Eintrag
in `ROUTE_READY_HINTS` haben.

#### 1e. `waitForAppReady` erweitern

Auch auf `'Loading...'` prüfen (englische Locale im Default-Variant):
```typescript
return !text.includes('Laden...') && !text.includes('Loading...');
```

#### Verifizierung Phase 1

```bash
cd frontend && npx playwright test ui-consistency-screenshots --project=e2e
```

Erwartetes Ergebnis:
- `/recipes` zeigt Rezeptliste mit 3 Rezepten (nicht mehr Home!)
- `/graph` zeigt 6+ Nodes
- `/note/{id}` Editor zeigt reichhaltigen Content
- `/journal` zeigt weiterhin "Encryption locked" (wird in Phase 2 gelöst)

---

### Phase 2: Encryption-Unlock für Journal (mittleres Risiko)

#### Ansatz

Nach Auth-Fixture-Setup (Login via API + Navigation zu Home), auf der **initialen Seite**
Encryption programmatisch unlocken:

1. User-Daten von `/api/auth/me` holen → `encryption_salt` (base64) + `id`
   - **Bestätigt:** Frische User haben IMMER einen Salt (wird bei Registrierung generiert,
     `auth_register.go:85`)
2. Via `page.evaluate` + dynamischem Import `setupEncryption()` aufrufen
3. KEK wird in IndexedDB persistiert (`kek-persistence.ts:persistKEK`)
4. Neue Pages in `captureOnNewPage()` stellen KEK automatisch wieder her:
   - Layout-Init Phase 2 → `tryRestoreKEK(userId)` (`initialize.ts:216`)
   - IndexedDB ist pro Browser-Context geteilt
   - `tryRestoreKEK` setzt `isUnlocked = true` im Encryption-Store
   - Journal-`$effect` reagiert reaktiv → lädt Einträge

#### Primärer Weg: Dynamic Import in page.evaluate

```typescript
// Nach initialem Page-Load in der Fixture/Test
const meResponse = await page.request.get(`${baseURL}/api/auth/me`);
const userData = await meResponse.json();

await page.evaluate(async ({ password, userId, saltBase64 }) => {
  const enc = await import('/src/lib/stores/encryption.svelte.ts');
  const binary = atob(saltBase64);
  const salt = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) salt[i] = binary.charCodeAt(i);
  await enc.setupEncryption(password, userId, salt, 'balanced');
}, {
  password: credentials.password,
  userId: userData.id,
  saltBase64: userData.encryption_salt,
});
```

**Warum das funktionieren sollte:** In Vite Dev Mode werden ES-Module per URL gecacht.
Der App-Code importiert `$lib/stores/encryption.svelte` → Vite löst das zu
`/src/lib/stores/encryption.svelte.ts` auf. Ein `import()` desselben Pfads in
`page.evaluate` trifft den gleichen Browser-Module-Cache → selbe Instanz → selber
`$state`-Wert.

**Risiko:** Wenn Vite die URL anders auflöst (z.B. `/@id/...`), entsteht eine zweite
Modul-Instanz. `isUnlocked` wäre dann in der falschen Instanz `true`.

#### Fallback: Dev-Mode Window-Helper

Falls der Dynamic Import eine separate Modul-Instanz erzeugt, minimaler Helper
in `+layout.svelte`:

```typescript
// In +layout.svelte, nach den bestehenden Imports
if (import.meta.env.DEV) {
  (window as any).__testSetupEncryption = encryption.setupEncryption;
}
```

Test-Code:
```typescript
await page.evaluate(async ({ password, userId, salt }) => {
  const binary = atob(salt);
  const saltArr = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) saltArr[i] = binary.charCodeAt(i);
  await (window as any).__testSetupEncryption(password, userId, saltArr, 'balanced');
}, { password, userId, salt: saltBase64 });
```

Vorteil: Garantiert dieselbe Modul-Instanz, da es die bereits importierte Funktion ist.
Nachteil: Neues Pattern im Projekt (kein bestehendes `window.__test`-Pattern).
Sicherheit: Nur in DEV-Mode, wird in Production-Builds wegoptimiert.

#### Timing für KEK-Restore auf neuen Pages

`tryRestoreKEK` läuft in Layout-Init Phase 2 (nach first paint, `initialize.ts:216`).
Der Journal-`$effect` ist reaktiv und feuert erneut wenn `isUnlocked` sich ändert.
Trotzdem muss der route-spezifische Wait für `/journal` ausreichend sein:

```typescript
'/journal': {
  selector: '.journal-heatmap, .journal-entries, [data-journal-loaded]',
  label: 'journal content',
  timeout: 15000,  // Mehr Zeit für KEK-Restore + Daten-Laden
}
```

#### Journal-Testdaten

3 Journal-Einträge via API erstellen (nach Encryption-Unlock nicht nötig —
Journal-Einträge werden unverschlüsselt erstellt, die Seite zeigt sie aber nur wenn
Encryption unlocked ist):

```
POST /api/notes × 3:
- note_type: "journal", journal_date: "2026-02-24", title: "Productive Monday", content: "..."
- note_type: "journal", journal_date: "2026-02-23", title: "Weekend Review", content: "..."
- note_type: "journal", journal_date: "2026-02-22", title: "Project Kickoff", content: "..."
```

#### Verifizierung Phase 2

Screenshots prüfen:
- `journal--en-gruvbox-light.png` → Kalender + Heatmap + 3 Einträge (nicht Lock-Screen!)
- `journal-mobile--en-gruvbox-light.png` → Mobile Journal mit Einträgen

Falls Dynamic Import nicht funktioniert → Fallback implementieren und dokumentieren.

---

### Phase 3: Rezept-Tab-Screenshots (niedrigstes Risiko, aufbauend auf Phase 1)

#### Ansatz

Rezept-Tabs (ingredients, instructions, preview) sind **Tabs innerhalb von `/note/{id}`**,
keine separaten Routes (`RecipeEditor.svelte:29`). URL ändert sich nicht beim Tab-Wechsel.

Eigene Capture-Funktion (keine Erweiterung von `captureOnNewPage`):

```typescript
async function captureRecipeTabs(
  context: BrowserContext,
  recipeId: string,
  outDir: string,
  viewport: { width: number; height: number },
  variant: UiVariant = DEFAULT_VARIANT
): Promise<string[]> {
  const page = await context.newPage();
  const failures: string[] = [];
  try {
    await page.setViewportSize(viewport);
    await applyUiVariant(page, variant);
    await page.goto(`/note/${recipeId}`, { waitUntil: 'domcontentloaded' });
    await waitForAppReady(page);

    // Warte auf Recipe-Tab-Slider (role="tablist")
    await page.waitForSelector('[role="tablist"]', { timeout: 10000 });

    const tabs = ['ingredients', 'instructions', 'preview'] as const;
    for (const tab of tabs) {
      // Klick auf den Tab-Button via aria-selected Pattern
      const tabButtons = page.locator('[role="tab"]');
      const tabIndex = tabs.indexOf(tab);
      await tabButtons.nth(tabIndex).click();
      await page.waitForTimeout(500); // Tab-Animation abwarten

      const filename = `recipe-${tab}--${variant.suffix}.png`;
      try {
        await page.screenshot({ path: path.join(outDir, filename), animations: 'disabled' });
      } catch {
        failures.push(`recipe-${tab}: screenshot failed`);
      }
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error);
    failures.push(`recipe-tabs: ${msg}`);
  } finally {
    await page.close().catch(() => {});
  }
  return failures;
}
```

Aufrufen in beiden Test-Funktionen (Desktop + Mobile), nach den bestehenden Screenshots:
```typescript
// Desktop
const recipeTabFailures = await captureRecipeTabs(
  context, testRecipeId, outDir, { width: 1440, height: 900 }, DEFAULT_VARIANT
);
failures.push(...recipeTabFailures);

// Mobile
const recipeTabFailures = await captureRecipeTabs(
  context, testRecipeId, mobileOutDir, { width: 393, height: 852 }, DEFAULT_VARIANT
);
failures.push(...recipeTabFailures);
```

#### Neue Screenshots

| Datei | Inhalt |
|-------|--------|
| `recipe-ingredients--{suffix}.png` | Zutaten-Tab mit Metadata + Ingredients |
| `recipe-instructions--{suffix}.png` | Anleitungs-Tab mit Markdown-Editor |
| `recipe-preview--{suffix}.png` | Vorschau-Tab mit formatiertem Rezept |
| Jeweils Desktop (1440×900) + Mobile (393×852) | |

---

## Betroffene Dateien

| Datei | Phase | Änderung |
|-------|-------|----------|
| `frontend/src/routes/recipes/+page.svelte` | 1 | Feature-Guard fixen (Production-Bug) |
| `frontend/tests/fixtures/auth.fixture.ts` | 1 | baseURL + credentials + reicherer Content |
| `frontend/tests/e2e/ui-consistency-screenshots.spec.ts` | 1–3 | Seeding, Waits, Tabs |
| `frontend/src/routes/+layout.svelte` | 2* | Nur falls Fallback nötig (window helper) |

*Nur wenn der primäre Dynamic-Import-Ansatz nicht funktioniert.

## Risikomatrix

| Risiko | Schwere | Wahrsch. | Mitigation |
|--------|---------|----------|------------|
| Dynamic Import erzeugt separate Modul-Instanz | Hoch | Mittel | Fallback: window helper in DEV |
| KEK-Restore Timing zu langsam für Screenshot | Mittel | Mittel | Erhöhter Timeout (15s) für Journal |
| Recipe Feature Guard Fix hat Seiteneffekte | Niedrig | Niedrig | Identisches Pattern wie Journal (bewährt) |
| CSRF-Token fehlt beim Seeding | Mittel | Niedrig | Explizit aus Cookies extrahieren |
| Tab-Animation nicht abgeschlossen bei Screenshot | Niedrig | Mittel | 500ms Wait + `animations: 'disabled'` |

## Rollback

- Phase 1: Recipe Guard Fix ist ein eigenständiger Commit, revertierbar
- Phase 2: Encryption-Unlock ist Test-only, kein Produktionscode betroffen (außer Fallback)
- Phase 3: Additive Änderung, bestehende Screenshots nicht betroffen
