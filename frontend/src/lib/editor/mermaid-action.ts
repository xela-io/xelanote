// Svelte action that renders Mermaid diagrams in the preview container.
// Scans for <code class="language-mermaid"> blocks and replaces with SVG.
// SVG output is sanitized via DOMPurify (Mermaid renders user input!)
// and isolated in Shadow DOM to prevent CSS leakage.

import type { ActionReturn } from 'svelte/action';

import { sanitizeSvg } from './markdown/html-sanitizer';
import { renderDiagram } from './mermaid-loader';

export interface MermaidRendererOptions {
  revision?: string | number;
}

let debounceTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Svelte action on the preview container.
 * Finds code blocks with language-mermaid and renders them as diagrams.
 */
export function mermaidRenderer(
  node: HTMLElement,
  _options?: MermaidRendererOptions
): ActionReturn<MermaidRendererOptions | undefined> {
  function render() {
    // Find pre > code.language-mermaid blocks
    const codeBlocks = node.querySelectorAll<HTMLElement>(
      'pre code.language-mermaid, pre[data-lang="mermaid"] code'
    );

    for (const codeEl of codeBlocks) {
      const preEl = codeEl.parentElement;
      if (!preEl || preEl.classList.contains('mermaid-rendered')) continue;

      const definition = codeEl.textContent?.trim();
      if (!definition) continue;

      // Mark as processing
      preEl.classList.add('mermaid-processing');

      renderDiagram(definition).then((svg) => {
        if (!svg || !preEl.parentNode) {
          preEl.classList.remove('mermaid-processing');
          return;
        }

        // Sanitize SVG output (Mermaid renders user input — XSS risk)
        const sanitizedSvg = sanitizeSvg(svg);

        // Create wrapper with Shadow DOM for CSS isolation
        const wrapper = document.createElement('div');
        wrapper.className = 'mermaid-diagram';

        const shadow = wrapper.attachShadow({ mode: 'open' });
        shadow.innerHTML = sanitizedSvg;

        // Replace the pre block with the diagram
        preEl.parentNode.replaceChild(wrapper, preEl);
      });
    }
  }

  function debouncedRender() {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(render, 500);
  }

  debouncedRender();

  return {
    update() {
      debouncedRender();
    },
    destroy() {
      if (debounceTimer) {
        clearTimeout(debounceTimer);
        debounceTimer = null;
      }
    },
  };
}
