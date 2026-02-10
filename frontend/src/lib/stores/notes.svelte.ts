// Notes store using Svelte 5 runes

import { SvelteDate, SvelteSet } from 'svelte/reactivity';
import * as api from '$lib/api';
import { ApiError } from '$lib/api';
import type { Note, Backlink, Job, OfflineNoteContext } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import * as autosave from '$lib/stores/autosave.svelte';
import * as toast from '$lib/stores/toast.svelte';
import * as encryption from '$lib/stores/encryption.svelte';
import * as searchIndex from '$lib/stores/search-index.svelte';
import * as foldersStore from '$lib/stores/folders.svelte';
import { extractWikilinks, extractDueDatesDetailed } from '$lib/editor/markdown';
import { hasPendingForNote } from '$lib/offline/offline-queue';

// --- Helper functions to reduce duplication ---

/**
 * Extract wikilinks from content and deduplicate by normalized title.
 */
function extractUniqueWikilinks(content: string) {
  const rawLinks = extractWikilinks(content);
  const seenTitles = new SvelteSet<string>();
  return rawLinks.filter((l) => {
    const norm = l.title.toLowerCase().trim();
    if (seenTitles.has(norm)) return false;
    seenTitles.add(norm);
    return true;
  });
}

/**
 * Decrypt an encrypted note response from the API in-place.
 * Returns true on success, false on failure (sets error message).
 */
function decryptNoteFields(note: Note): boolean {
  try {
    const encryptedPayload: EncryptedPayload = {
      ciphertext: note.encrypted_content,
      metadata: JSON.parse(note.encryption_metadata || '{}'),
    };
    const { title, content } = encryption.decryptNote(
      note.encrypted_title || null,
      encryptedPayload
    );
    note.title = title || note.title;
    note.content = content;
    return true;
  } catch (decryptError) {
    console.error('[NOTES] Failed to decrypt note:', decryptError);
    return false;
  }
}

/**
 * Guard for offline writes: throws if paranoid mode is active while offline.
 * If checkEncryptionLock is true, also throws if encryption is locked while offline.
 */
function assertOnlineForParanoidMode(checkEncryptionLock = false): void {
  if (!navigator.onLine) {
    if (encryption.getSecurityLevel() === 'paranoid') {
      throw new Error('Offline-Schreiben im Paranoid-Modus nicht verfuegbar');
    }
    if (checkEncryptionLock && !encryption.isEncryptionUnlocked()) {
      throw new Error('Verschluesselung gesperrt - bitte online gehen');
    }
  }
}

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

// Task event queue (flushed on save)
interface PendingTaskEvent {
  noteId: string;
  taskText: string;
  taskIndex: number;
  eventType: 'completed' | 'reopened';
}

let pendingTaskEvents: PendingTaskEvent[] = [];

// Restore from sessionStorage on module load (SSR-safe)
if (typeof window !== 'undefined') {
  try {
    const stored = sessionStorage.getItem('pendingTaskEvents');
    if (stored) pendingTaskEvents = JSON.parse(stored);
  } catch {
    /* ignore */
  }
}

function persistTaskEventQueue() {
  if (typeof window === 'undefined') return;
  try {
    sessionStorage.setItem('pendingTaskEvents', JSON.stringify(pendingTaskEvents));
  } catch {
    /* ignore */
  }
}

export function queueTaskEvent(
  noteId: string,
  taskText: string,
  taskIndex: number,
  eventType: 'completed' | 'reopened'
) {
  pendingTaskEvents.push({ noteId, taskText, taskIndex, eventType });
  persistTaskEventQueue();
}

// Export functions to access and modify state
export function getCurrentNote() {
  return currentNote;
}

export function getBacklinks() {
  return currentNoteBacklinks;
}

export function getIsLoading() {
  return isLoading;
}

export function getIsSaving() {
  return isSaving;
}

export function getError() {
  return error;
}

export function clearError() {
  error = null;
}

export function getIsDirty() {
  return isDirty;
}

export function setDirty(dirty: boolean) {
  isDirty = dirty;
}

export function getNotes() {
  return notes;
}

export function getRecentNotes(limit = 5) {
  return [...notes]
    .sort((a, b) => new SvelteDate(b.updated_at).getTime() - new SvelteDate(a.updated_at).getTime())
    .slice(0, limit);
}

