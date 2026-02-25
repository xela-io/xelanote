# xelanote Design System

> **See also:** [Design System Overview](../docs/design-system.md) for design tokens, typography scale, spacing, shadows, and high-level design principles.

Comprehensive documentation for the xelanote frontend theme and design system.

## Overview

The xelanote design system is built on **Tailwind CSS v4** using CSS custom properties and the **OKLch color space** for perceptually uniform colors across all themes. The system leverages Tailwind v4's auto-generation feature where `--color-*` CSS variables automatically generate corresponding utility classes.

### Key Features

- **OKLch color space**: Provides perceptually uniform colors with consistent lightness and chroma across themes
- **Automatic utility generation**: `--color-success` generates `text-success`, `bg-success`, `border-success`, and opacity variants like `bg-success/10`
- **Theme class system**: Switch themes by applying `.theme-gruvbox-light` or `.theme-gruvbox-dark` to the root element
- **Semantic color naming**: Colors are named by purpose (e.g., `primary`, `success`) rather than appearance (e.g., `blue`, `green`)

## Theme Blocks

The design system defines three theme blocks in `/src/app.css`:

### 1. Default Theme (`@theme`)

The base theme (Gruvbox Light) loaded by default. Variables defined here apply globally unless overridden by theme classes.

```css
@theme {
  --color-background: oklch(97% 0.03 85);
  --color-foreground: oklch(24% 0.02 60);
  /* ... */
}
```

### 2. Light Theme (`.theme-gruvbox-light`)

Explicit class for Gruvbox Light theme. Contains identical values to `@theme` for consistency.

```css
.theme-gruvbox-light {
  --color-background: oklch(97% 0.03 85);
  --color-foreground: oklch(24% 0.02 60);
  /* ... */
}
```

### 3. Dark Theme (`.theme-gruvbox-dark`)

Gruvbox Dark theme with warm, retro colors optimized for low-light environments.

```css
.theme-gruvbox-dark {
  --color-background: oklch(22% 0.02 60);
  --color-foreground: oklch(95% 0.03 85);
  /* ... */
}
```

## Color Variables Reference

All color variables use the `--color-*` prefix (required by Tailwind v4). Each variable automatically generates corresponding utility classes.

### Layout Colors

