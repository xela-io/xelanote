import type { ActionReturn } from 'svelte/action';

import type { EncryptionVersion } from '$lib/crypto/e2e';
import * as encryption from '$lib/stores/encryption.svelte';

import {
  decodeEncryptedAttachmentMetadata,
  inferMimeTypeFromFilename,
} from './encrypted-attachment-metadata';

const ENCRYPTED_FILE_EXT = '.xenc';
const DEFAULT_ATTACHMENT_NAME = 'attachment';
const DEFAULT_ATTACHMENT_MIME = 'application/octet-stream';

const FILENAME_PREFIX_RE =
  /^(Encrypted attachment|Verschluesselter Anhang|Verschl\u00fcsselter Anhang):\s*/i;

export interface EncryptedAttachmentContext {
  noteID: string;
  wrappedDEK: string;
  metadataVersion: EncryptionVersion;
}

export interface EncryptedAttachmentRendererOptions {
  revision: string;
  context: EncryptedAttachmentContext | null;
}

type TargetKind = 'image' | 'link';

interface AttachmentTarget {
  kind: TargetKind;
  element: HTMLImageElement | HTMLAnchorElement;
  sourceUrl: string; // absolute URL, without hash
  filename: string;
  mimeType: string;
}

interface ResolvedAttachment {
  objectUrl: string;
  filename: string;
  mimeType: string;
}

function cleanAttachmentLabel(text: string): string {
  return text.replace(FILENAME_PREFIX_RE, '').trim();
}

function mimeFromTitle(title: string | null): string | null {
  if (!title) return null;
  if (!title.startsWith('xenc:')) return null;
  const value = title.slice('xenc:'.length).trim();
  return value.length > 0 ? value : null;
}

function inferFilenameFromURL(url: URL): string {
  const base = decodeURIComponent(url.pathname.split('/').pop() || DEFAULT_ATTACHMENT_NAME);
  return base.endsWith(ENCRYPTED_FILE_EXT)
    ? base.slice(0, -ENCRYPTED_FILE_EXT.length) || DEFAULT_ATTACHMENT_NAME
    : base;
}

function resolveFilename(url: URL, element: HTMLElement, markerName: string | null): string {
  if (markerName) {
    return cleanAttachmentLabel(markerName) || DEFAULT_ATTACHMENT_NAME;
  }

  if (element instanceof HTMLImageElement) {
    const fromAlt = cleanAttachmentLabel(element.alt || '');
    if (fromAlt) return fromAlt;
  }

  if (element instanceof HTMLAnchorElement) {
    const fromText = cleanAttachmentLabel(element.textContent || '');
    if (fromText) return fromText;
  }

  return inferFilenameFromURL(url);
}

