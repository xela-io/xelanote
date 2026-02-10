# xelanote Design System

> **See also:** [Frontend Design System](../frontend/DESIGN_SYSTEM.md) for the detailed CSS/theme implementation with Tailwind v4, OKLch color space, and CSS custom properties.

## Overview

The xelanote design system provides a centralized, consistent approach to UI design. It combines modern design principles with accessibility standards to create a polished, professional interface.

## Design Tokens

### Location
- `frontend/src/lib/design/tokens.ts` - Core design tokens (spacing, typography, shadows, animations)
- `frontend/src/lib/design/colors.ts` - Color system and semantic naming
- `frontend/src/lib/design/animations.ts` - Animation definitions and microinteractions

### Typography Scale

| Level | Size | Line Height | Weight | Use Case |
|-------|------|-------------|--------|----------|
| Display | 2.5rem | 1.2 | 700 | Page titles |
| Headline | 2rem | 1.3 | 600 | Section headers |
| Title | 1.5rem | 1.4 | 600 | Card titles, major headings |
| Subtitle | 1.25rem | 1.4 | 500 | Form labels, subsections |
| Body | 1rem | 1.5 | 400 | Main content, paragraphs |
| Label | 0.875rem | 1.4 | 500 | Small labels, badges |
| Caption | 0.75rem | 1.4 | 400 | Helper text, timestamps |

### Color System

#### Semantic Colors
- **Primary**: Main brand color, CTAs, primary actions
- **Secondary**: Less emphasis than primary, secondary UI
- **Accent**: Hover states, highlights, badges
- **Destructive**: Delete, danger actions (red)
- **Muted**: Disabled states, secondary text, subtle backgrounds

#### Theme Support
xelanote supports 23 themes, all using OKLch color space for perceptually uniform colors:

**Light Themes (11):**
1. Default Light - Clean, modern light theme
2. Nord Light - Arctic, blue-tinted color scheme
3. Solarized Light - Precision colors by Ethan Schoonover
4. Catppuccin Latte - Soft pastel colors
5. Gruvbox Light - Warm retro colors with cream background
6. One Light - Iconic VSCode light theme
7. Ayu Light - Modern bright theme
8. Ayu Mirage - Mid-tone between light and dark
9. Rosé Pine Dawn - Warm light theme with soft pastels
10. Everforest Light - Nature-inspired with warm earth tones

**Dark Themes (12):**
11. Default Dark - Clean, modern dark theme
12. Nord Dark - Arctic, blue-tinted color scheme
13. Solarized Dark - Precision colors by Ethan Schoonover
14. Dracula - Dark theme with vibrant colors
15. Catppuccin Mocha - Soft pastel colors
16. Dark Pastels - Inspired by KDE Konsole with pastel accents
17. Gruvbox Dark - Warm retro colors with dark background
18. Tokyo Night - Inspired by Tokyo at night
19. One Dark - Iconic VSCode/Atom dark theme
20. Monokai - Classic warm dark theme
21. Rosé Pine Moon - Elegant dark theme with soft pastels
22. Kanagawa Wave - Japanese-inspired dark theme
23. Everforest Dark - Nature-inspired green-based forest theme

### Shadow System

| Level | Use | Example |
|-------|-----|---------|
| sm | Subtle cards | Cards, badges |
| md | Standard elevation | Dropdowns, popovers |
| lg | Strong elevation | Modals, floating panels |
| hover | Interactive feedback | Buttons on hover |

### Spacing Scale

Uses a consistent scale in rems: 0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 5, 6, 7, 8, 9, 10, 12, 14, 16, 20, 24, 28, 32

### Animation Durations

- **Fast** (150ms): Focus states, state changes
- **Base** (200ms): Standard transitions, entrances/exits
- **Slow** (300ms): Collapse/expand, page transitions
- **Slower** (500ms): Complex animations

### Easing Functions

- **default** (standard): General purpose transitions
- **entrance** (ease-out): Elements appearing
- **exit** (ease-in): Elements disappearing
- **in/out** (cubic-bezier): Smooth acceleration/deceleration

## Components

### Button.svelte

Reusable button component with multiple variants and sizes.

```svelte
<Button
  variant="primary|secondary|ghost|outline|destructive"
  size="sm|md|lg"
  icon="lucide-icon-name"
  iconPosition="left|right"
  disabled={false}
  on:click={handler}
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

**States:**
- **default**: Normal state
- **hover**: Background shift + subtle shadow + lift effect
- **active**: Press effect (scale 0.98)
- **focus**: Focus ring outline
- **disabled**: Reduced opacity, no interactions

### SidebarItem.svelte

Component for folder/note items in the sidebar.

```svelte
<SidebarItem
  icon="folder|file-text"
  label="Name"
  isActive={false}
  count={0}
  isDraggable={false}
  on:click={handler}
  on:contextmenu={handler}
  on:drag={handler}
