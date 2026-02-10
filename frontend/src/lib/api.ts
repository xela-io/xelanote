// API client for xelanote backend

import { getApiBaseUrl, isDesktop } from './config';
import type { OfflineNoteContext } from './offline/types';
import {
  enqueueOperation as enqueueOfflineOp,
  getQueueCount as getOfflineQueueCount,
} from './offline/offline-queue';
import type {
  OfflineOperation,
  OfflineCreatePayload,
  OfflineUpdatePayload,
  OfflineDeletePayload,
} from './offline/types';

// Re-export for use by other modules
export type { OfflineNoteContext };

// Import auth store functions (will be available at runtime)
let getAccessToken: () => string | null;
let getRefreshToken: () => string | null;
let updateTokens: (accessToken: string, refreshToken: string) => void;
let logout: () => void;

// Mutex for token refresh to prevent race conditions (Token Rotation)
// Note: refreshResultPromise is declared near refreshWithMutex() function

// Initialize auth store references (called by auth store)
export function initApiAuth(
  getToken: () => string | null,
  getRefresh: () => string | null,
  updateFn: (accessToken: string, refreshToken: string) => void,
  logoutFn: () => void
) {
  getAccessToken = getToken;
  getRefreshToken = getRefresh;
  updateTokens = updateFn;
  logout = logoutFn;
}

export interface User {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  encryption_salt?: string; // Base64-encoded salt for E2E encryption (optional)
}

export interface AuthResponse {
  access_token?: string;
  refresh_token?: string;
  user?: User;
  requires_two_factor?: boolean;
  two_factor_methods?: string[];
  pending_login_token?: string;
  encryption_salt?: string; // Base64-encoded salt for E2E encryption
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

// Phase 2: Differentiated refresh result for better error handling
export type RefreshResult =
  | { success: true; tokens: RefreshResponse }
  | { success: false; reason: 'auth_error' | 'network_error' | 'server_error' | 'timeout' };

// Helper to categorize HTTP errors
function categorizeHttpError(status: number): 'auth_error' | 'server_error' | 'network_error' {
  if (status === 401 || status === 403) return 'auth_error';
  if (status >= 500) return 'server_error';
  return 'network_error'; // 4xx (Rate-Limit etc.)
}

export interface TwoFactorSetup {
  secret: string;
  qr_code_url: string;
  backup_codes: string[];
}

export interface TwoFactorStatus {
  enabled: boolean;
  totp_enabled: boolean;
  fido2_enabled: boolean;
  fido2_key_count: number;
  verified_at: string;
  unused_backup_codes: number;
}

export interface FIDO2CredentialInfo {
  id: number;
  device_name: string;
  created_at: string;
  last_used_at?: string;
  transports?: string[];
}

export interface Note {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  display_order?: number;
  color?: string | null;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  // Encryption fields
  encrypted_content?: string; // Base64
  content_encrypted?: boolean;
  encrypted_title?: string | null; // JSON string
  title_encrypted?: boolean;
  wrapped_dek?: string; // Base64
  encryption_version?: number;
  encryption_metadata?: string; // JSON
  // Summary fields (LLM-generated)
  summary?: string | null;
  encrypted_summary?: string | null;
  summary_encrypted?: boolean;
  content_hash?: string | null;
  summary_generated_at?: string | null;
  // Journal fields
  note_type?: string; // "note" (default) or "journal"
  journal_date?: string; // YYYY-MM-DD for journal notes
  // AI-Enabled (Claude API opt-in)
  ai_enabled?: boolean; // true = Cloud-KI (Claude) allowed for this note
}

export interface Backlink {
  id: string;
  title: string;
}

export interface SearchResult {
  id: string;
  title: string;
  snippet: string;
  rank: number;
  encrypted?: boolean;
  title_encrypted?: boolean;
  encrypted_title?: string | null;
  matched_keywords?: string[];
}

export interface RenameResult {
  note: Note;
  updated_note_count: number;
}

export interface Job {
  id: string;
  type: string;
  user_id: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  result?: unknown;
  error?: string;
  created_at: string;
  updated_at: string;
  metadata: Record<string, unknown>;
}

export interface FolderInfo {
  path: string;
  note_count: number;
}

export interface Folder {
  id: number;
  path: string;
  parent_id?: number;
  name: string;
  note_count: number;
  display_order?: number;
  color?: string | null;
  created_at: string;
  updated_at: string;
  // AI-Enabled Default (Claude API opt-in)
  ai_enabled_default?: boolean; // New notes in this folder inherit this setting
  // Encryption Default
  encryption_default?: boolean; // New notes in this folder inherit this setting (true=encrypted)
}

export interface GraphNode {
  id: string;
  title: string;
  folder_path: string;
  is_resolved: boolean;
}

export interface GraphEdge {
  source_id: string;
  target_id: string;
  type: 'resolved' | 'unresolved';
}

export interface GraphMetadata {
  node_count: number;
  edge_count: number;
  truncated: boolean;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: GraphMetadata;
}

export interface Tag {
  id: number;
  name: string;
  user_id: number;
}

export interface Template {
  id: number;
  user_id: number;
  name: string;
  description: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateRequest {
  name: string;
  description: string;
  title: string;
  content: string;
}

export interface UpdateTemplateRequest {
  name: string;
  description: string;
  title: string;
  content: string;
}

export interface Snippet {
  id: number;
  user_id: number;
  name: string;
  description: string;
  content: string;
  shortcut: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSnippetRequest {
  name: string;
  description: string;
  content: string;
  shortcut?: string;
}

export interface UpdateSnippetRequest {
  name: string;
  description: string;
  content: string;
  shortcut?: string;
}

export interface QuickSearchFilters {
  query?: string;
  folders?: string[];
  tags?: string[];
  created_after?: string;
  created_before?: string;
  updated_after?: string;
  updated_before?: string;
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}

// Mutex for token refresh - stores RefreshResult instead of RefreshResponse | null
let refreshResultPromise: Promise<RefreshResult> | null = null;

// Grace period: cache last successful refresh result to prevent double-rotation
// When multiple requests get 401 near-simultaneously, the second 401 arrives after
// the mutex promise resolves. Without a grace period, a NEW refresh starts and
// tries to rotate an already-rotated token, which fails → logout.
let lastRefreshResult: RefreshResult | null = null;
let lastRefreshTime = 0;
const REFRESH_GRACE_MS = 3000; // 3-second grace window after successful refresh

/**
 * Central refresh function with mutex.
 * Prevents parallel refresh requests (Token Rotation Race Condition).
 * @returns RefreshResult with success/failure and reason
 */
export async function refreshWithMutex(): Promise<RefreshResult> {
  // If a refresh is already in progress, wait for its result
  if (refreshResultPromise) {
    console.log('[API] Waiting for ongoing refresh...');
    return refreshResultPromise;
  }

  // Grace period: if a refresh just succeeded, reuse the result
  // This prevents a second token rotation when the cookie hasn't propagated yet
  if (lastRefreshResult?.success && Date.now() - lastRefreshTime < REFRESH_GRACE_MS) {
    console.log('[API] Using cached refresh result (grace period)');
    return lastRefreshResult;
  }

  // Start new refresh
  refreshResultPromise = doRefresh();
  try {
    const result = await refreshResultPromise;
    if (result.success) {
      lastRefreshResult = result;
      lastRefreshTime = Date.now();
    }
    return result;
  } finally {
    refreshResultPromise = null;
  }
}

const REFRESH_TIMEOUT_MS = 10000; // 10s timeout for refresh request

async function doRefresh(): Promise<RefreshResult> {
  const refreshToken = getRefreshToken?.();
  // Cookie is sent automatically via credentials: 'include'

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), REFRESH_TIMEOUT_MS);

