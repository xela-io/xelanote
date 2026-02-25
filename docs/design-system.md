# xelanote Design System

> **See also:** [Frontend Design System](../frontend/DESIGN_SYSTEM.md) for the detailed CSS/theme implementation with Tailwind v4, OKLch color space, and CSS custom properties.
> **See also:** [Design Audit (Feb 2026)](design-audit-2026-02.md) for the latest Playwright-based design consistency audit.

## Overview

The xelanote design system provides a centralized, consistent approach to UI design. It combines modern design principles with accessibility standards to create a polished, professional interface.

## Design Tokens

### Location

Design tokens are defined entirely in CSS using Tailwind v4's `@theme` directive:

- `frontend/src/app.css` – All design tokens (colors, radii, motion, fonts) and theme definitions
- `frontend/src/lib/themes/index.ts` – TypeScript theme metadata (IDs, names, CSS class names)

> **Note:** There are no separate TypeScript design-token files. All tokens live in CSS and are consumed via Tailwind v4's auto-generated utility classes (e.g., `--color-success` → `text-success`, `bg-success`).

### Typography

**Font**: `'Inter', ui-sans-serif, system-ui, sans-serif` (weights: 400, 500, 600, 700)
**Mono**: `ui-monospace, monospace`

| Element         | Desktop   | Mobile    | Weight | Line-Height |
|-----------------|-----------|-----------|--------|-------------|
| Page title      | 20px      | 18px      | 700    | 1.2         |
| Page subtitle   | 12.8px    | 12.8px    | 400    | 1.4         |
| Kicker text     | 11.52px   | 11.52px   | 500    | 1.4         |
| Body / Editor   | 14.4px    | 16px*     | 400    | 1.6         |
| Form labels     | 14px      | 14px      | 500    | 1.4         |
| Buttons (md)    | 16px      | 16px      | 500    | 1.0         |
| Buttons (sm)    | 14px      | 14px      | 500    | 1.0         |

*\* 16px on mobile prevents iOS Safari auto-zoom on input focus.*

### Color System

#### Semantic Colors
- **Primary**: Main brand color, CTAs, primary actions
- **Secondary**: Less emphasis than primary, secondary UI
- **Accent**: Subtle hover states (NOT saturated – designed for backgrounds)
- **Destructive**: Delete, danger actions (red)
- **Success**: Positive status, checkmarks, encryption badges (Gruvbox Aqua)
- **Warning**: Caution states, sync conflicts (Gruvbox Yellow)
- **Muted**: Disabled states, secondary text, subtle backgrounds

#### Theme Support

xelanote currently supports **2 themes**, both using the OKLch color space:

1. **Gruvbox Light** (`gruvbox-light`) – Warm retro colors with cream background
2. **Gruvbox Dark** (`gruvbox-dark`) – Warm retro colors with dark background

> **Backend note:** `backend/internal/service/user_types.go` still validates 22 legacy theme IDs for backwards compatibility with older user preferences. Only the 2 Gruvbox themes are implemented in the frontend.

### Border Radius Scale

| Token          | Value    |
|----------------|----------|
| `--radius-xs`  | 0.125rem |
| `--radius-sm`  | 0.25rem  |
| `--radius-md`  | 0.375rem |
| `--radius-lg`  | 0.5rem   |
| `--radius-xl`  | 0.75rem  |
| `--radius-2xl` | 1rem     |
| `--radius-3xl` | 1.5rem   |

### Animation Durations

- **Fast** (150ms): Focus states, state changes
- **Base** (200ms): Standard transitions, entrances/exits
- **Slow** (300ms): Collapse/expand, page transitions

### Easing Functions

- **default**: `cubic-bezier(0.4, 0, 0.2, 1)` – General purpose
- **entrance**: `cubic-bezier(0.2, 0, 0, 1)` – Elements appearing
- **exit**: `cubic-bezier(0.3, 0, 0.8, 0.15)` – Elements disappearing
- **in**: `cubic-bezier(0.4, 0, 1, 1)` – Accelerating
- **out**: `cubic-bezier(0, 0, 0.2, 1)` – Decelerating

## Components