| Variable                       | Purpose                               | Light Value           | Dark Value            |
| ------------------------------ | ------------------------------------- | --------------------- | --------------------- |
| `--color-background`           | Main page background                  | `oklch(97% 0.03 85)`  | `oklch(22% 0.02 60)`  |
| `--color-foreground`           | Primary text color                    | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-card`                 | Elevated surface (cards, panels)      | `oklch(95% 0.03 85)`  | `oklch(24% 0.02 60)`  |
| `--color-card-foreground`      | Text on card surfaces                 | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-popover`              | Floating menus, tooltips              | `oklch(95% 0.03 85)`  | `oklch(24% 0.02 60)`  |
| `--color-popover-foreground`   | Text in popovers                      | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-muted`                | Subtle background for disabled states | `oklch(92% 0.03 85)`  | `oklch(30% 0.02 60)`  |
| `--color-muted-foreground`     | Secondary/disabled text               | `oklch(38% 0.08 60)`  | `oklch(78% 0.10 75)`  |
| `--color-border`               | Default border color                  | `oklch(88% 0.03 85)`  | `oklch(30% 0.02 60)`  |
| `--color-input`                | Input field borders                   | `oklch(88% 0.03 85)`  | `oklch(30% 0.02 60)`  |
| `--color-ring`                 | Focus ring outline                    | `oklch(38% 0.10 230)` | `oklch(72% 0.10 155)` |
| `--color-selection`            | Text selection background             | `oklch(75% 0.08 60)`  | `oklch(52% 0.12 60)`  |
| `--color-selection-foreground` | Text selection text color             | `oklch(15% 0.02 60)`  | `oklch(98% 0.01 60)`  |

### Action Colors

| Variable                         | Purpose                                                         | Light Value           | Dark Value            |
| -------------------------------- | --------------------------------------------------------------- | --------------------- | --------------------- |
| `--color-primary`                | Links, CTA buttons, active tabs, focus states                   | `oklch(38% 0.10 230)` | `oklch(72% 0.10 155)` |
| `--color-primary-foreground`     | Text on primary backgrounds                                     | `oklch(97% 0.03 85)`  | `oklch(22% 0.02 60)`  |
| `--color-secondary`              | Secondary buttons, less emphasis                                | `oklch(92% 0.03 85)`  | `oklch(30% 0.02 60)`  |
| `--color-secondary-foreground`   | Text on secondary backgrounds                                   | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-accent`                 | **Subtle hover states** (NOT saturated)                         | `oklch(92% 0.04 160)` | `oklch(28% 0.06 155)` |
| `--color-accent-foreground`      | Text on accent backgrounds                                      | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-destructive`            | Delete actions, errors                                          | `oklch(30% 0.16 25)`  | `oklch(65% 0.18 25)`  |
| `--color-destructive-foreground` | Text on destructive backgrounds                                 | `oklch(97% 0.03 85)`  | `oklch(97% 0.03 85)`  |
| `--color-success`                | Status indicators, checkmarks, encryption status (Gruvbox Aqua) | `oklch(45% 0.10 160)` | `oklch(75% 0.10 160)` |
| `--color-success-foreground`     | Text on success backgrounds                                     | `oklch(97% 0.03 85)`  | `oklch(22% 0.02 60)`  |
| `--color-warning`                | Warnings, caution states (Gruvbox Yellow)                       | `oklch(55% 0.14 70)`  | `oklch(78% 0.14 70)`  |
| `--color-warning-foreground`     | Text on warning backgrounds                                     | `oklch(97% 0.03 85)`  | `oklch(22% 0.02 60)`  |

### Sidebar Colors

Sidebar-specific colors for consistent sidebar styling across light and dark themes.

| Variable                             | Purpose                 | Light Value           | Dark Value            |
| ------------------------------------ | ----------------------- | --------------------- | --------------------- |
| `--color-sidebar-background`         | Sidebar background      | `oklch(95% 0.03 85)`  | `oklch(20% 0.02 60)`  |
| `--color-sidebar-foreground`         | Sidebar text            | `oklch(38% 0.08 60)`  | `oklch(72% 0.12 155)` |
| `--color-sidebar-primary`            | Sidebar active items    | `oklch(38% 0.10 230)` | `oklch(72% 0.10 155)` |
| `--color-sidebar-primary-foreground` | Text on sidebar primary | `oklch(97% 0.03 85)`  | `oklch(22% 0.02 60)`  |
| `--color-sidebar-accent`             | Sidebar hover states    | `oklch(92% 0.04 160)` | `oklch(30% 0.06 155)` |
| `--color-sidebar-accent-foreground`  | Text on sidebar accent  | `oklch(24% 0.02 60)`  | `oklch(95% 0.03 85)`  |
| `--color-sidebar-border`             | Sidebar borders         | `oklch(88% 0.03 85)`  | `oklch(30% 0.02 60)`  |
| `--color-sidebar-ring`               | Sidebar focus rings     | `oklch(38% 0.10 230)` | `oklch(72% 0.10 155)` |

## Auto-Generated Tailwind Utilities

Tailwind v4 automatically generates utility classes from `--color-*` variables:

```css
/* Variable definition */
--color-success: oklch(45% 0.10 160);

/* Auto-generated utilities */
.text-success          /* color: var(--color-success) */
.bg-success            /* background-color: var(--color-success) */
.border-success        /* border-color: var(--color-success) */
.bg-success/10         /* background with 10% opacity */
.border-success/30     /* border with 30% opacity */
```

### Usage Examples

```html
<!-- Text color -->
<p class="text-success">Success message</p>

<!-- Background color -->
<div class="bg-primary text-primary-foreground">Primary button</div>

<!-- Background with opacity -->
<div class="bg-destructive/10 border border-destructive/30">Error container</div>

