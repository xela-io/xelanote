// DOM morphing wrapper for preview content using Idiomorph.
// Morphs only changed nodes instead of replacing the entire DOM tree.

import { Idiomorph } from 'idiomorph';

/**
 * Morph the preview container's innerHTML to match the new HTML content.
 * Preserves DOM state like <details> open, scroll position, focus, etc.
 */
export function morphPreview(container: HTMLElement, newHtml: string): void {
  Idiomorph.morph(container, newHtml, {
    morphStyle: 'innerHTML',
    ignoreActiveValue: true,
    restoreFocus: true,
    callbacks: {
      // Preserve <details> open state (task collapse groups)
      beforeNodeMorphed(oldNode, newNode) {
        if (oldNode instanceof HTMLDetailsElement && newNode instanceof HTMLDetailsElement) {
          newNode.open = oldNode.open;
        }
        return true;
      },
    },
  });
}
