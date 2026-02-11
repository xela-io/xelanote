import { getApiBaseUrl } from '../config';
import { getCSRFToken, request } from './client';
import type {
  AIAction,
  AIEnabledResponse,
  AIEnabledUpdateResponse,
  AITransformResponse,
  FormatMarkdownResponse,
  LinkSuggestion,
  NoteTitleInfo,
  SpellCheckResponse,
  SpellIssue,
  SuggestLinksResponse,
  SuggestTagsResponse,
  SummarizeRequest,
  SummarizeResponse,
  TagSuggestion,
  TaskEventPayload,
} from './types';

function parseJsonEventData(raw: string): unknown | null {
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function asEventToken(value: unknown): string | null {
  return typeof value === 'string' ? value : null;
}

function asSummaryFromEvent(value: unknown): string | null {
  if (!value || typeof value !== 'object') return null;
  const maybe = value as { summary?: unknown };
  return typeof maybe.summary === 'string' ? maybe.summary : null;
}

function asErrorFromEvent(value: unknown): string | null {
  if (!value || typeof value !== 'object') return null;
  const maybe = value as { error?: unknown };
  return typeof maybe.error === 'string' ? maybe.error : null;
}

function asStreamToken(value: unknown): string | null {
  if (!value || typeof value !== 'object') return null;
  const maybe = value as { stream_token?: unknown };
  return typeof maybe.stream_token === 'string' ? maybe.stream_token : null;
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
      const preparePayload = await prepareResponse.json();
      const streamToken = asStreamToken(preparePayload);
      if (!streamToken) {
        onError('Failed to parse stream preparation response');
        return;
      }
      url += `?token=${encodeURIComponent(streamToken)}`;
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
            const parsed = parseJsonEventData(eventData);
            const token = asEventToken(parsed);
            if (token !== null) {
              fullSummary += token;
              onToken(token);
            } else {
              // Fallback for older backend versions
              const fallbackToken = eventData.replace(/\\n/g, '\n');
              fullSummary += fallbackToken;
              onToken(fallbackToken);
            }
          } else if (eventType === 'cached') {
            const parsed = parseJsonEventData(eventData);
            const summary = asSummaryFromEvent(parsed);
            if (summary !== null) {
              onComplete(summary);
              return;
            } else {
              onError('Failed to parse cached summary');
              return;
            }
          } else if (eventType === 'done') {
            onComplete(fullSummary);
            return;
          } else if (eventType === 'error') {
            const parsed = parseJsonEventData(eventData);
            const message = asErrorFromEvent(parsed);
            if (message !== null) {
              onError(message);
            } else {
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

/**
 * Get a lightweight list of note titles for link suggestions.
 * Only returns unencrypted titles (privacy-first).
 */
export async function getNoteTitles(): Promise<NoteTitleInfo[]> {
  const response = await request<{ titles: NoteTitleInfo[] }>('/notes/titles');
  return response.titles || [];
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

export async function recordTaskEvent(noteId: string, payload: TaskEventPayload): Promise<void> {
  await request(`/notes/${noteId}/task-events`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
