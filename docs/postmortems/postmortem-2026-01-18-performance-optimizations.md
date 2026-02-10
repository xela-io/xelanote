# Post-Mortem: Performance-Optimierungen (2026-01-18)

## Zusammenfassung

Nach dem Commit `b300113` (Umfassende Performance-Optimierungen) war die Anwendung unbenutzbar. Der Browser fror ein, die Login-Seite erschien nicht, und nach mehreren Fixes zeigten sich weitere Probleme.

**Betroffener Commit:** `b300113 feat: Umfassende Performance-Optimierungen (4 Phasen)`

**Ausfallzeit:** ~2 Stunden Debugging und Fixes

---

## Identifizierte Probleme

### 1. WebSocket Reconnect-Loop (Kritisch)

**Symptom:** Browser friert ein, Konsole zeigt endlose "WebSocket: Connecting..." / "Disconnected 1000" Meldungen.

**Root Cause:**
```svelte
// +layout.svelte - FEHLERHAFT
$effect(() => {
    if (auth.getCurrentUser()) {
        websocket.connect();
        return () => {
            websocket.disconnect();  // ← Cleanup wird bei jedem $effect-Run aufgerufen
        };
    }
});
```

Der `$effect` in Svelte 5 ruft bei jeder Neu-Evaluation zuerst die Cleanup-Funktion auf, dann den neuen Effect-Body. Da `connected` im WebSocket-Store ein `$state` ist, triggert die Änderung von `connected = true` eine Reaktivitätskaskade:

1. `$effect` läuft → `connect()` → `connected = true`
2. Svelte bemerkt State-Änderung → $effect wird neu evaluiert
3. Cleanup `disconnect()` wird aufgerufen → `connected = false`
4. `connect()` wird wieder aufgerufen → Endlosschleife

**Fix:**
- `$effect` für WebSocket entfernt
- WebSocket-Verbindung nur einmal in `onMount()` herstellen
- Zusätzliche Guards (`connecting`, `intentionalDisconnect` Flags) hinzugefügt

**Datei:** `frontend/src/routes/+layout.svelte`, `frontend/src/lib/stores/websocket.svelte.ts`

---

### 2. CodeMirror `EditorState.reconfigure` existiert nicht

**Symptom:** `TypeError: can't access property "of", (intermediate value).reconfigure is undefined`

**Root Cause:**
```typescript
// codemirror.ts - FEHLERHAFT
view.dispatch({
    effects: EditorState.reconfigure.of([...baseExtensions, ...lazyExtensions])
});
```

`EditorState.reconfigure` wurde in neueren CodeMirror 6 Versionen entfernt. Stattdessen muss ein `Compartment` verwendet werden.

**Fix:**
```typescript
// codemirror.ts - KORREKT
import { Compartment } from '@codemirror/state';

const lazyCompartment = new Compartment();
// Im baseExtensions Array:
lazyCompartment.of([])

// Beim Laden:
view.dispatch({
    effects: lazyCompartment.reconfigure(lazyExtensions)
});
```

**Datei:** `frontend/src/lib/editor/codemirror.ts`

---

### 3. Folders API gibt `null` statt leeres Array zurück

**Symptom:** `TypeError: can't access property Symbol.iterator, folders is null`

**Root Cause:**
```typescript
// tree.svelte.ts - FEHLERHAFT
treeData = buildTree(foldersResult.folders, notesResult.notes);
// foldersResult.folders ist null wenn keine Ordner existieren
```

**Fix:**
```typescript
// tree.svelte.ts - KORREKT
const folders = foldersResult.folders ?? [];
const notes = notesResult.notes ?? [];
treeData = buildTree(folders, notes);
```

**Datei:** `frontend/src/lib/stores/tree.svelte.ts`

---

### 4. Notizen ohne Ordner werden nicht angezeigt

**Symptom:** Neue Notizen werden erstellt, erscheinen aber nicht in der Sidebar.

**Root Cause:**
```typescript
// tree.svelte.ts - FEHLERHAFT
const folder = Array.from(folderMap.values()).find(f => f.path === note.folder_path);
if (folder) {
    folder.children.push(noteNode);
}
// Wenn kein Ordner existiert (folderMap ist leer), wird die Notiz nirgends hinzugefügt
```

**Fix:**
```typescript
// tree.svelte.ts - KORREKT
const orphanNotes: NoteTreeNode[] = [];

// In der Schleife:
if (folder) {
    folder.children.push(noteNode);
} else {
    orphanNotes.push(noteNode);
}

// Am Ende:
if (orphanNotes.length > 0) {
    virtualRoot.children.unshift(...orphanNotes);
}
```

