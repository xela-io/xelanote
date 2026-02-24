// Lazy KaTeX math renderer.
// renderToString does NOT require DOM — safe for Web Workers too.

let katexModule: typeof import('katex') | null = null;

async function getKaTeX(): Promise<typeof import('katex')> {
  if (!katexModule) {
    // Load KaTeX JS and CSS in parallel (CSS for fonts and layout)
    const [katex] = await Promise.all([import('katex'), import('katex/dist/katex.min.css')]);
    katexModule = katex;
  }
  return katexModule;
}

/**
 * Render a TeX expression to HTML.
 * Returns the rendered HTML string, or an error span if parsing fails.
 */
export async function renderMath(tex: string, displayMode: boolean): Promise<string> {
  const katex = await getKaTeX();
  try {
    return katex.default.renderToString(tex, {
      throwOnError: false,
      displayMode,
      output: 'htmlAndMathml',
    });
  } catch {
    return `<span class="katex-error" title="Invalid math expression">${escapeHtml(tex)}</span>`;
  }
}

/**
 * Synchronous render for use in markdown-it plugin (after KaTeX is loaded).
 * Returns null if KaTeX hasn't been loaded yet.
 */
export function renderMathSync(tex: string, displayMode: boolean): string | null {
  if (!katexModule) return null;
  try {
    return katexModule.default.renderToString(tex, {
      throwOnError: false,
      displayMode,
      output: 'htmlAndMathml',
    });
  } catch {
    return `<span class="katex-error" title="Invalid math expression">${escapeHtml(tex)}</span>`;
  }
}

/**
 * Preload KaTeX module. Call this when math content is detected.
 */
export async function preloadKaTeX(): Promise<void> {
  await getKaTeX();
}

function escapeHtml(text: string): string {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
