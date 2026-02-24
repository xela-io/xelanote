// Client wrapper for the markdown rendering web worker.
// Auto-cancels previous requests, falls back to sync rendering on timeout.

import { FEATURE_FLAGS } from '$lib/config';

import type { WorkerCancelRequest, WorkerRenderRequest, WorkerResponse } from './markdown.worker';
import { sanitizeRenderedHtml } from './markdown/html-sanitizer';

export interface AsyncRenderOptions {
  resolvedTitles?: Set<string>;
  titleToIdMap?: Map<string, string>;
}

let worker: Worker | null = null;
let nextRequestId = 1;
let pendingResolve: ((html: string) => void) | null = null;
let pendingId = 0;
let pendingTimer: ReturnType<typeof setTimeout> | null = null;

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(new URL('./markdown.worker.ts', import.meta.url), { type: 'module' });
    worker.onmessage = (event: MessageEvent<WorkerResponse>) => {
      const msg = event.data;
      if (msg.type === 'render-done' && msg.id === pendingId && pendingResolve) {
        if (pendingTimer) {
          clearTimeout(pendingTimer);
          pendingTimer = null;
        }
        // Sanitize on main thread (DOMPurify needs DOM)
        const sanitized = sanitizeRenderedHtml(msg.html);
        pendingResolve(sanitized);
        pendingResolve = null;
      }
    };
  }
  return worker;
}

/**
 * Render markdown asynchronously via web worker.
 * Auto-cancels any previous pending request.
 * Falls back to synchronous rendering on timeout (500ms).
 */
export function renderMarkdownAsync(
  content: string,
  options: AsyncRenderOptions = {}
): Promise<string> {
  // Cancel previous pending request
  if (pendingResolve) {
    // Tell the worker to skip the old request
    if (worker && pendingId) {
      const cancel: WorkerCancelRequest = { type: 'cancel', id: pendingId };
      worker.postMessage(cancel);
    }
    pendingResolve(''); // Resolve with empty to prevent hanging promises
    pendingResolve = null;
  }
  if (pendingTimer) {
    clearTimeout(pendingTimer);
    pendingTimer = null;
  }

  const id = nextRequestId++;
  pendingId = id;

  return new Promise<string>((resolve) => {
    pendingResolve = resolve;

    // Timeout fallback: if worker takes too long, render synchronously
    pendingTimer = setTimeout(async () => {
      if (pendingId === id && pendingResolve) {
        pendingResolve = null;
        // Dynamic import to avoid circular dependency
        const { renderMarkdown } = await import('./markdown');
        resolve(renderMarkdown(content, options));
      }
    }, 500);

    const w = getWorker();
    const request: WorkerRenderRequest = {
      type: 'render',
      id,
      content,
      titleToIdMap: options.titleToIdMap ? [...options.titleToIdMap.entries()] : undefined,
      resolvedTitles: options.resolvedTitles ? [...options.resolvedTitles] : undefined,
      imageResize: FEATURE_FLAGS.imageResize,
      taskLists: FEATURE_FLAGS.taskLists,
    };
    w.postMessage(request);
  });
}

/**
 * Terminate the worker (call on cleanup/navigation).
 */
export function terminateMarkdownWorker(): void {
  if (worker) {
    worker.terminate();
    worker = null;
  }
  if (pendingTimer) {
    clearTimeout(pendingTimer);
    pendingTimer = null;
  }
  pendingResolve = null;
}