/>
```

**Features:**
- Unread count badge on the right
- Active state with left border accent
- Drag handle on hover
- Context menu support
- Smooth hover effects

### Section.svelte

Wrapper for sidebar sections with optional collapse.

```svelte
<Section title="Notebooks" collapsible={true} on:toggle={handler}>
  <!-- items go here -->
</Section>
```

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
- **Breakpoints**: Uses Tailwind breakpoints (sm: 640px, md: 768px, lg: 1024px, xl: 1280px)
- **Sidebar**: Mobile drawer (fixed, off-canvas), Desktop sidebar (persistent)
- **Touch targets**: Minimum 44px × 44px on mobile

## Themes

### Theme Selection
Users can choose from 12 carefully crafted themes via the settings page:
- Accessible through the palette icon in the theme selector
- Themes are categorized into Light and Dark sections
- Theme preference is synced to the server and persists across sessions

### Theme Implementation
- All themes use OKLch color space for perceptually uniform colors
- Themes are defined in `frontend/src/lib/themes/index.ts`
- CSS variables are defined in `frontend/src/app.css`
- Backend validates theme IDs in `backend/internal/service/user.go`
- Smooth transition between themes (300ms fade)

#### FOUC Prevention (Flash of Unstyled Content)

**Problem:** Beim Neuladen der Seite gab es einen sichtbaren "Flash" (weißer/blauer Bildschirm) für ca. 100-300ms, wenn ein dunkles Theme eingestellt war. Dies passierte, weil das Theme erst nach dem Laden von JavaScript angewendet wurde.

**Lösung (2026-01):** Dreistufiger Ansatz für sofortiges Theme-Loading:

1. **Inline-Script in `app.html`** (ausgeführt BEVOR CSS/JS geladen wird):
   - Liest Theme aus `localStorage` sofort im `<head>`
   - Wendet Theme-Klasse auf `<html>` an (z.B. `class="dark"`)
   - Setzt `background-color` als Inline-Style auf `<html>` Element
   - Erstellt dynamisch `<meta name="theme-color">` für Mobile-Browser
   - Behandelt Migration von alten Theme-IDs (`light` → `default-light`)
   - Fallback auf System-Präferenz (`prefers-color-scheme`) wenn kein Theme gespeichert

2. **CSS-Variablen im Loading-State** (`+layout.svelte`):
   - Verwendet `style="background-color: var(--color-background, #111)"` statt Tailwind-Klassen
   - Ermöglicht sofortige Anwendung des Themes (noch vor hydration)
   - Fallback-Wert `#111` für absolute Sicherheit

3. **Service Worker in Development deaktiviert** (`vite.config.ts`):
   - `devOptions.enabled: false` verhindert Caching-Probleme
   - Vermeidet, dass alte Theme-Versionen aus dem Cache geladen werden
   - Erleichtert Theme-Entwicklung und Testing

**Technische Details:**
- Theme-Mapping im Inline-Script muss identisch sein mit `frontend/src/lib/themes/index.ts`
- Background-Colors müssen mit CSS-Variablen in `app.css` übereinstimmen
- Script läuft synchron im `<head>` → blockiert Rendering bis Theme angewendet ist (gewollt!)
- Keine externe JavaScript-Abhängigkeiten → funktioniert auch bei langsamer Verbindung

**Wartung:**
Bei Hinzufügen neuer Themes müssen **alle drei** Stellen aktualisiert werden:
- `frontend/src/lib/themes/index.ts` (Theme-Definition)
- `frontend/src/app.html` (Inline-Script Theme-Mapping)
- `frontend/src/app.css` (CSS-Variablen für das Theme)

**Beispiel Inline-Script Struktur:**
```javascript
// In frontend/src/app.html <head>
<script>
(function() {
  try {
    var theme = localStorage.getItem('xelanote-theme');
    // Migration alte Werte
    if (theme === 'light') theme = 'default-light';
    if (theme === 'dark') theme = 'default-dark';

    // Theme Mapping: { id: { cls: 'className', bg: '#color', tc: '#theme-color' } }
    var themes = {
      'default-light':     { cls: '',                      bg: '#ffffff', tc: '#3b82f6' },
      'default-dark':      { cls: 'dark',                  bg: '#1a1a2e', tc: '#1a1a2e' },
      // ... alle anderen Themes
    };

    // Fallback auf System-Präferenz
    if (!theme || !(theme in themes)) {
      theme = window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'default-dark' : 'default-light';
    }

    var t = themes[theme];
    if (t.cls) document.documentElement.classList.add(t.cls);
    document.documentElement.style.backgroundColor = t.bg;

    // Theme-color meta tag für Mobile
    var meta = document.createElement('meta');
    meta.name = 'theme-color';
    meta.content = t.tc;
    document.head.appendChild(meta);
  } catch(e) {}
})();
</script>
```

