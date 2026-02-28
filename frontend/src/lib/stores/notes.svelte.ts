// Notes store using Svelte 5 runes

import { SvelteMap } from 'svelte/reactivity';
import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';

import type { Backlink, Note } from '$lib/api';
import * as api from '$lib/api';
import { ApiError } from '$lib/api';
import { extractDueDatesDetailed } from '$lib/editor/markdown';
import { hasPendingForNote } from '$lib/offline/offline-queue';
import * as autosave from '$lib/stores/autosave.svelte';
import * as encryption from '$lib/stores/encryption.svelte';
import * as foldersStore from '$lib/stores/folders.svelte';
import { createNotesAccessors } from '$lib/stores/notes/accessors';
import {
  cancelAutoSave as cancelAutoSaveHelper,
  scheduleAutoSave as scheduleAutoSaveHelper,
  triggerAutoSave as triggerAutoSaveHelper,
} from '$lib/stores/notes/auto-save';
import { createNote as createNoteHelper } from '$lib/stores/notes/creator';
import { toggleEncryption as toggleEncryptionHelper } from '$lib/stores/notes/encryption-toggle';
import { assertOnlineForParanoidMode, extractUniqueWikilinks } from '$lib/stores/notes/helpers';
import {
  loadNote as loadNoteHelper,
  loadNotes as loadNotesHelper,
} from '$lib/stores/notes/loaders';
import {
  deleteCurrentNote as deleteCurrentNoteHelper,
  moveNote as moveNoteHelper,
} from '$lib/stores/notes/mutations';
import { handleRemoteUpdateWithPendingCheck } from '$lib/stores/notes/remote-update-gate';
import {
  handleRemoteCreate as handleRemoteCreateHelper,
  handleRemoteDelete as handleRemoteDeleteHelper,
  handleRemoteUpdate as handleRemoteUpdateHelper,
} from '$lib/stores/notes/remote-updates';
import { renameCurrentNote as renameCurrentNoteHelper } from '$lib/stores/notes/rename';
import { saveNote as saveNoteHelper } from '$lib/stores/notes/saver';
import {
  clearCurrentNote as clearCurrentNoteHelper,
  replaceTempId as replaceTempIdHelper,
  updateCurrentNoteAIEnabled as updateCurrentNoteAIEnabledHelper,
  updateCurrentNoteContent as updateCurrentNoteContentHelper,
  updateCurrentNoteTitle as updateCurrentNoteTitleHelper,
} from '$lib/stores/notes/state-updates';
import { createTaskEventQueue } from '$lib/stores/notes/task-events';
import * as searchIndex from '$lib/stores/search-index.svelte';
import { removeTabByNoteId, replaceTempId as replaceTabTempId } from '$lib/stores/tabs.svelte';
import * as toast from '$lib/stores/toast.svelte';

const taskEventQueue = createTaskEventQueue();

// Current note state
let currentNote = $state<Note | null>(null);
let currentNoteBacklinks = $state<Backlink[]>([]);
let isLoading = $state(false); // Only for loading notes
let isSaving = $state(false); // Only for saving notes
let error = $state<string | null>(null);
let isDirty = $state(false);

// Last saved version tracker (to detect and ignore WebSocket echoes of own saves)
let lastSavedVersion: number | null = null;
let lastSaveTimestamp: number | null = null;

// Counter to track if content changed during save (prevents isDirty race condition)
let saveInProgressCounter = 0;

// Auto-save state
let autoSaveTimeout = $state<ReturnType<typeof setTimeout> | null>(null);
let lastAutoSave = $state<Date | null>(null);
let autoSaveStatus = $state<'idle' | 'pending' | 'saving' | 'saved' | 'error'>('idle');
let autoSaveError = $state<string | null>(null);

// Notes list + Map for O(1) lookups by ID
let notes = $state<Note[]>([]);
let noteMapVersion = 0;
let lastSyncedVersion = -1;
const noteMap = new SvelteMap<string, Note>();
let notesLoading = $state(false);

// Delta-sync token (in-memory only — reset on page reload triggers full load, which is correct)
let syncToken = $state<string | null>(null);

/** Mark the map as needing a rebuild. Called implicitly when notes array changes. */
function invalidateNoteMap() {
  noteMapVersion++;
}

/** Ensure the map is in sync with the array. */
function ensureNoteMap() {
  if (lastSyncedVersion === noteMapVersion) return;
  noteMap.clear();
  for (const note of notes) {
    noteMap.set(note.id, note);
  }
  lastSyncedVersion = noteMapVersion;
}

/**
 * Get a note by ID in O(1) via the internal map.
 */