export function getNotesLoading() {
  return notesLoading;
}

export function getAutoSaveStatus() {
  return autoSaveStatus;
}

export function getLastAutoSave() {
  return lastAutoSave;
}

export function getAutoSaveError() {
  return autoSaveError;
}

export async function loadNotes() {
  notesLoading = true;
  try {
    // MVP LIMIT: 1000 Notes - same as tree store
    const result = await api.listNotes({ limit: 1000 });
    notes = result.notes;
  } catch (e) {
    console.error('Failed to load notes:', e);
  } finally {
    notesLoading = false;
  }
}

export async function loadNote(id: string) {
  isLoading = true;
  error = null;
  // Reset auto-save state when loading new note
  autoSaveStatus = 'idle';
  autoSaveError = null;
  lastSavedVersion = null; // Reset echo tracker
  try {
    let note: Note;

    // Offline fallback: use note from local notes[] array instead of server
    if (!navigator.onLine) {
      const localNote = notes.find((n) => n.id === id);
      if (localNote) {
        note = { ...localNote };
        console.log('[NOTES] Using local note (offline), id:', note.id);
      } else {
        throw new Error('Notiz offline nicht verfuegbar');
      }
    } else {
      note = await api.getNote(id);
      console.log(
        '[NOTES] Loaded note from API, id:',
        note.id,
        'content_encrypted:',
        note.content_encrypted,
        'encrypted_content (base64) length:',
        note.encrypted_content?.length || 0,
        'first 50 chars:',
        note.encrypted_content?.substring(0, 50) || ''
      );
    }

    // Decrypt if encrypted
    if (note.content_encrypted && note.encrypted_content) {
      if (!encryption.isEncryptionUnlocked()) {
        error = 'ENCRYPTION_LOCKED';
        currentNote = null;
        isLoading = false;
        // Don't logout - let UI handle re-authentication
        throw new Error('ENCRYPTION_LOCKED');
      }

      try {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: note.encrypted_content,
          metadata: JSON.parse(note.encryption_metadata || '{}'),
        };
        console.log(
          '[NOTES] Decrypting loaded note, wrapped_dek length:',
          encryptedPayload.metadata.wrapped_dek?.length || 0
        );

        const { title, content } = encryption.decryptNote(
          note.encrypted_title || null,
          encryptedPayload
        );

        console.log('[NOTES] Note decrypted after load, content length:', content.length);
        note.title = title || note.title; // Use decrypted title if available
        note.content = content;
      } catch (decryptError) {
        console.error('[NOTES] Failed to decrypt note:', decryptError);
        error = 'Failed to decrypt note - encryption key may be invalid';
        currentNote = null;
        isLoading = false;
        return;
      }
    }

    currentNote = note;
    console.log('[NOTES] currentNote set after load, content length:', currentNote.content.length);
    isDirty = false;
    // Load backlinks (skip offline - not critical)
    if (navigator.onLine) {
      try {
        const result = await api.getBacklinks(id);
        currentNoteBacklinks = result.backlinks;
      } catch {
        // Backlinks not available offline
      }
    } else {
      currentNoteBacklinks = [];
    }
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load note';
    currentNote = null;
  } finally {
    isLoading = false;
  }
}

