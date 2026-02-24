// Lazy Shiki syntax highlighter with CSS variable theme for Gruvbox integration.

type ShikiHighlighterLike = {
  loadLanguage: (lang: unknown) => Promise<void>;
  codeToHtml: (code: string, options: { lang: string; theme: string }) => string;
};

let highlighterPromise: Promise<ShikiHighlighterLike> | null = null;

// Languages loaded on first encounter
const loadedLanguages = new Set<string>();

async function getHighlighter(): Promise<ShikiHighlighterLike> {
  if (!highlighterPromise) {
    highlighterPromise = import('shiki/bundle/web').then(async ({ createHighlighter }) => {
      const highlighter = await createHighlighter({
        themes: ['css-variables'],
        langs: [],
      });
      return highlighter as unknown as ShikiHighlighterLike;
    });
  }
  return highlighterPromise;
}

/**
 * Common language aliases that map to shiki bundle/web language IDs.
 */
const LANG_ALIASES: Record<string, string> = {
  js: 'javascript',
  ts: 'typescript',
  tsx: 'tsx',
  jsx: 'jsx',
  sh: 'shellscript',
  bash: 'shellscript',
  shell: 'shellscript',
  zsh: 'shellscript',
  yml: 'yaml',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  md: 'markdown',
  dockerfile: 'docker',
  tf: 'hcl',
};

function resolveLanguage(lang: string): string {
  const lower = lang.toLowerCase();
  return LANG_ALIASES[lower] ?? lower;
}

/**
 * Highlight code with Shiki. Loads language grammar on demand.
 * Returns highlighted HTML string or null if language is not available.
 */
export async function highlightCode(code: string, lang: string): Promise<string | null> {
  const resolvedLang = resolveLanguage(lang);
  try {
    const highlighter = await getHighlighter();

    // Load language on demand if not yet loaded
    if (!loadedLanguages.has(resolvedLang)) {
      try {
        await highlighter.loadLanguage(resolvedLang);
        loadedLanguages.add(resolvedLang);
      } catch {
        // Language not available in shiki/bundle/web — return null for fallback
        return null;
      }
    }

    return highlighter.codeToHtml(code, {
      lang: resolvedLang,
      theme: 'css-variables',
    });
  } catch {
    return null;
  }
}
