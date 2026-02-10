# Generisches Listenarten-System

## Übersicht

Ein erweiterbares Plugin-System für verschiedene Listenverhalten im Editor:
- **Standard ToDo**: checked = durchgestrichen
- **Shopping**: checked = ausgegraut + ans Ende verschoben
- **Zukünftig**: Packing, Habits, etc.

## Syntax-Entscheidung: Block-Container `:::`

```markdown
:::shopping
- [ ] Milch
- [ ] Brot
- [x] Käse  ← checked, wird ans Ende + ausgegraut
:::

:::todo
- [ ] Task 1
- [x] Task 2  ← durchgestrichen
:::
```

**Begründung:**
- Kompatibel mit markdown-it-container Standard
- Klare Scope-Definition
- Einfach erweiterbar für weitere Typen

---

## Architektur

```
frontend/src/lib/editor/list-behaviors/
├── types.ts           # Interfaces
├── registry.ts        # ListBehaviorRegistry
├── todo-behavior.ts   # Standard-Verhalten
├── shopping-behavior.ts
└── index.ts           # Export & Registration
```

### Core Interface

```typescript
interface ListBehavior {
  id: string;                    // "shopping", "todo"
  displayName: string;           // "Einkaufsliste"
  icon: string;                  // Lucide icon name

  onCheck(context: ListContext): CheckResult;
  onUncheck(context: ListContext): CheckResult;
  sortItems?(items: ListItem[]): ListItem[];
}

interface CheckResult {
  checked: boolean;
  newIndex?: number;             // Für Reordering
  cssClasses?: string[];
}
```

---

## Implementierungsschritte

### Phase 1: Behavior-System (Foundation)

**Neue Dateien erstellen:**
- `frontend/src/lib/editor/list-behaviors/types.ts`
- `frontend/src/lib/editor/list-behaviors/registry.ts`
- `frontend/src/lib/editor/list-behaviors/todo-behavior.ts`
- `frontend/src/lib/editor/list-behaviors/shopping-behavior.ts`
- `frontend/src/lib/editor/list-behaviors/index.ts`

**Feature Flag in config.ts:**
```typescript
export const FEATURE_FLAGS = {
  // ...existing
  listTypes: true
};
```

### Phase 2: markdown-it Container-Integration

**Datei:** `frontend/src/lib/editor/markdown.ts`

1. `markdown-it-container` installieren:
   ```bash
   cd frontend && npm install markdown-it-container
   ```

2. Container-Plugin für jeden registrierten Listentyp:
   ```typescript
   import container from 'markdown-it-container';

   for (const type of listBehaviorRegistry.getTypes()) {
     md.use(container, type, {
       validate: (params) => params.trim() === type,
       render: (tokens, idx) => {
         if (tokens[idx].nesting === 1) {
           return `<div class="list-container list-${type}" data-list-type="${type}">`;
         }
         return '</div>';
       }
     });
   }
   ```

### Phase 3: CodeMirror Container-Plugin

**Datei:** `frontend/src/lib/editor/codemirror.ts`

ViewPlugin für `:::type` Syntax-Highlighting:
```typescript
const listContainerPlugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    // Regex: /^:::(shopping|todo|packing)$/
    // Decorate container markers
  },
  { decorations: (v) => v.decorations }
);
```

### Phase 4: Editor.svelte Toggle-Logik

**Datei:** `frontend/src/lib/components/Editor.svelte`

`toggleTaskByIndex()` erweitern:
1. Container-Kontext erkennen (welcher `:::type`?)
2. Behavior aus Registry holen
3. `onCheck`/`onUncheck` aufrufen
4. Bei `newIndex`: Source-Reordering durchführen

```typescript
function toggleTaskByIndex(checkboxIndex: number, checked: boolean) {
  const containerType = findContainerTypeAtPosition(checkboxIndex);
  const behavior = listBehaviorRegistry.get(containerType ?? 'todo');

  const context = buildListContext(containerType, checkboxIndex);
  const result = checked ? behavior.onCheck(context) : behavior.onUncheck(context);

  if (result.newIndex !== undefined) {
    reorderListItem(context, result.newIndex);
  }
  // Apply checkbox state change
}
```

