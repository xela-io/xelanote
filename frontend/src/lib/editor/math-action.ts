// Svelte action that renders math expressions marked with .math-pending.
// Lazily loads KaTeX on demand.

import type { ActionReturn } from 'svelte/action';

import { renderMath } from './math-loader';

export interface MathRendererOptions {
  revision?: string | number;
}

/**
 * Svelte action on the preview container.
 * Scans for .math-pending elements and replaces with KaTeX output.
 */
export function mathRenderer(
  node: HTMLElement,
  _options?: MathRendererOptions
): ActionReturn<MathRendererOptions | undefined> {
  function render() {
    const pendingElements = node.querySelectorAll<HTMLElement>('.math-pending');
    for (const el of pendingElements) {
      const tex = el.dataset.math;
      if (!tex) continue;

      const displayMode = el.dataset.display === 'true';

      // Mark as processing to avoid double-rendering
      el.classList.remove('math-pending');
      el.classList.add('math-processing');

      renderMath(tex, displayMode).then((html) => {
        if (el.parentNode) {
          el.innerHTML = html;
          el.classList.remove('math-processing');
          el.classList.add('math-rendered');
        }
      });
    }
  }

  render();

  return {
    update() {
      render();
    },
  };
}