**Beispiel Loading-State mit CSS-Variablen:**
```svelte
<!-- In frontend/src/routes/+layout.svelte -->
{#if !authInitialized}
  <div
    class="flex-1 flex items-center justify-center"
    style="background-color: var(--color-background, #111);"
  >
    <div class="text-center">
      <div
        class="animate-spin rounded-full h-12 w-12 border-b-2 mx-auto mb-4"
        style="border-color: var(--color-muted-foreground, #666);"
      ></div>
      <p style="color: var(--color-muted-foreground, #888);">Laden...</p>
    </div>
  </div>
{/if}
```

### Adding New Themes
To add a new theme:

1. **Frontend Theme Definition** (`frontend/src/lib/themes/index.ts`):
   ```typescript
   export type ThemeId =
     | 'existing-themes'
     | 'your-new-theme';

   export const THEMES: Record<ThemeId, Theme> = {
     'your-new-theme': {
       id: 'your-new-theme',
       name: 'Your New Theme',
       variant: 'dark',
       description: 'Description of your theme',
       className: 'theme-your-new-theme'
     }
   };
   ```

2. **CSS Variables** (`frontend/src/app.css`):
   ```css
   .theme-your-new-theme {
     --color-background: oklch(20% 0.02 60);
     --color-foreground: oklch(95% 0.03 85);
     /* ... all other color variables */
   }
   ```

3. **Scrollbar Styling** (add to dark theme lists if dark):
   ```css
   .theme-your-new-theme,
   .theme-your-new-theme * {
     scrollbar-color: rgba(255, 255, 255, 0.3) transparent !important;
   }
   ```

4. **Backend Validation** (`backend/internal/service/user.go`):
   ```go
   var validThemes = map[string]bool{
     "your-new-theme": true,
   }
   ```

### Recent Theme Updates

- **2026-01-26**: 10 neue Themes hinzugefügt (13→23 Themes total)
  - **Popular Developer Themes**: One Dark, One Light, Monokai, Ayu Light
  - **Pastel & Soft Themes**: Ayu Mirage, Rosé Pine Moon, Rosé Pine Dawn, Kanagawa Wave
  - **Nature & Earth Tones**: Everforest Dark, Everforest Light
  - Alle Themes verwenden OKLch Farbräume für konsistente Wahrnehmung
  - Backend-Validierung erweitert (inkl. fehlende tokyo-night)
  - FOUC-Prevention für alle neuen Themes
  - Scrollbar-Styling für neue Dark Themes
  - Markdown List Markers optimiert für bessere Lesbarkeit
  - **Ergebnis**: Ausgewogene Balance mit 11 Light & 12 Dark Themes

- **2026-01-26**: FOUC (Flash of Unstyled Content) Fix implementiert
  - Inline-Script in `app.html` lädt Theme synchron vor CSS/JS
  - Loading-State verwendet CSS-Variablen statt Tailwind für instant theming
  - Service Worker in Development deaktiviert (verhindert Theme-Caching-Probleme)
  - Beseitigt weißen/blauen Flash beim Neuladen mit dunklen Themes

- **2026-01-22**: Added Dark Pastels, Gruvbox Light, and Gruvbox Dark themes
  - Dark Pastels: Inspired by KDE Konsole colorscheme with soft, pastel accent colors
  - Gruvbox Light: Warm retro colors with a cream background (#fbf1c7)
  - Gruvbox Dark: Warm retro colors with a dark background (#282828)

## Performance

- **GPU Acceleration**: Use transform and opacity for animations
- **Will-change**: Applied to animated elements
- **CSS Containment**: Isolate component styles
- **Lazy Loading**: Code split heavy components

## Usage Examples

### Using Design Tokens in Components

```svelte
<script>
  import { typography, spacing, shadows, animationDurations, easing } from '$lib/design/tokens';
  import { colorUsage } from '$lib/design/colors';
</script>

<button
  style={`
    padding: ${spacing[3]};
    font-size: ${typography.body.size};
    background-color: ${colorUsage.primary};
    box-shadow: ${shadows.md};
    transition: all ${animationDurations.base}ms ${easing.default};
  `}
>
  Click me
</button>
```

### Creating New Components

1. Import design tokens and colors
2. Use consistent spacing, colors, and animations
3. Implement proper accessibility (focus, ARIA)
4. Test across all 9 themes
5. Ensure keyboard navigation works

## Future Enhancements

- Command palette (Cmd/Ctrl+K)
- Advanced search with filters
- Custom theme builder
- Component storybook
- Figma design file sync
- Animation preferences (respects prefers-reduced-motion)
