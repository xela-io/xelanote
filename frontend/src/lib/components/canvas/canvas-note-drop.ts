export const CANVAS_TREE_ITEM_MIME = 'application/x-xelanote-item';
export const CANVAS_LEGACY_NOTE_MIME = 'application/x-xelanote-note';
export const CANVAS_TEXT_PLAIN_MIME = 'text/plain';

export type CanvasDroppedNote = {
  id: string;
  title: string;
};

type UnifiedTreeNotePayload = {
  type: 'note';
  id: string;
  title: string;
};

type LegacyNotePayload = {
  id: string;
  title: string;
};

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object';
}

function parseJsonObject(raw: string): Record<string, unknown> | null {
  try {
    const parsed: unknown = JSON.parse(raw);
    return isObjectRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function parseUnifiedTreeNotePayload(raw: string): UnifiedTreeNotePayload | null {
  const parsed = parseJsonObject(raw);
  if (!parsed) return null;

  if (parsed.type !== 'note' || typeof parsed.id !== 'string' || typeof parsed.title !== 'string') {
    return null;
  }

  return {
    type: 'note',
    id: parsed.id,
    title: parsed.title,
  };
}

function parseLegacyNotePayload(raw: string): LegacyNotePayload | null {
  const parsed = parseJsonObject(raw);
  if (!parsed) return null;

  if (typeof parsed.id !== 'string' || typeof parsed.title !== 'string') {
    return null;
  }

  return {
    id: parsed.id,
    title: parsed.title,
  };
}

export function parseDroppedSidebarNote(dataTransfer: DataTransfer): CanvasDroppedNote | null {
  const treeItemRaw = dataTransfer.getData(CANVAS_TREE_ITEM_MIME);
  if (treeItemRaw) {
    const payload = parseUnifiedTreeNotePayload(treeItemRaw);
    if (payload) {
      return { id: payload.id, title: payload.title };
    }
  }

  const legacyRaw = dataTransfer.getData(CANVAS_LEGACY_NOTE_MIME);
  if (legacyRaw) {
    const payload = parseLegacyNotePayload(legacyRaw);
    if (payload) {
      return { id: payload.id, title: payload.title };
    }
  }

  const textPlainRaw = dataTransfer.getData(CANVAS_TEXT_PLAIN_MIME);
  if (textPlainRaw) {
    const unifiedPayload = parseUnifiedTreeNotePayload(textPlainRaw);
    if (unifiedPayload) {
      return { id: unifiedPayload.id, title: unifiedPayload.title };
    }
    const legacyPayload = parseLegacyNotePayload(textPlainRaw);
    if (legacyPayload) {
      return { id: legacyPayload.id, title: legacyPayload.title };
    }
  }

  return null;
}