export function getNoteById(id: string): Note | undefined {
  ensureNoteMap();
  return noteMap.get(id);
}

export function queueTaskEvent(
  noteId: string,
  taskText: string,
  taskIndex: number,
  eventType: 'completed' | 'reopened'
) {
  taskEventQueue.add({ noteId, taskText, taskIndex, eventType });
}

// Export functions to access and modify state
const accessors = createNotesAccessors({
  getCurrentNote: () => currentNote,
  getBacklinks: () => currentNoteBacklinks,
  getIsLoading: () => isLoading,
  getIsSaving: () => isSaving,
  getError: () => error,
  setError: (value) => {
    error = value;
  },
  getIsDirty: () => isDirty,
  setDirty: (dirty) => {
    isDirty = dirty;
  },
  getNotes: () => notes,
  getNotesLoading: () => notesLoading,
  getAutoSaveStatus: () => autoSaveStatus,
  getLastAutoSave: () => lastAutoSave,
  getAutoSaveError: () => autoSaveError,
});

export const getCurrentNote = accessors.getCurrentNote;
export const getBacklinks = accessors.getBacklinks;
export const getIsLoading = accessors.getIsLoading;
export const getIsSaving = accessors.getIsSaving;
export const getError = accessors.getError;
export const clearError = accessors.clearError;
export const getIsDirty = accessors.getIsDirty;
export const setDirty = accessors.setDirty;
export const getNotes = accessors.getNotes;
export const getRecentNotes = accessors.getRecentNotes;
export const getNotesLoading = accessors.getNotesLoading;
export const getAutoSaveStatus = accessors.getAutoSaveStatus;
export const getLastAutoSave = accessors.getLastAutoSave;
export const getAutoSaveError = accessors.getAutoSaveError;

export function signalEncryptionLocked(): void {
  error = 'ENCRYPTION_LOCKED';
}

export async function loadNotes(mode: 'full' | 'delta' = 'full') {
  await loadNotesHelper(
    {
      listNotes: (options) => api.listNotes(options),
      setNotes: (next) => {
        notes = next;
        invalidateNoteMap();
      },
      getNotes: () => notes,
      setLoading: (value) => {
        notesLoading = value;
      },
      getSyncToken: () => syncToken,
      setSyncToken: (token) => {
        syncToken = token;
      },
    },
    mode
  );
}

