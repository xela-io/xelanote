// API client for xelanote backend

import { getApiBaseUrl, isDesktop } from '../config';
import {
  enqueueOperation as enqueueOfflineOp,
  getQueueCount as getOfflineQueueCount,
} from '../offline/offline-queue';
import type { OfflineNoteContext } from '../offline/types';
import type {
  OfflineCreatePayload,
  OfflineDeletePayload,
  OfflineOperation,
  OfflineUpdatePayload,
} from '../offline/types';
import type { Note, RefreshResponse, RefreshResult } from './types';

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

export function getAccessTokenValue(): string | null {
  return getAccessToken?.() ?? null;
}

export function logoutAndRedirect(): void {
  logout?.();
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
}

// Phase 2: Differentiated refresh result for better error handling

// Helper to categorize HTTP errors
function categorizeHttpError(status: number): 'auth_error' | 'server_error' | 'network_error' {
  if (status === 401 || status === 403) return 'auth_error';
  if (status >= 500) return 'server_error';
  return 'network_error'; // 4xx (Rate-Limit etc.)
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

export function getCSRFToken(): string | null {
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

export async function request<T>(
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