### Phase 5: Styling

**Datei:** `frontend/src/app.css`

```css
/* Container-Styling */
.list-container {
  margin: 1rem 0;
  padding: 0.5rem;
  border-radius: 0.25rem;
}

.list-shopping {
  background: var(--color-muted) / 0.1;
}

/* Shopping checked items */
.list-shopping .task-list-item-checked {
  opacity: 0.4;
  text-decoration: none;  /* Kein Durchstreichen */
  color: var(--color-muted-foreground);
}

/* Collapsed categories */
.list-item-collapsed > ul {
  display: none;
}
.list-item-collapsed::before {
  content: "▶";
  /* Expand-Icon */
}
```

### Phase 6: Sub-Item-Logik

**Dateien:** `Editor.svelte`, `shopping-behavior.ts`

1. **Kategorie-Erkennung**: Item mit Sub-Items = Kategorie
2. **Alle-gecheckt-Tracking**: Bei jedem Toggle prüfen ob alle Siblings gecheckt
3. **Automatisches Collapse**: Wenn alle Sub-Items gecheckt → Parent wird `[>]`, ans Ende
4. **Uncheck-Logik**: Bei Uncheck in collapsed Kategorie → Kategorie nach oben, `[ ]`

```typescript
function checkAllSubItemsCompleted(parentItem: ListItem): boolean {
  return parentItem.children.every(child => child.checked);
}

function handleSubItemCheck(item: ListItem, checked: boolean) {
  // 1. Toggle item
  // 2. Reorder within parent
  // 3. Check if all siblings done → collapse parent, move to end
}
```

### Phase 7: NewNoteModal

**Neue Dateien:**
- `frontend/src/lib/components/NewNoteModal.svelte`
- `frontend/src/lib/editor/list-behaviors/templates.ts`

**Anpassen:**
- "Neue Notiz" Button → öffnet Modal statt direkte Erstellung

```svelte
<!-- NewNoteModal.svelte -->
<Dialog>
  <h2>Neue Notiz erstellen</h2>
  <button onclick={() => createNote('note')}>📝 Notiz</button>
  <button onclick={() => createNote('shopping')}>🛒 Einkaufsliste</button>
  <button onclick={() => createNote('todo')}>✅ Todo-Liste</button>
</Dialog>
```

```typescript
// templates.ts
export const noteTemplates = {
  note: '',
  shopping: ':::shopping\n- [ ] \n:::',
  todo: ':::todo\n- [ ] \n:::'
};
```

---

## Kritische Dateien

| Datei | Änderung |
|-------|----------|
| `frontend/src/lib/editor/list-behaviors/*` | NEU: Behavior-System, Templates |
| `frontend/src/lib/editor/markdown.ts` | Container-Plugin, `[>]` Syntax |
| `frontend/src/lib/editor/codemirror.ts` | Container-Decoration |
| `frontend/src/lib/components/Editor.svelte` | Toggle-Logik, Sub-Item-Handling |
| `frontend/src/lib/components/NewNoteModal.svelte` | NEU: Notiztyp-Auswahl |
| `frontend/src/lib/config.ts` | Feature Flag |
| `frontend/src/app.css` | Listentyp-Styling, Collapse-CSS |
| `frontend/package.json` | markdown-it-container |

---

## Persistenz

**Source-Reordering:** Checked Items werden tatsächlich im Markdown verschoben.

Vor Check:
```markdown
:::shopping
- [ ] Milch
- [ ] Brot
:::
```

Nach Check von "Milch":
```markdown
:::shopping
- [ ] Brot
- [x] Milch
:::
```

**Vorteile:** Keine Metadaten, funktioniert in anderen Editoren, einfache Sync.

---

## Erweiterbarkeit

Neuen Listentyp hinzufügen:

1. `packing-behavior.ts` erstellen mit `ListBehavior` Interface
2. In `index.ts` registrieren: `registry.register(packingBehavior)`
3. Optional: CSS für `.list-packing`