**Datei:** `frontend/src/lib/stores/tree.svelte.ts`

---

## Ursachenanalyse: Warum traten diese Fehler auf?

### 1. Unzureichende Tests nach großen Änderungen

Der Commit `b300113` enthielt **10.000+ Zeilen Änderungen** über 31 Dateien. Die Commit-Message enthielt zwar "Verifizierung: ✅ Frontend Build erfolgreich", aber:

- **Kein manueller Smoke-Test** der Anwendung im Browser
- **Keine Integration-Tests** für die neuen Features
- **Build-Erfolg ≠ Runtime-Erfolg**

### 2. Svelte 5 Runes Verständnislücken

Der `$effect` mit Cleanup-Funktion verhält sich anders als erwartet:

```svelte
// Erwartung: Cleanup nur bei Unmount
// Realität: Cleanup bei JEDER Neu-Evaluation des Effects
$effect(() => {
    doSomething();
    return () => cleanup();  // Wird oft aufgerufen!
});
```

**Learning:** `$effect` Cleanups sind für Ressourcen gedacht, die bei jeder Änderung neu erstellt werden müssen (z.B. Event Listener auf einem sich ändernden Element). Für einmalige Verbindungen wie WebSockets ist `onMount` besser geeignet.

### 3. API-Vertrag nicht defensiv gehandhabt

Die Frontend-Code ging davon aus, dass die API immer Arrays zurückgibt:
```typescript
// Annahme: folders ist immer ein Array
foldersResult.folders.map(...)
```

**Learning:** Immer Nullish Coalescing (`??`) oder Default-Werte verwenden:
```typescript
const folders = response.folders ?? [];
```

### 4. Edge Cases nicht bedacht

Der Tree-Builder ging davon aus, dass jede Notiz einen passenden Ordner hat. Bei einem frischen System ohne Ordner fielen Notizen durch das Raster.

**Learning:** Immer an den "leeren Zustand" denken:
- Was passiert ohne Ordner?
- Was passiert ohne Notizen?
- Was passiert bei der ersten Nutzung?

---

## Präventionsmaßnahmen für die Zukunft

### 1. Smoke-Test Checkliste nach großen Änderungen

Vor jedem Commit mit UI-Änderungen:
- [ ] Browser öffnen und Login testen
- [ ] Eine Notiz erstellen und bearbeiten
- [ ] Browser-Konsole auf Fehler prüfen
- [ ] Network-Tab auf fehlgeschlagene Requests prüfen

### 2. Defensive Programmierung

```typescript
// IMMER:
const items = response.items ?? [];
const value = obj?.property ?? defaultValue;

// NICHT:
const items = response.items;  // Kann null/undefined sein!
```

### 3. Svelte 5 $effect Best Practices

```svelte
// Für einmalige Verbindungen → onMount
onMount(() => {
    connect();
    return () => disconnect();
});

// Für reaktive Ressourcen → $effect mit explizitem State-Tracking
$effect(() => {
    const currentValue = someReactiveValue;
    // Effect-Body verwendet currentValue
});
```

### 4. Inkrementelle Commits

Statt eines 10.000-Zeilen-Commits besser aufteilen:
1. Commit: Code Splitting
2. Commit: Virtual Scrolling
3. Commit: Service Worker
4. Commit: WebSocket

Jeder Commit einzeln testbar und bei Problemen leicht revertierbar.

---

## Betroffene Dateien (Fixes)

| Datei | Problem | Fix |
|-------|---------|-----|
| `frontend/src/routes/+layout.svelte` | WebSocket $effect Loop | $effect entfernt, onMount verwendet |
| `frontend/src/lib/stores/websocket.svelte.ts` | Fehlende Guards | `connecting`, `intentionalDisconnect` Flags |
| `frontend/src/lib/editor/codemirror.ts` | reconfigure API | Compartment statt EditorState.reconfigure |
| `frontend/src/lib/stores/tree.svelte.ts` | null Handling, orphan notes | Nullish coalescing, orphanNotes Array |

---

## Zeitaufwand

- Debugging: ~1.5 Stunden
- Fixes implementieren: ~30 Minuten
- Dokumentation: ~15 Minuten

**Total:** ~2 Stunden 15 Minuten

Ein manueller 5-Minuten-Smoke-Test vor dem Commit hätte dies verhindert.
