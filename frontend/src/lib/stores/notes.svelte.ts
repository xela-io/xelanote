// Notes store using Svelte 5 runes

import type { Backlink, Job, Note } from '$lib/api';
import * as api from '$lib/api';
import { ApiError } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { extractDueDatesDetailed } from '$lib/editor/markdown';
import { hasPendingForNote } from '$lib/offline/offline-queue';
import { createNotesAccessors } from '$lib/stores/notes/accessors';
import {
  cancelAutoSave as cancelAutoSaveHelper,
  scheduleAutoSave as scheduleAutoSaveHelper,
  triggerAutoSave as triggerAutoSaveHelper,
} from '$lib/stores/notes/auto-save';
import { createNote as createNoteHelper } from '$lib/stores/notes/creator';
import { toggleEncryption as toggleEncryptionHelper } from '$lib/stores/notes/encryption-toggle';
import {
  assertOnlineForParanoidMode,
  extractUniqueWikilinks,
} from '$lib/stores/notes/helpers';
import { loadNote as loadNoteHelper, loadNotes as loadNotesHelper } from '$lib/stores/notes/loaders';
import {
  deleteCurrentNote as deleteCurrentNoteHelper,
  moveNote as moveNoteHelper,
} from '$lib/stores/notes/mutations';
import { saveNote as saveNoteHelper } from '$lib/stores/notes/saver';
import { createTaskEventQueue } from '$lib/stores/notes/task-events';
import * as autosave from '$lib/stores/autosave.svelte';
import * as encryption from '$lib/stores/encryption.svelte';
import * as foldersStore from '$lib/stores/folders.svelte';
import * as searchIndex from '$lib/stores/search-index.svelte';
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

// Notes list
let notes = $state<Note[]>([]);
let notesLoading = $state(false);

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

export async function loadNotes() {
  await loadNotesHelper({
    listNotes: (options) => api.listNotes(options),
    setNotes: (next) => {
      notes = next;
    },
    setLoading: (value) => {
      notesLoading = value;
    },
  });
}