---

## Verifizierung

1. **Unit Tests:** Behavior-Logik (onCheck, onUncheck, sortItems)
2. **Integration Test:** Container im Editor erstellen, Items togglen
3. **E2E:** Shopping-Liste erstellen, Items abhaken, Reihenfolge prüfen
4. **Manual:** Preview-Darstellung, Styling, Keyboard-Navigation

---

## Review-Notizen (für Implementierung beachten)

### Sub-Item-Verhalten (Shopping-Listen)

```markdown
:::shopping
- [ ] Obst
  - [ ] Äpfel
  - [x] Bananen    ← gecheckt → ans Ende der Sub-Kategorie "Obst"
- [>] Getränke     ← [>] = collapsed, alle Sub-Items gecheckt
  - [x] Wasser
  - [x] Saft
:::
```

**Logik:**
1. **Einzelnes Sub-Item gecheckt** → ans Ende seiner Sub-Kategorie verschieben
2. **Alle Sub-Items einer Kategorie gecheckt** → Ober-Kategorie ans Ende der Gesamtliste, Marker wird `[>]` (collapsed)
3. **Uncheck in collapsed Kategorie** → Kategorie wandert zurück nach oben, Marker wird `[ ]`, expanded

**Collapse-Syntax im Markdown:**
- `[ ]` = normale Kategorie/Item (expanded)
- `[>]` = collapsed (alle Sub-Items erledigt)
- `[x]` = gecheckt (für Items ohne Sub-Items)

**Zusätzliche Implementierung nötig:**
- Collapse/Expand Toggle im Preview (klick auf `[>]` klappt auf/zu)
- Tracking: "Sind alle Sub-Items gecheckt?" → automatisch `[>]` setzen
- CSS für collapsed state (Sub-Items hidden)
- markdown-it-task-lists erweitern für `[>]` Syntax

### UI: "Neue Notiz" mit Typauswahl

Bei Klick auf "Neue Notiz" → Auswahldialog:

```
┌─────────────────────────────┐
│  Neue Notiz erstellen       │
├─────────────────────────────┤
│  📝 Notiz (Standard)        │
│  🛒 Einkaufsliste           │
│  ✅ Todo-Liste              │
│  📦 Packliste (später)      │
└─────────────────────────────┘
```

**Implementierung:**
1. Modal-Komponente für Notiztyp-Auswahl
2. Templates pro Typ (z.B. Shopping startet mit `:::shopping\n- [ ] \n:::`)
3. "Neue Notiz" Button ruft Modal auf statt direkt zu erstellen

**Dateien:**
- `frontend/src/lib/components/NewNoteModal.svelte` (NEU)
- `frontend/src/lib/editor/list-behaviors/templates.ts` (NEU)
- Bestehende "Neue Notiz" Logik anpassen

### Offene Entscheidungen

1. **Konvertierung bestehender Listen**: Später als Feature hinzufügen

2. **Syntax `:::` nicht Standard-Markdown**: Akzeptiert als App-spezifische Erweiterung (xelanote-only Nutzung)

### Technische Risiken

- **Performance**: Container-Parsing bei großen Docs cachen
- **Undo/Redo**: Reordering muss mit CodeMirror History funktionieren
- **DOM-Index vs Source**: `toggleTaskByIndex()` arbeitet mit DOM-Index, Container-Kontext muss aus Source extrahiert werden

### MVP-Scope (aktualisiert)

**Phase 1 (Core):**
- `shopping` und `todo` Behaviors
- Container-Syntax `:::shopping`
- Basis-Reordering (checked ans Ende)

**Phase 2 (Sub-Items):**
- Sub-Item-Logik mit Kategorie-Tracking
- `[>]` Collapse-Syntax
- Automatisches Collapse bei allen Sub-Items gecheckt

**Phase 3 (UI):**
- NewNoteModal mit Typauswahl
- Templates pro Notiztyp

**Graceful Fallback:** Wenn Container nicht erkannt → Standard-Todo-Verhalten
