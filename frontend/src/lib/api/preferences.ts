import { request } from './client';
import { listNotes } from './notes';
import type {
  AIAvailableModelsResponse,
  AIModelPreferences,
  AIProvider,
  ClaudeAPIKeyStatus,
  DietaryPreference,
  GeminiAPIKeyStatus,
  HomeDashboardLayoutPreference,
  Note,
  NoteVersion,
  OpenAIAPIKeyStatus,
  OpenTabsPreference,
  StorageQuota,
  UpdateEncryptionPreferencesRequest,
  UpdatePreferencesRequest,
  UpdateSecurityPreferencesRequest,
  UserPreferences,
  WebAuthnCredentialInfo,
} from './types';
import { listVersions } from './versions';

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

export async function updateHomeDashboardLayoutPreference(
  layout: HomeDashboardLayoutPreference | null
): Promise<UserPreferences> {
  return request('/users/preferences', {
    method: 'PATCH',
    body: JSON.stringify({ home_dashboard_layout: layout }),
  });
}

export async function updateOpenTabsPreference(
  tabs: OpenTabsPreference | null
): Promise<UserPreferences> {
  return request('/users/preferences', {
    method: 'PATCH',
    body: JSON.stringify({ open_tabs: tabs }),
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

// OpenAI/ChatGPT API Key management (BYOK - Bring Your Own Key)
export async function getOpenAIAPIKeyStatus(): Promise<OpenAIAPIKeyStatus> {
  return request('/users/openai-api-key/status');
}

export async function setOpenAIAPIKey(apiKey: string): Promise<{ message: string }> {
  return request('/users/openai-api-key', {
    method: 'PUT',
    body: JSON.stringify({ api_key: apiKey }),
  });
}

export async function deleteOpenAIAPIKey(): Promise<{ message: string }> {
  return request('/users/openai-api-key', {
    method: 'DELETE',
  });
}

export async function getAIProviderPreference(): Promise<{ provider: AIProvider }> {
  return request('/users/ai-provider');
}

export async function setAIProviderPreference(
  provider: AIProvider
): Promise<{ provider: AIProvider }> {
  return request('/users/ai-provider', {
    method: 'PUT',
    body: JSON.stringify({ provider }),
  });
}

export async function getAIModelPreferences(): Promise<AIModelPreferences> {
  return request('/users/ai-models');
}

export async function setAIModelPreferences(data: AIModelPreferences): Promise<AIModelPreferences> {
  return request('/users/ai-models', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function getAvailableAIModels(): Promise<AIAvailableModelsResponse> {
  return request('/users/ai-models/available');
}

export async function getDietaryPreference(): Promise<{ dietary_preference: DietaryPreference }> {
  return request('/users/dietary-preference');
}

export async function setDietaryPreference(
  preference: DietaryPreference
): Promise<{ dietary_preference: DietaryPreference }> {
  return request('/users/dietary-preference', {
    method: 'PUT',
    body: JSON.stringify({ dietary_preference: preference }),
  });
}

export async function getStorageQuota(): Promise<StorageQuota> {
  return request('/users/storage-quota');
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
