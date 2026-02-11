import type { ComponentType } from 'svelte';

export function loadSvelteComponentFromModule(module: unknown, context: string): ComponentType {
  if (!module || typeof module !== 'object') {
    throw new Error(`Invalid lazy component module in ${context}`);
  }

  const maybe = module as { default?: unknown };
  if (!maybe.default) {
    throw new Error(`Lazy component module missing default export in ${context}`);
  }

  return maybe.default as ComponentType;
}