export async function createNote(
  title: string,
  content = '',
  folderPath = '/',
  journalOptions?: { note_type: string; journal_date?: string }
) {
  isLoading = true;
  error = null;
  try {
    assertOnlineForParanoidMode();

    // Check folder encryption_default
    const allFolders = foldersStore.getFolders();
    const targetFolder = allFolders.find((f) => f.path === folderPath);
    // Recipes are always plaintext by default (folder may not exist yet on first creation)
    const isRecipe = journalOptions?.note_type === 'recipe';
    const shouldEncrypt = isRecipe ? false : targetFolder?.encryption_default !== false;

    let note: api.Note;
    let processedNote: api.Note;

    if (!shouldEncrypt) {
      // Create plaintext note (folder has encryption disabled)
      console.log('[NOTES] Creating plaintext note (folder encryption_default=false)');

      // Extract and deduplicate wiki-links from content (for graph view)
      const uniqueLinks = extractUniqueWikilinks(content);

      const payload: api.NotePayload = {
        title,
        content,
        folder_path: folderPath,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
        ...(journalOptions && {
          note_type: journalOptions.note_type,
          journal_date: journalOptions.journal_date,
        }),
      };

      note = await api.createNote(payload);
      processedNote = note;
    } else {
      // Check if encryption is unlocked
      const isUnlocked = encryption.isEncryptionUnlocked();
      console.log('[NOTES] Creating note, encryption unlocked:', isUnlocked);

      if (!isUnlocked) {
        error = 'ENCRYPTION_LOCKED';
        currentNote = null;
        isLoading = false;
        // Don't logout - let UI handle re-authentication via the global modal
        throw new Error('ENCRYPTION_LOCKED');
      }

      // Encrypt before sending
      const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(title, content);

      // Extract and deduplicate wiki-links from content (for graph view)
      const uniqueLinks = extractUniqueWikilinks(content);

      const payload: api.NotePayload = {
        title: encryptedTitle ? '' : title, // Send empty string if title encrypted
        encrypted_title: encryptedTitle,
        title_encrypted: !!encryptedTitle,
        encrypted_content: encryptedContent.ciphertext,
        wrapped_dek: encryptedContent.metadata.wrapped_dek,
        encryption_metadata: JSON.stringify(encryptedContent.metadata),
        keywords: keywords,
        folder_path: folderPath,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
        due_dates: extractDueDatesDetailed(content),
        // Journal fields (optional)
        ...(journalOptions && {
          note_type: journalOptions.note_type,
          journal_date: journalOptions.journal_date,
        }),
      };

      // Offline context: metadata for synthetic response (no plaintext)
      const offlineContext: OfflineNoteContext = {
        note_type: journalOptions?.note_type || 'note',
        journal_date: journalOptions?.journal_date,
        encryption_version: encryptedContent.metadata.version as number | undefined,
      };

      note = await api.createNote(payload, offlineContext);
      console.log('[NOTES] Note created, version:', note.version, 'id:', note.id);

      // Decrypt the note we just created (backend returns encrypted)
      processedNote = note;
      if (note.content_encrypted && note.encrypted_content) {
        try {
          const encryptedPayload: EncryptedPayload = {
            ciphertext: note.encrypted_content,
            metadata: JSON.parse(note.encryption_metadata || '{}'),
          };

          const decrypted = encryption.decryptNote(note.encrypted_title || null, encryptedPayload);

          processedNote = {
            ...note,
            title: decrypted.title || note.title,
            content: decrypted.content,
          };
          console.log('[NOTES] Note decrypted, content length:', decrypted.content.length);
        } catch (err) {
          console.error('[NOTES] Failed to decrypt created note:', err);
          throw new Error('Failed to decrypt created note');
        }
      }
    }

    notes = [processedNote, ...notes];
    currentNote = processedNote;
    console.log(
      '[NOTES] currentNote set, version:',
      currentNote.version,
      'content length:',
      currentNote.content.length
    );
    isDirty = false;
    currentNoteBacklinks = [];

    // Update search index for encrypted notes
    if (processedNote.content_encrypted) {
      searchIndex.addToIndex(processedNote.id, processedNote.title, processedNote.content);
    }

    return processedNote;
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to create note';
    throw e;
  } finally {
    isLoading = false;
  }
}

