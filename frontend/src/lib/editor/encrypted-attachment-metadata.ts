export interface EncryptedAttachmentMetadata {
  version: 1;
  name: string;
  type: string;
}

const METADATA_PREFIX = 'xela-enc-v1:';
const DEFAULT_ATTACHMENT_NAME = 'attachment';
const DEFAULT_ATTACHMENT_MIME = 'application/octet-stream';

function normalizeName(name: string): string {
  const trimmed = name.trim();
  return trimmed || DEFAULT_ATTACHMENT_NAME;
}

function normalizeMimeType(type: string): string {
  const trimmed = type.trim().toLowerCase();
  return trimmed || DEFAULT_ATTACHMENT_MIME;
}

function bytesToBase64Url(bytes: Uint8Array): string {
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

function base64UrlToBytes(value: string): Uint8Array | null {
  if (!value) return null;
  let base64 = value.replace(/-/g, '+').replace(/_/g, '/');
  while (base64.length % 4 !== 0) {
    base64 += '=';
  }

  try {
    const binary = atob(base64);
    const out = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) {
      out[i] = binary.charCodeAt(i);
    }
    return out;
  } catch {
    return null;
  }
}

export function inferMimeTypeFromFilename(filename: string): string {
  const lower = filename.toLowerCase();
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
  if (lower.endsWith('.gif')) return 'image/gif';
  if (lower.endsWith('.webp')) return 'image/webp';
  if (lower.endsWith('.svg')) return 'image/svg+xml';
  if (lower.endsWith('.avif')) return 'image/avif';
  if (lower.endsWith('.pdf')) return 'application/pdf';
  if (lower.endsWith('.txt') || lower.endsWith('.md')) return 'text/plain';
  return DEFAULT_ATTACHMENT_MIME;
}

export function isImageMimeType(mimeType: string): boolean {
  return normalizeMimeType(mimeType).startsWith('image/');
}

export function encodeEncryptedAttachmentMetadata(input: { name: string; type: string }): string {
  const metadata: EncryptedAttachmentMetadata = {
    version: 1,
    name: normalizeName(input.name),
    type: normalizeMimeType(input.type),
  };
  const bytes = new TextEncoder().encode(JSON.stringify(metadata));
  return `${METADATA_PREFIX}${bytesToBase64Url(bytes)}`;
}

export function decodeEncryptedAttachmentMetadata(
  rawTitle: string | null | undefined
): EncryptedAttachmentMetadata | null {
  if (!rawTitle || !rawTitle.startsWith(METADATA_PREFIX)) return null;
  const encoded = rawTitle.slice(METADATA_PREFIX.length);
  const bytes = base64UrlToBytes(encoded);
  if (!bytes) return null;

  try {
    const parsed = JSON.parse(new TextDecoder().decode(bytes)) as {
      version?: unknown;
      name?: unknown;
      type?: unknown;
    };
    if (
      parsed.version !== 1 ||
      typeof parsed.name !== 'string' ||
      typeof parsed.type !== 'string'
    ) {
      return null;
    }
    return {
      version: 1,
      name: normalizeName(parsed.name),
      type: normalizeMimeType(parsed.type),
    };
  } catch {
    return null;
  }
}

export function stripFragment(url: string): string {
  const hashIndex = url.indexOf('#');
  return hashIndex >= 0 ? url.slice(0, hashIndex) : url;
}
