import type { ComponentType } from 'svelte';

export interface DialogLoaderState {
  moveToFolderDialog?: ComponentType | null;
  markdownGuideDialog?: ComponentType | null;
  markdownGuideDropdown?: ComponentType | null;
  versionHistoryDialog?: ComponentType | null;
}

export async function loadMoveToFolderDialog(state: DialogLoaderState) {
  if (state.moveToFolderDialog) return state;
  const module = await import('$lib/components/MoveToFolderDialog.svelte');
  return { ...state, moveToFolderDialog: module.default as unknown as ComponentType };
}

export async function loadMarkdownGuideDialog(state: DialogLoaderState) {
  if (state.markdownGuideDialog) return state;
  const module = await import('$lib/components/MarkdownGuideDialog.svelte');
  return { ...state, markdownGuideDialog: module.default as unknown as ComponentType };
}

export async function loadMarkdownGuideDropdown(state: DialogLoaderState) {
  if (state.markdownGuideDropdown) return state;
  const module = await import('$lib/components/MarkdownGuideDropdown.svelte');
  return { ...state, markdownGuideDropdown: module.default as unknown as ComponentType };
}

export async function loadVersionHistoryDialog(state: DialogLoaderState) {
  if (state.versionHistoryDialog) return state;
  const module = await import('$lib/components/VersionHistoryDialog.svelte');
  return { ...state, versionHistoryDialog: module.default as unknown as ComponentType };
}
