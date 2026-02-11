import type { ComponentType } from 'svelte';

import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

export interface DialogLoaderState {
  moveToFolderDialog?: ComponentType | null;
  markdownGuideDialog?: ComponentType | null;
  markdownGuideDropdown?: ComponentType | null;
  versionHistoryDialog?: ComponentType | null;
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
