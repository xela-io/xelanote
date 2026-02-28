import type { ComponentType } from 'svelte';

import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

export interface DialogLoaderState {
  moveToFolderDialog?: ComponentType | null;
  markdownGuideDialog?: ComponentType | null;
  markdownGuideDropdown?: ComponentType | null;
  versionHistoryDialog?: ComponentType | null;
  shareDialog?: ComponentType | null;
  aiActionsDropdown?: ComponentType | null;
  dictationPanel?: ComponentType | null;
  conflictDialog?: ComponentType | null;
}

export function maybeLoadDialog(
  shouldLoad: boolean,
  state: DialogLoaderState,
  loader: (state: DialogLoaderState) => Promise<DialogLoaderState>,
  setState: (state: DialogLoaderState) => void
): void {
  if (!shouldLoad) return;
  loader(state).then((next) => {
    setState(next);
  });
}

export async function loadMoveToFolderDialog(state: DialogLoaderState) {
  if (state.moveToFolderDialog) return state;
  const module = await import('$lib/components/MoveToFolderDialog.svelte');
  return {
    ...state,
    moveToFolderDialog: loadSvelteComponentFromModule(module, 'MoveToFolderDialog'),
  };
}

export async function loadMarkdownGuideDialog(state: DialogLoaderState) {
  if (state.markdownGuideDialog) return state;
  const module = await import('$lib/components/MarkdownGuideDialog.svelte');
  return {
    ...state,
    markdownGuideDialog: loadSvelteComponentFromModule(module, 'MarkdownGuideDialog'),
  };
}

export async function loadMarkdownGuideDropdown(state: DialogLoaderState) {
  if (state.markdownGuideDropdown) return state;
  const module = await import('$lib/components/MarkdownGuideDropdown.svelte');
  return {
    ...state,
    markdownGuideDropdown: loadSvelteComponentFromModule(module, 'MarkdownGuideDropdown'),
  };
}

export async function loadVersionHistoryDialog(state: DialogLoaderState) {
  if (state.versionHistoryDialog) return state;
  const module = await import('$lib/components/VersionHistoryDialog.svelte');
  return {
    ...state,
    versionHistoryDialog: loadSvelteComponentFromModule(module, 'VersionHistoryDialog'),
  };
}

export async function loadShareDialog(state: DialogLoaderState) {
  if (state.shareDialog) return state;
  const module = await import('$lib/components/ShareDialog.svelte');
  return {
    ...state,
    shareDialog: loadSvelteComponentFromModule(module, 'ShareDialog'),
  };
}

export async function loadAIActionsDropdown(state: DialogLoaderState) {
  if (state.aiActionsDropdown) return state;
  const module = await import('$lib/components/AIActionsDropdown.svelte');
  return {
    ...state,
    aiActionsDropdown: loadSvelteComponentFromModule(module, 'AIActionsDropdown'),
  };
}

export async function loadDictationPanel(state: DialogLoaderState) {
  if (state.dictationPanel) return state;
  const module = await import('$lib/components/DictationPanel.svelte');
  return {
    ...state,
    dictationPanel: loadSvelteComponentFromModule(module, 'DictationPanel'),
  };
}

export async function loadConflictDialog(state: DialogLoaderState) {
  if (state.conflictDialog) return state;
  const module = await import('$lib/components/ConflictDialog.svelte');
  return {
    ...state,
    conflictDialog: loadSvelteComponentFromModule(module, 'ConflictDialog'),
  };
}