export async function saveNote() {
  if (!currentNote || !isDirty) return;

  // Prevent concurrent saves (guard against race conditions)
  if (isSaving) {
    console.log('Save already in progress, skipping...');
    return;
  }

  // Cancel pending auto-save (manual save takes precedence)
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout);
    autoSaveTimeout = null;
    autoSaveStatus = 'idle';
  }

  isSaving = true;
  error = null;

  // Track save start to detect if content changed during save
  const saveStartCounter = ++saveInProgressCounter;

  try {
    assertOnlineForParanoidMode();

    // Extract and deduplicate wiki-links from content (for graph view)
    const uniqueLinks = extractUniqueWikilinks(currentNote.content);

    let updated: api.Note;
    let processedUpdate: api.Note;

    // Check if note is plaintext (decrypted) - save without encryption
    if (currentNote.content_encrypted === false) {
      const payload: api.NotePayload = {
        title: currentNote.title,
        content: currentNote.content,
        folder_path: currentNote.folder_path,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
      };

      console.log(
        '[NOTES] Saving plaintext note, current version:',
        currentNote.version,
        'id:',
        currentNote.id,
        'content length:',
        currentNote.content.length,
        'links:',
        uniqueLinks.length
      );
      updated = await api.updateNote(currentNote.id, payload, currentNote.version);
      processedUpdate = updated;
    } else {
      // Check if encryption is unlocked
      if (!encryption.isEncryptionUnlocked()) {
        error = 'Encryption locked - please re-login';
        autoSaveStatus = 'error';
        autoSaveError = 'Encryption locked - please re-login';
        throw new Error('Encryption locked');
      }

      // Encrypt before sending
      const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(
        currentNote.title,
        currentNote.content
      );

      const payload = {
        title: encryptedTitle ? '' : currentNote.title, // Send empty string if title encrypted
        encrypted_title: encryptedTitle,
        title_encrypted: !!encryptedTitle,
        encrypted_content: encryptedContent.ciphertext,
        wrapped_dek: encryptedContent.metadata.wrapped_dek,
        encryption_metadata: JSON.stringify(encryptedContent.metadata),
        keywords: keywords,
        folder_path: currentNote.folder_path,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
        due_dates: extractDueDatesDetailed(currentNote.content),
      };

      // Offline context: metadata for synthetic response (no plaintext)
      // INVARIANT: payload is always complete (encrypted_content, wrapped_dek, folder_path, links)
      const offlineContext: OfflineNoteContext = {
        created_at: currentNote.created_at,
        folder_path: currentNote.folder_path,
        note_type: currentNote.note_type,
        journal_date: currentNote.journal_date,
        ai_enabled: currentNote.ai_enabled,
        encryption_version: currentNote.encryption_version,
      };

      console.log(
        '[NOTES] Saving note, current version:',
        currentNote.version,
        'id:',
        currentNote.id,
        'content length:',
        currentNote.content.length,
        'links:',
        uniqueLinks.length
      );
      console.log(
        '[NOTES] Payload encrypted_content (base64) length:',
        encryptedContent.ciphertext.length,
        'first 50 chars:',
        encryptedContent.ciphertext.substring(0, 50)
      );
      updated = await api.updateNote(currentNote.id, payload, currentNote.version, offlineContext);
      console.log(
        '[NOTES] Save successful, backend returned version:',
        updated.version,
        'encrypted_content from backend length:',
        updated.encrypted_content?.length || 0,
        'first 50 chars:',
        updated.encrypted_content?.substring(0, 50) || ''
      );

      // Decrypt the note we just saved (backend returns encrypted)
      processedUpdate = updated;
      if (updated.content_encrypted && updated.encrypted_content) {
        try {
          const encryptedPayload: EncryptedPayload = {
            ciphertext: updated.encrypted_content,
            metadata: JSON.parse(updated.encryption_metadata || '{}'),
          };

          const decrypted = encryption.decryptNote(
            updated.encrypted_title || null,
            encryptedPayload
          );

          processedUpdate = {
            ...updated,
            title: decrypted.title || updated.title,
            content: decrypted.content,
          };
          console.log('[NOTES] Update decrypted, content length:', decrypted.content.length);
        } catch (err) {
          console.error('[NOTES] Failed to decrypt updated note:', err);
          throw new Error('Failed to decrypt updated note');
        }
      }
    }

    currentNote = processedUpdate;
    lastSavedVersion = processedUpdate.version; // Track to ignore WebSocket echo
    lastSaveTimestamp = Date.now(); // Track save timestamp for grace period

    // Update search index for encrypted notes
    if (processedUpdate.content_encrypted) {
      searchIndex.updateInIndex(processedUpdate.id, processedUpdate.title, processedUpdate.content);
    }

    // Only clear isDirty if no further changes happened during save
    if (saveInProgressCounter === saveStartCounter) {
      isDirty = false;
    } else {
      console.log('[Save] Weitere Änderungen während Save erkannt, isDirty bleibt true');
    }

    // Update in list
    notes = notes.map((n) => (n.id === processedUpdate.id ? processedUpdate : n));

    // Flush queued task events for THIS note after successful save
    const noteEvents = pendingTaskEvents.filter((e) => e.noteId === processedUpdate.id);
    if (noteEvents.length > 0) {
      pendingTaskEvents = pendingTaskEvents.filter((e) => e.noteId !== processedUpdate.id);
      persistTaskEventQueue();
      for (const evt of noteEvents) {
        const payload: api.TaskEventPayload = {
          event_type: evt.eventType,
          task_index: evt.taskIndex,
        };
        if (processedUpdate.content_encrypted) {
          const encrypted = encryption.encryptTaskText(evt.taskText);
          payload.encrypted_task_text = encrypted.ciphertext;
          payload.wrapped_dek = encrypted.metadata.wrapped_dek;
          payload.encryption_metadata = JSON.stringify(encrypted.metadata);
        } else {
          payload.task_text = evt.taskText.substring(0, 500);
        }
        api
          .recordTaskEvent(processedUpdate.id, payload)
          .catch((err) => console.warn('[TASK-EVENTS] Failed to record:', err));
      }
    }

    // Show saved status (auto-resets after 3s)
    autoSaveStatus = 'saved';
    setTimeout(() => {
      if (autoSaveStatus === 'saved') autoSaveStatus = 'idle';
    }, 3000);

    return processedUpdate;
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      console.error(
        '[NOTES] Save failed: Version conflict! Local version:',
        currentNote?.version,
        'Error:',
        e.message
      );
    }
    error = e instanceof Error ? e.message : 'Failed to save note';
    autoSaveStatus = 'error';
    throw e;
  } finally {
    isSaving = false;
  }
}