  try {
    const response = await fetch(`${getApiBaseUrl()}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: refreshToken ? JSON.stringify({ refresh_token: refreshToken }) : undefined,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      const reason = categorizeHttpError(response.status);
      console.warn(`[API] Refresh failed: ${response.status} (${reason})`);
      return { success: false, reason };
    }

    const tokens: RefreshResponse = await response.json();
    updateTokens?.(tokens.access_token, tokens.refresh_token);
    console.log('[API] Token refreshed successfully');
    return { success: true, tokens };
  } catch (error) {
    clearTimeout(timeoutId);
    if (error instanceof Error && error.name === 'AbortError') {
      console.error('[API] Refresh request timed out');
      return { success: false, reason: 'timeout' };
    }
    console.error('[API] Refresh error:', error);
    return { success: false, reason: 'network_error' };
  }
}

function getCSRFToken(): string | null {
  const match = document.cookie.match(/csrf_token=([^;]+)/);
  return match ? match[1] : null;
}

// --- Offline Write Mode: Extended request options ---

interface ExtendedRequestInit extends RequestInit {
  _offlineContext?: OfflineNoteContext;
  _offlineAllowed?: boolean; // Set by notes.svelte.ts after Paranoid/Encryption checks
}

// Callbacks for offline sync events (set by sync-manager)
let onOfflineEnqueue: ((count: number) => void) | null = null;

export function setOnOfflineEnqueue(callback: ((count: number) => void) | null) {
  onOfflineEnqueue = callback;
}

function isNoteRoute(path: string): boolean {
  // Match: POST /notes, PUT /notes/{id}, DELETE /notes/{id}
  return /^\/notes(\/[^/]+)?$/.test(path);
}

function safeJsonParse(s: string | undefined | null): Record<string, unknown> {
  if (!s) return {};
  try {
    return JSON.parse(s);
  } catch {
    return {};
  }
}

async function handleOfflineMutation<T>(path: string, options: ExtendedRequestInit): Promise<T> {
  // Only note mutations are allowed offline
  if (!isNoteRoute(path)) {
    throw new ApiError('Diese Operation ist offline nicht verfuegbar.', 0);
  }

  const method = options.method || 'GET';
  const body = options.body ? JSON.parse(options.body as string) : {};
  const headers = new Headers(options.headers);
  const ctx = options._offlineContext || {};

  let syntheticNote: Note;
  let operation: OfflineOperation;

  if (method === 'POST' && path === '/notes') {
    // CREATE
    const tempId = `temp_${crypto.randomUUID()}`;
    const encMetadata = safeJsonParse(body.encryption_metadata);

    syntheticNote = {
      id: tempId,
      title: '',
      content: '',
      folder_path: body.folder_path || '/',
      version: 1,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      encrypted_content: body.encrypted_content,
      content_encrypted: true,
      encrypted_title: body.encrypted_title || null,
      title_encrypted: body.title_encrypted || false,
      wrapped_dek: body.wrapped_dek,
      encryption_version: (encMetadata.version as number) || 2,
      encryption_metadata: body.encryption_metadata,
      note_type: body.note_type || 'note',
      journal_date: body.journal_date,
      ai_enabled: ctx.ai_enabled,
    };

    const createPayload: OfflineCreatePayload = {
      type: 'create',
      notePayload: body,
      folderPath: body.folder_path || '/',
    };

    operation = {
      id: crypto.randomUUID(),
      type: 'create',
      noteId: tempId,
      tempId: tempId,
      timestamp: Date.now(),
      status: 'pending',
      retryCount: 0,
      payload: createPayload,
    };
  } else if (method === 'PUT' && path.startsWith('/notes/')) {
    // UPDATE
    const noteId = path.split('/')[2];
    const version = parseInt(headers.get('If-Match') || '1');
    const encMetadata = safeJsonParse(body.encryption_metadata);

    syntheticNote = {
      id: noteId,
      title: '',
      content: '',
      folder_path: body.folder_path || ctx.folder_path || '/',
      version: version + 1,
      created_at: ctx.created_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
      encrypted_content: body.encrypted_content,
      content_encrypted: true,
      encrypted_title: body.encrypted_title || null,
      title_encrypted: body.title_encrypted || false,
      wrapped_dek: body.wrapped_dek,
      encryption_version: ctx.encryption_version || (encMetadata.version as number) || 2,
      encryption_metadata: body.encryption_metadata,
      note_type: ctx.note_type || 'note',
      journal_date: ctx.journal_date,
      ai_enabled: ctx.ai_enabled || false,
    };

    const updatePayload: OfflineUpdatePayload = {
      type: 'update',
      notePayload: body,
      expectedVersion: version,
    };

    operation = {
      id: crypto.randomUUID(),
      type: 'update',
      noteId: noteId,
      timestamp: Date.now(),
      status: 'pending',
      retryCount: 0,
      payload: updatePayload,
    };
  } else if (method === 'DELETE' && path.startsWith('/notes/')) {
    // DELETE
    const noteId = path.split('/')[2];

    const deletePayload: OfflineDeletePayload = {
      type: 'delete',
    };

    operation = {
      id: crypto.randomUUID(),
      type: 'delete',
      noteId: noteId,
      timestamp: Date.now(),
      status: 'pending',
      retryCount: 0,
      payload: deletePayload,
    };

    // Enqueue and return undefined (like 204 No Content)
    await enqueueOfflineOp(operation);
    if (onOfflineEnqueue) {
      const count = await getOfflineQueueCount();
      onOfflineEnqueue(count);
    }
    return undefined as T;
  } else {
    throw new ApiError('Diese Operation ist offline nicht verfuegbar.', 0);
  }

  // Enqueue the operation
  await enqueueOfflineOp(operation);
  if (onOfflineEnqueue) {
    const count = await getOfflineQueueCount();
    onOfflineEnqueue(count);
  }

  console.log(`[API] Offline mutation queued: ${operation.type} ${operation.noteId}`);
  return syntheticNote as T;
}

async function request<T>(
  path: string,
  options: ExtendedRequestInit = {},
  retryWithRefresh = true
): Promise<T> {
  // Offline interception for note mutations
  const method = options.method || 'GET';
  const isMutation = ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method);

  if (typeof navigator !== 'undefined' && !navigator.onLine && isMutation) {
    if (isNoteRoute(path) && options._offlineAllowed) {
      return handleOfflineMutation<T>(path, options);
    }
    throw new ApiError('Offline - Changes not allowed. Please go online to save.', 0);
  }

  // Add Authorization header if authenticated
  const accessToken = getAccessToken?.();
  const headers = new Headers(options.headers);

  if (!(options.body instanceof FormData)) {
    if (!headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }
  }

  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  // Add CSRF token for state-changing requests
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken);
    }
  }

  // Add desktop client identifier (allows CAPTCHA bypass on backend)
  if (isDesktop()) {
    headers.set('X-Client-Type', 'desktop');
  }

  // SEC-006: Always include credentials to send HttpOnly cookies
  let response: Response;
  try {
    response = await fetch(`${getApiBaseUrl()}${path}`, {
      ...options,
      headers,
      credentials: 'include',
    });
  } catch (fetchError) {
    // TypeError = network unreachable (despite navigator.onLine === true)
    if (fetchError instanceof TypeError && isMutation && isNoteRoute(path)) {
      // Gating check: _offlineAllowed must be true (set by notes.svelte.ts)
      // Prevents bypass of Paranoid/Encryption checks
      if (options._offlineAllowed) {
        console.warn('[API] Fetch failed despite navigator.onLine, queuing offline:', fetchError);
        return handleOfflineMutation<T>(path, options);
      }
    }
    throw fetchError;
  }

  // If 401 Unauthorized and we have a refresh token, try to refresh using central mutex
  if (response.status === 401 && accessToken && retryWithRefresh) {
    // Check if access token was already rotated by a concurrent request's refresh.
    // If the current in-memory token differs from the one we used, just retry
    // with the new token instead of triggering another refresh cycle.
    const currentToken = getAccessToken?.();
    if (currentToken && currentToken !== accessToken) {
      console.log('[API] Token already refreshed by concurrent request, retrying');
      return request<T>(path, options, false);
    }

    const result = await refreshWithMutex();

    if (result.success) {
      // Retry the original request with new token (prevent infinite loop)
      return request<T>(path, options, false);
    } else {
      // Refresh failed, logout and redirect to login
      logout?.();
      if (typeof window !== 'undefined') {
        window.location.href = '/login';
      }
      throw new ApiError('Session expired', 401);
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(error.error || 'Request failed', response.status);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

// Notes API

export async function listNotes(
  options: {
    limit?: number;
    cursor?: string;
    folder?: string;
  } = {}
): Promise<{ notes: Note[]; next_cursor?: string }> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.cursor) params.set('cursor', options.cursor);
  if (options.folder) params.set('folder', options.folder);

  const query = params.toString();
  return request(`/notes${query ? '?' + query : ''}`);
}

export async function getNote(id: string): Promise<Note> {
  return request(`/notes/${id}`);
}

export interface NotePayload {
  title: string;
  content?: string; // Optional for encrypted notes
  folder_path?: string;
  // Encryption fields
  encrypted_title?: string | null;
  title_encrypted?: boolean;
  encrypted_content?: string; // Base64
  wrapped_dek?: string; // Base64
  encryption_metadata?: string; // JSON
  keywords?: string[];
  // Client-side extracted links (for E2E encrypted notes where server can't parse content)
  links?: Array<{ target_title: string }>;
  // Client-side extracted due dates (for E2E encrypted notes)
  due_dates?: Array<{
    due_date: string;
    line_text: string;
    line_index: number;
    is_task_item: boolean;
    is_completed: boolean;
  }>;
  // Journal fields
  note_type?: string; // "note" (default) or "journal"
  journal_date?: string; // YYYY-MM-DD for journal notes
}

export async function createNote(
  data: NotePayload,
  offlineContext?: OfflineNoteContext
): Promise<Note> {
  return request('/notes', {
    method: 'POST',
    body: JSON.stringify(data),
    _offlineContext: offlineContext,
    _offlineAllowed: !!offlineContext,
  });
}

export async function updateNote(
  id: string,
  data: NotePayload,
  version: number,
  offlineContext?: OfflineNoteContext
): Promise<Note> {
  return request(`/notes/${id}`, {
    method: 'PUT',
    headers: {
      'If-Match': version.toString(),
    },
    body: JSON.stringify(data),
    _offlineContext: offlineContext,
    _offlineAllowed: !!offlineContext,
  });
}

export async function moveNote(id: string, folderPath: string, version: number): Promise<Note> {
  const note = await getNote(id);
  const payload: NotePayload = { title: note.title, folder_path: folderPath };

  if (note.content_encrypted && note.encrypted_content && note.wrapped_dek) {
    // Preserve all encryption fields for encrypted notes
    payload.encrypted_content = note.encrypted_content;
    payload.wrapped_dek = note.wrapped_dek;
    payload.encryption_metadata = note.encryption_metadata;
    payload.encrypted_title = note.encrypted_title;
    payload.title_encrypted = note.title_encrypted;
  } else {
    payload.content = note.content;
  }

  return updateNote(id, payload, version);
}

export async function deleteNote(id: string, offlineAllowed = false): Promise<void> {
  return request(`/notes/${id}`, {
    method: 'DELETE',
    _offlineAllowed: offlineAllowed,
  });
}

export async function renameNote(id: string, newTitle: string): Promise<RenameResult> {
  return request(`/notes/${id}/rename`, {
    method: 'POST',
    body: JSON.stringify({ newTitle }),
  });
}

export async function renameNoteAsync(
  id: string,
  newTitle: string
): Promise<{ job_id: string; status: string }> {
  return request(`/notes/${id}/rename?async=true`, {
    method: 'POST',
    body: JSON.stringify({ newTitle }),
  });
}

export async function getJobStatus(jobId: string): Promise<Job> {
  return request(`/jobs/${jobId}`);
}

export async function getBacklinks(noteId: string): Promise<{ backlinks: Backlink[] }> {
  return request(`/notes/${noteId}/backlinks`);
}

// Search API

function validateSearchQuery(query: string): string {
  query = query.trim();

  if (query.length > 500) {
    throw new Error('Search query too long (max 500 characters)');
  }

  const terms = query.split(/\s+/).filter((t) => t.length > 0);
  if (terms.length > 20) {
    throw new Error('Too many search terms (max 20)');
  }

  for (const term of terms) {
    if (term.length > 100) {
      throw new Error('Search term too long (max 100 characters)');
    }
  }

  return query;
}

export async function search(query: string, limit = 20): Promise<{ results: SearchResult[] }> {
  query = validateSearchQuery(query);
  const params = new URLSearchParams({ q: query, limit: limit.toString() });
  return request(`/search?${params}`);
}

export async function quickSearch(
  query: string,
  limit = 10,
  filters?: QuickSearchFilters
): Promise<{ notes: Note[] }> {
  const params = new URLSearchParams({ q: query, limit: limit.toString() });

  // Add filter parameters if provided
  if (filters) {
    if (filters.folders && filters.folders.length > 0) {
      params.set('folders', filters.folders.join(','));
    }
    if (filters.tags && filters.tags.length > 0) {
      params.set('tags', filters.tags.join(','));
    }
    if (filters.created_after) {
      params.set('created_after', filters.created_after);
    }
    if (filters.created_before) {
      params.set('created_before', filters.created_before);
    }
    if (filters.updated_after) {
      params.set('updated_after', filters.updated_after);
    }
    if (filters.updated_before) {
      params.set('updated_before', filters.updated_before);
    }
  }

  return request(`/quick-search?${params}`);
}

// Folders API (legacy)

export async function getFoldersLegacy(): Promise<{ folders: FolderInfo[] }> {
  return request('/folders-legacy');
}

// Folders API (new folders table)

export async function getFolders(): Promise<{ folders: Folder[] }> {
  return request('/folders');
}

export async function createFolder(path: string): Promise<Folder> {
  return request('/folders', {
    method: 'POST',
    body: JSON.stringify({ path }),
  });
}

export async function moveFolder(id: number, newParentPath: string): Promise<void> {
  return request(`/folders/${id}/move`, {
    method: 'PUT',
    body: JSON.stringify({ new_parent_path: newParentPath }),
  });
}

export async function deleteFolder(id: number): Promise<void> {
  return request(`/folders/${id}`, {
    method: 'DELETE',
  });
}

export async function renameFolder(id: number, newName: string): Promise<void> {
  return request(`/folders/${id}/rename`, {
    method: 'PUT',
    body: JSON.stringify({ new_name: newName }),
  });
}

export async function reorderFolders(parentID: number | null, items: number[]): Promise<void> {
  return request(`/folders/reorder`, {
    method: 'POST',
    body: JSON.stringify({ parent_id: parentID, items }),
  });
}

export async function reorderNotes(folderPath: string, items: string[]): Promise<void> {
  return request(`/notes/reorder`, {
    method: 'POST',
    body: JSON.stringify({ folder_path: folderPath, items }),
  });
}

export async function updateFolderColor(id: number, color: string | null): Promise<void> {
  return request(`/folders/${id}/color`, {
    method: 'PUT',
    body: JSON.stringify({ color }),
  });
}

export async function updateNoteColor(id: string, color: string | null): Promise<void> {
  return request(`/notes/${id}/color`, {
    method: 'PUT',
    body: JSON.stringify({ color }),
  });
}

// Tags API

export async function getTags(): Promise<Tag[]> {
  const result = await request<Tag[] | null>('/tags');
  return result || [];
}

export async function getNoteTags(noteId: string): Promise<Tag[]> {
  const result = await request<Tag[] | null>(`/notes/${noteId}/tags`);
  return result || [];
}

export async function setNoteTags(noteId: string, tags: string[]): Promise<Tag[]> {
  const result = await request<Tag[] | null>(`/notes/${noteId}/tags`, {
    method: 'PUT',
    body: JSON.stringify({ tags }),
  });
  return result || [];
}

export async function deleteTag(tagId: number): Promise<void> {
  return request(`/tags/${tagId}`, {
    method: 'DELETE',
  });
}

// Export API

export function getExportUrl(): string {
  return `${getApiBaseUrl()}/export/markdown`;
}

// Config API

export interface AppConfig {
  captcha_enabled: boolean;
  captcha_site_key?: string;
  captcha_iframe_url?: string;
  version?: string;
  error_reporting_enabled?: boolean;
}

export async function getConfig(): Promise<AppConfig> {
  return request('/config');
}

export async function getChangelog(): Promise<string> {
  const response = await fetch(`${getApiBaseUrl()}/changelog`, {
    credentials: 'include',
  });
  if (!response.ok) {
    throw new Error('Failed to fetch changelog');
  }
  return response.text();
}

// Auth API

export async function register(
  username: string,
  email: string,
  password: string,
  captchaToken?: string
): Promise<AuthResponse> {
  return request('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password, captcha_token: captchaToken }),
  });
}

export async function login(
  usernameOrEmail: string,
  password: string,
  captchaToken?: string,
  totpCode?: string,
  backupCode?: string
): Promise<AuthResponse> {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({
      username_or_email: usernameOrEmail,
      password,
      captcha_token: captchaToken,
      totp_code: totpCode,
      backup_code: backupCode,
    }),
  });
}

export async function refreshToken(refreshToken: string): Promise<RefreshResponse> {
  return request('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

/**
 * SEC-006: Refresh token using HttpOnly cookie (no body needed).
 * Used by proactive token refresh after page reload when token is not in memory.
 * credentials: 'include' sends the refresh_token cookie automatically.
 * @returns RefreshResult with differentiated error reasons
 */
export async function refreshTokenViaCookie(): Promise<RefreshResult> {
  return refreshWithMutex();
}

export async function logoutApi(refreshToken: string): Promise<void> {
  return request('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export async function getCurrentUser(): Promise<User> {
  return request('/auth/me');
}

// Two-Factor Authentication
export async function setup2FA(): Promise<TwoFactorSetup> {
  return request('/2fa/setup', {
    method: 'POST',
  });
}

export async function verify2FA(code: string): Promise<{ message: string }> {
  return request('/2fa/verify', {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
}

export async function disable2FA(
  password: string,
  totpCode?: string,
  backupCode?: string
): Promise<{ message: string }> {
  return request('/2fa', {
    method: 'DELETE',
    body: JSON.stringify({
      password,
      totp_code: totpCode,
      backup_code: backupCode,
    }),
  });
}

export async function get2FAStatus(): Promise<TwoFactorStatus> {
  return request('/2fa/status');
}

// SEC-009: Requires password re-authentication
export async function regenerateBackupCodes(password: string): Promise<{ backup_codes: string[] }> {
  return request('/2fa/backup-codes/regenerate', {
    method: 'POST',
    body: JSON.stringify({ password }),
  });
}

// FIDO2/WebAuthn 2FA API

export async function beginFIDO2Registration(): Promise<PublicKeyCredentialCreationOptions> {
  return request('/2fa/fido2/register/begin', { method: 'POST' });
}

export async function finishFIDO2Registration(
  deviceName: string,
  credential: Credential
): Promise<{ credential_id: number; backup_codes?: string[] }> {
  const response = await fetch(
    `${getApiBaseUrl()}/2fa/fido2/register/finish?device_name=${encodeURIComponent(deviceName)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(credential),
    }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Registration failed' }));
    throw new Error(error.error || 'Registration failed');
  }
  return response.json();
}

export async function listFIDO2Credentials(): Promise<FIDO2CredentialInfo[]> {
  return request('/2fa/fido2/credentials');
}

export async function deleteFIDO2Credential(id: number): Promise<void> {
  return request(`/2fa/fido2/credentials/${id}`, { method: 'DELETE' });
}

export async function beginFIDO2Auth(
  pendingLoginToken: string
): Promise<PublicKeyCredentialRequestOptions> {
  const response = await fetch(`${getApiBaseUrl()}/auth/fido2/begin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ pending_login_token: pendingLoginToken }),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Authentication failed' }));
    throw new Error(error.error || 'Authentication failed');
  }
  return response.json();
}

export async function finishFIDO2Auth(
  pendingLoginToken: string,
  credential: Credential
): Promise<AuthResponse> {
  const response = await fetch(
    `${getApiBaseUrl()}/auth/fido2/finish?pending_login_token=${encodeURIComponent(pendingLoginToken)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(credential),
    }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Authentication failed' }));
    throw new Error(error.error || 'Authentication failed');
  }
  return response.json();
}

// Upload API

export interface UploadResponse {
  url: string;
  filename: string;
}

export async function uploadImage(file: File): Promise<UploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const accessToken = getAccessToken?.();
  const headers = new Headers();
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  // Add CSRF token for state-changing requests (required when cookies are sent)
  const csrfToken = getCSRFToken();
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  // SEC-006: Include credentials for cookie-based authentication
  const response = await fetch(`${getApiBaseUrl()}/uploads`, {
    method: 'POST',
    headers: headers,
    body: formData,
    credentials: 'include',
  });

