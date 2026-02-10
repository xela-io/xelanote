export interface ApiKeyFormState {
  apiKey: string;
  showKey: boolean;
  error: string;
  isSaving: boolean;
  isDeleting: boolean;
}

export interface ApiKeyStatusDeps<TStatus> {
  getStatus: () => Promise<TStatus>;
  setStatus: (status: TStatus | null) => void;
  setLoading: (value: boolean) => void;
  errorContext: string;
}

export async function loadApiKeyStatus<TStatus>(deps: ApiKeyStatusDeps<TStatus>) {
  deps.setLoading(true);
  try {
    const status = await deps.getStatus();
    deps.setStatus(status);
  } catch (err) {
    console.error(`Failed to load ${deps.errorContext} API key status:`, err);
    deps.setStatus(null);
  } finally {
    deps.setLoading(false);
  }
}

export interface ApiKeySaveDeps {
  form: ApiKeyFormState;
  setForm: (next: Partial<ApiKeyFormState>) => void;
  validate: (value: string) => string | null;
  save: (value: string) => Promise<void>;
  reloadStatus: () => Promise<void>;
  saveError: string;
}

export async function saveApiKey(e: Event, deps: ApiKeySaveDeps) {
  e.preventDefault();
  deps.setForm({ error: '' });

  const value = deps.form.apiKey.trim();
  const validationError = deps.validate(value);
  if (validationError) {
    deps.setForm({ error: validationError });
    return;
  }

  deps.setForm({ isSaving: true });
  try {
    await deps.save(value);
    deps.setForm({ apiKey: '', showKey: false });
    await deps.reloadStatus();
  } catch (err) {
    console.error('Failed to save API key:', err);
    deps.setForm({ error: deps.saveError });
  } finally {
    deps.setForm({ isSaving: false });
  }
}

export interface ApiKeyDeleteDeps {
  confirm: () => Promise<boolean>;
  setForm: (next: Partial<ApiKeyFormState>) => void;
  deleteKey: () => Promise<void>;
  reloadStatus: () => Promise<void>;
}

export async function deleteApiKey(deps: ApiKeyDeleteDeps) {
  const confirmed = await deps.confirm();
  if (!confirmed) return;

  deps.setForm({ isDeleting: true });
  try {
    await deps.deleteKey();
    await deps.reloadStatus();
  } catch (err) {
    console.error('Failed to delete API key:', err);
  } finally {
    deps.setForm({ isDeleting: false });
  }
}
