<script lang="ts">
  import {
    AlertTriangle,
    ArrowLeft,
    Check,
    Database,
    Download,
    Edit3,
    Globe,
    Loader2,
    Palette,
    RefreshCw,
    Shield,
    Sparkles,
    Upload,
    User,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _, locale } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as api from '$lib/api';
  import { getExportUrl } from '$lib/api';
  import BackupCodesDisplay from '$lib/components/BackupCodesDisplay.svelte';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import TwoFactorDisable from '$lib/components/TwoFactorDisable.svelte';
  import TwoFactorSetup from '$lib/components/TwoFactorSetup.svelte';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import { FEATURE_FLAGS } from '$lib/config';
  import { getDefaultServerUrl, getServerUrl, isTauri, setServerUrl } from '$lib/config';
  import { e2eEncryption } from '$lib/crypto/e2e';
  import type { WebAuthnCredential } from '$lib/crypto/webauthn';
  import {
    type EmailFormState,
    handleEmailSubmit as handleEmailSubmitHelper,
  } from '$lib/routes/settings/account-forms';
  import {
    type ApiKeyFormState,
    deleteApiKey,
    loadApiKeyStatus,
    saveApiKey,
  } from '$lib/routes/settings/ai-keys';
  import {
    handleExport as handleExportHelper,
    handleImportClick as handleImportClickHelper,
    handleImportFiles as handleImportFilesHelper,
  } from '$lib/routes/settings/import-export';
  import {
    loadMigrationStats as loadMigrationStatsHelper,
    type MigrationStats,
  } from '$lib/routes/settings/migration-stats';
  import {
    handlePasswordChange,
    type PasswordFormState,
  } from '$lib/routes/settings/password-change';
  import {
    handleAutoLockTimeoutChange as handleAutoLockTimeoutChangeHelper,
    handleSecurityLevelChange as handleSecurityLevelChangeHelper,
    loadSecurityPreferences as loadSecurityPreferencesHelper,
    type SecurityLevel,
  } from '$lib/routes/settings/security-preferences';
  import AccountTab from '$lib/routes/settings/tabs/AccountTab.svelte';
  import AiTab from '$lib/routes/settings/tabs/AiTab.svelte';
  import SecurityTab from '$lib/routes/settings/tabs/SecurityTab.svelte';
  import {
    confirmBackupCodesRegeneration,
    handleTwoFactorDisableSuccess,
    handleTwoFactorSetupSuccess,
    loadTwoFactorStatus,
    requestBackupCodesRegeneration,
  } from '$lib/routes/settings/two-factor';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { type ThemeId, THEMES } from '$lib/themes';

  // Tab state (connection tab only visible in Tauri)
  let activeTab = $state<
    'appearance' | 'editor' | 'account' | 'security' | 'ai' | 'data' | 'connection'
  >('appearance');

  // Import/Export state
  let importInput: HTMLInputElement;
  let importing = $state(false);

  // Form states
  const emailForm = $state<EmailFormState>({
    newEmail: '',
    password: '',
    error: '',
  });

  const passwordForm = $state<PasswordFormState>({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
    error: '',
    isChanging: false,
    reWrappingProgress: '',
  });

  const updatePasswordForm = (next: Partial<PasswordFormState>) => {
    Object.assign(passwordForm, next);
  };

  // 2FA state
  let tfaStatus = $state<api.TwoFactorStatus | null>(null);
  let isLoadingTfa = $state(false);
  let showSetupDialog = $state(false);
  let showDisableDialog = $state(false);
  let showBackupCodesDialog = $state(false);
  let newBackupCodes = $state<string[] | null>(null);
  let isRegeneratingCodes = $state(false);
  // SEC-009: Password prompt for backup code regeneration
  let showRegeneratePasswordPrompt = $state(false);
  let regeneratePassword = $state('');

  // Security preferences state
  let securityLevel = $state<SecurityLevel>('balanced');
  let autoLockTimeout = $state(15);
  let isSavingSecurityLevel = $state(false);
  let isSavingAutoLockTimeout = $state(false);
  let webAuthnCredentials = $state<WebAuthnCredential[]>([]);

  // Migration statistics state
  let migrationStats = $state<MigrationStats | null>(null);
  let isLoadingMigrationStats = $state(false);

  // Claude API Key state (BYOK)
  let claudeApiKeyStatus = $state<api.ClaudeAPIKeyStatus | null>(null);
  let isLoadingClaudeKeyStatus = $state(false);
  const claudeKeyForm = $state<ApiKeyFormState>({
    apiKey: '',
    showKey: false,
    error: '',
    isSaving: false,
    isDeleting: false,
  });

  // Gemini API Key state (BYOK)
  let geminiApiKeyStatus = $state<api.GeminiAPIKeyStatus | null>(null);
  let isLoadingGeminiKeyStatus = $state(false);
  const geminiKeyForm = $state<ApiKeyFormState>({
    apiKey: '',
    showKey: false,
    error: '',
    isSaving: false,
    isDeleting: false,
  });

  // OpenAI/ChatGPT API Key state (BYOK)
  let openAIApiKeyStatus = $state<api.OpenAIAPIKeyStatus | null>(null);
  let isLoadingOpenAIKeyStatus = $state(false);
  const openAIKeyForm = $state<ApiKeyFormState>({
    apiKey: '',
    showKey: false,
    error: '',
    isSaving: false,
    isDeleting: false,
  });

  let activeAIProvider = $state<api.AIProvider>('auto');
  let isSavingAIProvider = $state(false);
  let dietaryPreference = $state<api.DietaryPreference>('none');
  let isSavingDietaryPreference = $state(false);
  const aiModels = $state<api.AIModelPreferences>({
    claude_model: '',
    gemini_model: '',
    chatgpt_model: '',
  });
  let availableAIModels = $state<api.AIAvailableModelsResponse | null>(null);
  let isLoadingAvailableAIModels = $state(false);
  let isSavingAIModels = $state(false);

  // Reactive Tauri detection (handles timing issues with preload script)
  let isTauriApp = $state(false);

  // Server URL state (Tauri only)
  const serverUrlForm = $state({
    url: '',
    error: '',
    isSaving: false,
    showRestartWarning: false,
  });

  // Theme grid data
  const themeList = Object.values(THEMES);
  const lightThemes = themeList.filter((t) => t.variant === 'light');
  const darkThemes = themeList.filter((t) => t.variant === 'dark');

  // Editor modes
  const editorModes = [
    {
      id: 'edit' as const,
      label: $_('page.settings.editor.mode_edit_label'),
      description: $_('page.settings.editor.mode_edit_description'),
    },
    {
      id: 'preview' as const,
      label: $_('page.settings.editor.mode_preview_label'),
      description: $_('page.settings.editor.mode_preview_description'),
    },
    {
      id: 'split' as const,
      label: $_('page.settings.editor.mode_split_label'),
      description: $_('page.settings.editor.mode_split_description'),
    },
    ...(FEATURE_FLAGS.livePreview
      ? [
          {
            id: 'live' as const,
            label: $_('page.settings.editor.mode_live_label'),
            description: $_('page.settings.editor.mode_live_description'),
          },
        ]
      : []),
  ];

  onMount(() => {
    // Detect Tauri at mount time (preload script should be done)
    isTauriApp = isTauri();
    if (isTauriApp) {
      serverUrlForm.url = getServerUrl();
    }

    load2FAStatus();
    loadSecurityPreferences();
    loadMigrationStats();
    loadClaudeApiKeyStatus();
    loadGeminiApiKeyStatus();
    loadOpenAIApiKeyStatus();
    loadAIProviderPreference();
    loadDietaryPreference();
    loadAIModelPreferences();
    loadAvailableAIModels();
    features.loadJournalFeature();
    features.loadRecipeFeature();
    features.loadCanvasFeature();
  });

  const updateClaudeKeyForm = (next: Partial<ApiKeyFormState>) => {
    Object.assign(claudeKeyForm, next);
  };

  const updateGeminiKeyForm = (next: Partial<ApiKeyFormState>) => {
    Object.assign(geminiKeyForm, next);
  };

  const updateOpenAIKeyForm = (next: Partial<ApiKeyFormState>) => {
    Object.assign(openAIKeyForm, next);
  };

  async function loadClaudeApiKeyStatus() {
    await loadApiKeyStatus({
      getStatus: () => api.getClaudeAPIKeyStatus(),
      setStatus: (status) => {
        claudeApiKeyStatus = status;
      },
      setLoading: (value) => {
        isLoadingClaudeKeyStatus = value;
      },
      errorContext: 'Claude',
    });
  }

  async function loadGeminiApiKeyStatus() {
    await loadApiKeyStatus({
      getStatus: () => api.getGeminiAPIKeyStatus(),
      setStatus: (status) => {
        geminiApiKeyStatus = status;
      },
      setLoading: (value) => {
        isLoadingGeminiKeyStatus = value;
      },
      errorContext: 'Gemini',
    });
  }

  async function loadOpenAIApiKeyStatus() {
    await loadApiKeyStatus({
      getStatus: () => api.getOpenAIAPIKeyStatus(),
      setStatus: (status) => {
        openAIApiKeyStatus = status;
      },
      setLoading: (value) => {
        isLoadingOpenAIKeyStatus = value;
      },
      errorContext: 'OpenAI',
    });
  }

  async function loadAIProviderPreference() {
    try {
      const res = await api.getAIProviderPreference();
      activeAIProvider = res.provider;
    } catch (err) {
      console.error('Failed to load AI provider preference:', err);
      activeAIProvider = 'auto';
    }
  }

  async function loadDietaryPreference() {
    try {
      const res = await api.getDietaryPreference();
      dietaryPreference = res.dietary_preference;
    } catch (err) {
      console.error('Failed to load dietary preference:', err);
      dietaryPreference = 'none';
    }
  }

  async function handleDietaryPreferenceChange(pref: api.DietaryPreference) {
    if (isSavingDietaryPreference || pref === dietaryPreference) return;
    isSavingDietaryPreference = true;
    try {
      await api.setDietaryPreference(pref);
      dietaryPreference = pref;
    } catch (err) {
      console.error('Failed to save dietary preference:', err);
    } finally {
      isSavingDietaryPreference = false;
    }
  }

  async function handleSaveClaudeApiKey(e: Event) {
    await saveApiKey(e, {
      form: claudeKeyForm,
      setForm: updateClaudeKeyForm,
      validate: (value) => {
        if (!value) return $_('page.settings.ai.api_key_required');
        if (!value.startsWith('sk-ant-')) {
          return $_('page.settings.ai.claude_key_invalid_format');
        }
        return null;
      },
      save: (value) => api.setClaudeAPIKey(value),
      reloadStatus: loadClaudeApiKeyStatus,
      saveError: $_('page.settings.ai.api_key_save_failed'),
    });
  }

  async function handleDeleteClaudeApiKey() {
    await deleteApiKey({
      confirm: () =>
        dialog.confirm({
          title: $_('page.settings.ai.delete_claude_key_title'),
          message: $_('page.settings.ai.delete_claude_key_confirm'),
          confirmText: $_('common.delete'),
          cancelText: $_('common.cancel'),
          variant: 'danger',
        }),
      setForm: updateClaudeKeyForm,
      deleteKey: () => api.deleteClaudeAPIKey(),
      reloadStatus: loadClaudeApiKeyStatus,
    });
  }

  async function handleSaveGeminiApiKey(e: Event) {
    await saveApiKey(e, {
      form: geminiKeyForm,
      setForm: updateGeminiKeyForm,
      validate: (value) => {
        if (!value) return $_('page.settings.ai.api_key_required');
        if (!value.startsWith('AIza')) {
          return $_('page.settings.ai.gemini_key_invalid_format');
        }
        return null;
      },
      save: (value) => api.setGeminiAPIKey(value),
      reloadStatus: loadGeminiApiKeyStatus,
      saveError: $_('page.settings.ai.api_key_save_failed'),
    });
  }

  async function handleDeleteGeminiApiKey() {
    await deleteApiKey({
      confirm: () =>
        dialog.confirm({
          title: $_('page.settings.ai.delete_gemini_key_title'),
          message: $_('page.settings.ai.delete_gemini_key_confirm'),
          confirmText: $_('common.delete'),
          cancelText: $_('common.cancel'),
          variant: 'danger',
        }),
      setForm: updateGeminiKeyForm,
      deleteKey: () => api.deleteGeminiAPIKey(),
      reloadStatus: loadGeminiApiKeyStatus,
    });
  }

  async function handleSaveOpenAIApiKey(e: Event) {
    await saveApiKey(e, {
      form: openAIKeyForm,
      setForm: updateOpenAIKeyForm,
      validate: (value) => {
        if (!value) return $_('page.settings.ai.api_key_required');
        if (!value.startsWith('sk-')) {
          return $_('page.settings.ai.openai_key_invalid_format');
        }
        return null;
      },
      save: (value) => api.setOpenAIAPIKey(value),
      reloadStatus: loadOpenAIApiKeyStatus,
      saveError: $_('page.settings.ai.api_key_save_failed'),
    });
  }

  async function handleDeleteOpenAIApiKey() {
    await deleteApiKey({
      confirm: () =>
        dialog.confirm({
          title: $_('page.settings.ai.delete_openai_key_title'),
          message: $_('page.settings.ai.delete_openai_key_confirm'),
          confirmText: $_('common.delete'),
          cancelText: $_('common.cancel'),
          variant: 'danger',
        }),
      setForm: updateOpenAIKeyForm,
      deleteKey: () => api.deleteOpenAIAPIKey(),
      reloadStatus: loadOpenAIApiKeyStatus,
    });
  }

  async function handleAIProviderChange(provider: api.AIProvider) {
    if (isSavingAIProvider || provider === activeAIProvider) return;
    isSavingAIProvider = true;
    try {
      await api.setAIProviderPreference(provider);
      activeAIProvider = provider;
    } catch (err) {
      console.error('Failed to save AI provider preference:', err);
    } finally {
      isSavingAIProvider = false;
    }
  }

  async function loadAIModelPreferences() {
    try {
      const models = await api.getAIModelPreferences();
      aiModels.claude_model = models.claude_model || '';
      aiModels.gemini_model = models.gemini_model || '';
      aiModels.chatgpt_model = models.chatgpt_model || '';
    } catch (err) {
      console.error('Failed to load AI model preferences:', err);
    }
  }

  async function loadAvailableAIModels() {
    isLoadingAvailableAIModels = true;
    try {
      availableAIModels = await api.getAvailableAIModels();
    } catch (err) {
      console.error('Failed to load available AI models:', err);
      availableAIModels = null;
    } finally {
      isLoadingAvailableAIModels = false;
    }
  }

  async function handleSaveAIModels(e: Event) {
    e.preventDefault();
    if (isSavingAIModels) return;

    isSavingAIModels = true;
    try {
      await api.setAIModelPreferences({
        claude_model: aiModels.claude_model.trim(),
        gemini_model: aiModels.gemini_model.trim(),
        chatgpt_model: aiModels.chatgpt_model.trim(),
      });
    } catch (err) {
      console.error('Failed to save AI model preferences:', err);
    } finally {
      isSavingAIModels = false;
    }
  }

  async function loadMigrationStats() {
    await loadMigrationStatsHelper({
      listNotes: (options) => api.listNotes(options),
      isEncrypted: (note) => Boolean(note.content_encrypted) && note.encryption_version === 1,
      isPlaintext: (note) => !note.content_encrypted || note.encryption_version === 0,
      setStats: (stats) => {
        migrationStats = stats;
      },
      setLoading: (value) => {
        isLoadingMigrationStats = value;
      },
    });
  }

  async function load2FAStatus() {
    await loadTwoFactorStatus({
      getStatus: () => api.get2FAStatus(),
      setStatus: (status) => {
        tfaStatus = status;
      },
      setLoading: (value) => {
        isLoadingTfa = value;
      },
    });
  }

  function handle2FASetupSuccess() {
    handleTwoFactorSetupSuccess({
      closeDialog: () => {
        showSetupDialog = false;
      },
      reloadStatus: load2FAStatus,
    });
  }

  function handle2FADisableSuccess() {
    handleTwoFactorDisableSuccess({
      closeDialog: () => {
        showDisableDialog = false;
      },
      reloadStatus: load2FAStatus,
    });
  }

  // SEC-009: Show password prompt before regenerating backup codes
  function handleRegenerateBackupCodes() {
    requestBackupCodesRegeneration({
      openPrompt: () => {
        showRegeneratePasswordPrompt = true;
      },
    });
  }

  async function confirmRegenerateBackupCodes() {
    await confirmBackupCodesRegeneration({
      password: regeneratePassword,
      setIsRegenerating: (value) => {
        isRegeneratingCodes = value;
      },
      regenerate: (password) => api.regenerateBackupCodes(password),
      setNewBackupCodes: (codes) => {
        newBackupCodes = codes;
      },
      setShowBackupCodesDialog: (value) => {
        showBackupCodesDialog = value;
      },
      setShowPrompt: (value) => {
        showRegeneratePasswordPrompt = value;
      },
      setPassword: (value) => {
        regeneratePassword = value;
      },
      reloadStatus: load2FAStatus,
    });
  }

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString('de-DE', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  }

  async function handleThemeChange(themeId: ThemeId) {
    await settings.setThemePreference(themeId);
  }

  async function handleEditorModeChange(mode: 'edit' | 'preview' | 'split' | 'live') {
    await settings.setEditorModePreference(mode);
  }

  async function handleJournalToggle(enabled: boolean) {
    try {
      await features.toggleJournalFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle journal feature:', error);
    }
  }

  async function handleRecipeToggle(enabled: boolean) {
    try {
      await features.toggleRecipeFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle recipe feature:', error);
    }
  }

  async function handleCanvasToggle(enabled: boolean) {
    try {
      await features.toggleCanvasFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle canvas feature:', error);
    }
  }

  async function handleEmailSubmit(e: Event) {
    await handleEmailSubmitHelper(e, {
      form: emailForm,
      setForm: (next) => {
        Object.assign(emailForm, next);
      },
      changeEmail: (newEmail, password) => settings.changeEmail(newEmail, password),
      reload: () => window.location.reload(),
      validationMessages: {
        emailRequired: $_('page.settings.account.email_required'),
        passwordRequired: $_('page.settings.account.password_required'),
        changeEmailFailed: $_('page.settings.account.change_email_failed'),
      },
    });
  }

  async function handlePasswordSubmit(e: Event) {
    await handlePasswordChange(e, {
      form: passwordForm,
      setForm: updatePasswordForm,
      validationMessages: {
        currentPasswordRequired: $_('page.settings.account.current_password_required'),
        newPasswordRequired: $_('page.settings.account.new_password_required'),
        newPasswordMinLength: $_('page.settings.account.new_password_min_length'),
        passwordsDoNotMatch: $_('page.settings.account.passwords_do_not_match'),
      },
      getEncryptionEnabled: () => encryption.isEncryptionUnlocked(),
      getAllEncryptedNotes: () => api.getAllEncryptedNotes(),
      getAllEncryptedVersions: () => api.getAllEncryptedVersions(),
      getCurrentUser: () => auth.getCurrentUser(),
      reWrapAllDEKs: (...args) => e2eEncryption.reWrapAllDEKs(...args),
      changePassword: (...args) => api.changePassword(...args),
      setupKEK: (...args) => e2eEncryption.setupKEK(...args),
      alert: (options) => dialog.alert(options),
    });
  }

  async function loadSecurityPreferences() {
    await loadSecurityPreferencesHelper({
      getPreferences: () => api.getPreferences(),
      setSecurityLevel: (level) => {
        securityLevel = level;
      },
      setAutoLockTimeout: (timeout) => {
        autoLockTimeout = timeout;
      },
      setWebAuthnCredentials: (credentials) => {
        webAuthnCredentials = credentials as WebAuthnCredential[];
      },
    });
  }

  async function handleSecurityLevelChange(newLevel: SecurityLevel) {
    await handleSecurityLevelChangeHelper(newLevel, {
      getIsSaving: () => isSavingSecurityLevel,
      setIsSaving: (value) => {
        isSavingSecurityLevel = value;
      },
      getSecurityLevel: () => securityLevel,
      getAutoLockTimeout: () => autoLockTimeout,
      setSecurityLevel: (level) => {
        securityLevel = level;
      },
      confirm: (options) => dialog.confirm(options),
      updateSecurityLevel: (level) => encryption.updateSecurityLevel(level),
      updateSecurityPreferences: (prefs) => settings.updateSecurityPreferences(prefs),
      stopAutoLock: () => autoLock.stopAutoLock(),
      initAutoLock: (timeout) => autoLock.initAutoLock(timeout),
      texts: {
        confirmTitle: $_('dialog.confirm_title'),
        confirmCancel: $_('dialog.cancel'),
        confirmLabel: $_('common.confirm'),
        confirmParanoid: $_('page.settings.security.paranoid_confirm'),
        confirmBalanced: $_('page.settings.security.balanced_confirm'),
      },
    });
  }

  async function handleAutoLockTimeoutChange() {
    await handleAutoLockTimeoutChangeHelper({
      getIsSaving: () => isSavingAutoLockTimeout,
      setIsSaving: (value) => {
        isSavingAutoLockTimeout = value;
      },
      getAutoLockTimeout: () => autoLockTimeout,
      setAutoLockTimeout: (timeout) => {
        autoLockTimeout = timeout;
      },
      getSecurityLevel: () => securityLevel,
      updateSecurityPreferences: (prefs) => settings.updateSecurityPreferences(prefs),
      stopAutoLock: () => autoLock.stopAutoLock(),
      initAutoLock: (timeout) => autoLock.initAutoLock(timeout),
    });
  }

  async function handleSettingsLogout() {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('page.sidebar.confirm_logout'),
      confirmText: $_('common.logout'),
      cancelText: $_('dialog.cancel'),
    });

    if (!confirmed) return;

    try {
      autoLock.stopAutoLock();
      await auth.logoutAsync();
      window.location.href = '/login';
    } catch {
      window.location.href = '/login';
    }
  }

  // Server URL handlers (Tauri only)
  function validateServerUrl(url: string): boolean {
    try {
      const parsed = new URL(url);
      return parsed.protocol === 'https:' || parsed.protocol === 'http:';
    } catch {
      return false;
    }
  }

  async function handleServerUrlSubmit(e: Event) {
    e.preventDefault();
    serverUrlForm.error = '';

    const url = serverUrlForm.url.trim();

    if (!url) {
      serverUrlForm.error = 'Server URL ist erforderlich';
      return;
    }

    if (!validateServerUrl(url)) {
      serverUrlForm.error = 'Ungültige URL. Bitte geben Sie eine gültige HTTP(S) URL ein.';
      return;
    }

    serverUrlForm.isSaving = true;

    try {
      // Test connection to new server
      const testResponse = await fetch(`${url}/health`);
      if (!testResponse.ok) {
        serverUrlForm.error = 'Server nicht erreichbar oder nicht kompatibel';
        return;
      }

      // Save the new URL
      setServerUrl(url);

      // Show restart warning - user needs to re-login
      serverUrlForm.showRestartWarning = true;
    } catch (err) {
      console.error('Failed to connect to server:', err);
      serverUrlForm.error = 'Verbindung zum Server fehlgeschlagen';
    } finally {
      serverUrlForm.isSaving = false;
    }
  }

  // Import/Export handlers
  function handleExport() {
    handleExportHelper({
      openWindow: (url, target) => window.open(url, target),
      getExportUrl: () => getExportUrl(),
    });
  }

  function handleImportClick() {
    handleImportClickHelper({
      triggerFileDialog: () => importInput.click(),
    });
  }

  async function handleImportFiles(e: Event) {
    await handleImportFilesHelper(e, {
      setImporting: (value) => {
        importing = value;
      },
      importMarkdown: (files, merge) => api.importMarkdown(files, merge),
      alert: (options) => dialog.alert(options),
      messages: {
        noteTitle: $_('common.note'),
        errorTitle: $_('common.error'),
        noMdSelected: $_('page.settings.data.no_md_files_selected'),
        importCompleted: $_('page.settings.data.import_completed'),
        notesImported: (count) => $_('page.settings.data.notes_imported', { values: { count } }),
        foldersCreated: (count) => $_('page.settings.data.folders_created', { values: { count } }),
        skippedNotes: (count) => $_('page.settings.data.skipped_notes', { values: { count } }),
        failedNotes: (count) => $_('page.settings.data.failed_notes', { values: { count } }),
        errorsLabel: $_('page.settings.data.errors'),
        importFailed: (error) => $_('page.settings.data.import_failed', { values: { error } }),
      },
    });
  }

  function handleResetServerUrl() {
    serverUrlForm.url = getDefaultServerUrl();
    serverUrlForm.error = '';
    serverUrlForm.showRestartWarning = false;
  }
