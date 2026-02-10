/**
 * Design Tokens for xelanote
 * Centralized design system definitions (colors, typography, spacing, shadows, animations)
 * Used across all components to ensure consistency
 */

// Typography Scale (rem)
export const typography = {
  display: {
    size: '2.5rem',
    lineHeight: '1.2',
    weight: 700,
  },
  headline: {
    size: '2rem',
    lineHeight: '1.3',
    weight: 600,
  },
  title: {
    size: '1.5rem',
    lineHeight: '1.4',
    weight: 600,
  },
  subtitle: {
    size: '1.25rem',
    lineHeight: '1.4',
    weight: 500,
  },
  body: {
    size: '1rem',
    lineHeight: '1.5',
    weight: 400,
  },
  label: {
    size: '0.875rem',
    lineHeight: '1.4',
    weight: 500,
  },
  caption: {
    size: '0.75rem',
    lineHeight: '1.4',
    weight: 400,
  },
} as const;

// Spacing Scale (rem)
export const spacing = {
  0: '0',
  0.5: '0.125rem',
  1: '0.25rem',
  1.5: '0.375rem',
  2: '0.5rem',
  2.5: '0.625rem',
  3: '0.75rem',
  3.5: '0.875rem',
  4: '1rem',
  5: '1.25rem',
  6: '1.5rem',
  7: '1.75rem',
  8: '2rem',
  9: '2.25rem',
  10: '2.5rem',
  12: '3rem',
  14: '3.5rem',
  16: '4rem',
  20: '5rem',
  24: '6rem',
  28: '7rem',
  32: '8rem',
} as const;

// Shadow System (depth levels)
export const shadows = {
  none: 'none',
  sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -1px rgb(0 0 0 / 0.06)',
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -2px rgb(0 0 0 / 0.05)',
  xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
  '2xl': '0 25px 50px -12px rgb(0 0 0 / 0.25)',
  inner: 'inset 0 2px 4px 0 rgb(0 0 0 / 0.05)',
  hover: '0 10px 20px -5px rgb(0 0 0 / 0.12)',
  focus: '0 0 0 3px rgb(59 130 246 / 0.1)',
} as const;

// Animation Durations (milliseconds)
export const animationDurations = {
  fast: 150,
  base: 200,
  slow: 300,
  slower: 500,
} as const;

// Easing Functions (cubic-bezier)
export const easing = {
  default: 'cubic-bezier(0.4, 0, 0.2, 1)', // material-standard
  in: 'cubic-bezier(0.4, 0, 1, 1)', // ease-in
  out: 'cubic-bezier(0, 0, 0.2, 1)', // ease-out
  inOut: 'cubic-bezier(0.4, 0, 0.2, 1)', // ease-in-out
  entrance: 'cubic-bezier(0.2, 0, 0, 1)', // entrance
  exit: 'cubic-bezier(0.3, 0, 0.8, 0.15)', // exit
} as const;

// Border Radius
export const borderRadius = {
  none: '0',
  sm: '0.125rem',
  base: '0.5rem',
  md: '0.75rem',
  lg: '1rem',
  xl: '1.5rem',
  full: '9999px',
} as const;

// Z-index Scale
export const zIndex = {
  hide: -1,
  base: 0,
  dropdown: 10,
  sticky: 20,
  fixed: 30,
  modal: 40,
  popover: 50,
  tooltip: 60,
  notification: 70,
  debug: 9999,
} as const;

// Button Sizes
export const buttonSizes = {
  sm: {
    padding: '0.375rem 0.75rem',
    fontSize: '0.875rem',
    height: '2rem',
  },
  md: {
    padding: '0.5rem 1rem',
    fontSize: '1rem',
    height: '2.5rem',
  },
  lg: {
    padding: '0.75rem 1.5rem',
    fontSize: '1rem',
    height: '3rem',
  },
} as const;

// Transitions (CSS transition shorthand)
export const transitions = {
  colors: `color ${animationDurations.fast}ms ${easing.default}`,
  background: `background-color ${animationDurations.fast}ms ${easing.default}`,
  border: `border-color ${animationDurations.fast}ms ${easing.default}`,
  shadow: `box-shadow ${animationDurations.base}ms ${easing.default}`,
  all: `all ${animationDurations.base}ms ${easing.default}`,
  smooth: `all ${animationDurations.slow}ms ${easing.default}`,
} as const;

// Focus Ring
export const focusRing = {
  width: '2px',
  offset: '2px',
  color: 'var(--color-ring)',
  style: 'solid',
} as const;