<!-- Hover states -->
<button class="hover:bg-accent hover:text-accent-foreground">Hover me</button>
```

## Usage Guidelines

### When to Use Each Color

#### `primary`

- **Use for**: Links, active tabs, CTA buttons, focus rings, primary navigation
- **Examples**: Active note in sidebar, "Save" button, focused input rings
- **Usage count**: Most frequently used action color

#### `accent`

- **Use for**: Subtle hover states, non-primary highlights
- **Important**: `accent` is NOT saturated - it's designed for subtle backgrounds
- **Usage count**: 77+ instances as `hover:bg-accent`
- **Examples**: Button hover states, menu item hover, tree item hover

#### `success` (Gruvbox Aqua)

- **Use for**: Positive status indicators, checkmarks, encryption badges, completed tasks
- **Examples**: "Encryption enabled" badge, task completion checkmark `[x]`
- **Do not use**: As a primary action color

#### `warning` (Gruvbox Yellow)

- **Use for**: Caution states, warnings that need attention
- **Examples**: "Browser not supported" warnings, sync conflicts
- **Do not use**: For general highlights

#### `destructive`

- **Use for**: Delete actions, error messages, dangerous operations
- **Examples**: "Delete note" button, error toasts, failed sync status
- **Always pair with**: Confirmation dialogs for destructive actions

#### `secondary`

- **Use for**: Less emphasis than primary, alternative actions
- **Examples**: "Cancel" buttons, secondary navigation items

#### `muted`

- **Use for**: Disabled states, secondary text, subtle backgrounds
- **Examples**: Disabled buttons, placeholder text, inline code backgrounds

### Sidebar-Specific Guidelines

Use `sidebar-*` variables for all sidebar components to ensure consistent theming:

```html
<!-- Sidebar container -->
<aside class="bg-sidebar-background text-sidebar-foreground">
  <!-- Active item -->
  <button class="bg-sidebar-primary text-sidebar-primary-foreground">Active Note</button>

  <!-- Hover item -->
  <button class="hover:bg-sidebar-accent hover:text-sidebar-accent-foreground">Hover Me</button>
</aside>
```

## Scoped Styles with CSS Variables

When using CSS variables in Svelte `<style>` blocks, use `color-mix()` for transparency:

```svelte
<style>
  .custom-element {
    /* Correct: Use color-mix for transparency */
    background: color-mix(in oklch, var(--color-primary), transparent 80%);
    color: var(--color-foreground);
  }

  /* WRONG: Do not use hardcoded colors */
  .bad-example {
    background: rgba(59, 130, 246, 0.1); /* ❌ Hardcoded blue */
  }
</style>
```

### OKLch vs SRGB

- **Use `oklch`** for theme-consistent transparency (preferred)
- **Use `srgb`** only when mixing with theme variables already in sRGB space

```css
/* OKLch (preferred) */
background: color-mix(in oklch, var(--color-accent), transparent 85%);

/* SRGB (when necessary) */
background: color-mix(in srgb, var(--color-muted) 15%, transparent);
```

## Legacy Aliases (Deprecated)

The following legacy aliases exist for backwards compatibility. These are scheduled for removal in v2.0:

```css
/* Legacy aliases - DO NOT USE in new code */
--bg-primary → var(--color-background)
--bg-secondary → var(--color-card)
--text-primary → var(--color-foreground)
--text-secondary → var(--color-muted-foreground)
--text-tertiary → var(--color-muted-foreground)
--text-muted → var(--color-muted-foreground)
--border → var(--color-border)
--border-color → var(--color-border)
--border-hover → var(--color-border)
--accent → var(--color-primary)
--accent-primary → var(--color-primary)
--accent-light → color-mix(in oklch, var(--color-primary), transparent 80%)
--accent-dark → color-mix(in oklch, var(--color-primary), black 15%)
--accent-hover → color-mix(in oklch, var(--color-primary), black 10%)
--accent-foreground → var(--color-primary-foreground)
--error → var(--color-destructive)
--error-bg → color-mix(in oklch, var(--color-destructive), transparent 85%)
--info → var(--color-primary)
--info-bg → color-mix(in oklch, var(--color-primary), transparent 85%)
```

**Migration**: Replace legacy aliases with their modern `--color-*` equivalents.

## CodeMirror Syntax Theme

The editor uses a custom syntax highlighting theme that integrates with the design system.

### Theme Definition

Location: `/src/lib/editor/codemirror.ts` (lines 267-284)

```typescript
const markdownSyntaxStyle = HighlightStyle.define([
  {
    tag: [tags.meta, tags.punctuation],
    color: 'var(--color-muted-foreground)',
    fontWeight: '600',
  },
  {
    tag: tags.atom, // Task markers [ ]/[x]
    color: 'var(--color-muted-foreground)',
  },
  {
    tag: [tags.link, tags.url],
    color: 'var(--color-primary)',
  },
]);
```

### Load Order (Critical)

The custom syntax theme **MUST** load BEFORE `defaultHighlightStyle` with `{fallback: true}`:

```typescript
// CORRECT order
syntaxHighlighting(markdownSyntaxStyle),
syntaxHighlighting(defaultHighlightStyle, { fallback: true }),

