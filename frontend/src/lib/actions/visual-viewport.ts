/**
 * Visual Viewport Action for handling iOS keyboard
 *
 * Problem: When iOS keyboard appears, 100dvh changes dynamically,
 * causing layout reflow and cursor jumping in the editor.
 *
 * Solution: Use Visual Viewport API to:
 * 1. Detect when keyboard opens/closes
 * 2. Adjust container height to prevent reflow
 * 3. Scroll focused element into view smoothly
 */

import { browser } from '$app/environment';

export interface VisualViewportOptions {
  /** Called when keyboard state changes */
  onKeyboardChange?: (isOpen: boolean, keyboardHeight: number) => void;
  /** Debounce delay in ms (default: 100) */
  debounce?: number;
}

interface ViewportState {
  initialHeight: number;
  isKeyboardOpen: boolean;
  keyboardHeight: number;
}

/**
 * Svelte action that handles Visual Viewport changes for keyboard handling
 */
export function visualViewport(node: HTMLElement, options: VisualViewportOptions = {}) {
  if (!browser) return;

  const { onKeyboardChange, debounce = 100 } = options;

  // Check if Visual Viewport API is available
  if (!window.visualViewport) {
    console.warn('Visual Viewport API not supported');
    return;
  }

  const state: ViewportState = {
    initialHeight: window.innerHeight,
    isKeyboardOpen: false,
    keyboardHeight: 0,
  };

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let lastHeight = window.visualViewport.height;

  function handleResize() {
    if (!window.visualViewport) return;

    const viewportHeight = window.visualViewport.height;
    const windowHeight = window.innerHeight;

    // Keyboard is considered open if visual viewport is significantly smaller
    // than window height (threshold: 150px to account for browser chrome)
    const heightDiff = windowHeight - viewportHeight;
    const keyboardOpen = heightDiff > 150;
    const keyboardHeight = keyboardOpen ? heightDiff : 0;

    // Only update if state changed significantly
    if (Math.abs(lastHeight - viewportHeight) < 10) return;
    lastHeight = viewportHeight;

    // Debounce to avoid rapid updates during keyboard animation
    if (debounceTimer) clearTimeout(debounceTimer);

    debounceTimer = setTimeout(() => {
      const wasOpen = state.isKeyboardOpen;
      state.isKeyboardOpen = keyboardOpen;
      state.keyboardHeight = keyboardHeight;

      if (keyboardOpen) {
        // Keyboard opened - set fixed height to prevent reflow
        node.style.height = `${viewportHeight}px`;
        node.style.overflow = 'hidden';

        // NOTE: Don't auto-scroll here - CodeMirror handles cursor visibility itself
        // scrollFocusedElementIntoView() was causing unwanted jumps
      } else {
        // Keyboard closed - restore flexible height
        node.style.height = '';
        node.style.overflow = '';
      }

      // Notify callback if state changed
      if (wasOpen !== keyboardOpen && onKeyboardChange) {
        onKeyboardChange(keyboardOpen, keyboardHeight);
      }
    }, debounce);
  }

  function _scrollFocusedElementIntoView() {
    const activeElement = document.activeElement;
    if (!activeElement || !(activeElement instanceof HTMLElement)) return;

    // Check if it's an input element
    const isInput =
      activeElement.tagName === 'INPUT' ||
      activeElement.tagName === 'TEXTAREA' ||
      activeElement.isContentEditable ||
      activeElement.closest('.cm-editor'); // CodeMirror

    if (!isInput) return;

    // Use scrollIntoView with smooth behavior
    // 'center' ensures the element is visible even with keyboard
    requestAnimationFrame(() => {
      activeElement.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
        inline: 'nearest',
      });
    });
  }

  function handleScroll() {
    // Visual Viewport scroll event fires when the viewport scrolls
    // relative to the layout viewport (e.g., when keyboard pushes content)
    // We can use this to re-center the focused element if needed
  }

  // Add event listeners
  window.visualViewport.addEventListener('resize', handleResize);
  window.visualViewport.addEventListener('scroll', handleScroll);

  // Initial check
  handleResize();

  return {
    update(_newOptions: VisualViewportOptions) {
      // Options can be updated if needed
    },
    destroy() {
      if (debounceTimer) clearTimeout(debounceTimer);

      // Restore styles
      node.style.height = '';
      node.style.overflow = '';

      // Remove listeners
      if (window.visualViewport) {
        window.visualViewport.removeEventListener('resize', handleResize);
        window.visualViewport.removeEventListener('scroll', handleScroll);
      }
    },
  };
}

/**
 * Store-based approach for components that need to react to keyboard state
 */
export function createKeyboardStore() {
  let isOpen = $state(false);
  let height = $state(0);

  function update(open: boolean, h: number) {
    isOpen = open;
    height = h;
  }

  return {
    get isOpen() {
      return isOpen;
    },
    get height() {
      return height;
    },
    update,
  };
}
