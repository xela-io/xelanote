import type { EditorView } from '@codemirror/view';

import { updateImageWidthByIndex } from '$lib/editor/image-resize';
import { calculateMoveChanges } from '$lib/utils/task-reorder';

interface TaskReorderActionParams {
  editorView: EditorView | undefined;
  fromTaskIndex: number;
  toTaskIndex: number;
  scheduleAutoSave: () => void;
  scrollIntoView?: boolean;
}

export function handleTaskReorderAction(params: TaskReorderActionParams): void {
  const {
    editorView,
    fromTaskIndex,
    toTaskIndex,
    scheduleAutoSave,
    scrollIntoView = true,
  } = params;
  if (!editorView) return;

  const doc = editorView.state.doc;
  const changes = calculateMoveChanges(doc, fromTaskIndex, toTaskIndex);
  if (changes.length === 0) return;

  editorView.dispatch({ changes, scrollIntoView });
  scheduleAutoSave();
}

interface ImageResizeActionParams {
  editorView: EditorView | undefined;
  imageIndex: number;
  newWidth: number;
  getFallbackContent: () => string;
  setFallbackContent: (content: string) => void;
  scheduleAutoSave: () => void;
}

export function handleImageResizeAction(params: ImageResizeActionParams): void {
  const {
    editorView,
    imageIndex,
    newWidth,
    getFallbackContent,
    setFallbackContent,
    scheduleAutoSave,
  } = params;

  const content = editorView ? editorView.state.doc.toString() : getFallbackContent();
  const newContent = updateImageWidthByIndex(content, imageIndex, newWidth);

  if (editorView) {
    editorView.dispatch({
      changes: { from: 0, to: editorView.state.doc.length, insert: newContent },
    });
  } else {
    setFallbackContent(newContent);
  }

  scheduleAutoSave();
}
