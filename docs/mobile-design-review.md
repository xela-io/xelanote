# Mobile Design Review — Xelanote

Date: 2026-02-26

## Overall Impression

The Gruvbox theme is a strong foundation — warm, distinctive, avoids generic AI aesthetic. But the execution plays it too safe. The interface feels like a competently themed utility rather than a product with character.

---

## 1. Typography — The Biggest Issue

**Problem: Inter is the primary font.** Inter is the default choice of every AI-generated UI and every Tailwind starter template. Technically fine but completely devoid of personality.

### Recommendations

| Role | Current | Suggested Alternatives |
|------|---------|----------------------|
| **Display/Headings** | Inter 600/700 | **Fraunces** (optical-size variable, warm serifs that match Gruvbox's warmth), **Literata** (editorial feel, excellent for a notes app), or **Playfair Display** (high contrast, editorial) |
| **Body/UI** | Inter 400/500 | **Source Serif 4** (readable, warm, distinctive), **Crimson Pro** (elegant, bookish), or **DM Sans** (geometric but warmer than Inter) |
| **Monospace/Code** | system monospace | **JetBrains Mono** or **Fira Code** (both pair well with Gruvbox, ligatures optional) |

A notes app deserves a **literary, editorial** typographic voice. The combination of a characterful serif for headings + a clean sans for UI would instantly differentiate this from every other app.

**Section headers** ("ALL RECIPES", "NOTES", "MORE OPTIONS") currently use 0.68rem uppercase tracking — fine structurally but would benefit from a slightly different weight or a small-caps variant from the chosen heading font.

---

## 2. Color & Theme — Cohesive but Flat

The Gruvbox palette is well-implemented with OKLCH (modern, good choice). But the application of color lacks hierarchy and surprise.

### Issues
- The bright green `--color-primary` on the "New Recipe" button clashes with the warm amber/gold tone used everywhere else. Feels like it belongs to a different app
- Section label gold (`--canvas-yellow` range) and the sidebar green (`--color-sidebar-foreground`) compete rather than complement
- All cards share the same `--color-card` background with identical borders — nothing guides the eye

### Recommendations
- **Establish a clear accent hierarchy**: Use the warm amber/gold (`oklch(78% 0.14 70)`) as primary action color instead of the green. Reserve the green/teal for success states and active navigation only
- **Differentiate card types visually**: Recipe cards could have a subtle warm gradient or left-border accent. Journal entries could use a date-based subtle color shift. Note tree items could use depth-based opacity
- **Add a "hero" accent**: One unexpected color used very sparingly — perhaps a muted terracotta or rust (`oklch(55% 0.12 35)`) for pinned items, starred notes, or important badges. Adds moments of visual surprise within the warm palette

---

## 3. Motion — Mostly Static

Solid animation system defined (keyframes, easing tokens, utility classes) but barely used in the actual UI.

### What's Missing
- **List stagger animations**: When the recipe list, journal entries, or note tree loads, items should fade/slide in with incremental `animation-delay`. Single highest-impact motion improvement.
  ```css
  .recipe-card:nth-child(1) { animation-delay: 0ms; }
  .recipe-card:nth-child(2) { animation-delay: 50ms; }
  .recipe-card:nth-child(3) { animation-delay: 100ms; }
  ```
- **Bottom nav tab transitions**: Active indicator should animate between tabs (sliding underline or pill) rather than just appearing
- **Tree expand/collapse**: Currently uses `heightExpand`/`heightCollapse` but children should stagger in when a folder opens
- **Page transitions**: Moving between Notes/Journal/Recipes/Settings should have a coordinated exit-enter animation, not just a cut
- **Card press feedback**: Subtle `scale(0.98)` on tap with `duration-fast` easing adds tactile quality

---

## 4. Backgrounds & Depth — Too Flat

The dark background (`oklch(22% 0.02 60)`) is a uniform void. Frosted glass on bottom nav is nice but stands alone.

### Recommendations
- **Add a subtle noise/grain texture** to the background (CSS-only approach):
  ```css
  body::after {
    content: '';
    position: fixed;
    inset: 0;
    background-image: url("data:image/svg+xml,..."); /* tiny noise SVG */
    opacity: 0.03;
    pointer-events: none;
    z-index: 9999;
  }
  ```
  Adds warmth and tactility matching Gruvbox's analog feel

- **Gradient bleed on page headers**: Subtle radial gradient from primary color at ~3-5% opacity behind page title creates depth:
  ```css
  .ui-page-header::before {
    background: radial-gradient(ellipse at top, oklch(78% 0.14 70 / 0.06), transparent 70%);
  }
  ```

- **Card elevation hierarchy**: Currently all `.ui-panel` elements have the same treatment. Create 2-3 elevation levels with progressively stronger `box-shadow` and slightly lighter backgrounds for elevated elements

---

## 5. Component-Specific Feedback

### Bottom Navigation
- Good: Frosted glass, compact height, safe-area support
- Improve: Add a **sliding active indicator** (small pill or line that animates between tabs using `transform: translateX()`)
- The three-dot "More" icon is ambiguous. Consider replacing with a **hamburger or grid icon** that more clearly suggests "menu"

### Sidebar/Note Tree
- Indentation + chevron pattern is functional but visually monotonous
- **Add depth cues**: Nested levels could have progressively reduced opacity or a subtle left-border matching depth level
- Active item highlight (green background pill) is good, but the **pinned icon** (sparkle) is too subtle at this size — consider a small dot indicator instead

### Recipe Views
- **Ingredient cards are too tall**: Each ingredient takes ~80px for data that could fit in 44px. Most critical UX issue — excessive scrolling
- Ingredient layout should be a **compact table/list** rather than individual cards:
  ```
  400g   Ganze geschälte Tomaten    ○  🗑
  250ml  Gemüsebrühe                ○  🗑
  ```
- Preview tab with its clean tabular layout is actually much better — make edit view match this density
- **"+ New Recipe"** button's bright green is the loudest element on screen and fights for attention. Tone down to match warm palette

### Journal
- **Streak visualization** (tiny colored squares) is a good concept but too small to parse on mobile. Make squares at least 12px with more spacing
- **Redundant date display**: Each card shows both English AND German date. Show only the localized version
- Cards are visually identical — differentiate **today** with a subtle glow or accent border

### Settings
- **Icon-only tab row** at top has no labels and no tooltips — usability issue on mobile (no hover). Add labels below icons or use segmented control with text
- **Theme cards** are plain rectangles. Show a mini-preview of the theme's color palette inside each card
- Page feels empty — tighten spacing between sections

### More Options Sheet
- Clean and functional
- Active state on "Live preview" (green highlight) works well
- Consider adding **icons** to the Formatting section items (Indent/Outdent) for faster scanning

---

## 6. Priority Ranking (Impact-to-Effort)

1. **Replace Inter with a distinctive font pairing** — Highest visual impact, moderate effort (font swap + size adjustments)
2. **Add staggered list animations** — High delight factor, low effort (CSS-only, keyframes already exist)
3. **Compact the recipe ingredient layout** — Biggest UX win, moderate effort
4. **Add noise texture to background** — Instant atmosphere upgrade, very low effort
5. **Unify the accent color hierarchy** — Fixes visual confusion, moderate effort (CSS variable changes)
6. **Animated bottom nav indicator** — Polish detail, low effort
7. **Fix settings icon labels** — Usability fix, low effort
