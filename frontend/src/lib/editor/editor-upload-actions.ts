import type { EditorView } from '@codemirror/view';

import { uploadImages } from '$lib/editor/image-upload';

interface UploadToast {
  success: (message: string) => void;
  warning: (message: string) => void;
  error: (message: string) => void;
}

interface UploadMessages {
  success: (filename?: string) => string;
  copiedToClipboard: string;
  fallback: (url?: string) => string;
  error: (filename?: string, message?: string) => string;
}

interface UploadImagesFromEditorParams {
  files: File[];
  editorView: EditorView | undefined;
  setUploading: (value: boolean) => void;
  toast: UploadToast;
  messages: UploadMessages;
}

export async function uploadImagesFromEditorAction(
  params: UploadImagesFromEditorParams
): Promise<void> {
  const { files, editorView, setUploading, toast, messages } = params;

  await uploadImages(files, {
    editorView,
    onStatus: (value) => {
      setUploading(value);
    },
    onSuccess: (_event, ctx) => {
      toast.success(messages.success(ctx?.filename));
    },
    onWarning: (event, ctx) => {
      if (event === 'copied_to_clipboard') {
        toast.warning(messages.copiedToClipboard);
      } else {
        toast.warning(messages.fallback(ctx?.url));
      }
    },
    onError: (_event, ctx) => {
      toast.error(messages.error(ctx?.filename, ctx?.error));
    },
  });
}
