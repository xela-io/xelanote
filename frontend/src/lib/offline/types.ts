// Offline Write Mode - Type Definitions
// All types for the offline queue, conflict resolution, and local cache.

import type { Note, NotePayload } from '$lib/api';

// --- Queue Operation Types ---

export type OfflineOperationType = 'create' | 'update' | 'delete';

export type OfflineOperationStatus = 'pending' | 'syncing' | 'completed' | 'conflict' | 'failed';

export interface OfflineCreatePayload {
  type: 'create';
  notePayload: NotePayload;
  folderPath: string;
}

export interface OfflineUpdatePayload {
  type: 'update';
  notePayload: NotePayload;
  expectedVersion: number;
}

export interface OfflineDeletePayload {
  type: 'delete';
  // No additional fields needed - noteId is sufficient.
  // Backend DELETE does not require If-Match (no version check, only WHERE id + user_id).
}

export type OfflinePayload = OfflineCreatePayload | OfflineUpdatePayload | OfflineDeletePayload;

export interface OfflineOperation {
  id: string; // UUID
  type: OfflineOperationType;
  noteId: string; // Real ID or temp_xxx for offline-created notes
  tempId?: string; // Only set for 'create' operations (temp_xxx)
  timestamp: number; // Date.now() at enqueue time
  status: OfflineOperationStatus;
  retryCount: number;
  payload: OfflinePayload;
  error?: string; // Last error message (for failed operations)
}

// --- Temp-ID Mapping ---

export interface TempIdMapping {
  tempId: string; // e.g. "temp_abc-123"
  realId: string; // Server-assigned UUID
  realVersion: number; // Server-assigned initial version
}

// --- Conflict Resolution ---

export interface ConflictData {
  operationId: string; // Queue operation ID
  noteId: string;
  localTitle: string; // Decrypted in memory (never persisted)
  localContent: string; // Decrypted in memory (never persisted)
  remoteTitle: string; // Decrypted in memory (never persisted)
  remoteContent: string; // Decrypted in memory (never persisted)
  remoteVersion: number;
  isDelete: boolean; // True if local operation was a delete
}

export type ConflictResolution = 'keep_local' | 'keep_remote' | 'keep_both';

// --- Offline Note Context ---
// Metadata passed from notes.svelte.ts to api.ts via _offlineContext.
// Contains ONLY metadata - NO plaintext content.

export interface OfflineNoteContext {
  created_at?: string;
  folder_path?: string;
  note_type?: string;
  journal_date?: string;
  ai_enabled?: boolean;
  encryption_version?: number;
}

// --- Sync Manager State ---

export interface SyncProgress {
  current: number;
  total: number;
}

// --- Local Note Cache (for offline display) ---

export interface CachedNote {
  id: string; // Note ID (real or temp)
  note: Note; // Full note object (encrypted fields only)
  cachedAt: number; // Timestamp
}
