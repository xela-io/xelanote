/**
 * Animation Definitions for xelanote
 * Standardized animations and transitions for consistent, professional feel
 * All animations are GPU-accelerated (using transform/opacity)
 */

import { animationDurations, easing } from './tokens';

export const animations = {
  // Entrance animations (200ms)
  fadeIn: {
    keyframes: `@keyframes fadeIn {
			from { opacity: 0; }
			to { opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  slideUp: {
    keyframes: `@keyframes slideUp {
			from { transform: translateY(16px); opacity: 0; }
			to { transform: translateY(0); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  slideDown: {
    keyframes: `@keyframes slideDown {
			from { transform: translateY(-16px); opacity: 0; }
			to { transform: translateY(0); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  slideRight: {
    keyframes: `@keyframes slideRight {
			from { transform: translateX(-16px); opacity: 0; }
			to { transform: translateX(0); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  slideLeft: {
    keyframes: `@keyframes slideLeft {
			from { transform: translateX(16px); opacity: 0; }
			to { transform: translateX(0); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  scaleUp: {
    keyframes: `@keyframes scaleUp {
			from { transform: scale(0.95); opacity: 0; }
			to { transform: scale(1); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  // Exit animations (200ms)
  fadeOut: {
    keyframes: `@keyframes fadeOut {
			from { opacity: 1; }
			to { opacity: 0; }
		}`,
    duration: animationDurations.base,
    easing: easing.exit,
  },

  slideUpExit: {
    keyframes: `@keyframes slideUpExit {
			from { transform: translateY(0); opacity: 1; }
			to { transform: translateY(-16px); opacity: 0; }
		}`,
    duration: animationDurations.base,
    easing: easing.exit,
  },

  slideDownExit: {
    keyframes: `@keyframes slideDownExit {
			from { transform: translateY(0); opacity: 1; }
			to { transform: translateY(16px); opacity: 0; }
		}`,
    duration: animationDurations.base,
    easing: easing.exit,
  },

  // Interactive feedback (150ms)
  buttonPress: {
    keyframes: `@keyframes buttonPress {
			from { transform: scale(1); }
			to { transform: scale(0.98); }
		}`,
    duration: animationDurations.fast,
    easing: easing.default,
  },

  hoverLift: {
    keyframes: `@keyframes hoverLift {
			from { transform: translateY(0); }
			to { transform: translateY(-2px); }
		}`,
    duration: animationDurations.base,
    easing: easing.default,
  },

  focusExpand: {
    keyframes: `@keyframes focusExpand {
			from { outline-width: 1px; outline-offset: 0; }
			to { outline-width: 2px; outline-offset: 2px; }
		}`,
    duration: animationDurations.fast,
    easing: easing.default,
  },

  // Loading animations (infinite)
  spin: {
    keyframes: `@keyframes spin {
			from { transform: rotate(0deg); }
			to { transform: rotate(360deg); }
		}`,
    duration: 1000,
    easing: 'linear',
  },

  pulse: {
    keyframes: `@keyframes pulse {
			0%, 100% { opacity: 1; }
			50% { opacity: 0.5; }
		}`,
    duration: 2000,
    easing: 'cubic-bezier(0.4, 0, 0.6, 1)',
  },

  shimmer: {
    keyframes: `@keyframes shimmer {
			0% { background-position: -1000px 0; }
			100% { background-position: 1000px 0; }
		}`,
    duration: 2000,
    easing: 'linear',
  },

  // Collapse/Expand animations (250ms)
  heightExpand: {
    keyframes: `@keyframes heightExpand {
			from { max-height: 0; opacity: 0; }
			to { max-height: 500px; opacity: 1; }
		}`,
    duration: animationDurations.slow,
    easing: easing.out,
  },

  heightCollapse: {
    keyframes: `@keyframes heightCollapse {
			from { max-height: 500px; opacity: 1; }
			to { max-height: 0; opacity: 0; }
		}`,
    duration: animationDurations.slow,
    easing: easing.in,
  },

  // Tooltip animations (200ms)
  popIn: {
    keyframes: `@keyframes popIn {
			from { transform: scale(0.8) translateY(-8px); opacity: 0; }
			to { transform: scale(1) translateY(0); opacity: 1; }
		}`,
    duration: animationDurations.base,
    easing: easing.entrance,
  },

  popOut: {
    keyframes: `@keyframes popOut {
			from { transform: scale(1) translateY(0); opacity: 1; }
			to { transform: scale(0.8) translateY(-8px); opacity: 0; }
		}`,
    duration: animationDurations.base,
    easing: easing.exit,
  },
} as const;

/**
 * Microinteraction definitions
 * Standard interactions for consistent UX
 */
export const microinteractions = {
  // Button hover: background shift + slight lift
  buttonHover: `
		transition: background-color ${animationDurations.fast}ms ${easing.default},
		            box-shadow ${animationDurations.base}ms ${easing.default};
	`,

  // Button active: press effect
  buttonActive: `
		transition: transform ${animationDurations.fast}ms ${easing.default},
		            background-color ${animationDurations.fast}ms ${easing.default};
	`,

  // Focus ring
  focusRing: `
		transition: outline-offset ${animationDurations.fast}ms ${easing.default},
		            outline-width ${animationDurations.fast}ms ${easing.default};
	`,

  // Smooth color transition
  smoothColorTransition: `
		transition: color ${animationDurations.fast}ms ${easing.default},
		            background-color ${animationDurations.fast}ms ${easing.default},
		            border-color ${animationDurations.fast}ms ${easing.default};
	`,

  // Collapse/expand items
  collapsibleItem: `
		transition: max-height ${animationDurations.slow}ms ${easing.default},
		            opacity ${animationDurations.base}ms ${easing.default};
	`,
} as const;
