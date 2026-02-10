# Logo Design and Assets

This document outlines the design concepts for the xelanote project logo and provides assets that can be used.

## 1. Design Concept

The logo for xelanote should convey its core principles:
*   **Connection & Linking:** The application's main feature is the wiki-style linking of notes to build a personal knowledge graph.
*   **Structure & Knowledge:** The app helps organize thoughts and build a structured knowledge base.
*   **Modern & Fast:** The technology stack (Go + Svelte) is modern and performant, and the logo should reflect this clean, efficient aesthetic.
*   **Identity:** The name "xelanote" provides the unique "X" which can be a central part of the brand.

The chosen design concept is a **stylized 'X' representing interconnected notes**. This is unique, memorable, and directly tied to the application's function.

## 2. AI Image Generator Prompt

For creating a high-quality, professional logo with a generative AI tool like Midjourney or DALL-E, use the following detailed prompt:

> **"Minimalist vector logo for a modern note-taking app named 'xelanote'. The central icon should be an abstract, stylized letter 'X'. The 'X' should be formed by two intersecting, dynamic lines or arrows, symbolizing the connection and linking of ideas. The design must be clean, geometric, and professional, suitable for a tech application. Use a sophisticated color palette of deep teal and dark slate gray. The logo should be presented on a plain white background, and a version for dark mode should also be included."**

## 3. SVG Placeholder Logo

As an immediate, usable asset, the following SVG code provides a clean, minimalist logo that can be placed directly into the project.

**Suggested Location:** `frontend/static/logo.svg` (noch nicht im Repo vorhanden)

### Advantages of this SVG Logo:
*   **Scalable:** Looks sharp at any size.
*   **Lightweight:** Tiny file size.
*   **Theme-Aware:** Automatically adapts to the system's light or dark mode using CSS media queries.
*   **Self-Contained:** No external files or images needed.

### SVG Code

```xml
<svg width="128" height="128" viewBox="0 0 128 128" fill="none" xmlns="http://www.w3.org/2000/svg">
  <style>
    .bg { fill: #2D3748; }
    .stroke-primary { stroke: #38B2AC; }
    .text-color { fill: #E2E8F0; }
    @media (prefers-color-scheme: light) {
      .bg { fill: #F7FAFC; }
      .stroke-primary { stroke: #319795; }
      .text-color { fill: #2D3748; }
    }
  </style>
  <rect class="bg" width="128" height="128" rx="24"/>
  <g transform="translate(24, 24) scale(0.625)">
    <path d="M16 112L112 16" class="stroke-primary" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="M16 16L112 112" class="stroke-primary" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
    <circle cx="64" cy="64" r="16" fill="#2D3748"/>
    <circle cx="64" cy="64" r="10" class="stroke-primary" stroke-width="4"/>
  </g>
</svg>
```

---

## 4. Logo-Komponente (Svelte)

Die wiederverwendbare Logo-Komponente befindet sich in `frontend/src/lib/components/Logo.svelte`.

### Props

| Prop | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `size` | `'sm' \| 'md' \| 'lg' \| 'xl'` | `'md'` | Schriftgröße (0.875rem / 1rem / 1.5rem / 2rem) |
| `variant` | `'default' \| 'badge'` | `'default'` | Darstellungsvariante |
| `uppercase` | `boolean` | `false` | Text in Großbuchstaben |
| `showIcon` | `boolean` | `true` | PenLine-Icon anzeigen |

### Icon-Größen

Das PenLine-Icon (lucide-svelte) skaliert passend zur Textgröße:

| Size | Icon-Größe |
|------|------------|
| `sm` | 14px |
| `md` | 16px |
| `lg` | 22px |
| `xl` | 28px |

### Hover-Effekte

Die Komponente hat interaktive Hover-Effekte (nur bei `showIcon={true}`):

- **Scale**: Leichte Vergrößerung auf 1.03
- **Glow**: Drop-Shadow mit Akzentfarbe
- **Icon-Rotation**: Stift dreht sich um -15°
- **Animation**: Gradient-Animation beschleunigt (8s → 3s)

### Varianten

**Default** (mit Gradient-Animation):
```svelte
<Logo />
<Logo size="lg" />
<Logo showIcon={false} />
```

**Badge** (solid, ohne Icon/Animation):
```svelte
<Logo variant="badge" uppercase />
```

### Barrierefreiheit

- `role="img"` und `aria-label="xelanote"` für Screenreader
- `user-select: none` verhindert Textmarkierung
- `prefers-reduced-motion`: Alle Animationen und Hover-Effekte deaktiviert

### Verwendungsbeispiele

```svelte
<!-- Sidebar-Header -->
<Logo size="lg" />

<!-- Login-Seite -->
<Logo size="xl" />

<!-- Footer-Badge -->
<Logo variant="badge" size="sm" uppercase />

<!-- Kompakt ohne Icon -->
<Logo size="sm" showIcon={false} />
```

---
This document can be used as a reference for any future branding or design work.