  // Handle 401 with token refresh using central mutex
  if (response.status === 401 && accessToken) {
    const result = await refreshWithMutex();

    if (result.success) {
      // Retry upload with new token
      const retryHeaders = new Headers();
      retryHeaders.set('Authorization', `Bearer ${result.tokens.access_token}`);

      // CSRF token is refreshed along with access token, get the new one
      const newCsrfToken = getCSRFToken();
      if (newCsrfToken) {
        retryHeaders.set('X-CSRF-Token', newCsrfToken);
      }

      const retryResponse = await fetch(`${getApiBaseUrl()}/uploads`, {
        method: 'POST',
        headers: retryHeaders,
        body: formData,
        credentials: 'include', // SEC-006: Include credentials
      });

      if (!retryResponse.ok) {
        const error = await retryResponse.json().catch(() => ({ error: 'Upload failed' }));
        throw new ApiError(error.error || 'Upload failed', retryResponse.status);
      }

      return retryResponse.json();
    } else {
      // Refresh failed, logout
      logout?.();
      if (typeof window !== 'undefined') {
        window.location.href = '/login';
      }
      throw new ApiError('Session expired', 401);
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Upload failed' }));
    throw new ApiError(error.error || 'Upload failed', response.status);
  }

  return response.json();
}

// Import API

export interface ImportFile {
  path: string;
  filename: string;
  content: string;
}

export interface ImportResult {
  imported: number;
  skipped: number;
  failed: number;
  folders_created: number;
  errors?: string[];
}

export async function importMarkdown(
  files: ImportFile[],
  preserveStructure = true
): Promise<ImportResult> {
  return request('/import/markdown', {
    method: 'POST',
    body: JSON.stringify({
      files,
      preserve_structure: preserveStructure,
    }),
  });
}

// Trash API

export async function listTrash(
  options: {
    limit?: number;
    cursor?: string;
  } = {}
): Promise<{ notes: Note[]; next_cursor?: string }> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.cursor) params.set('cursor', options.cursor);