All components use **Svelte 5 Runes** (`$props()`, `$derived`, Snippets). No Svelte 4 stores or `on:` event handlers.

### Button.svelte

Path: `frontend/src/lib/components/Button.svelte`

```svelte
<Button
  variant="primary|secondary|ghost|outline|destructive"
  size="sm|md|lg"
  icon={LucideIconComponent}
  iconPosition="left|right"
  loading={false}
  fullWidth={false}
  iconOnly={false}
  disabled={false}
  onclick={handler}
>
  Label
</Button>
```

**Variants:**
- **primary**: Solid background, prominent
- **secondary**: Subtle background
- **ghost**: Transparent, text-only appearance
- **outline**: Bordered, transparent background
- **destructive**: Red background for danger actions

### UI Components (`frontend/src/lib/components/ui/`)

| Component | Purpose |
|-----------|---------|
| `BaseDialog.svelte` | Modal dialog with focus trap, size variants (sm/md/lg/xl/2xl) |
| `AlertDialog.svelte` | Confirmation/alert dialogs |
| `ConfirmDialog.svelte` | Confirm/cancel prompts |
| `DialogActions.svelte` | Footer action buttons container |
| `DialogField.svelte` | Form field wrapper with label/help/error |
| `PageHeader.svelte` | Page title + subtitle + action buttons |
| `SettingsSection.svelte` | Settings panel with 3 variants (panel/soft/form) |

### CSS Component Classes (`.ui-*`)

| Prefix            | Purpose                             |
|-------------------|-------------------------------------|
| `.ui-page-*`      | Page layout (header, body, title)   |
| `.ui-panel-*`     | Card/panel styling (panel, soft)    |
| `.ui-button-*`    | Button variants                     |
| `.ui-icon-button-*` | Icon buttons (sm/md)              |
| `.ui-tab*`        | Tab navigation                      |
| `.ui-form-*`      | Form elements (label, input, etc.)  |
| `.ui-list-item`   | List entries                        |
| `.ui-empty-state` | Empty state layouts                 |
| `.ui-mobile-*`    | Mobile-specific (topbar, flat)      |

## Accessibility

### Standards
- **WCAG AA** minimum for all components
- **Focus indicators** always visible
- **Semantic HTML** for proper document structure
- **ARIA labels** for icon-only buttons
- **Keyboard navigation** for all interactive elements
- **Color contrast** 4.5:1 for normal text, 7:1 for AAA

### Best Practices

1. **Focus Management**: All interactive elements must be focusable via keyboard
2. **Focus Indicators**: Custom focus styles should never be removed
3. **Semantic HTML**: Use proper HTML elements (buttons, links, labels)
4. **ARIA Labels**: Icon-only buttons need aria-label
5. **Keyboard Support**: Implement arrow keys for lists, Tab for navigation

## Interactions

### Button Interactions
- **Hover**: Background color shift + slight shadow increase + lift (translateY -2px)
- **Press**: Scale 0.98 while mouse down
- **Focus**: 2px outline ring with 2px offset
- **Disabled**: Reduced opacity (0.5) + no interactions

### Focus Management
- Always visible (never `outline: none`)
- 3px ring using `--color-ring` CSS variable
- 2px offset for clear visibility
- Smooth transition (150ms)

## Responsive Design

- **Mobile first**: Design starts with mobile, enhanced for larger screens
- **Breakpoints**: `sm: 640px`, `md: 768px` (primary mobile/desktop switch)
- **Sidebar**: Desktop icon sidebar (persistent), Mobile bottom navigation bar
- **Touch targets**: Minimum 44px x 44px on mobile (WCAG AA)
- **iOS PWA**: Safe area insets, dynamic viewport height, standalone mode detection
- **Touch detection**: `@media (pointer: coarse)` for touch-friendly sizing

## Themes

### Theme Selection

Users can choose from 2 Gruvbox themes via Settings → Darstellung:
- **Gruvbox Hell** (light) – Default
- **Gruvbox Dunkel** (dark)
- Theme preference is synced to the server and persists across sessions

### Theme Implementation