/**
 * Toggle encryption state of the current note.
 * Decrypt: clears all encryption fields, stores plaintext on server.
 * Encrypt: encrypts content and sends encrypted payload.
 */
export async function toggleEncryption(): Promise<void> {
  if (!currentNote) return;

  // Prevent concurrent operations
  if (isSaving) {
    console.log('[NOTES] Save in progress, skipping encryption toggle');
    return;
  }

  // Cancel pending auto-save
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout);
    autoSaveTimeout = null;
    autoSaveStatus = 'idle';
  }

  isSaving = true;
  error = null;

  try {
    if (currentNote.content_encrypted !== false) {
      // Currently encrypted → decrypt
      console.log('[NOTES] Decrypting note:', currentNote.id);

      // For recipe notes: extract recipe data from decrypted content
      let recipeData:
        | { recipe_metadata?: api.RecipeMetadata; recipe_ingredients?: api.RecipeIngredient[] }
        | undefined;
      if (currentNote.note_type === 'recipe' && currentNote.content) {
        try {
          const parsed = JSON.parse(currentNote.content);
          if (parsed.recipe_metadata || parsed.recipe_ingredients) {
            recipeData = {
              recipe_metadata: parsed.recipe_metadata,
              recipe_ingredients: parsed.recipe_ingredients,
            };
            // Use the actual note content from the parsed payload
            currentNote = { ...currentNote, content: parsed.content ?? '' };
          }
        } catch {
          // Content is not JSON (legacy), no recipe data to restore
        }
      }

      const decrypted = await api.decryptNote(
        currentNote.id,
        currentNote.title,
        currentNote.content,
        currentNote.version,
        recipeData
      );

      currentNote = decrypted;
      lastSavedVersion = decrypted.version;
      lastSaveTimestamp = Date.now();
      isDirty = false;

      // Remove from client search index (now plaintext, server handles search)
      searchIndex.removeFromIndex(decrypted.id);

      // Update in list
      notes = notes.map((n) => (n.id === decrypted.id ? decrypted : n));
    } else {
      // Currently plaintext → encrypt
      console.log('[NOTES] Encrypting note:', currentNote.id);

      if (!encryption.isEncryptionUnlocked()) {
        throw new Error('Encryption locked - please re-login');
      }

      // For recipe notes: load metadata+ingredients and serialize into content
      let contentToEncrypt = currentNote.content;
      if (currentNote.note_type === 'recipe') {
        try {
          const detail = await api.getRecipeDetail(currentNote.id);
          if (detail.metadata || (detail.ingredients && detail.ingredients.length > 0)) {
            contentToEncrypt = JSON.stringify({
              content: currentNote.content,
              recipe_metadata: detail.metadata,
              recipe_ingredients: detail.ingredients,
            });
          }
        } catch (err) {
          console.warn(
            '[NOTES] Failed to load recipe data for encryption, encrypting content only:',
            err
          );
        }
      }

      const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(
        currentNote.title,
        contentToEncrypt
      );

      // Extract links
      const uniqueLinks = extractUniqueWikilinks(currentNote.content);

      const payload: api.NotePayload = {
        title: encryptedTitle ? '' : currentNote.title,
        encrypted_title: encryptedTitle,
        title_encrypted: !!encryptedTitle,
        encrypted_content: encryptedContent.ciphertext,
        wrapped_dek: encryptedContent.metadata.wrapped_dek,
        encryption_metadata: JSON.stringify(encryptedContent.metadata),
        keywords: keywords,
        folder_path: currentNote.folder_path,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
      };

      const updated = await api.updateNote(currentNote.id, payload, currentNote.version);

      // Decrypt response back into memory
      let processedUpdate = updated;
      if (updated.content_encrypted && updated.encrypted_content) {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: updated.encrypted_content,
          metadata: JSON.parse(updated.encryption_metadata || '{}'),
        };

        const dec = encryption.decryptNote(updated.encrypted_title || null, encryptedPayload);

        processedUpdate = {
          ...updated,
          title: dec.title || updated.title,
          content: dec.content,
        };
      }

      currentNote = processedUpdate;
      lastSavedVersion = processedUpdate.version;
      lastSaveTimestamp = Date.now();
      isDirty = false;

      // Add to client search index (now encrypted, server can't search)
      searchIndex.addToIndex(processedUpdate.id, processedUpdate.title, processedUpdate.content);

      // Update in list
      notes = notes.map((n) => (n.id === processedUpdate.id ? processedUpdate : n));
    }
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to toggle encryption';
    throw e;
  } finally {
    isSaving = false;
  }
}

