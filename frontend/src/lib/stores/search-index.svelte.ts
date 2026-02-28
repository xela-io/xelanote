// Client-side fulltext search index for encrypted notes
// Uses MiniSearch for in-memory indexing of decrypted content

import MiniSearch from 'minisearch';
import { SvelteMap, SvelteSet } from 'svelte/reactivity';

import type { SearchResult } from '$lib/api';
import { getAllEncryptedNotes } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import * as encryption from '$lib/stores/encryption.svelte';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

const MAX_CONTENT_LENGTH = 50_000; // 50 KB per note

// MiniSearch instance and content map (for snippets)
let index: MiniSearch | null = null;
const contentMap = new SvelteMap<string, string>();
const truncatedNotes = new SvelteSet<string>();

// Svelte 5 runes
let indexState = $state<'idle' | 'building' | 'ready' | 'error'>('idle');
let indexError = $state<string | null>(null);
let indexProgress = $state<{ current: number; total: number }>({ current: 0, total: 0 });
let buildController: AbortController | null = null;

// Queue for updates arriving during index build
interface PendingUpdate {
  type: 'add' | 'update' | 'remove';
  noteId: string;
  title?: string;
  content?: string;
}
const pendingUpdates = new SvelteMap<string, PendingUpdate>();

// --- Public API: State getters ---

export function getIndexState(): string {
  return indexState;
}

export function getIndexError(): string | null {
  return indexError;
}

export function getIndexProgress(): { current: number; total: number } {
  return indexProgress;
}

// --- Private helpers ---

function resetState(): void {
  if (buildController) {
    buildController.abort();
    buildController = null;
  }
  index = null;
  contentMap.clear();
  truncatedNotes.clear();
  pendingUpdates.clear();
  indexState = 'idle';
  indexError = null;
  indexProgress = { current: 0, total: 0 };
}

function createMiniSearchInstance(): MiniSearch {
  return new MiniSearch({
    fields: ['title', 'content'],
    storeFields: ['title'],
    searchOptions: {
      prefix: true,
      boost: { title: 2 },
    },
  });
}