- All themes use OKLch color space for perceptually uniform colors
- TypeScript metadata: `frontend/src/lib/themes/index.ts` (ThemeId, THEMES record)
- CSS variables: `frontend/src/app.css` (`.theme-gruvbox-light`, `.theme-gruvbox-dark`)
- Backend validation: `backend/internal/service/user_types.go` (validThemes map)

#### FOUC Prevention (Flash of Unstyled Content)

Inline-Script in `app.html` applies theme class + background color synchron im `<head>`, bevor CSS/JS geladen wird:

```javascript
// Actual code in frontend/src/app.html
var themes = {
  'gruvbox-light': { cls: 'theme-gruvbox-light', bg: '#fbf1c7', tc: '#458588' },
  'gruvbox-dark':  { cls: 'theme-gruvbox-dark',  bg: '#282828', tc: '#282828' },
};
```

**Technische Details:**
- Theme-Mapping im Inline-Script muss identisch sein mit `frontend/src/lib/themes/index.ts`
- Background-Colors muessen mit CSS-Variablen in `app.css` uebereinstimmen
- Script laeuft synchron im `<head>` → blockiert Rendering bis Theme angewendet ist (gewollt!)
- Fallback auf System-Praeferenz (`prefers-color-scheme`) wenn kein Theme gespeichert

### Adding New Themes

Bei Hinzufuegen neuer Themes muessen **vier Stellen** aktualisiert werden:

1. **TypeScript-Definition** (`frontend/src/lib/themes/index.ts`):
   - `ThemeId` Union Type erweitern
   - `THEMES` Record ergaenzen (id, name, variant, description, className)

2. **CSS-Variablen** (`frontend/src/app.css`):
   - Neue `.theme-*` Klasse mit allen `--color-*` Variablen (OKLCH)
   - Canvas-Farb-Presets (`--canvas-*`) falls Canvas-Feature genutzt wird
   - Scrollbar-Styling fuer Dark Themes

3. **FOUC-Script** (`frontend/src/app.html`):
   - Theme-Mapping im Inline-Script ergaenzen (cls, bg, tc)

4. **Backend-Validierung** (`backend/internal/service/user_types.go`):
   - Theme-ID in `validThemes` Map aufnehmen

### Theme History

- **2026-02**: Konsolidierung auf 2 Gruvbox-Themes (frontend). 20 Legacy-Theme-IDs bleiben im Backend fuer Abwaertskompatibilitaet.
- **2026-01-26**: FOUC Prevention implementiert (Inline-Script in app.html)
- **2026-01-22**: Gruvbox Light + Dark Themes eingefuehrt

## Usage Examples

### Using Design Tokens in Components

Design tokens werden ausschliesslich ueber CSS-Variablen und Tailwind-Utility-Klassen konsumiert:

```svelte
<!-- Tailwind utilities (bevorzugt) -->
<button class="bg-primary text-primary-foreground rounded-lg px-4 py-2
               transition-colors duration-base ease-default
               hover:bg-primary/90">
  Speichern
</button>

<!-- CSS-Variablen (fuer Spezialfaelle) -->
<div style="background: color-mix(in oklch, var(--color-primary), transparent 80%)">
  Transparenter Hintergrund
</div>
```

### Creating New Components

1. Nutze `--color-*` Tokens via Tailwind-Klassen (`bg-primary`, `text-success`, etc.)
2. **Keine hardcoded Tailwind-Farben** (`bg-blue-500`, `text-green-600`) – verwende semantische Tokens
3. Implementiere Accessibility (focus-visible, ARIA labels)
4. Teste in beiden Themes (Light + Dark)
5. Stelle Keyboard-Navigation sicher

## Testing

### Design-Test-Kommandos

```bash
npm run test:design          # Design-Audit (Layout, Touch, Kontrast)
npm run test:e2e:visual      # Visual Regression (Chromium)
npm run test:e2e:visual:all  # Visual Regression (alle Browser)
npm run test:a11y            # Accessibility (axe-core)
npm run test:design:vision   # AI Design Review (benoetigt Claude CLI)
```

Siehe [Design Audit (Feb 2026)](design-audit-2026-02.md) fuer den aktuellen Audit-Bericht.