  const query = params.toString();
  return request(`/trash${query ? '?' + query : ''}`);
}

export async function getTrashCount(): Promise<{ count: number }> {
  return request('/trash/count');
}

export async function restoreNote(id: string): Promise<Note> {
  return request(`/notes/${id}/restore`, {
    method: 'POST',
  });
}

export async function permanentlyDeleteNote(id: string): Promise<void> {
  return request(`/notes/${id}/permanent`, {
    method: 'DELETE',
  });
}

export async function emptyTrash(): Promise<{ deleted_count: number }> {
  return request('/trash', {
    method: 'DELETE',
  });
}

// Due Dates API

export interface DueDateItem {
  id: number;
  note_id: string;
  note_title: string;
  due_date: string;
  line_text: string;
  line_index: number;
  is_task_item: boolean;
  is_completed: boolean;
}

export async function getDueDates(showCompleted = false): Promise<DueDateItem[]> {
  const params = showCompleted ? '?show_completed=true' : '';
  const data = await request<{ due_dates: DueDateItem[] }>(`/due-dates${params}`);
  return data.due_dates;
}

// Version History API

export interface NoteVersion {
  id: number;
  note_id: string;
  user_id: number;
  version: number;
  title: string;
  content: string;
  snapshot_at: string;
  // Encryption fields (only present for encrypted notes)
  encrypted_content?: string;
  wrapped_dek?: string;
  content_encrypted?: boolean;
  title_encrypted?: boolean;
  encrypted_title?: string | null;
  encryption_version?: number;
}

export interface VersionListResponse {
  versions: NoteVersion[];
  next_cursor?: string;
  total: number;
}

export interface CompareResponse {
  version1: NoteVersion;
  version2: NoteVersion;
}

export async function listVersions(
  noteId: string,
  options: { limit?: number; cursor?: string } = {}
): Promise<VersionListResponse> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.cursor) params.set('cursor', options.cursor);

  const query = params.toString();
  return request(`/notes/${noteId}/versions${query ? '?' + query : ''}`);
}

export async function getVersion(noteId: string, version: number): Promise<NoteVersion> {
  return request(`/notes/${noteId}/versions/${version}`);
}

export async function compareVersions(
  noteId: string,
  v1: number,
  v2: number
): Promise<CompareResponse> {
  const params = new URLSearchParams({
    v1: v1.toString(),
    v2: v2.toString(),
  });
  return request(`/notes/${noteId}/versions/compare?${params}`);
}

export async function restoreVersion(
  noteId: string,
  version: number,
  currentVersion: number
): Promise<Note> {
  return request(`/notes/${noteId}/versions/${version}/restore`, {
    method: 'POST',
    headers: {
      'If-Match': currentVersion.toString(),
    },
  });
}

// ===== Graph API =====

export async function getGlobalGraph(
  options: {
    folder?: string;
    max_nodes?: number;
  } = {}
): Promise<GraphData> {
  const params = new URLSearchParams();
  if (options.folder) params.set('folder', options.folder);
  if (options.max_nodes) params.set('max_nodes', options.max_nodes.toString());

  const query = params.toString();
  return request(`/graph${query ? '?' + query : ''}`);
}

// ===== Templates API =====

export async function listTemplates(): Promise<{ templates: Template[] }> {
  return request('/templates');
}

export async function getTemplate(id: number): Promise<Template> {
  return request(`/templates/${id}`);
}

