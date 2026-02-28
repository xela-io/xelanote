import {
  decodeEncryptedAttachmentMetadata,
  encodeEncryptedAttachmentMetadata,
  inferMimeTypeFromFilename,
  isImageMimeType,
  stripFragment,
} from './encrypted-attachment-metadata';

const LEGACY_ATTACHMENT_PREFIX_RE =
  /^(Encrypted attachment|Verschluesselter Anhang|Verschl\u00fcsselter Anhang):\s*/i;
const IMAGE_TOKEN_RE = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g;
const LINK_TOKEN_RE = /\[(.*?)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g;

function isEncryptedAttachmentURL(url: string): boolean {
  const pathOnly = stripFragment(url).split('?', 1)[0].toLowerCase();
  return pathOnly.endsWith('.xenc');
}

function isLegacyAttachmentLabel(label: string): boolean {
  return LEGACY_ATTACHMENT_PREFIX_RE.test(label);
}

function normalizeAttachmentName(raw: string): string {
  const stripped = raw.replace(LEGACY_ATTACHMENT_PREFIX_RE, '').trim();
  return stripped || 'attachment';
}

function inferNameFromURL(url: string): string {
  const basename = decodeURIComponent(stripFragment(url).split('/').pop() || 'attachment');
  return basename.endsWith('.xenc') ? basename.slice(0, -'.xenc'.length) || 'attachment' : basename;
}

function extractMimeFromLegacyTitle(title: string | undefined): string | null {
  if (!title || !title.startsWith('xenc:')) return null;
  const value = title.slice('xenc:'.length).trim().toLowerCase();
  return value || null;
}

function extractHashParams(url: string): URLSearchParams {
  const [, hash = ''] = url.split('#', 2);
  return new URLSearchParams(hash);
}

function escapeMarkdownText(text: string): string {
  return text.replace(/[[\]\\]/g, '\\$&');
}

function buildImageToken(alt: string, url: string, metadataTitle: string): string {
  return `![${escapeMarkdownText(alt)}](${url} "${metadataTitle}")`;
}

function buildLinkToken(label: string, url: string, metadataTitle: string): string {
  return `[${label}](${url} "${metadataTitle}")`;
}

function resolveAttachmentInfo(
  label: string,
  url: string,
  title: string | undefined
): {
  name: string;
  mimeType: string;
  cleanURL: string;
} {
  const structured = decodeEncryptedAttachmentMetadata(title);
  const hashParams = extractHashParams(url);
  const hashName = hashParams.get('name')?.trim();
  const hashType = hashParams.get('type')?.trim().toLowerCase();

  const name =
    structured?.name ||
    hashName ||
    (isLegacyAttachmentLabel(label) ? normalizeAttachmentName(label) : label.trim()) ||
    inferNameFromURL(url);

  const mimeType =
    structured?.type ||
    hashType ||
    extractMimeFromLegacyTitle(title) ||
    inferMimeTypeFromFilename(name);

  return {
    name,
    mimeType,
    cleanURL: stripFragment(url),
  };
}

export function migrateLegacyEncryptedAttachmentLinks(content: string): {
  content: string;
  migratedCount: number;
} {
  let migratedCount = 0;

  const migrateImage = (full: string, alt: string, url: string, title?: string): string => {
    if (!isEncryptedAttachmentURL(url)) return full;

    const info = resolveAttachmentInfo(alt, url, title);
    const metadataTitle = encodeEncryptedAttachmentMetadata({
      name: info.name,
      type: info.mimeType,
    });
    const next = buildImageToken(alt.trim() || info.name, info.cleanURL, metadataTitle);
    if (next !== full) migratedCount++;
    return next;
  };

  const afterImages = content.replace(IMAGE_TOKEN_RE, migrateImage);

  const migratedContent = afterImages.replace(
    LINK_TOKEN_RE,
    (
      full,
      label: string,
      url: string,
      title: string | undefined,
      offset: number,
      source: string
    ) => {
      // Skip image tokens, they already start with "!["
      if (offset > 0 && source[offset - 1] === '!') return full;
      if (!isEncryptedAttachmentURL(url)) return full;

      const info = resolveAttachmentInfo(label, url, title);
      const metadataTitle = encodeEncryptedAttachmentMetadata({
        name: info.name,
        type: info.mimeType,
      });

      if (isLegacyAttachmentLabel(label) && isImageMimeType(info.mimeType)) {
        const nextImage = buildImageToken(info.name, info.cleanURL, metadataTitle);
        if (nextImage !== full) migratedCount++;
        return nextImage;
      }

      const nextLink = buildLinkToken(label, info.cleanURL, metadataTitle);
      if (nextLink !== full) migratedCount++;
      return nextLink;
    }
  );

  return { content: migratedContent, migratedCount };
}