function decryptNotePayload(note: {
  id: string;
  encrypted_title?: string | null;
  encrypted_content?: string | null;
  encryption_metadata?: string | null;
}): { title: string | null; content: string } {
  const encryptedPayload: EncryptedPayload = {
    ciphertext: note.encrypted_content!,
    metadata: parseEncryptionMetadata(note.encryption_metadata),
  };
  return encryption.decryptNote(note.encrypted_title || null, encryptedPayload, note.id);
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function generateSnippet(content: string, queryTerms: string[]): string {
  const lowerContent = content.toLowerCase();
  const lowerTerms = queryTerms.map((t) => t.toLowerCase());
  let bestPos = -1;

  for (const term of lowerTerms) {
    const pos = lowerContent.indexOf(term);
    if (pos !== -1 && (bestPos === -1 || pos < bestPos)) bestPos = pos;
  }
  if (bestPos === -1) bestPos = 0;

  const start = Math.max(0, bestPos - 60);
  const end = Math.min(content.length, bestPos + 90);
  let snippet = content.substring(start, end);
  if (start > 0) snippet = '...' + snippet;
  if (end < content.length) snippet += '...';

  // HTML-escape everything first
  snippet = escapeHtml(snippet);

  // Then wrap matched terms with <mark> (on escaped text)
  for (const term of lowerTerms) {
    if (!term) continue;
    const escapedTerm = escapeHtml(term);
    const regex = new RegExp(`(${escapeRegex(escapedTerm)})`, 'gi');
    snippet = snippet.replace(regex, '<mark>$1</mark>');
  }

  return snippet;
}

function applyPendingUpdates(): void {
  if (!index) return;

  for (const update of pendingUpdates.values()) {
    switch (update.type) {
      case 'add':
        if (!index.has(update.noteId)) {
          index.add({ id: update.noteId, title: update.title, content: update.content });
          contentMap.set(update.noteId, update.content || '');
        }
        break;
      case 'update':
        if (index.has(update.noteId)) {
          index.discard(update.noteId);
        }
        index.add({ id: update.noteId, title: update.title, content: update.content });
        contentMap.set(update.noteId, update.content || '');
        break;
      case 'remove':
        if (index.has(update.noteId)) {
          index.discard(update.noteId);
        }
        contentMap.delete(update.noteId);
        break;
    }
  }
  pendingUpdates.clear();
}

// --- Public API: Lifecycle ---

export async function buildIndex(): Promise<void> {
  // Don't build if already building or if encryption is locked
  if (indexState === 'building') return;
  if (!encryption.isEncryptionUnlocked()) return;

  indexState = 'building';
  indexError = null;
  buildController = new AbortController();

  try {
    const notes = await getAllEncryptedNotes();

    if (buildController.signal.aborted) {
      resetState();
      return;
    }

    index = createMiniSearchInstance();
    contentMap.clear();
    truncatedNotes.clear();

    let batchSize = 20;
    for (let i = 0; i < notes.length; i += batchSize) {
      if (buildController.signal.aborted) {
        resetState();
        return;
      }

      const start = performance.now();
      const batch = notes.slice(i, i + batchSize);

      for (const note of batch) {
        try {
          if (!note.encrypted_content) continue;

          const decrypted = decryptNotePayload(note);
          const title = decrypted.title || note.title || '';
          let content = decrypted.content || '';

          if (content.length > MAX_CONTENT_LENGTH) {
            console.warn(
              `[SEARCH-INDEX] Note ${note.id} truncated: ${content.length} -> ${MAX_CONTENT_LENGTH}`
            );
            content = content.substring(0, MAX_CONTENT_LENGTH);
            truncatedNotes.add(note.id);
          }

          index!.add({ id: note.id, title, content });
          contentMap.set(note.id, content);
        } catch (err) {
          console.warn(`[SEARCH-INDEX] Failed to decrypt note ${note.id}, skipping:`, err);
        }
      }

      const elapsed = performance.now() - start;
      // Adaptive batch sizing: target 16ms per batch (1 frame)
      if (elapsed > 20) batchSize = Math.max(5, Math.floor(batchSize * 0.7));
      else if (elapsed < 8) batchSize = Math.min(50, Math.floor(batchSize * 1.3));

      indexProgress = { current: Math.min(i + batch.length, notes.length), total: notes.length };
      await new Promise((r) => setTimeout(r, 0)); // Yield to UI
    }

    // Apply any updates that arrived during build
    applyPendingUpdates();

    indexState = 'ready';
    console.log(`[SEARCH-INDEX] Ready (${contentMap.size} notes indexed)`);
  } catch (err) {
    console.error('[SEARCH-INDEX] Build failed:', err);
    indexState = 'error';
    indexError = err instanceof Error ? err.message : 'Unknown error';
    index = null;
    contentMap.clear();
    truncatedNotes.clear();
    pendingUpdates.clear();
  }
}

export function destroyIndex(): void {
  resetState();
}

export function cancelBuild(): void {
  resetState();
}

// --- Public API: Search ---

export function searchEncrypted(query: string, limit = 50): SearchResult[] {
  if (!index || indexState !== 'ready') {
    console.log(`[SEARCH-INDEX] Search skipped: index=${!!index}, state=${indexState}`);
    return [];
  }
  if (!query.trim()) return [];

  const fuzzy = query.length >= 3 ? 0.2 : false;
  const results = index.search(query, { fuzzy, prefix: true });
  const limitedResults = results.slice(0, limit);
  console.log(
    `[SEARCH-INDEX] Search "${query}": ${results.length} results (index has ${contentMap.size} docs)`
  );

  // Extract query terms for snippet generation
  const queryTerms = query
    .trim()
    .split(/\s+/)
    .filter((t) => t.length > 0);

  return limitedResults.map((result) => {
    const content = contentMap.get(result.id) || '';
    const snippet = generateSnippet(content, queryTerms);

    return {
      id: result.id,
      title: (result.title as string) || '',
      snippet,
      rank: 0, // Not used for client results
      encrypted: true,
    };
  });
}

// --- Public API: Incremental Updates ---

export function addToIndex(noteId: string, title: string, content: string): void {
  const truncated = content.substring(0, MAX_CONTENT_LENGTH);
  if (indexState === 'building') {
    pendingUpdates.set(noteId, { type: 'add', noteId, title, content: truncated });
    return;
  }
  if (!index || indexState !== 'ready') return;
  if (!index.has(noteId)) {
    index.add({ id: noteId, title, content: truncated });
  }
  contentMap.set(noteId, truncated);
}

export function updateInIndex(noteId: string, title: string, content: string): void {
  const truncated = content.substring(0, MAX_CONTENT_LENGTH);
  if (indexState === 'building') {
    pendingUpdates.set(noteId, { type: 'update', noteId, title, content: truncated });
    return;
  }
  if (!index || indexState !== 'ready') return;
  if (index.has(noteId)) {
    index.discard(noteId);
  }
  index.add({ id: noteId, title, content: truncated });
  contentMap.set(noteId, truncated);
}

export function removeFromIndex(noteId: string): void {
  if (indexState === 'building') {
    pendingUpdates.set(noteId, { type: 'remove', noteId });
    return;
  }
  if (!index || indexState !== 'ready') return;
  if (index.has(noteId)) {
    index.discard(noteId);
  }
  contentMap.delete(noteId);
}