export async function loadNote(id: string) {
  await loadNoteHelper({
    id,
    isOnline: () => navigator.onLine,
    getLocalNotes: () => notes,
    getNote: (noteId) => api.getNote(noteId),
    getBacklinks: (noteId) => api.getBacklinks(noteId),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    decryptNote: (encryptedTitle, payload) => encryption.decryptNote(encryptedTitle, payload),
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
    encryptNote: (noteTitle, noteContent) => encryption.encryptNote(noteTitle, noteContent),
    decryptNote: (encryptedTitle, payload) =>
      encryption.decryptNote(encryptedTitle, payload),
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
    encryptNote: (title, content) => encryption.encryptNote(title, content),
    decryptNote: (encryptedTitle, payload) =>
      encryption.decryptNote(encryptedTitle, payload),
    encryptTaskText: (text) => encryption.encryptTaskText(text),
    extractUniqueLinks: (content) => extractUniqueWikilinks(content),
    extractDueDates: (content) => extractDueDatesDetailed(content),
    updateNote: (id, payload, version, offlineContext) =>
      api.updateNote(id, payload, version, offlineContext),
    updateSearchIndex: (id, title, content) =>
      searchIndex.updateInIndex(id, title, content),
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
    encryptNote: (noteTitle, noteContent) => encryption.encryptNote(noteTitle, noteContent),
    decryptNote: (encryptedTitle, payload) =>
      encryption.decryptNote(encryptedTitle, payload),
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
}

/**
 * Poll a job until it completes or fails
 */
async function pollJobCompletion(jobId: string, maxAttempts = 60): Promise<Job> {
  for (let i = 0; i < maxAttempts; i++) {
    const job = await api.getJobStatus(jobId);
    if (job.status === 'completed' || job.status === 'failed') {
      return job;
    }
    // Wait 1 second between polls
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error('Job timeout - operation took too long');
}

function isRenameResult(value: unknown): value is api.RenameResult {
  if (!value || typeof value !== 'object') return false;
  const result = value as { note?: unknown; updated_note_count?: unknown };
  if (!result.note || typeof result.note !== 'object') return false;
  return typeof result.updated_note_count === 'number';
}

export async function renameCurrentNote(newTitle: string) {
  if (!currentNote) return;

  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = {
    note: { ...currentNote },
    notesList: [...notes],
  };

  // Optimistic UI update - apply changes immediately
  currentNote = { ...currentNote, title: newTitle };
  notes = notes.map((n) => (n.id === currentNote!.id ? currentNote! : n));
  isDirty = false;
  error = null;

  try {
    // Check backlinks count to decide sync vs async
    const backlinksResult = await api.getBacklinks(snapshot.note.id);
    const backlinksCount = backlinksResult.backlinks.length;

    // Heuristic: Use async mode for >100 backlinks
    const useAsync = backlinksCount > 100;

    if (useAsync) {
      // Async mode - submit job and poll for completion
      toast.info('Renaming note in background...');

      const { job_id } = await api.renameNoteAsync(snapshot.note.id, newTitle);

      // Poll for completion
      const job = await pollJobCompletion(job_id);

      if (job.status === 'completed') {
        if (!isRenameResult(job.result)) {
          throw new Error('Unexpected job result from rename');
        }
        // Success - update with result
        currentNote = job.result.note;
        await loadNotes();
        toast.success(
          `Note renamed successfully (${job.result.updated_note_count} references updated)`
        );
        return job.result;
      } else {
        // Failed - revert and show error
        currentNote = snapshot.note;
        notes = snapshot.notesList;
        const errorMsg = job.error || 'Failed to rename note';
        toast.error(errorMsg);
        throw new Error(errorMsg);
      }
    } else {
      // Sync mode - execute immediately (existing behavior)
      const result = await api.renameNote(snapshot.note.id, newTitle);
      currentNote = result.note;
      await loadNotes();
      return result;
    }
  } catch (e) {
    // Revert to snapshot on error
    currentNote = snapshot.note;
    notes = snapshot.notesList;
    error = e instanceof Error ? e.message : 'Failed to rename note';
    throw e;
  }
}

export function updateCurrentNoteContent(content: string) {
  if (currentNote) {
    // Only mark as dirty if content actually changed
    if (currentNote.content !== content) {
      currentNote = { ...currentNote, content };
      isDirty = true;
      // Increment counter to detect changes during save
      saveInProgressCounter++;
    }
  }
}

export function updateCurrentNoteTitle(title: string) {
  if (currentNote) {
    // Only mark as dirty if title actually changed
    if (currentNote.title !== title) {
      currentNote = { ...currentNote, title };
      isDirty = true;
      // Increment counter to detect changes during save
      saveInProgressCounter++;
    }
  }
}

export function updateCurrentNoteAIEnabled(aiEnabled: boolean) {
  if (currentNote) {
    currentNote = { ...currentNote, ai_enabled: aiEnabled };
    // Also update in the notes list
    const idx = notes.findIndex((n) => n.id === currentNote!.id);
    if (idx !== -1) {
      notes[idx] = { ...notes[idx], ai_enabled: aiEnabled };
      notes = [...notes]; // Trigger reactivity
    }
  }
}

export function clearCurrentNote() {
  // Cancel pending auto-save
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout);
    autoSaveTimeout = null;
  }
  autoSaveStatus = 'idle';
  autoSaveError = null;
  lastSavedVersion = null; // Reset echo tracker

  currentNote = null;
  currentNoteBacklinks = [];
  isDirty = false;
  error = null;
}

export async function moveNote(id: string, folderPath: string) {
  return moveNoteHelper({
    id,
    folderPath,
    assertOnline: () => assertOnlineForParanoidMode(true),
    getNotes: () => notes,
    setNotes: (nextNotes) => {
      notes = nextNotes;
    },
    getCurrentNote: () => currentNote,
    setCurrentNote: (note) => {
      currentNote = note;
    },
    setError: (value) => {
      error = value;
    },
    getNote: (noteId) => api.getNote(noteId),
    decryptNote: (encryptedTitle, payload) =>
      encryption.decryptNote(encryptedTitle, payload),
    isEncryptionUnlocked: () => encryption.isEncryptionUnlocked(),
    encryptNote: (title, content) => encryption.encryptNote(title, content),
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
        toast.warning('Auto-Save Konflikt. Notiz wurde remote geändert.', {
          label: 'Neu laden',
          handler: () => {
            if (currentNote) loadNote(currentNote.id);
          },
        });
      } catch (err) {
        console.error('Failed to fetch remote version:', err);
      }
    },
    conflictMessage: 'Konflikt erkannt. Notiz wurde extern geändert.',
    defaultError: 'Auto-save failed',
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
  _handleRemoteUpdateAsync(remoteNote);
}

async function _handleRemoteUpdateAsync(remoteNote: Note) {
  try {
    const hasPending = await hasPendingForNote(remoteNote.id);
    if (hasPending) {
      console.log(
        '[WebSocket] Skipping remote update for note with pending offline ops:',
        remoteNote.id
      );
      return;
    }
  } catch {
    // IndexedDB error - continue with normal flow
  }

  _handleRemoteUpdateSync(remoteNote);
}

function _handleRemoteUpdateSync(remoteNote: Note) {
  const localNote = currentNote;

  // Skip WebSocket updates while save is in progress - they're likely our own echo
  // The echo detection below won't work yet because lastSavedVersion isn't set until after API response
  if (isSaving && localNote && localNote.id === remoteNote.id) {
    console.log('[WebSocket] Update während Save ignoriert (potentielles Echo)', {
      remoteVersion: remoteNote.version,
      isSaving,
    });
    return;
  }

  // ENHANCED ECHO DETECTION: 2-second Grace Period
  // Ignores WebSocket echoes even if user continues typing during save
  const isEcho =
    localNote &&
    localNote.id === remoteNote.id &&
    lastSavedVersion !== null &&
    remoteNote.version === lastSavedVersion &&
    lastSaveTimestamp !== null &&
    Date.now() - lastSaveTimestamp < 2000; // 2s grace period

  if (isEcho) {
    console.log('[WebSocket] Echo erkannt, ignoriere', {
      version: remoteNote.version,
      timeSinceSave: Date.now() - (lastSaveTimestamp || 0),
    });
    lastSavedVersion = null;
    lastSaveTimestamp = null;
    return; // Echo → Ignore completely
  }

  // Decrypt remote note if encrypted
  let processedNote = remoteNote;
  if (remoteNote.content_encrypted && remoteNote.encrypted_content) {
    // Only decrypt if encryption is unlocked
    if (encryption.isEncryptionUnlocked()) {
      try {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: remoteNote.encrypted_content,
          metadata: JSON.parse(remoteNote.encryption_metadata || '{}'),
        };

        const decrypted = encryption.decryptNote(
          remoteNote.encrypted_title || null,
          encryptedPayload
        );

        processedNote = {
          ...remoteNote,
          title: decrypted.title || remoteNote.title,
          content: decrypted.content,
        };
      } catch (err) {
        console.error('[WebSocket] Failed to decrypt remote note:', err);
        // Don't update currentNote if decryption fails, but still update list
        updateNoteInList(remoteNote);
        return;
      }
    } else {
      // Encryption locked - can't decrypt, only update list
      console.log('[WebSocket] Encryption locked, skipping currentNote update');
      updateNoteInList(remoteNote);
      return;
    }
  }

  // CONFLICT DETECTION: Only show warning for true conflicts
  if (localNote && localNote.id === processedNote.id) {
    const versionDiverged = processedNote.version !== localNote.version;

    if (isDirty && versionDiverged) {
      // TRUE CONFLICT: Local changes + Remote update from different source
      const localChanges = localNote.content.length - processedNote.content.length;
      const changeInfo = Math.abs(localChanges) > 0 ? ` (±${Math.abs(localChanges)} Zeichen)` : '';

      console.warn('[Konflikt] Remote Update mit lokalen Änderungen', {
        localVersion: localNote.version,
        remoteVersion: processedNote.version,
        localChanges,
      });

      toast.warning(
        `Remote-Update erkannt (Version ${processedNote.version}). Du hast lokale Änderungen${changeInfo}. Speichern überschreibt Remote-Version.`,
        {
          label: 'Remote laden',
          handler: () => loadNote(processedNote.id),
        }
      );
      return;
    }

    // No conflict or no local changes → Accept update
    if (!isDirty || !versionDiverged) {
      currentNote = processedNote;
      updateNoteInList(processedNote);
      return;
    }
  }

  // Different note → Update list only
  if (!localNote || localNote.id !== processedNote.id) {
    updateNoteInList(processedNote);
  }
}