function buildAttachmentTarget(element: HTMLElement): AttachmentTarget | null {
  const isImage = element instanceof HTMLImageElement;
  const rawUrl = isImage ? element.getAttribute('src') : element.getAttribute('href');
  if (!rawUrl) return null;

  let parsed: URL;
  try {
    parsed = new URL(rawUrl, window.location.origin);
  } catch {
    return null;
  }

  if (!parsed.pathname.toLowerCase().endsWith(ENCRYPTED_FILE_EXT)) {
    return null;
  }

  const hashParams = new URLSearchParams(parsed.hash.replace(/^#/, ''));
  const structured = decodeEncryptedAttachmentMetadata(element.getAttribute('title'));
  const markerName = hashParams.get('name');
  const markerType = hashParams.get('type');
  const filename = structured?.name || resolveFilename(parsed, element, markerName);
  const mimeType =
    structured?.type ||
    markerType ||
    mimeFromTitle(element.getAttribute('title')) ||
    inferMimeTypeFromFilename(filename) ||
    DEFAULT_ATTACHMENT_MIME;

  return {
    kind: isImage ? 'image' : 'link',
    element: isImage ? element : (element as HTMLAnchorElement),
    sourceUrl: `${parsed.origin}${parsed.pathname}${parsed.search}`,
    filename: filename || DEFAULT_ATTACHMENT_NAME,
    mimeType,
  };
}

async function fetchAndDecryptAttachment(
  target: AttachmentTarget,
  context: EncryptedAttachmentContext
): Promise<ResolvedAttachment> {
  const response = await fetch(target.sourceUrl, {
    method: 'GET',
    credentials: 'include',
  });
  if (!response.ok) {
    throw new Error(`download failed (${response.status})`);
  }

  const encryptedBytes = new Uint8Array(await response.arrayBuffer());
  const decryptedBytes = encryption.decryptAttachment(encryptedBytes, {
    noteID: context.noteID,
    wrappedDEK: context.wrappedDEK,
    metadataVersion: context.metadataVersion,
    filename: target.filename,
  });
  const decryptedBuffer = new Uint8Array(decryptedBytes).buffer;
  const blob = new Blob([decryptedBuffer], { type: target.mimeType || DEFAULT_ATTACHMENT_MIME });
  const objectUrl = URL.createObjectURL(blob);

  return {
    objectUrl,
    filename: target.filename,
    mimeType: target.mimeType || DEFAULT_ATTACHMENT_MIME,
  };
}

function applyResolvedAttachment(target: AttachmentTarget, resolved: ResolvedAttachment): void {
  if (target.kind === 'image') {
    const image = target.element as HTMLImageElement;
    image.src = resolved.objectUrl;
    image.setAttribute('data-xela-encrypted-image', 'decrypted');
    return;
  }

  const link = target.element as HTMLAnchorElement;
  link.href = resolved.objectUrl;
  link.setAttribute('download', resolved.filename);
  link.setAttribute('data-xela-encrypted-link', 'decrypted');
}

export function encryptedAttachmentRenderer(
  node: HTMLElement,
  options: EncryptedAttachmentRendererOptions
): ActionReturn<EncryptedAttachmentRendererOptions> {
  let currentOptions = options;
  let contextSignature = '';
  let generation = 0;
  let destroyed = false;

  const resolvedCache = new Map<string, ResolvedAttachment>();
  const pendingCache = new Map<string, Promise<ResolvedAttachment>>();

  function clearCaches(): void {
    for (const entry of resolvedCache.values()) {
      URL.revokeObjectURL(entry.objectUrl);
    }
    resolvedCache.clear();
    pendingCache.clear();
  }

  async function resolveAttachment(
    target: AttachmentTarget,
    context: EncryptedAttachmentContext
  ): Promise<ResolvedAttachment> {
    const cacheKey = `${context.noteID}:${context.metadataVersion}:${context.wrappedDEK}:${target.sourceUrl}:${target.filename}`;
    const cached = resolvedCache.get(cacheKey);
    if (cached) return cached;

    const pending = pendingCache.get(cacheKey);
    if (pending) return pending;

    const promise = fetchAndDecryptAttachment(target, context)
      .then((resolved) => {
        resolvedCache.set(cacheKey, resolved);
        pendingCache.delete(cacheKey);
        return resolved;
      })
      .catch((err) => {
        pendingCache.delete(cacheKey);
        throw err;
      });

    pendingCache.set(cacheKey, promise);
    return promise;
  }

  async function process(): Promise<void> {
    const context = currentOptions.context;
    const runGeneration = ++generation;
    if (!context || !encryption.isEncryptionUnlocked()) {
      clearCaches();
      contextSignature = '';
      return;
    }

    const signature = `${context.noteID}:${context.metadataVersion}:${context.wrappedDEK}`;
    if (signature !== contextSignature) {
      clearCaches();
      contextSignature = signature;
    }

    const elements = Array.from(node.querySelectorAll<HTMLElement>('img[src], a[href]'));
    const targets = elements
      .map((el) => buildAttachmentTarget(el))
      .filter((target): target is AttachmentTarget => target !== null);

    await Promise.all(
      targets.map(async (target) => {
        try {
          const resolved = await resolveAttachment(target, context);
          if (destroyed || runGeneration !== generation) return;
          applyResolvedAttachment(target, resolved);
        } catch (err) {
          console.warn('[ENCRYPTION] Failed to resolve encrypted attachment:', err);
        }
      })
    );
  }

  void process();

  return {
    update(newOptions: EncryptedAttachmentRendererOptions) {
      currentOptions = newOptions;
      void process();
    },
    destroy() {
      destroyed = true;
      clearCaches();
    },
  };
}
