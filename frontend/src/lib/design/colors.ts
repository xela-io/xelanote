/**
 * Color System for xelanote
 * Semantic color naming and palette organization
 * Uses OKLch color space for perceptually uniform colors
 */

export interface ColorPalette {
  primary: {
    background: string;
    foreground: string;
  };
  secondary: {
    background: string;
    foreground: string;
  };
  accent: {
    background: string;
    foreground: string;
  };
  destructive: {
    background: string;
    foreground: string;
  };
  success: {
    background: string;
    foreground: string;
  };
  warning: {
    background: string;
    foreground: string;
  };
  muted: {
    background: string;
    foreground: string;
  };
  border: string;
  ring: string;
  background: string;
  foreground: string;
  card: {
    background: string;
    foreground: string;
  };
  sidebar: {
    background: string;
    foreground: string;
    primary: {
      background: string;
      foreground: string;
    };
    accent: {
      background: string;
      foreground: string;
    };
    border: string;
    ring: string;
  };
}

/**
 * Semantic color usage guidelines
 */
export const colorUsage = {
  // Primary: Main brand color, CTA buttons, links
  primary: 'var(--color-primary)',
  primaryForeground: 'var(--color-primary-foreground)',

  // Secondary: Less emphasis than primary
  secondary: 'var(--color-secondary)',
  secondaryForeground: 'var(--color-secondary-foreground)',

  // Accent: Hover states, highlights, badges
  accent: 'var(--color-accent)',
  accentForeground: 'var(--color-accent-foreground)',

  // Destructive: Delete, danger actions (red)
  destructive: 'var(--color-destructive)',
  destructiveForeground: 'var(--color-destructive-foreground)',

  // Success: Positive feedback, completed states (Gruvbox Aqua)
  success: 'var(--color-success)',
  successForeground: 'var(--color-success-foreground)',

  // Warning: Caution, attention needed (Gruvbox Yellow)
  warning: 'var(--color-warning)',
  warningForeground: 'var(--color-warning-foreground)',

  // Muted: Disabled state, secondary text, subtle backgrounds
  muted: 'var(--color-muted)',
  mutedForeground: 'var(--color-muted-foreground)',

  // Backgrounds and text
  background: 'var(--color-background)',
  foreground: 'var(--color-foreground)',

  // Card: Elevated surfaces
  card: 'var(--color-card)',
  cardForeground: 'var(--color-card-foreground)',

  // Border and focus ring
  border: 'var(--color-border)',
  input: 'var(--color-input)',
  ring: 'var(--color-ring)',

  // Sidebar specific
  sidebar: {
    background: 'var(--color-sidebar-background)',
    foreground: 'var(--color-sidebar-foreground)',
    primary: 'var(--color-sidebar-primary)',
    primaryForeground: 'var(--color-sidebar-primary-foreground)',
    accent: 'var(--color-sidebar-accent)',
    accentForeground: 'var(--color-sidebar-accent-foreground)',
    border: 'var(--color-sidebar-border)',
    ring: 'var(--color-sidebar-ring)',
  },
} as const;

/**
 * Contrast ratio helpers (for accessibility)
 * WCAG guidelines:
 * - Normal text: 4.5:1
 * - Large text (18pt+): 3:1
 * - AAA (enhanced): 7:1
 */
export const contrastLevels = {
  AA_NORMAL: 4.5,
  AA_LARGE: 3,
  AAA_NORMAL: 7,
  AAA_LARGE: 4.5,
} as const;

/**
 * Button color variants
 */
export const buttonVariants = {
  primary: {
    background: colorUsage.primary,
    foreground: colorUsage.primaryForeground,
    hover: 'var(--color-primary)',
    active: 'var(--color-primary)',
  },
  secondary: {
    background: colorUsage.secondary,
    foreground: colorUsage.secondaryForeground,
    hover: 'var(--color-secondary)',
    active: 'var(--color-secondary)',
  },
  ghost: {
    background: 'transparent',
    foreground: colorUsage.foreground,
    hover: colorUsage.accent,
    active: colorUsage.muted,
  },
  outline: {
    background: 'transparent',
    foreground: colorUsage.foreground,
    border: colorUsage.border,
    hover: colorUsage.accent,
    active: colorUsage.muted,
  },
  destructive: {
    background: colorUsage.destructive,
    foreground: colorUsage.destructiveForeground,
    hover: colorUsage.destructive,
    active: colorUsage.destructive,
  },
} as const;

/**
 * State colors for feedback
 */
export const stateColors = {
  success: 'var(--color-success)', // Gruvbox Aqua
  warning: 'var(--color-warning)', // Gruvbox Yellow
  error: 'oklch(57.71% 0.215 27.33)', // Red (matches destructive)
  info: 'oklch(60% 0.12 250)', // Blue
} as const;