/**
 * Handle a remote note creation (from WebSocket)
 */
export function handleRemoteCreate(note: Note) {
  // Add to notes list if not already present
  if (!notes.find((n) => n.id === note.id)) {
    notes = [note, ...notes];
    toast.info(`New note "${note.title}" created`);
  }
}

/**
 * Handle a remote note deletion (from WebSocket)
 */
export function handleRemoteDelete(id: string) {
  // Remove from notes list
  notes = notes.filter((n) => n.id !== id);

  // Clear current note if it was deleted
  if (currentNote?.id === id) {
    currentNote = null;
    toast.info('This note was deleted');
  } else {
    const deletedNote = notes.find((n) => n.id === id);
    if (deletedNote) {
      toast.info(`Note "${deletedNote.title}" was deleted`);
    }
  }
}

/**
 * Update a note in the notes list
 */
function updateNoteInList(updated: Note) {
  notes = notes.map((n) => (n.id === updated.id ? updated : n));
}

/**
 * Replace a temp-ID with a real ID in the notes list and currentNote.
 * Called by sync-manager after a successful offline create sync.
 */
export function replaceTempId(tempId: string, realNote: Note) {
  // Update in notes list
  notes = notes.map((n) => (n.id === tempId ? realNote : n));

  // Update currentNote if it has the temp ID
  if (currentNote?.id === tempId) {
    currentNote = realNote;
  }
}
