/// <reference lib="webworker" />

// Web Worker for markdown-it rendering.
// Performs the expensive markdown-it parse+render off the main thread.
// DOMPurify sanitization happens on the main thread (requires DOM).

import type MarkdownIt from 'markdown-it';

import { extractImageWidths } from './markdown/image-plugin';
import { addDragHandlesToTasks, getRenderedTaskLineNumbers } from './markdown/task-processor';
import { createConfiguredMarkdownIt } from './markdown-config';

let md: MarkdownIt | null = null;

function getMarkdownInstance(): MarkdownIt {
  if (!md) {
    md = createConfiguredMarkdownIt();
  }
  return md;
}

export type WorkerRenderRequest = {
  type: 'render';
  id: number;
  content: string;
  titleToIdMap?: [string, string][];
  resolvedTitles?: string[];
  imageResize: boolean;
  taskLists: boolean;
};

export type WorkerCancelRequest = {
  type: 'cancel';
  id: number;
};

export type WorkerRequest = WorkerRenderRequest | WorkerCancelRequest;

export type WorkerResponse = {
  type: 'render-done';
  id: number;
  html: string;
};

// Track cancelled request IDs so we can skip sending results
const cancelledIds = new Set<number>();

self.onmessage = (event: MessageEvent<WorkerRequest>) => {
  const msg = event.data;

  if (msg.type === 'cancel') {
    cancelledIds.add(msg.id);
    return;
  }

  if (msg.type !== 'render') return;

  const instance = getMarkdownInstance();

  // Extract image widths and clean content
  let processedContent = msg.content;
  let widthMap = new Map<number, string>();

  if (msg.imageResize) {
    const extracted = extractImageWidths(msg.content);
    processedContent = extracted.cleanContent;
    widthMap = extracted.widthMap;
  }

  // Reconstruct Maps/Sets from serialized data
  const env: Record<string, unknown> = {
    widthMap,
    titleToIdMap: msg.titleToIdMap ? new Map(msg.titleToIdMap) : undefined,
    resolvedTitles: msg.resolvedTitles ? new Set(msg.resolvedTitles) : undefined,
  };

  let html = instance.render(processedContent, env);

  // Add drag handles (string manipulation, no DOM needed)
  if (msg.taskLists) {
    html = addDragHandlesToTasks(html, getRenderedTaskLineNumbers(processedContent));
  }

  // Skip if cancelled while rendering
  if (cancelledIds.has(msg.id)) {
    cancelledIds.delete(msg.id);
    return;
  }

  // Return unsanitized HTML — DOMPurify runs on main thread
  const response: WorkerResponse = { type: 'render-done', id: msg.id, html };
  self.postMessage(response);
};
