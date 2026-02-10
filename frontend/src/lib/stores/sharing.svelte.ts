// Sharing store using Svelte 5 runes

import * as api from '$lib/api';
import type { SharedNote, SharedFolder, SharedCollection } from '$lib/api';

// State
let sharedNotes = $state<SharedNote[]>([]);
let sharedFolders = $state<SharedFolder[]>([]);
let sharedCollections = $state<SharedCollection[]>([]);
let currentSharedFolderNotes = $state<SharedNote[]>([]);
let isLoading = $state(false);
let currentSharedNote = $state<SharedNote | null>(null);

// Getters
export function getSharedNotes() {
  return sharedNotes;
}

export function getSharedFolders() {
  return sharedFolders;
}

export function getSharedCollections() {
  return sharedCollections;
}

export function getCurrentSharedFolderNotes() {
  return currentSharedFolderNotes;
}

export function getIsLoading() {
  return isLoading;
}

export function getCurrentSharedNote() {
  return currentSharedNote;
}

export function getSharedNoteCount() {
  return sharedNotes.length;
}

export function getTotalSharedCount() {
  return sharedNotes.length + sharedFolders.length + sharedCollections.length;
}

// Actions
export async function loadAllShared() {
  isLoading = true;
  try {
    const [notes, folders, collections] = await Promise.all([
      api.getSharedNotes(),
      api.getSharedFolders(),
      api.getSharedCollections().catch(() => [] as SharedCollection[]),
    ]);
    sharedNotes = notes;
    sharedFolders = folders;
    sharedCollections = collections;
  } catch (err) {
    console.error('Failed to load shared items:', err);
    sharedNotes = [];
    sharedFolders = [];
    sharedCollections = [];
  } finally {
    isLoading = false;
  }
}

export async function loadSharedNotes() {
  isLoading = true;
  try {
    sharedNotes = await api.getSharedNotes();
  } catch (err) {
    console.error('Failed to load shared notes:', err);
    sharedNotes = [];
  } finally {
    isLoading = false;
  }
}

export async function loadSharedFolders() {
  isLoading = true;
  try {
    sharedFolders = await api.getSharedFolders();
  } catch (err) {
    console.error('Failed to load shared folders:', err);
    sharedFolders = [];
  } finally {
    isLoading = false;
  }
}

export async function loadSharedFolderNotes(folderId: number) {
  isLoading = true;
  try {
    currentSharedFolderNotes = await api.getSharedFolderNotes(folderId);
  } catch (err) {
    console.error('Failed to load shared folder notes:', err);
    currentSharedFolderNotes = [];
    throw err;
  } finally {
    isLoading = false;
  }
}

export async function loadSharedNote(id: string) {
  isLoading = true;
  try {
    currentSharedNote = await api.getSharedNote(id);
  } catch (err) {
    console.error('Failed to load shared note:', err);
    currentSharedNote = null;
    throw err;
  } finally {
    isLoading = false;
  }
}

export function clearCurrentSharedNote() {
  currentSharedNote = null;
}

export function clearCurrentSharedFolderNotes() {
  currentSharedFolderNotes = [];
}
