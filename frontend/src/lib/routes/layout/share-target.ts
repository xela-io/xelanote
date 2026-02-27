import type { Note } from '$lib/api';

export interface ShareTargetDeps {
  isAuthenticated: () => boolean;
  createNote: (title: string, content?: string) => Promise<Note | undefined>;
  goto: (path: string) => void;
  notifySuccessfulAction: () => void;
}

/**
 * Stash share parameters to sessionStorage for processing after login.
 * Called when the user is not yet authenticated but received a Web Share Target intent.
 */
export function stashPendingShare(): void {
  try {
    const params = new URL(window.location.href).searchParams;
    const title = safeGetParam(params, 'title');
    const text = safeGetParam(params, 'text');
    const url = safeGetParam(params, 'url');
    if (!title && !text && !url) return;

    sessionStorage.setItem(
      'xelanote-pending-share',
      JSON.stringify({
        title: (title ?? '').slice(0, 200),
        text: (text ?? '').slice(0, 50_000),
        url: (url ?? '').slice(0, 2048),
      })
    );
    window.history.replaceState(null, '', window.location.pathname);
  } catch {
    // silent — sessionStorage unavailable
  }
}

/**
 * Process a pending share that was stashed before authentication.
 */
export async function processPendingShare(deps: ShareTargetDeps): Promise<void> {
  try {
    const raw = sessionStorage.getItem('xelanote-pending-share');
    if (!raw) return;
    sessionStorage.removeItem('xelanote-pending-share');

    const parsed = JSON.parse(raw);
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      typeof parsed.title !== 'string' ||
      typeof parsed.text !== 'string' ||
      typeof parsed.url !== 'string'
    ) {
      return;
    }

    await createNoteFromShare(parsed.title, parsed.text, parsed.url, deps);
  } catch {
    // Parse error or sessionStorage error — silently ignore
    try {
      sessionStorage.removeItem('xelanote-pending-share');
    } catch {
      /* silent */
    }
  }
}

/**
 * Process Web Share Target parameters from the current URL (Chromium: share text/URLs to create notes).
 */
export async function processShareTarget(deps: ShareTargetDeps): Promise<void> {
  const params = new URL(window.location.href).searchParams;
  const title = (safeGetParam(params, 'title') ?? '').slice(0, 200);
  const text = (safeGetParam(params, 'text') ?? '').slice(0, 50_000);
  const url = (safeGetParam(params, 'url') ?? '').slice(0, 2048);

  if (!title && !text && !url) return;

  try {
    window.history.replaceState(null, '', window.location.pathname);
  } catch {
    // replaceState may throw SecurityError on iOS Safari PWA
  }
  await createNoteFromShare(title, text, url, deps);
}

function safeGetParam(params: URLSearchParams, key: string): string | null {
  try {
    const val = params.get(key);
    return val ? val.trim() : null;
  } catch {
    // decodeURIComponent error on malformed %-encoded params
    return null;
  }
}

async function createNoteFromShare(
  title: string,
  text: string,
  url: string,
  deps: ShareTargetDeps
): Promise<void> {
  let content = text;
  if (url) {
    content = content ? content + '\n\n' + url : url;
  }
  if (!title && !content) return;

  const note = await deps.createNote(title || '', content);
  if (note?.id) {
    deps.goto(`/note/${note.id}`);
    deps.notifySuccessfulAction();
  }
}
