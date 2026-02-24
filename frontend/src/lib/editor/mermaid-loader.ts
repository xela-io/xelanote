// Lazy Mermaid diagram renderer with content-hash caching.

let mermaidModule: typeof import('mermaid') | null = null;
let initialized = false;

// FNV-1a hash for content-based caching
function fnv1a(str: string): string {
  let hash = 2166136261;
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

const svgCache = new Map<string, string>();

async function getMermaid(): Promise<typeof import('mermaid')> {
  if (!mermaidModule) {
    mermaidModule = await import('mermaid');
    if (!initialized) {
      mermaidModule.default.initialize({
        startOnLoad: false,
        securityLevel: 'strict',
        theme: 'neutral',
        fontFamily: 'inherit',
      });
      initialized = true;
    }
  }
  return mermaidModule;
}

/**
 * Render a Mermaid diagram definition to SVG.
 * Uses content-hash caching to avoid re-rendering identical diagrams.
 * Returns SVG string or null on error.
 */
export async function renderDiagram(definition: string): Promise<string | null> {
  const hash = fnv1a(definition);
  const cached = svgCache.get(hash);
  if (cached) return cached;

  try {
    const mermaid = await getMermaid();
    const id = `mermaid-${hash}-${Date.now()}`;
    const { svg } = await mermaid.default.render(id, definition);
    svgCache.set(hash, svg);
    return svg;
  } catch {
    return null;
  }
}