export async function deleteCurrentNote() {
  if (!currentNote) return;

  assertOnlineForParanoidMode(true);

  isLoading = true;
  error = null;
  try {
    const deletedId = currentNote.id;
    await api.deleteNote(currentNote.id, true);
    notes = notes.filter((n) => n.id !== deletedId);
    searchIndex.removeFromIndex(deletedId);
    currentNote = null;
    isDirty = false;
    currentNoteBacklinks = [];
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to delete note';
    throw e;
  } finally {
    isLoading = false;
  }
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
  assertOnlineForParanoidMode(true);

  // Try to find note in local list first
  let note = notes.find((n) => n.id === id);

  // If not in list, load from backend (for sidebar drag-drop)
  if (!note) {
    console.log('[NOTES] Note not in local list, loading from backend for move');
    try {
      note = await api.getNote(id);
      // Decrypt if encrypted
      if (note.content_encrypted && note.encrypted_content) {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: note.encrypted_content,
          metadata: JSON.parse(note.encryption_metadata || '{}'),
        };
        const decrypted = encryption.decryptNote(note.encrypted_title || null, encryptedPayload);
        note.content = decrypted.content;
        note.title = decrypted.title || note.title;
      }
    } catch (err) {
      console.error('[NOTES] Failed to load note for move:', err);
      throw new Error('Note not found');
    }
  }

  // Check if encryption is unlocked
  if (!encryption.isEncryptionUnlocked()) {
    error = 'Encryption locked - please re-login';
    throw new Error('Encryption locked');
  }

  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = {
    notesList: [...notes],
    currentNote: currentNote ? { ...currentNote } : null,
  };

  // Optimistic UI update - apply changes immediately (if note is in list)
  const optimisticUpdate = { ...note, folder_path: folderPath };
  notes = notes.map((n) => (n.id === id ? optimisticUpdate : n));
  if (currentNote?.id === id) {
    currentNote = optimisticUpdate;
  }
  error = null;

  try {
    // Encrypt before sending
    const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(
      note.title,
      note.content
    );

    // Extract and deduplicate wiki-links from content (for graph view)
    const uniqueLinks = extractUniqueWikilinks(note.content);

    const payload = {
      title: encryptedTitle ? '' : note.title,
      encrypted_title: encryptedTitle,
      title_encrypted: !!encryptedTitle,
      encrypted_content: encryptedContent.ciphertext,
      wrapped_dek: encryptedContent.metadata.wrapped_dek,
      encryption_metadata: JSON.stringify(encryptedContent.metadata),
      keywords: keywords,
      folder_path: folderPath,
      links: uniqueLinks.map((l) => ({ target_title: l.title })),
      due_dates: extractDueDatesDetailed(note.content),
    };

    // Offline context: metadata for synthetic response (no plaintext)
    // INVARIANT: payload is always complete (encrypted_content, wrapped_dek, folder_path, links)
    const offlineContext: OfflineNoteContext = {
      created_at: note.created_at,
      folder_path: folderPath,
      note_type: note.note_type,
      journal_date: note.journal_date,
      ai_enabled: note.ai_enabled,
      encryption_version: note.encryption_version,
    };

    // API call in background
    const updated = await api.updateNote(id, payload, note.version, offlineContext);

    // Decrypt the updated note
    let processedUpdate = updated;
    if (updated.content_encrypted && updated.encrypted_content) {
      const encryptedPayload: EncryptedPayload = {
        ciphertext: updated.encrypted_content,
        metadata: JSON.parse(updated.encryption_metadata || '{}'),
      };
      const decrypted = encryption.decryptNote(updated.encrypted_title || null, encryptedPayload);
      processedUpdate = {
        ...updated,
        title: decrypted.title || updated.title,
        content: decrypted.content,
      };
    }

    // Update with server response
    notes = notes.map((n) => (n.id === processedUpdate.id ? processedUpdate : n));
    if (currentNote?.id === id) {
      currentNote = processedUpdate;
    }
    return processedUpdate;
  } catch (e) {
    // Revert to snapshot on error
    notes = snapshot.notesList;
    currentNote = snapshot.currentNote;
    error = e instanceof Error ? e.message : 'Failed to move note';
    throw e;
  }
}