export async function createTemplate(data: CreateTemplateRequest): Promise<Template> {
  return request('/templates', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function updateTemplate(id: number, data: UpdateTemplateRequest): Promise<Template> {
  return request(`/templates/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function deleteTemplate(id: number): Promise<void> {
  return request(`/templates/${id}`, { method: 'DELETE' });
}

// ===== Snippets API =====

export async function listSnippets(): Promise<{ snippets: Snippet[] }> {
  return request('/snippets');
}

export async function getSnippet(id: number): Promise<Snippet> {
  return request(`/snippets/${id}`);
}

export async function createSnippet(data: CreateSnippetRequest): Promise<Snippet> {
  return request('/snippets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function updateSnippet(id: number, data: UpdateSnippetRequest): Promise<Snippet> {
  return request(`/snippets/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function deleteSnippet(id: number): Promise<void> {
  return request(`/snippets/${id}`, { method: 'DELETE' });
}

// ===== User Preferences API =====

export interface WebAuthnCredentialInfo {
  id: number;
  credential_id: string;
  device_name: string;
  created_at: string;
  last_used_at?: string; // CRITICAL: Display in Settings UI for device auditing
}

export interface UserPreferences {
  theme: string;
  editor_mode: 'edit' | 'preview' | 'split';
  keywords_enabled: boolean;
  encrypt_titles: boolean;
  security_level: 'paranoid' | 'balanced' | 'convenient';
  auto_lock_timeout: number; // minutes (0 = never)
  webauthn_credentials: WebAuthnCredentialInfo[];
  created: boolean;
}

export interface UpdatePreferencesRequest {
  theme: string;
  editor_mode: 'edit' | 'preview' | 'split';
}

export interface UpdateSecurityPreferencesRequest {
  security_level?: string;
  auto_lock_timeout?: number;
}

export async function getPreferences(): Promise<UserPreferences> {
  return request('/users/preferences');
}

// Alias for consistency with kek-persistence usage
export async function getUserPreferences(): Promise<UserPreferences> {
  return getPreferences();
}

export async function updatePreferences(data: UpdatePreferencesRequest): Promise<UserPreferences> {
  return request('/users/preferences', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function updateSecurityPreferences(
  data: UpdateSecurityPreferencesRequest
): Promise<UserPreferences> {
  return request('/users/preferences/security', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export interface UpdateEncryptionPreferencesRequest {
  keywords_enabled: boolean;
  encrypt_titles: boolean;
}

export async function updateEncryptionPreferences(
  data: UpdateEncryptionPreferencesRequest
): Promise<{ message: string }> {
  return request('/users/preferences/encryption', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

// WebAuthn credential management
export async function addWebAuthnCredential(
  credentialId: string,
  deviceName: string
): Promise<WebAuthnCredentialInfo> {
  return request('/users/webauthn/credentials', {
    method: 'POST',
    body: JSON.stringify({ credential_id: credentialId, device_name: deviceName }),
  });
}

export async function deleteWebAuthnCredential(credentialId: string): Promise<void> {
  return request(`/users/webauthn/credentials?credential_id=${encodeURIComponent(credentialId)}`, {
    method: 'DELETE',
  });
}

export async function touchWebAuthnCredential(credentialId: string): Promise<void> {
  return request(
    `/users/webauthn/credentials/touch?credential_id=${encodeURIComponent(credentialId)}`,
    {
      method: 'PATCH',
    }
  );
}

// Claude API Key management (BYOK - Bring Your Own Key)
export interface ClaudeAPIKeyStatus {
  has_key: boolean;
  updated_at?: string;
  masked_key?: string; // e.g., "sk-ant-api0...xxxx"
}

export async function getClaudeAPIKeyStatus(): Promise<ClaudeAPIKeyStatus> {
  return request('/users/api-key/status');
}

export async function setClaudeAPIKey(apiKey: string): Promise<{ message: string }> {
  return request('/users/api-key', {
    method: 'PUT',
    body: JSON.stringify({ api_key: apiKey }),
  });
}

export async function deleteClaudeAPIKey(): Promise<{ message: string }> {
  return request('/users/api-key', {
    method: 'DELETE',
  });
}

// Gemini API Key management (BYOK - Bring Your Own Key)
export interface GeminiAPIKeyStatus {
  has_key: boolean;
  updated_at?: string;
  masked_key?: string; // e.g., "AIzaSy...xxxx"
}

export async function getGeminiAPIKeyStatus(): Promise<GeminiAPIKeyStatus> {
  return request('/users/gemini-api-key/status');
}

export async function setGeminiAPIKey(apiKey: string): Promise<{ message: string }> {
  return request('/users/gemini-api-key', {
    method: 'PUT',
    body: JSON.stringify({ api_key: apiKey }),
  });
}

export async function deleteGeminiAPIKey(): Promise<{ message: string }> {
  return request('/users/gemini-api-key', {
    method: 'DELETE',
  });
}

export async function changeEmail(newEmail: string, currentPassword: string): Promise<void> {
  return request('/users/email', {
    method: 'PUT',
    body: JSON.stringify({ new_email: newEmail, current_password: currentPassword }),
  });
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
  reWrappedNoteDEKs?: Record<string, string>,
  reWrappedVersionDEKs?: Record<string, string>
): Promise<{ message: string; recovery_key_invalidated?: string }> {
  const body: {
    current_password: string;
    new_password: string;
    re_wrapped_note_deks?: Record<string, string>;
    re_wrapped_version_deks?: Record<string, string>;
  } = {
    current_password: currentPassword,
    new_password: newPassword,
  };

  // Add optional re-wrapped DEKs if provided
  if (reWrappedNoteDEKs && Object.keys(reWrappedNoteDEKs).length > 0) {
    body.re_wrapped_note_deks = reWrappedNoteDEKs;
  }
  if (reWrappedVersionDEKs && Object.keys(reWrappedVersionDEKs).length > 0) {
    body.re_wrapped_version_deks = reWrappedVersionDEKs;
  }

  return request('/users/password', {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

/**
 * Get all encrypted notes for the current user.
 * Uses pagination to fetch all notes with encryption enabled.
 */
export async function getAllEncryptedNotes(): Promise<Note[]> {
  const allNotes: Note[] = [];
  let cursor: string | undefined = undefined;

  // Fetch all notes using pagination
  do {
    const response = await listNotes({ limit: 100, cursor });
    allNotes.push(...response.notes);
    cursor = response.next_cursor;
  } while (cursor);

  // Filter to only encrypted notes
  return allNotes.filter((note) => note.content_encrypted || note.title_encrypted);
}

/**
 * Get all encrypted versions for a specific note.
 * Uses pagination to fetch all versions with encryption enabled.
 */
export async function getAllEncryptedVersionsForNote(noteId: string): Promise<NoteVersion[]> {
  const allVersions: NoteVersion[] = [];
  let cursor: string | undefined = undefined;

  // Fetch all versions using pagination
  do {
    const response = await listVersions(noteId, { limit: 100, cursor });
    allVersions.push(...response.versions);
    cursor = response.next_cursor;
  } while (cursor);

  // Filter to only encrypted versions
  return allVersions.filter((v) => v.content_encrypted);
}

/**
 * Get all encrypted versions for all notes of the current user.
 */
export async function getAllEncryptedVersions(): Promise<NoteVersion[]> {
  // First, get all notes
  const allNotes = await getAllEncryptedNotes();

  // Then, fetch versions for each encrypted note
  const versionPromises = allNotes.map((note) => getAllEncryptedVersionsForNote(note.id));
  const versionsArrays = await Promise.all(versionPromises);

  // Flatten the arrays
  return versionsArrays.flat();
}

// ===== Admin API =====

export interface AdminStats {
  total_users: number;
  total_notes: number;
  total_folders: number;
  total_tags: number;
  storage_used_mb: number;
}

export interface DailyCount {
  date: string;
  count: number;
}

export interface DailyFloat {
  date: string;
  value: number;
}

export interface DetailedStats {
  stats: AdminStats;
  user_growth: DailyCount[];
  note_growth: DailyCount[];
  storage_trend: DailyFloat[];
}

export interface AdminUser {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  note_count: number;
  storage_mb: number;
  created_at: string;
  totp_enabled: boolean;
  totp_verified_at?: string;
}

export interface ActivityLog {
  id: number;
  user_id: number | null;
  username: string | null;
  action: string;
  target_type: string | null;
  target_id: string | null;
  details: Record<string, unknown> | null;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
}

export interface ActivityLogsResponse {
  logs: ActivityLog[];
  total: number;
}

export interface SystemSettings {
  registration_enabled: string;
  max_notes_per_user: string;
  max_storage_mb_per_user: string;
  maintenance_mode: string;
  activity_retention_days: string;
}

export async function getAdminStats(): Promise<AdminStats> {
  return request('/admin/stats');
}

export async function getDetailedStats(): Promise<DetailedStats> {
  return request('/admin/stats/detailed');
}

export async function getAdminUsers(): Promise<AdminUser[]> {
  return request('/admin/users');
}

export async function getAdminUserDetails(id: number): Promise<AdminUser> {
  return request(`/admin/users/${id}`);
}

export async function toggleUserAdmin(id: number, isAdmin: boolean): Promise<void> {
  return request(`/admin/users/${id}/admin`, {
    method: 'PUT',
    body: JSON.stringify({ is_admin: isAdmin }),
  });
}

export async function deleteUserAdmin(id: number): Promise<void> {
  return request(`/admin/users/${id}`, {
    method: 'DELETE',
  });
}

export interface ActivityLogsOptions {
  limit?: number;
  page?: number;
  action?: string;
  user_id?: number;
  target_type?: string;
  date_from?: string;
  date_to?: string;
}

export async function getActivityLogs(
  options: ActivityLogsOptions = {}
): Promise<ActivityLogsResponse> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.page) params.set('page', options.page.toString());
  if (options.action) params.set('action', options.action);
  if (options.user_id) params.set('user_id', options.user_id.toString());
  if (options.target_type) params.set('target_type', options.target_type);
  if (options.date_from) params.set('date_from', options.date_from);
  if (options.date_to) params.set('date_to', options.date_to);

  const query = params.toString();
  return request(`/admin/activity${query ? '?' + query : ''}`);
}

export async function getSystemSettings(): Promise<SystemSettings> {
  return request('/admin/settings');
}

export async function updateSystemSettings(
  settings: Partial<SystemSettings>
): Promise<SystemSettings> {
  return request('/admin/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
}

// ===== Summary API =====

export interface SummarizeRequest {
  // For encrypted notes: decrypted content from frontend
  plaintext_content?: string;
  // For E2E notes: hash of plaintext (computed by frontend before encryption)
  plaintext_content_hash?: string;
  // For E2E notes: already encrypted summary (encrypted by frontend)
  encrypted_summary?: string;
}

export interface SummarizeResponse {
  summary: string;
}

/**
 * Generate or retrieve a summary for a note.
 * For plaintext notes: call without arguments to generate server-side
 * For encrypted notes: provide decrypted content for LLM processing
 */
export async function summarizeNote(
  noteId: string,
  plaintextContent?: string
): Promise<SummarizeResponse> {
  const body: SummarizeRequest = {};
  if (plaintextContent) {
    body.plaintext_content = plaintextContent;
  }

  return request(`/notes/${noteId}/summarize`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

/**
 * Generate a summary with streaming output for progress display.
 * Uses fetch with ReadableStream to stream tokens as they are generated.
 */
export async function summarizeNoteStream(
  noteId: string,
  onToken: (token: string) => void,
  onComplete: (summary: string) => void,
  onError: (error: string) => void,
  plaintextContent?: string
): Promise<void> {
  const baseUrl = getApiBaseUrl();
  let url = `${baseUrl}/notes/${noteId}/summarize/stream`;

  try {
    // If plaintext content provided, use prepare endpoint to avoid content in URL
    if (plaintextContent) {
      const prepareHeaders: Record<string, string> = { 'Content-Type': 'application/json' };
      const csrfToken = getCSRFToken();
      if (csrfToken) {
        prepareHeaders['X-CSRF-Token'] = csrfToken;
      }
      const prepareResponse = await fetch(`${baseUrl}/notes/${noteId}/summarize/prepare`, {
        method: 'POST',
        credentials: 'include',
        headers: prepareHeaders,
        body: JSON.stringify({ plaintext_content: plaintextContent }),
      });
      if (!prepareResponse.ok) {
        onError(`Failed to prepare stream: HTTP ${prepareResponse.status}`);
        return;
      }
      const { stream_token } = await prepareResponse.json();
      url += `?token=${encodeURIComponent(stream_token)}`;
    }

    const response = await fetch(url, {
      method: 'GET',
      credentials: 'include',
      headers: {
        Accept: 'text/event-stream',
      },
    });

    if (!response.ok) {
      const text = await response.text();
      onError(text || `HTTP ${response.status}`);
      return;
    }

    const reader = response.body?.getReader();
    if (!reader) {
      onError('Streaming not supported');
      return;
    }

    const decoder = new TextDecoder();
    let buffer = '';
    let fullSummary = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Parse SSE events from buffer
      const lines = buffer.split('\n');
      buffer = lines.pop() || ''; // Keep incomplete line in buffer

      let eventType = '';
      let eventData = '';

      for (const line of lines) {
        if (line.startsWith('event: ')) {
          eventType = line.slice(7);
        } else if (line.startsWith('data: ')) {
          eventData = line.slice(6);
        } else if (line === '' && eventType && eventData) {
          // Process complete event
          if (eventType === 'token') {
            try {
              const token = JSON.parse(eventData) as string;
              fullSummary += token;
              onToken(token);
            } catch {
              // Fallback for older backend versions
              const token = eventData.replace(/\\n/g, '\n');
              fullSummary += token;
              onToken(token);
            }
          } else if (eventType === 'cached') {
            try {
              const data = JSON.parse(eventData);
              onComplete(data.summary);
              return;
            } catch {
              onError('Failed to parse cached summary');
              return;
            }
          } else if (eventType === 'done') {
            onComplete(fullSummary);
            return;
          } else if (eventType === 'error') {
            try {
              const data = JSON.parse(eventData);
              onError(data.error || 'Unknown error');
            } catch {
              onError(eventData);
            }
            return;
          }
          eventType = '';
          eventData = '';
        }
      }
    }

    // If we get here without a done/cached event, complete with what we have
    if (fullSummary) {
      onComplete(fullSummary);
    }
  } catch (err) {
    onError(err instanceof Error ? err.message : 'Connection error');
  }
}

/**
 * Store a pre-encrypted summary for an E2E encrypted note.
 * The frontend encrypts the summary before sending it to the server.
 */
export async function storeEncryptedSummary(
  noteId: string,
  encryptedSummary: string,
  plaintextContentHash: string
): Promise<void> {
  const body: SummarizeRequest = {
    encrypted_summary: encryptedSummary,
    plaintext_content_hash: plaintextContentHash,
  };

  return request(`/notes/${noteId}/summarize`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

/**
 * Compute SHA256 hash of content (first 16 characters).
 * Used for change detection and E2E summary storage.
 */
export async function computeContentHash(content: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(content);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
  return hashHex.slice(0, 16);
}

// ===== LLM Feature: Tag Suggestions =====

export interface TagSuggestion {
  name: string;
  is_new: boolean;
  score: number;
}

export interface SuggestTagsResponse {
  suggestions: TagSuggestion[];
}

/**
 * Get LLM-based tag suggestions for a note.
 * For encrypted notes, provide the decrypted content.
 */
export async function suggestTags(
  noteId: string,
  plaintextContent?: string
): Promise<TagSuggestion[]> {
  const body: { plaintext_content?: string } = {};
  if (plaintextContent) {
    body.plaintext_content = plaintextContent;
  }

  const response = await request<SuggestTagsResponse>(`/notes/${noteId}/suggest-tags`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return response.suggestions || [];
}

// ===== LLM Feature: Link Suggestions =====

export interface LinkSuggestion {
  term: string;
  target_title: string;
  confidence: number;
}

export interface SuggestLinksResponse {
  suggestions: LinkSuggestion[];
}

/**
 * Get LLM-based wikilink suggestions for a note.
 * noteTitles: list of available note titles to link to
 * existingLinks: list of titles already linked in the note
 */
export async function suggestLinks(
  noteId: string,
  plaintextContent: string | undefined,
  noteTitles: string[],
  existingLinks: string[]
): Promise<LinkSuggestion[]> {
  const body: {
    plaintext_content?: string;
    note_titles: string[];
    existing_links: string[];
  } = {
    note_titles: noteTitles,
    existing_links: existingLinks,
  };

  if (plaintextContent) {
    body.plaintext_content = plaintextContent;
  }

  const response = await request<SuggestLinksResponse>(`/notes/${noteId}/suggest-links`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return response.suggestions || [];
}

// ===== LLM Feature: Spell Check =====

export interface SpellIssue {
  byte_offset: number;
  byte_length: number;
  original: string;
  message: string;
  suggestions: string[];
  type: 'spelling' | 'grammar';
}

export interface SpellCheckResponse {
  issues: SpellIssue[];
}

/**
 * Perform LLM-based spell check on text.
 * @param text Text to check
 * @param language "de" for German or "en" for English
 */
export async function spellCheck(
  text: string,
  language: 'de' | 'en' = 'en'
): Promise<SpellIssue[]> {
  const response = await request<SpellCheckResponse>('/llm/spell-check', {
    method: 'POST',
    body: JSON.stringify({ text, language }),
  });
  return response.issues || [];
}

// ===== Note Titles API (for link suggestions) =====

export interface NoteTitleInfo {
  id: string;
  title: string;
  encrypted: boolean;
}

/**
 * Get a lightweight list of note titles for link suggestions.
 * Only returns unencrypted titles (privacy-first).
 */
export async function getNoteTitles(): Promise<NoteTitleInfo[]> {
  const response = await request<{ titles: NoteTitleInfo[] }>('/notes/titles');
  return response.titles || [];
}

// ===== User Features API =====

export interface UserFeature {
  user_id: number;
  feature: string;
  enabled: boolean;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

/**
 * Get a specific feature configuration for the current user.
 */
export async function getFeature(feature: string): Promise<UserFeature> {
  return request(`/features/${feature}`);
}

/**
 * Set a feature configuration for the current user.
 */
export async function setFeature(
  feature: string,
  enabled: boolean,
  settings?: Record<string, unknown>
): Promise<{ success: boolean }> {
  return request(`/features/${feature}`, {
    method: 'PUT',
    body: JSON.stringify({ enabled, settings }),
  });
}

/**
 * List all feature configurations for the current user.
 */
export async function listFeatures(): Promise<UserFeature[]> {
  return request('/features');
}

// ===== Journal API =====

export interface JournalLookupResponse {
  exists: boolean;
  date: string;
  note_id: string; // Empty if not exists
}

export interface JournalCalendarResponse {
  year: number;
  month: number;
  dates: string[];
}

/**
 * Check if a journal exists for a specific date.
 * Returns the note ID if it exists.
 */
export async function lookupJournal(date: string): Promise<JournalLookupResponse> {
  return request(`/journal?date=${date}`);
}

/**
 * Get calendar data (dates with journal entries) for a specific month.
 */
export async function getJournalCalendar(
  year: number,
  month: number
): Promise<JournalCalendarResponse> {
  return request(`/journal/calendar?year=${year}&month=${month}`);
}

export interface JournalYearCalendarResponse {
  year: number;
  dates: string[];
}

/**
 * Get calendar data (dates with journal entries) for a full year.
 */
export async function getJournalYearCalendar(year: number): Promise<JournalYearCalendarResponse> {
  return request(`/journal/calendar/year?year=${year}`, { cache: 'no-store' });
}

export interface JournalEntry {
  id: string;
  title: string;
  journal_date: string;
  note_type: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
  content_encrypted: boolean;
}

export interface JournalEntriesResponse {
  entries: JournalEntry[];
}

/**
 * Get all journal entries for the current user.
 */
export async function getJournalEntries(): Promise<JournalEntriesResponse> {
  return request('/journal/entries');
}

// ===== AI-Enabled API (Claude API Opt-In) =====

export interface AIEnabledResponse {
  ai_enabled: boolean;
}

export interface AIEnabledUpdateResponse {
  status: string;
  ai_enabled: boolean;
}

/**
 * Get the ai_enabled status for a note.
 * Returns whether Claude API features are allowed for this note.
 */
export async function getNoteAIEnabled(noteId: string): Promise<boolean> {
  const response = await request<AIEnabledResponse>(`/notes/${noteId}/ai-enabled`);
  return response.ai_enabled;
}

/**
 * Update the ai_enabled status for a note.
 * When ai_enabled=true, Cloud-KI features (Claude API) are allowed.
 */
export async function updateNoteAIEnabled(noteId: string, aiEnabled: boolean): Promise<void> {
  await request<AIEnabledUpdateResponse>(`/notes/${noteId}/ai-enabled`, {
    method: 'PUT',
    body: JSON.stringify({ ai_enabled: aiEnabled }),
  });
}

/**
 * Get the ai_enabled_default status for a folder.
 * New notes created in this folder will inherit this setting.
 */
export async function getFolderAIEnabledDefault(folderId: number): Promise<boolean> {
  const response = await request<AIEnabledResponse>(`/folders/${folderId}/ai-enabled`);
  return response.ai_enabled;
}

/**
 * Update the ai_enabled_default status for a folder.
 * New notes created in this folder will inherit this setting.
 */
export async function updateFolderAIEnabledDefault(
  folderId: number,
  aiEnabled: boolean
): Promise<void> {
  await request<AIEnabledUpdateResponse>(`/folders/${folderId}/ai-enabled`, {
    method: 'PUT',
    body: JSON.stringify({ ai_enabled: aiEnabled }),
  });
}

/**
 * Get titles of notes with ai_enabled=true.
 * Used for Claude API link suggestions (only AI-enabled notes are included).
 */
export async function getNoteTitlesAIEnabled(): Promise<string[]> {
  const response = await request<{ titles: string[] }>('/notes/titles/ai-enabled');
  return response.titles || [];
}

// ===== LLM Feature: Format Markdown =====

export interface FormatMarkdownResponse {
  formatted_content: string;
}

/**
 * Format markdown content using an LLM provider.
 * @param noteId - The note ID (required for ai_enabled check)
 * @param content - The content or selection to format
 * @param plaintextContent - For encrypted notes: the decrypted content
 * @returns The formatted markdown content
 */
export async function formatMarkdown(
  noteId: string,
  content?: string,
  plaintextContent?: string
): Promise<string> {
  const body: { selection_only?: string; plaintext_content?: string } = {};

  if (plaintextContent) {
    body.plaintext_content = plaintextContent;
  } else if (content) {
    body.selection_only = content;
  }

  const response = await request<FormatMarkdownResponse>(`/notes/${noteId}/format-markdown`, {
    method: 'POST',
    body: JSON.stringify(body),
  });

  return response.formatted_content;
}

// ===== LLM Feature: AI Transform =====

/**
 * Available AI transformation actions.
 */
export type AIAction =
  | 'format'
  | 'summarize'
  | 'expand'
  | 'translate_de'
  | 'translate_en'
  | 'formal'
  | 'informal'
  | 'custom';

export interface AITransformResponse {
  transformed_content: string;
}

/**
 * Transform text using AI with various actions.
 * @param noteId - The note ID (required for ai_enabled check)
 * @param action - The transformation action to perform
 * @param content - The content or selection to transform
 * @param customPrompt - Custom instruction (only for action='custom')
 * @param plaintextContent - For encrypted notes: the decrypted content
 * @returns The transformed content
 */
export async function aiTransform(
  noteId: string,
  action: AIAction,
  content: string,
  customPrompt?: string,
  plaintextContent?: string
): Promise<string> {
  const body: {
    action: string;
    content?: string;
    plaintext_content?: string;
    custom_prompt?: string;
  } = {
    action,
  };

  if (plaintextContent) {
    body.plaintext_content = plaintextContent;
  } else if (content) {
    body.content = content;
  }

  if (action === 'custom' && customPrompt) {
    body.custom_prompt = customPrompt;
  }

  const response = await request<AITransformResponse>(`/notes/${noteId}/ai-transform`, {
    method: 'POST',
    body: JSON.stringify(body),
  });

  return response.transformed_content;
}

// ===== Task Events API =====

export interface TaskEventPayload {
  task_text?: string;
  task_index: number;
  encrypted_task_text?: string;
  wrapped_dek?: string;
  encryption_metadata?: string;
  event_type: 'completed' | 'reopened';
}

export async function recordTaskEvent(noteId: string, payload: TaskEventPayload): Promise<void> {
  await request(`/notes/${noteId}/task-events`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

// Sharing types
export interface NoteShare {
  id: number;
  note_id: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
}

export interface SharedNote {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  version: number;
  created_at: string;
  updated_at: string;
  note_type?: string;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
}

export interface UserSearchResult {
  id: number;
  username: string;
}

// Sharing API

export async function shareNote(
  noteId: string,
  identifier: string,
  role: string
): Promise<NoteShare> {
  return request<NoteShare>(`/notes/${noteId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

export async function getNoteShares(noteId: string): Promise<NoteShare[]> {
  const result = await request<{ shares: NoteShare[] }>(`/notes/${noteId}/shares`);
  return result.shares;
}

export async function updateShareRole(noteId: string, userId: number, role: string): Promise<void> {
  return request<void>(`/notes/${noteId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function removeShare(noteId: string, userId: number): Promise<void> {
  return request<void>(`/notes/${noteId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

export async function getSharedNotes(): Promise<SharedNote[]> {
  const result = await request<{ notes: SharedNote[] }>('/shared');
  return result.notes;
}

export async function getSharedNote(id: string): Promise<SharedNote> {
  return request<SharedNote>(`/shared/${id}`);
}

export async function updateSharedNote(
  id: string,
  title: string,
  content: string,
  expectedVersion: number
): Promise<SharedNote> {
  return request<SharedNote>(`/shared/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ title, content, expected_version: expectedVersion }),
  });
}

export async function searchUsers(query: string): Promise<UserSearchResult[]> {
  const result = await request<{ users: UserSearchResult[] }>(
    `/users/search?q=${encodeURIComponent(query)}`
  );
  return result.users;
}

// Folder Sharing types
export interface FolderShare {
  id: number;
  folder_id: number;
  folder_path: string;
  folder_name: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
  updated_at: string;
}

export interface SharedFolder {
  id: number;
  path: string;
  name: string;
  note_count: number;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
  created_at: string;
  updated_at: string;
}

// Folder Sharing API

export async function shareFolder(
  folderId: number,
  identifier: string,
  role: string
): Promise<FolderShare> {
  return request<FolderShare>(`/folders/${folderId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

export async function getFolderShares(folderId: number): Promise<FolderShare[]> {
  const result = await request<{ shares: FolderShare[] }>(`/folders/${folderId}/shares`);
  return result.shares;
}

export async function updateFolderShareRole(
  folderId: number,
  userId: number,
  role: string
): Promise<void> {
  return request<void>(`/folders/${folderId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function removeFolderShare(folderId: number, userId: number): Promise<void> {
  return request<void>(`/folders/${folderId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

export async function getSharedFolders(): Promise<SharedFolder[]> {
  const result = await request<{ folders: SharedFolder[] }>('/shared/folders');
  return result.folders;
}

export async function getSharedFolderNotes(folderId: number): Promise<SharedNote[]> {
  const result = await request<{ notes: SharedNote[] }>(`/shared/folders/${folderId}/notes`);
  return result.notes;
}

// Placement API

export async function placeSharedNote(noteId: string, folderId: number): Promise<void> {
  return request<void>(`/shared/${noteId}/placement`, {
    method: 'POST',
    body: JSON.stringify({ folder_id: folderId }),
  });
}

export async function removePlacement(noteId: string): Promise<void> {
  return request<void>(`/shared/${noteId}/placement`, {
    method: 'DELETE',
  });
}

// ===== Encryption Toggle =====

/**
 * Decrypt a note by clearing all encryption fields and setting plaintext content.
 * The client sends the already-decrypted title and content.
 */
export async function decryptNote(
  id: string,
  title: string,
  content: string,
  version: number,
  recipeData?: { recipe_metadata?: RecipeMetadata; recipe_ingredients?: RecipeIngredient[] }
): Promise<Note> {
  const body: Record<string, unknown> = { title, content };
  if (recipeData?.recipe_metadata) {
    body.recipe_metadata = recipeData.recipe_metadata;
  }
  if (recipeData?.recipe_ingredients) {
    body.recipe_ingredients = recipeData.recipe_ingredients;
  }
  return request(`/notes/${id}/decrypt`, {
    method: 'POST',
    headers: {
      'If-Match': version.toString(),
    },
    body: JSON.stringify(body),
  });
}

/**
 * Get the encryption_default status for a folder.
 */
export async function getFolderEncryptionDefault(folderId: number): Promise<boolean> {
  const response = await request<{ encrypted: boolean }>(`/folders/${folderId}/encryption-default`);
  return response.encrypted;
}

/**
 * Update the encryption_default status for a folder.
 */
export async function updateFolderEncryptionDefault(
  folderId: number,
  encrypted: boolean
): Promise<void> {
  await request(`/folders/${folderId}/encryption-default`, {
    method: 'PUT',
    body: JSON.stringify({ encrypted }),
  });
}

// ===== Recipe API =====

export interface RecipeMetadata {
  note_id: string;
  user_id: number;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  source_url?: string | null;
  difficulty?: 'easy' | 'medium' | 'hard' | null;
  updated_at: string;
}

export interface RecipeIngredient {
  id?: number;
  note_id?: string;
  amount?: number | null;
  amount_text?: string | null;
  unit?: string | null;
  name: string;
  group_name?: string | null;
  display_order: number;
  optional: boolean;
  scalable: boolean;
}

export interface ScaledIngredient extends RecipeIngredient {
  scaled_amount?: number | null;
  display_amount: string;
}

export interface RecipeCollection {
  id: number;
  user_id: number;
  name: string;
  description?: string | null;
  color?: string | null;
  display_order: number;
  recipe_count?: number;
}

export interface RecipeImage {
  id: number;
  note_id: string;
  user_id: number;
  image_url: string;
  caption?: string | null;
  display_order: number;
  created_at: string;
}

export interface RecipeDetail {
  note: Note;
  metadata: RecipeMetadata | null;
  ingredients: RecipeIngredient[];
  images: RecipeImage[];
  collections: RecipeCollection[];
  encrypted: boolean;
}

export interface RecipeListItem {
  id: string;
  title: string;
  folder_path: string;
  note_type: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: string | null;
  updated_at: string;
  content_encrypted: boolean;
}

/**
 * List all recipes for the current user (owner-only).
 */
export async function getRecipes(): Promise<RecipeListItem[]> {
  const result = await request<{ recipes: RecipeListItem[] }>('/recipes');
  return result.recipes || [];
}

/**
 * Get full recipe detail including metadata, ingredients, and collections.
 */
export async function getRecipeDetail(noteId: string): Promise<RecipeDetail> {
  return request<RecipeDetail>(`/recipes/${noteId}`);
}

/**
 * Update recipe metadata (servings, prep/cook time, difficulty, source URL).
 * Uses optimistic locking via expected_updated_at.
 */
export async function updateRecipeMetadata(
  noteId: string,
  metadata: {
    servings: number;
    prep_time_minutes?: number | null;
    cook_time_minutes?: number | null;
    source_url?: string | null;
    difficulty?: string | null;
  },
  expectedUpdatedAt: string
): Promise<RecipeMetadata> {
  return request<RecipeMetadata>(`/recipes/${noteId}/metadata`, {
    method: 'PUT',
    body: JSON.stringify({
      ...metadata,
      expected_updated_at: expectedUpdatedAt,
    }),
  });
}

/**
 * Replace all ingredients for a recipe (atomic operation).
 * Uses optimistic locking via expected_updated_at from recipe_metadata.
 */
export async function setRecipeIngredients(
  noteId: string,
  ingredients: RecipeIngredient[],
  expectedUpdatedAt: string
): Promise<void> {
  return request(`/recipes/${noteId}/ingredients`, {
    method: 'PUT',
    body: JSON.stringify({
      ingredients,
      expected_updated_at: expectedUpdatedAt,
    }),
  });
}

/**
 * Get scaled ingredients for a target number of servings.
 */
export async function getScaledIngredients(
  noteId: string,
  targetServings: number
): Promise<ScaledIngredient[]> {
  const result = await request<{ ingredients: ScaledIngredient[] }>(
    `/recipes/${noteId}/scaled?servings=${targetServings}`
  );
  return result.ingredients || [];
}

// Recipe Collections API

/**
 * List all recipe collections (cookbooks) for the current user.
 */
export async function getRecipeCollections(): Promise<RecipeCollection[]> {
  const result = await request<{ collections: RecipeCollection[] }>('/recipes/collections');
  return result.collections || [];
}

/**
 * Create a new recipe collection.
 */
export async function createRecipeCollection(
  name: string,
  description?: string | null,
  color?: string | null
): Promise<RecipeCollection> {
  return request<RecipeCollection>('/recipes/collections', {
    method: 'POST',
    body: JSON.stringify({ name, description, color }),
  });
}

/**
 * Update a recipe collection.
 */
export async function updateRecipeCollection(
  collectionId: number,
  name: string,
  description?: string | null,
  color?: string | null
): Promise<void> {
  return request(`/recipes/collections/${collectionId}`, {
    method: 'PUT',
    body: JSON.stringify({ name, description, color }),
  });
}

/**
 * Delete a recipe collection.
 */
export async function deleteRecipeCollection(collectionId: number): Promise<void> {
  return request(`/recipes/collections/${collectionId}`, {
    method: 'DELETE',
  });
}

/**
 * Add a recipe to a collection.
 */
export async function addRecipeToCollection(collectionId: number, noteId: string): Promise<void> {
  return request(`/recipes/collections/${collectionId}/items`, {
    method: 'POST',
    body: JSON.stringify({ note_id: noteId }),
  });
}

/**
 * Remove a recipe from a collection.
 */
export async function removeRecipeFromCollection(
  collectionId: number,
  noteId: string
): Promise<void> {
  return request(`/recipes/collections/${collectionId}/items/${noteId}`, {
    method: 'DELETE',
  });
}

/**
 * List recipes in a collection.
 */
export async function getCollectionItems(collectionId: number): Promise<RecipeListItem[]> {
  const result = await request<{ recipes: RecipeListItem[] }>(
    `/recipes/collections/${collectionId}/items`
  );
  return result.recipes || [];
}

// --- Recipe Images ---

/**
 * Add an image to a recipe.
 */
export async function addRecipeImage(
  noteId: string,
  imageUrl: string,
  caption?: string | null
): Promise<RecipeImage> {
  return request<RecipeImage>(`/recipes/${noteId}/images`, {
    method: 'POST',
    body: JSON.stringify({ image_url: imageUrl, caption }),
  });
}

/**
 * Update the caption of a recipe image.
 */
export async function updateRecipeImageCaption(
  noteId: string,
  imageId: number,
  caption: string | null
): Promise<void> {
  return request(`/recipes/${noteId}/images/${imageId}`, {
    method: 'PUT',
    body: JSON.stringify({ caption }),
  });
}

/**
 * Delete a recipe image.
 */
export async function deleteRecipeImage(noteId: string, imageId: number): Promise<void> {
  return request(`/recipes/${noteId}/images/${imageId}`, {
    method: 'DELETE',
  });
}

/**
 * Reorder recipe images.
 */
export async function reorderRecipeImages(noteId: string, imageIds: number[]): Promise<void> {
  return request(`/recipes/${noteId}/images/order`, {
    method: 'PUT',
    body: JSON.stringify({ image_ids: imageIds }),
  });
}

// --- Collection Sharing ---

export interface CollectionShare {
  id: number;
  collection_id: number;
  collection_name: string;
  owner_user_id: number;
  owner_username: string;
  shared_with_user_id: number;
  shared_with_username: string;
  role: 'viewer' | 'editor';
  created_at: string;
  updated_at: string;
}

export interface SharedCollection {
  id: number;
  name: string;
  description?: string | null;
  color?: string | null;
  recipe_count: number;
  shared_by: string;
  share_role: 'viewer' | 'editor';
  share_id: number;
  created_at: string;
  updated_at: string;
}

/**
 * Share a collection with another user.
 */
export async function shareCollection(
  collectionId: number,
  identifier: string,
  role: string
): Promise<CollectionShare> {
  return request<CollectionShare>(`/recipes/collections/${collectionId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

/**
 * Get all shares for a collection (owner-only).
 */
export async function getCollectionShares(collectionId: number): Promise<CollectionShare[]> {
  const result = await request<{ shares: CollectionShare[] }>(
    `/recipes/collections/${collectionId}/shares`
  );
  return result.shares || [];
}

/**
 * Update the role of a collection share.
 */
export async function updateCollectionShareRole(
  collectionId: number,
  userId: number,
  role: string
): Promise<void> {
  return request(`/recipes/collections/${collectionId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

/**
 * Remove a collection share.
 */
export async function removeCollectionShare(collectionId: number, userId: number): Promise<void> {
  return request(`/recipes/collections/${collectionId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

/**
 * Get all shared recipes for the current user.
 */
export async function getSharedRecipes(): Promise<SharedNote[]> {
  const result = await request<{ recipes: SharedNote[] }>('/shared/recipes');
  return result.recipes || [];
}

/**
 * Get all shared collections for the current user.
 */
export async function getSharedCollections(): Promise<SharedCollection[]> {
  const result = await request<{ collections: SharedCollection[] }>('/shared/collections');
  return result.collections || [];
}

/**
 * Get recipes in a shared collection.
 */
export async function getSharedCollectionItems(collectionId: number): Promise<Note[]> {
  const result = await request<{ recipes: Note[] }>(`/shared/collections/${collectionId}/items`);
  return result.recipes || [];
}

/**
 * Add a recipe to a shared collection (editor only).
 */
export async function addToSharedCollection(collectionId: number, noteId: string): Promise<void> {
  return request(`/shared/collections/${collectionId}/items`, {
    method: 'POST',
    body: JSON.stringify({ note_id: noteId }),
  });
}

/**
 * Remove a recipe from a shared collection (editor only).
 */
export async function removeFromSharedCollection(
  collectionId: number,
  noteId: string
): Promise<void> {
  return request(`/shared/collections/${collectionId}/items/${noteId}`, {
    method: 'DELETE',
  });
}

// --- Recipe Suggestions (AI) ---

export interface SimilarRecipeResult {
  note_id: string;
  title: string;
  similarity_score: number;
  reason: string;
}

export interface IngredientMatchResult {
  note_id: string;
  title: string;
  match_score: number;
  matched_ingredients: string[];
  missing_ingredients: string[];
}

export interface GeneratedIngredient {
  name: string;
  amount?: number | null;
  unit?: string | null;
  scalable: boolean;
  optional: boolean;
}

export interface GeneratedRecipe {
  title: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: 'easy' | 'medium' | 'hard' | null;
  ingredients: GeneratedIngredient[];
  instructions: string;
}

export interface IngredientSuggestionResult {
  matches: IngredientMatchResult[];
  generated: GeneratedRecipe[];
}

/**
 * Find recipes similar to the given recipe using AI.
 */
export async function findSimilarRecipes(
  noteId: string,
  locale: string,
  collectionId?: number | null
): Promise<SimilarRecipeResult[]> {
  const result = await request<{ results: SimilarRecipeResult[] }>('/recipes/suggestions/similar', {
    method: 'POST',
    body: JSON.stringify({
      note_id: noteId,
      collection_id: collectionId || undefined,
      locale,
    }),
  });
  return result.results || [];
}

/**
 * Suggest recipes based on available ingredients.
 */
export async function suggestByIngredients(
  ingredients: string[],
  locale: string,
  collectionId?: number | null
): Promise<IngredientSuggestionResult> {
  return request<IngredientSuggestionResult>('/recipes/suggestions/by-ingredients', {
    method: 'POST',
    body: JSON.stringify({
      ingredients,
      collection_id: collectionId || undefined,
      locale,
    }),
  });
}

/**
 * Save an AI-generated recipe as a new note.
 */
export async function saveGeneratedRecipe(data: {
  title: string;
  instructions: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: string | null;
  ingredients: GeneratedIngredient[];
  folder_path?: string;
}): Promise<{ note_id: string; title: string }> {
  return request('/recipes/suggestions/save-generated', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * Extract ingredients from a photo using AI vision.
 */
export async function extractIngredientsFromPhoto(file: File, locale: string): Promise<string[]> {
  const formData = new FormData();
  formData.append('image', file);
  formData.append('locale', locale);

  const result = await request<{ ingredients: string[] }>(
    '/recipes/suggestions/extract-ingredients',
    {
      method: 'POST',
      body: formData,
    }
  );
  return result.ingredients || [];
}
