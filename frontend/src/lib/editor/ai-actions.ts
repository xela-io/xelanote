import type { EditorView } from '@codemirror/view';
import type { AIAction } from '$lib/api';

export interface AITransformState {
  action: AIAction;
  customPrompt?: string;
  originalText: string;
  selectionFrom: number;
  selectionTo: number;
  isFullContent: boolean;
  initialContentHash: string;
}

export interface AITransformPlan {
  state: AITransformState;
  selectionFrom: number;
  selectionTo: number;
  transformedText?: string;
}

export interface AITransformHandlers {
  getCurrentContent: () => string;
  getEditorView: () => EditorView | undefined;
  setDialogOpen: (open: boolean) => void;
  setTransformState: (state: AITransformState | null) => void;
  showError: (message: string) => void;
}

export async function computeContentHash(content: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(content);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray
    .slice(0, 8)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export async function prepareAITransform(
  action: AIAction,
  customPrompt: string | undefined,
  handlers: AITransformHandlers
) {
  const editorView = handlers.getEditorView();
  const currentContent = handlers.getCurrentContent();

  if (!editorView) {
    return;
  }

  const selection = editorView.state.selection.main;
  const hasSelection = selection.from !== selection.to;

  let textToTransform: string;
  let selectionFrom: number;
  let selectionTo: number;
  let isFullContent: boolean;

  if (hasSelection) {
    textToTransform = editorView.state.doc.sliceString(selection.from, selection.to);
    selectionFrom = selection.from;
    selectionTo = selection.to;
    isFullContent = false;
  } else {
    textToTransform = currentContent;
    selectionFrom = 0;
    selectionTo = editorView.state.doc.length;
    isFullContent = true;
  }

  if (textToTransform.trim().length < 10) {
    handlers.showError('too_short');
    return;
  }

  const initialContentHash = await computeContentHash(currentContent);

  handlers.setTransformState({
    action,
    customPrompt,
    originalText: textToTransform,
    selectionFrom,
    selectionTo,
    isFullContent,
    initialContentHash,
  });
  handlers.setDialogOpen(true);
}

export function applyAITransform(
  editorView: EditorView | undefined,
  transformState: AITransformState | null,
  transformedText: string
) {
  if (!editorView || !transformState) return;

  editorView.dispatch({
    changes: {
      from: transformState.selectionFrom,
      to: transformState.selectionTo,
      insert: transformedText,
    },
  });
}