// WRONG order - defaultHighlightStyle will override custom theme
syntaxHighlighting(defaultHighlightStyle),
syntaxHighlighting(markdownSyntaxStyle),
```

### Why Load Order Matters

- `markdownSyntaxStyle` defines theme-aware colors for specific elements
- `defaultHighlightStyle` provides fallback styles for unstyled syntax elements
- Without `{fallback: true}`, `defaultHighlightStyle` overrides custom theme colors
- Incorrect order causes blue atoms (`#221199`) invisible on dark backgrounds

### Editor Theme Variables

```css
.cm-editor {
  /* Background and text */
  background: var(--color-background);
  color: var(--color-foreground);
}

.cm-activeLine {
  background: color-mix(in srgb, var(--color-muted) 15%, transparent);
}

.cm-selectionBackground {
  background: var(--color-selection);
}

.cm-gutters {
  background: var(--color-sidebar-background);
  color: var(--color-muted-foreground);
  border-right: 1px solid var(--color-border);
}

/* Bracket matching */
.cm-matchingBracket {
  background: color-mix(in srgb, var(--color-primary) 25%, transparent);
  outline: 1px solid color-mix(in srgb, var(--color-primary) 40%, transparent);
}

.cm-nonmatchingBracket {
  background: color-mix(in srgb, var(--color-destructive) 25%, transparent);
}

/* Task brackets [x] */
.cm-task-bracket {
  color: var(--color-success) !important;
}

/* Wikilinks */
.cm-wikilink {
  color: var(--color-primary);
  text-decoration: underline;
}

.cm-wikilink-unresolved {
  color: var(--color-destructive);
  text-decoration: underline dashed;
}
```

## Exceptions

### Diff Views

The following components use hardcoded green/red for semantic diff visualization:

- `AITransformDialog` (AI text transformation diffs)
- `FormatMarkdownDialog` (Markdown formatting diffs)
- `VersionHistoryDialog` (Version comparison diffs)

**Rationale**: Diff colors are universally understood as green = added, red = removed. Using theme colors would reduce clarity.

### Semantic Warnings

Using hardcoded orange for specific semantic warnings (e.g., browser compatibility) is acceptable when the color communicates critical meaning.

## Adding a New Theme

Follow these steps to add a new theme variant:

### 1. Define Theme Class

Add a new theme class in `/src/app.css`:

```css
.theme-my-new-theme {
  /* Layout colors */
  --color-background: oklch(...);
  --color-foreground: oklch(...);
  --color-card: oklch(...);
  --color-card-foreground: oklch(...);
  --color-popover: oklch(...);
  --color-popover-foreground: oklch(...);
  --color-muted: oklch(...);
  --color-muted-foreground: oklch(...);
  --color-border: oklch(...);
  --color-input: oklch(...);
  --color-ring: oklch(...);
  --color-selection: oklch(...);
  --color-selection-foreground: oklch(...);

  /* Action colors */
  --color-primary: oklch(...);
  --color-primary-foreground: oklch(...);
  --color-secondary: oklch(...);
  --color-secondary-foreground: oklch(...);
  --color-accent: oklch(...);
  --color-accent-foreground: oklch(...);
  --color-destructive: oklch(...);
  --color-destructive-foreground: oklch(...);
  --color-success: oklch(...);
  --color-success-foreground: oklch(...);
  --color-warning: oklch(...);
  --color-warning-foreground: oklch(...);

  /* Sidebar colors */
  --color-sidebar-background: oklch(...);
  --color-sidebar-foreground: oklch(...);
  --color-sidebar-primary: oklch(...);
  --color-sidebar-primary-foreground: oklch(...);
  --color-sidebar-accent: oklch(...);
  --color-sidebar-accent-foreground: oklch(...);
  --color-sidebar-border: oklch(...);
  --color-sidebar-ring: oklch(...);
}
```

### 2. Update TypeScript Definitions

Add theme ID to `/src/lib/themes/index.ts`:

```typescript
export type ThemeId = 'gruvbox-light' | 'gruvbox-dark' | 'your-new-theme';

export const THEMES: Record<ThemeId, Theme> = {
  // ... existing themes
  'your-new-theme': {
    id: 'your-new-theme',
    name: 'Your New Theme',
    variant: 'dark', // or 'light'
    description: 'Description',
    className: 'theme-your-new-theme',
  },
};
```

### 3. Update FOUC Script

Add theme mapping in `/src/app.html` inline script:

```javascript
var themes = {
  // ... existing themes
  'your-new-theme': { cls: 'theme-your-new-theme', bg: '#hexcolor', tc: '#hexcolor' },
};
```

### 4. Add Backend Validation

Add theme ID to `backend/internal/service/user_types.go`:

```go
var validThemes = map[string]bool{
  // ... existing themes
  "your-new-theme": true,
}
```

### 5. Add Theme Selector Option

The theme selector automatically reads from the `THEMES` record – no additional changes needed.

### 6. Test Accessibility

Verify WCAG contrast ratios:

- **Normal text**: Minimum 4.5:1 (AA)
- **Large text** (18pt+): Minimum 3:1 (AA)
- **AAA compliance** (recommended): 7:1 normal, 4.5:1 large

Use browser DevTools or online contrast checkers to validate.

### 7. Test Scrollbar Colors

Add scrollbar styling if needed (see lines 82-123 in `app.css`):

```css
.theme-my-new-theme,
.theme-my-new-theme * {
  scrollbar-color: rgba(...) transparent !important;
}
```

## Common Mistakes

### 1. Incorrect Variable Prefix

```css
/* ❌ WRONG - Missing --color- prefix */
color: var(--primary);

/* ✅ CORRECT - Tailwind v4 requires --color- prefix */
color: var(--color-primary);
```

### 2. Hardcoded Tailwind Colors

```html
<!-- ❌ WRONG - Hardcoded color, won't adapt to theme -->
<div class="text-blue-600 bg-gray-100">
  <!-- ✅ CORRECT - Theme-aware colors -->
  <div class="text-primary bg-muted"></div>
</div>
```

### 3. Hardcoded Colors in :global()

```svelte
<style>
  /* ❌ WRONG - Hardcoded color in global scope */
  :global(.my-class) {
    background: #3b82f6;
  }

  /* ✅ CORRECT - Use theme variables */
  :global(.my-class) {
    background: var(--color-primary);
  }
</style>
```

### 4. Using hsl() with OKLch Values

```css
/* ❌ WRONG - OKLch values in hsl() function */
color: hsl(72% 0.1 155);

/* ✅ CORRECT - Use oklch() function */
color: oklch(72% 0.1 155);
```

### 5. Incorrect CodeMirror Syntax Load Order

```typescript
// ❌ WRONG - defaultHighlightStyle overrides custom theme
syntaxHighlighting(defaultHighlightStyle),
syntaxHighlighting(markdownSyntaxStyle),

// ✅ CORRECT - Custom theme first, default as fallback
syntaxHighlighting(markdownSyntaxStyle),
syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
```

### 6. Missing -foreground Pairing

```html
<!-- ❌ WRONG - Text color may be unreadable -->
<button class="bg-primary">Click me</button>

<!-- ✅ CORRECT - Always pair background with foreground -->
<button class="bg-primary text-primary-foreground">Click me</button>
```

### 7. Using accent as Saturated Highlight

```html
<!-- ❌ WRONG - accent is subtle, not for prominent highlights -->
<div class="bg-accent border-accent">Important badge</div>

<!-- ✅ CORRECT - Use primary or success for prominent highlights -->
<div class="bg-success text-success-foreground">Important badge</div>
```

## Additional Resources

- **Tailwind CSS v4 Documentation**: [tailwindcss.com](https://tailwindcss.com)
- **OKLch Color Space**: [oklch.com](https://oklch.com) - Interactive OKLch color picker
- **WCAG Contrast Checker**: [webaim.org/resources/contrastchecker](https://webaim.org/resources/contrastchecker/)
- **Gruvbox Color Scheme**: [github.com/morhetz/gruvbox](https://github.com/morhetz/gruvbox)

## Questions & Contributions

For questions about the design system or to propose new colors/themes:

1. Check existing components for usage patterns
2. Verify accessibility with contrast checkers
3. Test in both light and dark themes
4. Submit changes with updated documentation
