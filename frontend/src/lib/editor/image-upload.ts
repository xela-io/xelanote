import type { EditorView } from '@codemirror/view';
import * as api from '$lib/api';
import { ApiError } from '$lib/api';

export interface ImageUploadHandlers {
  editorView?: EditorView | null;
  onStatus: (uploading: boolean) => void;
  onSuccess: (message: string, context?: { filename: string }) => void;
  onWarning: (message: string, context?: { filename?: string; url?: string }) => void;
  onError: (message: string, context?: { filename: string; error: string }) => void;
}

export function handleEditorDrop(e: DragEvent, upload: (files: File[]) => void) {
  const files = Array.from(e.dataTransfer?.files || []);
  const imageFiles = files.filter((f) => f.type.startsWith('image/'));

  if (imageFiles.length > 0) {
    e.preventDefault();
    upload(imageFiles);
  }
}

export function handleEditorDragOver(e: DragEvent) {
  // Check if dragging files (not notes/folders)
  const types = e.dataTransfer?.types || [];
  if (types.includes('Files')) {
    e.preventDefault();
    e.dataTransfer!.dropEffect = 'copy';
  }
}

export function handleEditorPaste(e: ClipboardEvent, upload: (files: File[]) => void) {
  const items = Array.from(e.clipboardData?.items || []);
  const imageItems = items.filter((item) => item.type.startsWith('image/'));

  if (imageItems.length > 0) {
    e.preventDefault();
    const files = imageItems.map((item) => item.getAsFile()).filter(Boolean) as File[];
    upload(files);
  }
}

export async function uploadImages(files: File[], handlers: ImageUploadHandlers) {
  handlers.onStatus(true);

  for (const file of files) {
    try {
      const { url } = await api.uploadImage(file);

      // Insert markdown at cursor
      const markdown = `\n![${file.name}](${url})\n`;
      const inserted = insertTextAtCursor(markdown, handlers);

      if (inserted) {
        handlers.onSuccess('uploaded', { filename: file.name });
      } else {
        // Fallback: Copy to clipboard if editor not ready
        try {
          await navigator.clipboard.writeText(markdown);
          handlers.onWarning('copied_to_clipboard', { filename: file.name });
        } catch {
          handlers.onWarning('copied_url', { url });
        }
      }
    } catch (err: unknown) {
      const message =
        err instanceof ApiError ? err.message : err instanceof Error ? err.message : String(err);
      handlers.onError('upload_failed', { filename: file.name, error: message });
    }
  }

  handlers.onStatus(false);
}

export function insertTextAtCursor(text: string, handlers: ImageUploadHandlers): boolean {
  const { editorView } = handlers;
  if (!editorView) {
    return false;
  }

  const selection = editorView.state.selection.main;
  editorView.dispatch({
    changes: {
      from: selection.from,
      to: selection.to,
      insert: text,
    },
    selection: { anchor: selection.from + text.length },
  });

  // Focus editor
  editorView.focus();
  return true;
}