// ============================================================================
// Auto-Save Logic
// ============================================================================

/**
 * Trigger auto-save after debounce delay
 * This is exported so the Editor component can call it
 */
export async function triggerAutoSave() {
  const noteIdAtStart = currentNote?.id;
  if (!currentNote || !isDirty || !noteIdAtStart) return;

  autoSaveStatus = 'saving';
  autoSaveError = null;

  try {
    // Race condition protection: verify note didn't change during debounce
    if (currentNote?.id !== noteIdAtStart) {
      console.log('Auto-save cancelled: note changed during debounce');
      autoSaveStatus = 'idle';
      return;
    }

    await saveNote(); // Reuse existing save function (sets autoSaveStatus)
    lastAutoSave = new SvelteDate();
  } catch (e) {
    // Handle 409 Conflict (Version Mismatch)
    if (e instanceof ApiError && e.status === 409) {
      autoSaveStatus = 'error';
      autoSaveError = 'Konflikt erkannt. Notiz wurde extern geändert.';

      // Fetch remote version for conflict resolution
      if (currentNote) {
        api
          .getNote(currentNote.id)
          .then((_latest) => {
            toast.warning('Auto-Save Konflikt. Notiz wurde remote geändert.', {
              label: 'Neu laden',
              handler: () => {
                if (currentNote) loadNote(currentNote.id);
              },
            });
          })
          .catch((err) => {
            console.error('Failed to fetch remote version:', err);
          });
      }
      // Don't completely disable auto-save, just pause on this error
    } else {
      autoSaveStatus = 'error';
      autoSaveError = e instanceof Error ? e.message : 'Auto-save failed';
    }
    console.error('Auto-save failed:', e);
  }
}

/**
 * Schedule auto-save with debounce
 * Called by components when content changes
 */
export function scheduleAutoSave() {
  if (!autosave.getAutoSaveEnabled() || !currentNote || !isDirty || isLoading) {
    return;
  }

  autoSaveStatus = 'pending';

  // Cancel existing timeout (debounce)
  if (autoSaveTimeout) clearTimeout(autoSaveTimeout);

  // Schedule auto-save after delay
  autoSaveTimeout = setTimeout(() => {
    triggerAutoSave();
  }, autosave.getAutoSaveDelay());
}

/**
 * Cancel any pending auto-save
 */
export function cancelAutoSave() {
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout);
    autoSaveTimeout = null;
  }
  if (autoSaveStatus === 'pending') {
    autoSaveStatus = 'idle';
  }
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
