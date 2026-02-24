// Svelte action for rendering preview HTML with optional DOM morphing.
// First render uses innerHTML, subsequent updates morph via Idiomorph.

import type { ActionReturn } from 'svelte/action';

import { FEATURE_FLAGS } from '$lib/config';

import { morphPreview } from './preview-morph';

export interface PreviewRendererOptions {
  html: string;
}

/**
 * Svelte action that manages preview HTML rendering.
 * When morphPreview feature flag is enabled, uses Idiomorph for updates.
 * Otherwise falls back to innerHTML replacement.
 */
export function previewRenderer(
  node: HTMLElement,
  options: PreviewRendererOptions
): ActionReturn<PreviewRendererOptions> {
  let isFirstRender = true;

  function render(html: string) {
    if (isFirstRender || !FEATURE_FLAGS.morphPreview) {
      node.innerHTML = html;
      isFirstRender = false;
    } else {
      morphPreview(node, html);
    }
  }

  render(options.html);

  return {
    update(newOptions: PreviewRendererOptions) {
      render(newOptions.html);
    },
  };
}