</script>

<div class="h-full bg-background overflow-y-auto overflow-x-hidden scrollbar-none">
  <div class="max-w-3xl mx-auto p-4 md:p-8">
    <!-- Header -->
    <div class="ui-panel flex items-center gap-2 sm:gap-4 mb-6 p-4 sm:p-5">
      <MobileSidebarInlineToggle />
      <button onclick={() => goto('/')} class="ui-icon-button p-2" title={$_('page.settings.back')}>
        <ArrowLeft size={20} />
      </button>
      <h1 class="text-2xl font-bold text-foreground">{$_('page.settings.title')}</h1>
    </div>

    <!-- Tabs: Option A - Icon-only on mobile, with text on desktop -->
    <div class="ui-panel p-1.5 sm:p-2 mb-8 overflow-x-auto scrollbar-none">
      <div class="ui-tablist min-w-max" role="tablist">
        <button
          onclick={() => (activeTab = 'appearance')}
          title={$_('page.settings.tabs.appearance')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'appearance'}
        >
          <Palette size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.appearance')}</span>
        </button>
        <button
          onclick={() => (activeTab = 'editor')}
          title={$_('page.settings.tabs.editor')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'editor'}
        >
          <Edit3 size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.editor')}</span>
        </button>
        <button
          onclick={() => (activeTab = 'security')}
          title={$_('page.settings.tabs.security')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'security'}
        >
          <Shield size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.security')}</span>
        </button>
        <button
          onclick={() => (activeTab = 'account')}
          title={$_('page.settings.tabs.account')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'account'}
        >
          <User size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.account')}</span>
        </button>
        <button
          onclick={() => (activeTab = 'ai')}
          title={$_('page.settings.tabs.ai')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'ai'}
        >
          <Sparkles size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.ai')}</span>
        </button>
        <button
          onclick={() => (activeTab = 'data')}
          title={$_('page.settings.tabs.data')}
          class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
          role="tab"
          aria-selected={activeTab === 'data'}
        >
          <Database size={18} />
          <span class="hidden sm:inline">{$_('page.settings.tabs.data')}</span>
        </button>
        {#if isTauriApp}
          <button
            onclick={() => (activeTab = 'connection')}
            title="Connection"
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'connection'}
          >
            <Globe size={18} />
            <span class="hidden sm:inline">Connection</span>
          </button>
        {/if}
      </div>
    </div>

    <!-- Tab Content -->
    <div class="ui-panel p-4 sm:p-5">
      {#if activeTab === 'appearance'}
        <div class="space-y-8">
          <!-- Language -->
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              {$_('page.settings.appearance.language')}
            </h3>
            <div class="space-y-2">
              <select
                value={$locale}
                onchange={(e) => {
                  const newLocale = (e.target as HTMLSelectElement).value;
                  locale.set(newLocale);
                  window.localStorage.setItem('locale', newLocale);
                }}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
								focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                <option value="de">Deutsch</option>
                <option value="en">English</option>
              </select>
            </div>
          </div>

          <!-- Dark Themes -->
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              {$_('page.settings.appearance.dark_themes')}
            </h3>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
              {#each darkThemes as theme (theme.id)}
                <button
                  onclick={() => handleThemeChange(theme.id)}
                  disabled={settings.getIsSavingPreferences()}
                  class="relative p-4 rounded-lg border-2 transition-all text-left
									{ui.getCurrentThemeId() === theme.id
                    ? 'border-primary bg-primary/5'
                    : 'border-border hover:border-primary/50 bg-card'}"
                >
                  {#if ui.getCurrentThemeId() === theme.id}
                    <div class="absolute top-2 right-2 text-primary">
                      <Check size={16} />
                    </div>
                  {/if}
                  <div class="font-medium text-foreground text-sm">{theme.name}</div>
                  {#if theme.description}
                    <div class="text-xs text-muted-foreground mt-1">{theme.description}</div>
                  {/if}
                </button>
              {/each}
            </div>
          </div>

          <!-- Light Themes -->
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              {$_('page.settings.appearance.light_themes')}
            </h3>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
              {#each lightThemes as theme (theme.id)}
                <button
                  onclick={() => handleThemeChange(theme.id)}
                  disabled={settings.getIsSavingPreferences()}
                  class="relative p-4 rounded-lg border-2 transition-all text-left
									{ui.getCurrentThemeId() === theme.id
                    ? 'border-primary bg-primary/5'
                    : 'border-border hover:border-primary/50 bg-card'}"
                >
                  {#if ui.getCurrentThemeId() === theme.id}
                    <div class="absolute top-2 right-2 text-primary">
                      <Check size={16} />
                    </div>
                  {/if}
                  <div class="font-medium text-foreground text-sm">{theme.name}</div>
                  {#if theme.description}
                    <div class="text-xs text-muted-foreground mt-1">{theme.description}</div>
                  {/if}
                </button>
              {/each}
            </div>
          </div>
        </div>
      {:else if activeTab === 'editor'}
        <div class="space-y-6">
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              {$_('page.settings.editor.default_editor_mode')}
            </h3>
            <div class="space-y-2">
              {#each editorModes as mode (mode.id)}
                <button
                  onclick={() => handleEditorModeChange(mode.id)}
                  disabled={settings.getIsSavingPreferences()}
                  class="w-full flex items-center gap-4 p-4 rounded-lg border-2 transition-all text-left
									{ui.getEditorMode() === mode.id
                    ? 'border-primary bg-primary/5'
                    : 'border-border hover:border-primary/50 bg-card'}"
                >
                  <div
                    class="w-5 h-5 rounded-full border-2 flex items-center justify-center
									{ui.getEditorMode() === mode.id ? 'border-primary' : 'border-muted-foreground'}"
                  >
                    {#if ui.getEditorMode() === mode.id}
                      <div class="w-2.5 h-2.5 rounded-full bg-primary"></div>
                    {/if}
                  </div>
                  <div class="flex-1">
                    <div class="font-medium text-foreground">{mode.label}</div>
                    <div class="text-sm text-muted-foreground">{mode.description}</div>
                  </div>
                </button>
              {/each}
            </div>
          </div>

          <!-- Performance Settings (Experimental) -->
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              Performance (Experimental)
            </h3>
            <div class="space-y-4">
              <label
                class="flex items-start gap-3 p-4 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={settings.getVirtualTreeEnabled()}
                  onchange={(e) => settings.setVirtualTreeEnabled(e.currentTarget.checked)}
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="font-medium text-foreground">Virtual Tree Scrolling</div>
                  <div class="text-sm text-muted-foreground mt-1">
                    Verbessert die Performance bei 500+ Notizen durch virtuelles Scrollen (nur
                    sichtbare Items werden gerendert).
                    <strong class="text-orange-600 dark:text-orange-400">Hinweis:</strong> Drag-and-Drop
                    ist auf sichtbare Items beschränkt.
                  </div>
                </div>
              </label>
            </div>
          </div>

          <!-- Feature Toggles -->
          <div>
            <h3 class="text-sm font-medium text-muted-foreground mb-4">
              {$_('page.settings.editor.features_title')}
            </h3>
            <div class="space-y-4">
              <label
                class="flex items-start gap-3 p-4 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={features.getJournalFeatureEnabled()}
                  disabled={features.getJournalFeatureLoading()}
                  onchange={(e) => handleJournalToggle(e.currentTarget.checked)}
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="font-medium text-foreground">
                    {$_('page.settings.editor.journal_feature_title')}
                  </div>
                  <div class="text-sm text-muted-foreground mt-1">
                    {$_('page.settings.editor.journal_feature_description')}
                  </div>
                </div>
                {#if features.getJournalFeatureLoading()}
                  <Loader2 size={16} class="animate-spin text-muted-foreground" />
                {/if}
              </label>

              <label
                class="flex items-start gap-3 p-4 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={features.getRecipeFeatureEnabled()}
                  disabled={features.getRecipeFeatureLoading()}
                  onchange={(e) => handleRecipeToggle(e.currentTarget.checked)}
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="font-medium text-foreground">
                    {$_('page.settings.editor.recipe_feature_title')}
                  </div>
                  <div class="text-sm text-muted-foreground mt-1">
                    {$_('page.settings.editor.recipe_feature_description')}
                  </div>
                </div>
                {#if features.getRecipeFeatureLoading()}
                  <Loader2 size={16} class="animate-spin text-muted-foreground" />
                {/if}
              </label>

              <label
                class="flex items-start gap-3 p-4 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={features.getCanvasFeatureEnabled()}
                  disabled={features.getCanvasFeatureLoading()}
                  onchange={(e) => handleCanvasToggle(e.currentTarget.checked)}
                  class="mt-1"
                />
                <div class="flex-1">
                  <div class="font-medium text-foreground">
                    {$_('page.settings.editor.canvas_feature_title')}
                  </div>
                  <div class="text-sm text-muted-foreground mt-1">
                    {$_('page.settings.editor.canvas_feature_description')}
                  </div>
                </div>
                {#if features.getCanvasFeatureLoading()}
                  <Loader2 size={16} class="animate-spin text-muted-foreground" />
                {/if}
              </label>

              {#if errorReporter.getServiceAvailable()}
                <label
                  class="flex items-start gap-3 p-4 rounded-lg border border-border bg-card cursor-pointer hover:border-primary/50 transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={errorReporter.isEnabled()}
                    onchange={(e) => errorReporter.setEnabled(e.currentTarget.checked)}
                    class="mt-1"
                  />
                  <div class="flex-1">
                    <div class="font-medium text-foreground">{$_('feedback.settings_title')}</div>
                    <div class="text-sm text-muted-foreground mt-1">
                      {$_('feedback.settings_description')}
                    </div>
                  </div>
                </label>
              {/if}
            </div>
          </div>
        </div>
      {:else if activeTab === 'security'}
        <SecurityTab
          {encryption}
          {securityLevel}
          {isSavingSecurityLevel}
          bind:autoLockTimeout
          {isSavingAutoLockTimeout}
          {handleSecurityLevelChange}
          {handleAutoLockTimeoutChange}
          {webAuthnCredentials}
          {load2FAStatus}
          {loadSecurityPreferences}
          {isLoadingMigrationStats}
          {migrationStats}
        />
      {:else if activeTab === 'account'}
        <AccountTab
          {auth}
          {emailForm}
          {passwordForm}
          {handleEmailSubmit}
          {handlePasswordSubmit}
          {tfaStatus}
          {isLoadingTfa}
          bind:showSetupDialog
          bind:showDisableDialog
          {handleRegenerateBackupCodes}
          {formatDate}
          {handleSettingsLogout}
        />
      {:else if activeTab === 'ai'}
        <AiTab
          {claudeApiKeyStatus}
          {isLoadingClaudeKeyStatus}
          {claudeKeyForm}
          {handleSaveClaudeApiKey}
          {handleDeleteClaudeApiKey}
          {geminiApiKeyStatus}
          {isLoadingGeminiKeyStatus}
          {geminiKeyForm}
          {handleSaveGeminiApiKey}
          {handleDeleteGeminiApiKey}
          {openAIApiKeyStatus}
          {isLoadingOpenAIKeyStatus}
          {openAIKeyForm}
          {handleSaveOpenAIApiKey}
          {handleDeleteOpenAIApiKey}
          {activeAIProvider}
          {isSavingAIProvider}
          {handleAIProviderChange}
          {dietaryPreference}
          {isSavingDietaryPreference}
          {handleDietaryPreferenceChange}
          {aiModels}
          {availableAIModels}
          {isLoadingAvailableAIModels}
          {isSavingAIModels}
          {handleSaveAIModels}
        />
      {:else if activeTab === 'data'}
        <div class="space-y-8">
          <!-- Export -->
          <div>
            <h3 class="text-lg font-medium text-foreground mb-2">
              {$_('page.settings.data.export_title')}
            </h3>
            <p class="text-sm text-muted-foreground mb-4">
              {$_('page.settings.data.export_description')}
            </p>
            <button onclick={handleExport} class="ui-button ui-button-primary">
              <Download size={16} />
              {$_('page.settings.data.export_button')}
            </button>
          </div>

          <!-- Import -->
          <div>
            <h3 class="text-lg font-medium text-foreground mb-2">
              {$_('page.settings.data.import_title')}
            </h3>
            <p class="text-sm text-muted-foreground mb-4">
              {$_('page.settings.data.import_description')}
            </p>
            <button
              onclick={handleImportClick}
              disabled={importing}
              class="ui-button ui-button-secondary"
            >
              {#if importing}
                <Loader2 size={16} class="animate-spin" />
                {$_('page.settings.data.importing')}
              {:else}
                <Upload size={16} />
                {$_('page.settings.data.import_button')}
              {/if}
            </button>
          </div>

          <!-- Info -->
          <div
            class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground"
          >
            <strong>{$_('common.note')}</strong>
            {$_('page.settings.data.info_note')}
          </div>
        </div>
      {:else if activeTab === 'connection' && isTauriApp}
        <div class="space-y-8">
          <!-- Server URL -->
          <div>
            <h3 class="text-lg font-medium text-foreground mb-4">Server-Verbindung</h3>
            <p class="text-sm text-muted-foreground mb-4">
              Verbinde dich mit xelanote.com oder deinem eigenen Server.
            </p>

            <form onsubmit={handleServerUrlSubmit} class="space-y-4">
              <div>
                <label for="server-url" class="block text-sm font-medium text-foreground mb-1">
                  Server URL
                </label>
                <input
                  id="server-url"
                  type="url"
                  bind:value={serverUrlForm.url}
                  disabled={serverUrlForm.isSaving}
                  class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                  placeholder="https://xelanote.com"
                />
              </div>

              {#if serverUrlForm.error}
                <div class="text-sm text-red-500">{serverUrlForm.error}</div>
              {/if}

              {#if serverUrlForm.showRestartWarning}
                <div
                  class="p-4 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
                >
                  <div class="flex items-start gap-3">
                    <AlertTriangle size={20} class="text-orange-700 dark:text-orange-400 mt-0.5" />
                    <div class="flex-1">
                      <div class="font-medium text-orange-950 dark:text-orange-200 mb-1">
                        Server URL geändert
                      </div>
                      <div class="text-sm text-orange-900 dark:text-orange-300">
                        Die neue Server-URL wurde gespeichert. Bitte melde dich ab und neu an, um
                        die Änderung anzuwenden.
                      </div>
                    </div>
                  </div>
                </div>
              {/if}

              <div class="flex gap-2">
                <button
                  type="submit"
                  disabled={serverUrlForm.isSaving}
                  class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
									font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {#if serverUrlForm.isSaving}
                    <Loader2 size={16} class="animate-spin" />
                    Verbinde...
                  {:else}
                    <Check size={16} />
                    Verbindung testen & speichern
                  {/if}
                </button>
                <button
                  type="button"
                  onclick={handleResetServerUrl}
                  disabled={serverUrlForm.isSaving}
                  class="flex items-center gap-2 px-4 py-2 rounded-lg border border-border
									bg-background text-foreground hover:bg-muted disabled:opacity-50 transition-colors"
                >
                  <RefreshCw size={16} />
                  Standard
                </button>
              </div>
            </form>
          </div>

          <!-- Self-hosted info -->
          <div
            class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground"
          >
            <strong>Hinweis:</strong> Du kannst xelanote selbst hosten. Weitere Informationen
            findest du in der
            <a
              href="https://github.com/xela-io/xelanote"
              target="_blank"
              rel="noopener noreferrer"
              class="underline hover:no-underline"
            >
              Dokumentation
            </a>.
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<!-- 2FA Setup Dialog -->
{#if showSetupDialog}
  <TwoFactorSetup onClose={() => (showSetupDialog = false)} onSuccess={handle2FASetupSuccess} />
{/if}

<!-- 2FA Disable Dialog -->
{#if showDisableDialog}
  <TwoFactorDisable
    onClose={() => (showDisableDialog = false)}
    onSuccess={handle2FADisableSuccess}
  />
{/if}

<!-- SEC-009: Password Prompt for Backup Code Regeneration -->
<BaseDialog
  open={showRegeneratePasswordPrompt}
  title={$_('page.settings.regenerate_backup_codes.password_prompt.title')}
  onClose={() => {
    showRegeneratePasswordPrompt = false;
    regeneratePassword = '';
  }}
  size="sm"
>
  {#snippet content()}
    <div class="space-y-4">
      <p class="text-sm text-muted-foreground">
        {$_('page.settings.regenerate_backup_codes.password_prompt.description')}
      </p>
      <div>
        <label for="regenerate-password" class="block text-sm font-medium mb-1.5">
          {$_('page.settings.password.label')}
        </label>
        <input
          id="regenerate-password"
          type="password"
          bind:value={regeneratePassword}
          placeholder={$_('page.settings.password.placeholder')}
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
          onkeydown={(e) => {
            if (e.key === 'Enter' && regeneratePassword) {
              confirmRegenerateBackupCodes();
            }
          }}
        />
      </div>
    </div>
  {/snippet}
  {#snippet footer()}
    <button
      type="button"
      onclick={() => {
        showRegeneratePasswordPrompt = false;
        regeneratePassword = '';
      }}
      class="btn btn-ghost"
    >
      {$_('common.cancel')}
    </button>
    <button
      type="button"
      onclick={confirmRegenerateBackupCodes}
      disabled={!regeneratePassword || isRegeneratingCodes}
      class="btn btn-primary"
    >
      {#if isRegeneratingCodes}
        <Loader2 class="h-4 w-4 animate-spin mr-2" />
      {/if}
      {$_('common.confirm')}
    </button>
  {/snippet}
</BaseDialog>

<!-- Hidden file input for markdown import -->
<input
  type="file"
  accept=".md"
  multiple
  webkitdirectory
  bind:this={importInput}
  onchange={handleImportFiles}
  style="display:none"
/>

<!-- Backup Codes Dialog -->
<BaseDialog
  open={showBackupCodesDialog && newBackupCodes !== null}
  title={$_('page.settings.backup_codes.title')}
  onClose={() => {
    showBackupCodesDialog = false;
    newBackupCodes = null;
  }}
  size="md"
  showCloseButton={false}
>
  {#snippet content()}
    {#if newBackupCodes}
      <BackupCodesDisplay
        codes={newBackupCodes}
        onConfirm={() => {
          showBackupCodesDialog = false;
          newBackupCodes = null;
        }}
      />
    {/if}
  {/snippet}
</BaseDialog>
