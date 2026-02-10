# Mobile-optimierte Versionshistorie

> **See also:** [Mobile App Plan](./planning/mobile-app.md) | [Editor Features](./editor-features.md)

**Datum**: 2026-01-19
**Commit**: `ac17f67`
**Status**: Implementiert

## Überblick

Die Versionshistorie wurde für mobile Geräte optimiert und bietet nun eine vollständig angepasste Benutzeroberfläche mit Tab-Navigation statt des herkömmlichen Side-by-Side-Layouts.

## Features

### Fullscreen-Dialog auf Mobilgeräten

- **Desktop**: Dialog mit fester Größe (max-w-5xl, 80vh) und Rounded Corners
- **Mobile**: Fullscreen-Dialog ohne Padding oder Rounded Corners (100% Viewport-Höhe/Breite)
- **Responsive Breakpoint**: Automatische Erkennung über `ui.getIsMobile()`

### Tab-Navigation (Nur Mobile)

Auf mobilen Geräten ersetzt eine Tab-Navigation das Side-by-Side-Layout:

#### Zwei Tabs

1. **Versionen-Tab**
   - Zeigt Liste aller verfügbaren Versionen (inkl. "Aktuell")
   - Versionsnummer, Titel und relative Zeitangabe
   - Aktuell gewählte Version ist hervorgehoben
   - Scroll-fähige Liste bei vielen Versionen

2. **Vorschau/Vergleich-Tab**
   - **Vorschau-Modus**: Zeigt vollständigen Content der gewählten Version
   - **Vergleich-Modus**: Zeigt Line-by-Line Diff zwischen zwei Versionen
   - Dynamischer Tab-Titel ändert sich mit dem Modus

#### Tab-Verhalten

- Standard: Versionen-Tab ist aktiv beim Öffnen
- Automatischer Wechsel: Nach Auswahl einer Version wechselt die UI automatisch zum Content-Tab
- Manueller Wechsel: Tabs sind jederzeit anklickbar
- Visuelles Feedback: Aktiver Tab wird mit Primary-Farbe und Unterstrich markiert

### Automatischer Tab-Wechsel

**Trigger**: Beim Klick auf eine Version in der Liste

```typescript
function selectVersion(v: VersionItem) {
    if (mode === 'compare' && selectedVersion) {
        // Don't compare with itself
        if (v.id === selectedVersion.id) return;
        compareVersion = v;
    } else {
        selectedVersion = v;
        compareVersion = null;
    }
    // On mobile, switch to content tab after selecting a version
    if (isMobile) {
        mobileTab = 'content';
    }
}
```

**Vorteil**: User muss nicht manuell zum Content-Tab wechseln, um die gewählte Version zu sehen.

### Desktop-Layout unverändert

Das Desktop-Layout bleibt komplett unberührt:

- Side-by-Side View: Versionsliste links (feste Breite 16rem), Content rechts
- Beide Bereiche gleichzeitig sichtbar
- Keine Tab-Navigation auf Desktop
- Gewohntes UX-Verhalten bleibt erhalten

## Technische Implementation

### Neue State-Variablen

```typescript
// Mobile-specific state
let mobileTab = $state<'versions' | 'content'>('versions');
const isMobile = $derived(ui.getIsMobile());
```

### Bedingte Rendering-Logik

```svelte
<!-- Mobile Tab Navigation -->
{#if isMobile}
    <div class="flex border-b border-border">
        <button onclick={() => mobileTab = 'versions'} ...>
            Versionen
        </button>
        <button onclick={() => mobileTab = 'content'} ...>
            {mode === 'compare' ? 'Vergleich' : 'Vorschau'}
        </button>
    </div>
{/if}

<!-- Version List Sidebar -->
<div class="... {isMobile && mobileTab !== 'versions' ? 'hidden' : ''}">
    <!-- Version list content -->
</div>

<!-- Content Area -->
<div class="... {isMobile && mobileTab !== 'content' ? 'hidden' : ''}">
    <!-- Preview or diff content -->
</div>
```

### Responsive Layout-Klassen

**Dialog-Container**:
```svelte
class="{isMobile ? 'h-full w-full rounded-none' : 'rounded-lg w-full max-w-5xl h-[80vh]'}"
```

**Backdrop-Padding**:
```svelte
class="... {isMobile ? 'p-0' : 'p-4'} ..."
```

**Sidebar-Breite**:
```svelte
class="{isMobile ? 'w-full' : 'w-64'} ..."
```

## User Experience

### Mobile Workflow

1. **Dialog öffnen**: "Versionen"-Button im Editor anklicken
2. **Versionen-Tab**: Liste aller Versionen wird angezeigt
3. **Version auswählen**: Tippen auf eine Version
4. **Automatischer Wechsel**: UI springt zu Vorschau-Tab
5. **Content ansehen**: Vollständiger Content ist sichtbar
6. **Vergleich aktivieren**: Button in der Header-Bar wechselt zu Vergleichsmodus
7. **Tab-Label ändert sich**: "Vorschau" → "Vergleich"
8. **Zweite Version wählen**: Zurück zu Versionen-Tab, andere Version antippen
9. **Diff ansehen**: Content-Tab zeigt farbcodiertes Line-Diff

### Desktop Workflow

Unverändert: Alles wie gewohnt mit Side-by-Side View.

## Vorteile

### Usability

- **Keine Platzverschwendung**: Fullscreen-Dialog nutzt gesamten verfügbaren Raum
- **Fokussierte UI**: Nur relevanter Content sichtbar (kein Split-Screen auf kleinem Display)
- **Natürliche Navigation**: Tab-Pattern ist vertraut aus vielen Mobile-Apps
- **Automatischer Flow**: Kein manuelles Tab-Switching nach Version-Auswahl nötig

### Technisch

- **Keine Code-Duplikation**: Gleiche Komponente für Desktop und Mobile
- **Bedingte Logik**: Einfache `if (isMobile)` Checks
- **Wartbarkeit**: Zentrale Responsive-Logik in einer Datei
- **Performance**: Keine zusätzlichen Requests, nur CSS-basiertes Hiding

## Dateien

- **Frontend**: `frontend/src/lib/components/VersionHistoryDialog.svelte`
- **Store**: `frontend/src/lib/stores/ui.svelte.ts` (für `getIsMobile()`)

## Zukünftige Verbesserungen

- **Swipe-Gesten**: Links/rechts wischen zum Wechseln zwischen Versionen
- **Pull-to-Refresh**: Versionsliste aktualisieren per Geste
- **Tablet-Optimierung**: Hybrid-Layout für mittlere Bildschirmgrößen (Split-Screen mit größeren Tabs)
- **Landscape-Mode**: Angepasstes Layout im Querformat (evtl. Side-by-Side)