export async function loadNote(id: string) {
  await loadNoteHelper({
    id,
    isOnline: () => navigator.onLine,
    getLocalNotes: () => notes,
    getNote: async (noteId) => {
      try {
        return await api.getNote(noteId);
      } catch (err) {
        // Shared notes/recipes are loaded via /shared/{id}, not /notes/{id}.
        if (err instanceof ApiError && err.status === 404) {
          const shared = await api.getSharedNote(noteId);
          return {
            ...shared,
            is_shared: true,
          };
        }
        throw err;
      }
    },
    getBacklinks: (noteId) => api.getBacklinks(noteId),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    decryptNote: (encryptedTitle, payload, noteId) =>
      encryption.decryptNote(encryptedTitle, payload, noteId),
    encryptNote: (title, content, noteId) => encryption.encryptNote(title, content, noteId),
    updateNote: (noteId, payload, version) => api.updateNote(noteId, payload, version),
    isConflictError: (err) => err instanceof ApiError && err.status === 409,
    setIsLoading: (value) => {
      isLoading = value;
    },
    setError: (value) => {
      error = value;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    resetLastSaved: () => {
      lastSavedVersion = null;
    },
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setBacklinks: (backlinks) => {
      currentNoteBacklinks = backlinks;
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    offlineUnavailableMessage: get(_)('editor.autosave.offline_unavailable'),
    decryptFailedMessage: get(_)('editor.autosave.decrypt_failed'),
    defaultErrorMessage: get(_)('editor.autosave.load_failed'),
  });
}

export async function createNote(
  title: string,
  content = '',
  folderPath = '/',
  journalOptions?: { note_type: string; journal_date?: string }
) {
  return createNoteHelper({
    title,
    content,
    folderPath,
    journalOptions,
    assertOnline: () => assertOnlineForParanoidMode(),
    getFolders: () => foldersStore.getFolders(),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    encryptNote: (noteTitle, noteContent, noteId) =>
      encryption.encryptNote(noteTitle, noteContent, noteId),
    decryptNote: (encryptedTitle, payload, noteId) =>
      encryption.decryptNote(encryptedTitle, payload, noteId),
    extractUniqueLinks: (noteContent) => extractUniqueWikilinks(noteContent),
    extractDueDates: (noteContent) => extractDueDatesDetailed(noteContent),
    createNote: (payload, offlineContext) => api.createNote(payload, offlineContext),
    addToSearchIndex: (id, noteTitle, noteContent) =>
      searchIndex.addToIndex(id, noteTitle, noteContent),
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    getNotes: () => notes,
    setBacklinks: (backlinks) => {
      currentNoteBacklinks = backlinks;
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    setError: (value) => {
      error = value;
    },
    setIsLoading: (value) => {
      isLoading = value;
    },
  });
}

export async function saveNote() {
  return saveNoteHelper({
    getCurrentNote: () => currentNote,
    getIsDirty: () => isDirty,
    getIsSaving: () => isSaving,
    setIsSaving: (value) => {
      isSaving = value;
    },
    setError: (value) => {
      error = value;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    incrementSaveCounter: () => ++saveInProgressCounter,
    getSaveCounter: () => saveInProgressCounter,
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    setCurrentNote: (note) => {
      currentNote = note;
    },
    updateNotes: (updater) => {
      notes = updater(notes);
    },
    setLastSavedVersion: (version) => {
      lastSavedVersion = version;
    },
    setLastSaveTimestamp: (timestamp) => {
      lastSaveTimestamp = timestamp;
    },
    taskEventQueue,
    assertOnline: () => assertOnlineForParanoidMode(),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    encryptNote: (title, content, noteId) => encryption.encryptNote(title, content, noteId),
    decryptNote: (encryptedTitle, payload, noteId) =>
      encryption.decryptNote(encryptedTitle, payload, noteId),
    encryptTaskText: (text) => encryption.encryptTaskText(text),
    extractUniqueLinks: (content) => extractUniqueWikilinks(content),
    extractDueDates: (content) => extractDueDatesDetailed(content),
    updateNote: (id, payload, version, offlineContext) =>
      api.updateNote(id, payload, version, offlineContext),
    updateSearchIndex: (id, title, content) => searchIndex.updateInIndex(id, title, content),
    recordTaskEvent: (noteId, payload) => api.recordTaskEvent(noteId, payload),
    isConflictError: (err) => err instanceof ApiError && err.status === 409,
  });
}

/**
 * Toggle encryption state of the current note.
 * Decrypt: clears all encryption fields, stores plaintext on server.
 * Encrypt: encrypts content and sends encrypted payload.
 */
export async function toggleEncryption(): Promise<void> {
  return toggleEncryptionHelper({
    getCurrentNote: () => currentNote,
    getIsSaving: () => isSaving,
    setIsSaving: (value) => {
      isSaving = value;
    },
    setError: (value) => {
      error = value;
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    getNotes: () => notes,
    setLastSavedVersion: (version) => {
      lastSavedVersion = version;
    },
    setLastSaveTimestamp: (timestamp) => {
      lastSaveTimestamp = timestamp;
    },
    cancelAutoSave: () => cancelAutoSave(),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    encryptNote: (noteTitle, noteContent, noteId) =>
      encryption.encryptNote(noteTitle, noteContent, noteId),
    decryptNote: (encryptedTitle, payload, noteId) =>
      encryption.decryptNote(encryptedTitle, payload, noteId),
    extractUniqueLinks: (content) => extractUniqueWikilinks(content),
    updateNote: (noteId, payload, version) => api.updateNote(noteId, payload, version),
    decryptNoteApi: (noteId, noteTitle, noteContent, version, recipeData) =>
      api.decryptNote(noteId, noteTitle, noteContent, version, recipeData),
    getRecipeDetail: (noteId) => api.getRecipeDetail(noteId),
    removeFromSearchIndex: (noteId) => searchIndex.removeFromIndex(noteId),
    addToSearchIndex: (noteId, noteTitle, noteContent) =>
      searchIndex.addToIndex(noteId, noteTitle, noteContent),
  });
}

export async function deleteCurrentNote() {
  const noteId = currentNote?.id;
  await deleteCurrentNoteHelper({
    getCurrentNote: () => currentNote,
    assertOnline: () => assertOnlineForParanoidMode(true),
    setIsLoading: (value) => {
      isLoading = value;
    },
    setError: (value) => {
      error = value;
    },
    deleteNote: (id, soft) => api.deleteNote(id, soft),
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    getNotes: () => notes,
    removeFromSearchIndex: (id) => searchIndex.removeFromIndex(id),
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    setBacklinks: (backlinks) => {
      currentNoteBacklinks = backlinks;
    },
  });
  if (noteId) removeTabByNoteId(noteId);
}

export async function renameCurrentNote(newTitle: string) {
  return renameCurrentNoteHelper(newTitle, {
    getCurrentNote: () => currentNote,
    getNotes: () => notes,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    setError: (value) => {
      error = value;
    },
    getBacklinks: (noteId) => api.getBacklinks(noteId),
    renameNote: (noteId, title) => api.renameNote(noteId, title),
    renameNoteAsync: (noteId, title) => api.renameNoteAsync(noteId, title),
    getJobStatus: (jobId) => api.getJobStatus(jobId),
    notifyInfo: (message) => toast.info(message),
    notifySuccess: (message) => toast.success(message),
    notifyError: (message) => toast.error(message),
    loadNotes: () => loadNotes(),
  });
}

export function updateCurrentNoteContent(content: string) {
  updateCurrentNoteContentHelper(content, {
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    incrementSaveCounter: () => {
      saveInProgressCounter++;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    clearError: () => {
      error = null;
    },
    clearBacklinks: () => {
      currentNoteBacklinks = [];
    },
    clearLastSaved: () => {
      lastSavedVersion = null;
      lastSaveTimestamp = null;
    },
  });
}

export function updateCurrentNoteTitle(title: string) {
  updateCurrentNoteTitleHelper(title, {
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    incrementSaveCounter: () => {
      saveInProgressCounter++;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    clearError: () => {
      error = null;
    },
    clearBacklinks: () => {
      currentNoteBacklinks = [];
    },
    clearLastSaved: () => {
      lastSavedVersion = null;
      lastSaveTimestamp = null;
    },
  });
}

export function updateCurrentNoteAIEnabled(aiEnabled: boolean) {
  updateCurrentNoteAIEnabledHelper(aiEnabled, {
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    incrementSaveCounter: () => {
      saveInProgressCounter++;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    clearError: () => {
      error = null;
    },
    clearBacklinks: () => {
      currentNoteBacklinks = [];
    },
    clearLastSaved: () => {
      lastSavedVersion = null;
      lastSaveTimestamp = null;
    },
  });
}

export function clearCurrentNote() {
  clearCurrentNoteHelper({
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    incrementSaveCounter: () => {
      saveInProgressCounter++;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    clearError: () => {
      error = null;
    },
    clearBacklinks: () => {
      currentNoteBacklinks = [];
    },
    clearLastSaved: () => {
      lastSavedVersion = null;
      lastSaveTimestamp = null;
    },
  });
}

export async function moveNote(id: string, folderPath: string) {
  return moveNoteHelper({
    id,
    folderPath,
    assertOnline: () => assertOnlineForParanoidMode(true),
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setError: (value) => {
      error = value;
    },
    getNote: (noteId) => api.getNote(noteId),
    decryptNote: (encryptedTitle, payload, noteId) =>
      encryption.decryptNote(encryptedTitle, payload, noteId),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    encryptNote: (title, content, noteId) => encryption.encryptNote(title, content, noteId),
    extractUniqueLinks: (content) => extractUniqueWikilinks(content),
    extractDueDates: (content) => extractDueDatesDetailed(content),
    updateNote: (noteId, payload, version, offlineContext) =>
      api.updateNote(noteId, payload, version, offlineContext),
  });
}

// ============================================================================
// Auto-Save Logic
// ============================================================================

/**
 * Trigger auto-save after debounce delay
 * This is exported so the Editor component can call it
 */
export async function triggerAutoSave() {
  await triggerAutoSaveHelper({
    getCurrentNoteId: () => currentNote?.id ?? null,
    isDirty: () => isDirty,
    setStatus: (status) => {
      autoSaveStatus = status;
    },
    setError: (value) => {
      autoSaveError = value;
    },
    setLastAutoSave: (value) => {
      lastAutoSave = value;
    },
    saveNote: () => saveNote(),
    isConflictError: (err) => err instanceof ApiError && err.status === 409,
    handleConflict: async () => {
      if (!currentNote) return;
      try {
        await api.getNote(currentNote.id);
        toast.warning(get(_)('editor.autosave.conflict_toast'), {
          label: get(_)('editor.autosave.reload_label'),
          handler: () => {
            if (currentNote) loadNote(currentNote.id);
          },
        });
      } catch (err) {
        console.error('Failed to fetch remote version:', err);
      }
    },
    conflictMessage: get(_)('editor.autosave.conflict_message'),
    defaultError: get(_)('editor.autosave.default_error'),
  });
}

/**
 * Schedule auto-save with debounce
 * Called by components when content changes
 */
export function scheduleAutoSave() {
  scheduleAutoSaveHelper({
    isEnabled: () => autosave.getAutoSaveEnabled(),
    isDirty: () => isDirty,
    isLoading: () => isLoading,
    getCurrentNoteId: () => currentNote?.id ?? null,
    setStatus: (status) => {
      autoSaveStatus = status;
    },
    getTimeout: () => autoSaveTimeout,
    setTimeoutHandle: (handle) => {
      autoSaveTimeout = handle;
    },
    getDelay: () => autosave.getAutoSaveDelay(),
    trigger: () => {
      void triggerAutoSave();
    },
  });
}

/**
 * Cancel any pending auto-save
 */
export function cancelAutoSave() {
  cancelAutoSaveHelper({
    getTimeout: () => autoSaveTimeout,
    setTimeoutHandle: (handle) => {
      autoSaveTimeout = handle;
    },
    getStatus: () => autoSaveStatus,
    setStatus: (status) => {
      autoSaveStatus = status;
    },
  });
}

// ============================================================================
// WebSocket Remote Update Handlers
// ============================================================================

/**
 * Handle a remote note update (from WebSocket)
 */
export function handleRemoteUpdate(remoteNote: Note) {
  // Skip WebSocket updates for notes with pending offline operations.
  // After sync completes, loadNotes() will load fresh server state.
  // Note: This is async but we handle it via fire-and-forget pattern.
  // The check runs fast (IndexedDB) and calls the actual update logic.
  handleRemoteUpdateWithPendingCheck(remoteNote, {
    hasPendingForNote: (noteId) => hasPendingForNote(noteId),
    onUpdate: (note) =>
      handleRemoteUpdateHelper(note, {
        getCurrentNote: () => currentNote,
        getIsSaving: () => isSaving,
        getIsDirty: () => isDirty,
        getLastSavedVersion: () => lastSavedVersion,
        getLastSaveTimestamp: () => lastSaveTimestamp,
        clearLastSaved: () => {
          lastSavedVersion = null;
          lastSaveTimestamp = null;
        },
        setCurrentNote: (nextNote) => {
          currentNote = nextNote;
        },
        updateNoteInList: (nextNote) => updateNoteInList(nextNote),
        getNotes: () => notes,
        setNotes: (nextNotes) => {
          notes = nextNotes;
          invalidateNoteMap();
        },
        isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
        decryptNote: (encryptedTitle, payload, noteId) =>
          encryption.decryptNote(encryptedTitle, payload, noteId),
        warn: (message, options) => toast.warning(message, options),
        loadNote: (noteId) => {
          void loadNote(noteId);
        },
      }),
  });
}

/**
 * Handle a remote note creation (from WebSocket)
 */
export function handleRemoteCreate(note: Note) {
  handleRemoteCreateHelper(note, {
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    info: (message) => toast.info(message),
  });
}

/**
 * Handle a remote note deletion (from WebSocket)
 */
export function handleRemoteDelete(id: string) {
  handleRemoteDeleteHelper(id, {
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    info: (message) => toast.info(message),
  });
  removeTabByNoteId(id);
}

/**
 * Update a note in the notes list and map — O(1) map update + array splice.
 */
function updateNoteInList(updated: Note) {
  // Update map directly (skip full rebuild)
  ensureNoteMap();
  noteMap.set(updated.id, updated);
  const idx = notes.findIndex((n) => n.id === updated.id);
  if (idx !== -1) {
    notes[idx] = updated;
    // Keep versions in sync so ensureNoteMap doesn't rebuild
    noteMapVersion++;
    lastSyncedVersion = noteMapVersion;
  }
}

/**
 * Replace a temp-ID with a real ID in the notes list and currentNote.
 * Called by sync-manager after a successful offline create sync.
 */
export function replaceTempId(tempId: string, realNote: Note) {
  replaceTempIdHelper(tempId, realNote, {
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
      invalidateNoteMap();
    },
    setDirty: (dirty) => {
      isDirty = dirty;
    },
    incrementSaveCounter: () => {
      saveInProgressCounter++;
    },
    getAutoSaveTimeout: () => autoSaveTimeout,
    setAutoSaveTimeout: (handle) => {
      autoSaveTimeout = handle;
    },
    setAutoSaveStatus: (status) => {
      autoSaveStatus = status;
    },
    setAutoSaveError: (value) => {
      autoSaveError = value;
    },
    clearError: () => {
      error = null;
    },
    clearBacklinks: () => {
      currentNoteBacklinks = [];
    },
    clearLastSaved: () => {
      lastSavedVersion = null;
      lastSaveTimestamp = null;
    },
  });
  replaceTabTempId(tempId, realNote.id);
}
