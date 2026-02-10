<script lang="ts">
  import {
    AlertTriangle,
    ArrowLeft,
    ArrowRight,
    Check,
    Database,
    Download,
    Edit3,
    Eye,
    EyeOff,
    Globe,
    Key,
    Loader2,
    Lock,
    Palette,
    RefreshCw,
    Shield,
    ShieldCheck,
    ShieldOff,
    Sparkles,
    Trash2,
    Unlock,
    Upload,
    User,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _, locale } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as api from '$lib/api';
  import { getExportUrl } from '$lib/api';
  import BackupCodesDisplay from '$lib/components/BackupCodesDisplay.svelte';
  import SecurityKeyManager from '$lib/components/SecurityKeyManager.svelte';
  import TwoFactorDisable from '$lib/components/TwoFactorDisable.svelte';
  import TwoFactorSetup from '$lib/components/TwoFactorSetup.svelte';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import WebAuthnDeviceManager from '$lib/components/WebAuthnDeviceManager.svelte';
  import { getDefaultServerUrl,getServerUrl, isTauri, setServerUrl } from '$lib/config';
  import { e2eEncryption } from '$lib/crypto/e2e';
  import type { WebAuthnCredential } from '$lib/crypto/webauthn';
  import {
    deleteApiKey,
    loadApiKeyStatus,
    saveApiKey,
    type ApiKeyFormState,
  } from '$lib/routes/settings/ai-keys';
  import {
    loadMigrationStats as loadMigrationStatsHelper,
    type MigrationStats,
  } from '$lib/routes/settings/migration-stats';
  import {
    confirmBackupCodesRegeneration,
    handleTwoFactorDisableSuccess,
    handleTwoFactorSetupSuccess,
    loadTwoFactorStatus,
    requestBackupCodesRegeneration,
  } from '$lib/routes/settings/two-factor';
  import {
    handlePasswordChange,
    type PasswordFormState,
  } from '$lib/routes/settings/password-change';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { type ThemeId,THEMES } from '$lib/themes';

  // Tab state (connection tab only visible in Tauri)
  let activeTab = $state<
    'appearance' | 'editor' | 'account' | 'security' | 'ai' | 'data' | 'connection'
  >('appearance');

  // Import/Export state
  let importInput: HTMLInputElement;
  let importing = $state(false);

  // Form states
  const emailForm = $state({
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
  type SecurityLevel = 'paranoid' | 'balanced' | 'convenient';
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
    features.loadJournalFeature();
    features.loadRecipeFeature();
  });

  const updateClaudeKeyForm = (next: Partial<ApiKeyFormState>) => {
    Object.assign(claudeKeyForm, next);
  };

  const updateGeminiKeyForm = (next: Partial<ApiKeyFormState>) => {
    Object.assign(geminiKeyForm, next);
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

  async function loadMigrationStats() {
    await loadMigrationStatsHelper({
      listNotes: (options) => api.listNotes(options),
      isEncrypted: (note) => note.content_encrypted && note.encryption_version === 1,
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

  async function handleEditorModeChange(mode: 'edit' | 'preview' | 'split') {
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

  async function handleEmailSubmit(e: Event) {
    e.preventDefault();
    emailForm.error = '';

    if (!emailForm.newEmail.trim()) {
      emailForm.error = $_('page.settings.account.email_required');
      return;
    }

    if (!emailForm.password) {
      emailForm.error = $_('page.settings.account.password_required');
      return;
    }

    const result = await settings.changeEmail(emailForm.newEmail.trim(), emailForm.password);

    if (result.success) {
      emailForm.newEmail = '';
      emailForm.password = '';
      // Reload user info
      window.location.reload();
    } else {
      emailForm.error = result.error || $_('page.settings.account.change_email_failed');
    }
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
    try {
      const prefs = await api.getPreferences();
      securityLevel = prefs.security_level as SecurityLevel;
      autoLockTimeout = prefs.auto_lock_timeout;
      webAuthnCredentials = prefs.webauthn_credentials || [];
    } catch (err) {
      console.error('Failed to load security preferences:', err);
    }
  }

  async function handleSecurityLevelChange(newLevel: SecurityLevel) {
    if (isSavingSecurityLevel) return;

    // Show confirmation modal
    let confirmMessage = '';
    if (newLevel === 'paranoid' && securityLevel !== 'paranoid') {
      confirmMessage = $_('page.settings.security.paranoid_confirm');
    } else if (newLevel !== 'paranoid' && securityLevel === 'paranoid') {
      confirmMessage = $_('page.settings.security.balanced_confirm');
    }

    if (confirmMessage) {
      const confirmed = await dialog.confirm({
        title: $_('dialog.confirm_title'),
        message: confirmMessage,
        confirmText: $_('common.confirm'),
        cancelText: $_('dialog.cancel'),
        variant: newLevel === 'paranoid' ? 'danger' : 'default',
      });

      if (!confirmed) return;
    }

    // Buffer old state for rollback
    const oldSecurityLevel = securityLevel;
    const oldAutoLockRunning = autoLockTimeout > 0;

    isSavingSecurityLevel = true;

    try {
      // Update frontend encryption store (handles IndexedDB clear/persist)
      await encryption.updateSecurityLevel(newLevel);

      // Save to backend
      const success = await settings.updateSecurityPreferences({ security_level: newLevel });

      if (!success) {
        throw new Error('Backend save failed');
      }

      // Update local state
      securityLevel = newLevel;

      // Handle auto-lock timer
      if (newLevel === 'paranoid') {
        // Stop timer (KEK not persisted), but keep autoLockTimeout value
        autoLock.stopAutoLock();
      } else if (oldAutoLockRunning) {
        // Restart timer with existing timeout
        autoLock.stopAutoLock();
        autoLock.initAutoLock(autoLockTimeout);
      }
    } catch (err) {
      // Rollback on failure
      console.error('Failed to change security level:', err);
      securityLevel = oldSecurityLevel;

      // Attempt to rollback encryption store
      try {
        await encryption.updateSecurityLevel(oldSecurityLevel);
      } catch (rollbackErr) {
        console.error('Rollback failed:', rollbackErr);
      }
    } finally {
      isSavingSecurityLevel = false;
    }
  }

  async function handleAutoLockTimeoutChange() {
    if (isSavingAutoLockTimeout) return;

    // Buffer old state for rollback
    const oldTimeout = autoLockTimeout;

    isSavingAutoLockTimeout = true;

    try {
      // Save to backend
      const success = await settings.updateSecurityPreferences({
        auto_lock_timeout: autoLockTimeout,
      });

      if (!success) {
        throw new Error('Backend save failed');
      }

      // Update auto-lock timer
      autoLock.stopAutoLock();
      if (autoLockTimeout > 0 && securityLevel !== 'paranoid') {
        autoLock.initAutoLock(autoLockTimeout);
      }
    } catch (err) {
      // Rollback on failure
      console.error('Failed to change auto-lock timeout:', err);
      autoLockTimeout = oldTimeout;
    } finally {
      isSavingAutoLockTimeout = false;
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
    window.open(getExportUrl(), '_blank');
  }

  function handleImportClick() {
    importInput.click();
  }

  async function handleImportFiles(e: Event) {
    const input = e.target as HTMLInputElement;
    const files = Array.from(input.files || []);
    const mdFiles = files.filter((f) => f.name.endsWith('.md'));

    if (mdFiles.length === 0) {
      await dialog.alert({
        title: $_('common.note'),
        message: $_('page.settings.data.no_md_files_selected'),
        variant: 'warning',
      });
      return;
    }

    importing = true;

    try {
      const importFiles = await Promise.all(
        mdFiles.map(async (file) => ({
          path: (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name,
          filename: file.name,
          content: await file.text(),
        }))
      );

      const result = await api.importMarkdown(importFiles, true);

      let message = $_('page.settings.data.import_completed') + '\n\n';
      message +=
        $_('page.settings.data.notes_imported', { values: { count: result.imported } }) + '\n';
      message +=
        $_('page.settings.data.folders_created', { values: { count: result.folders_created } }) +
        '\n';

      if (result.skipped > 0) {
        message +=
          $_('page.settings.data.skipped_notes', { values: { count: result.skipped } }) + '\n';
      }

      if (result.failed > 0) {
        message +=
          $_('page.settings.data.failed_notes', { values: { count: result.failed } }) + '\n';
        if (result.errors) {
          message += `\n${$_('page.settings.data.errors')}:\n${result.errors.slice(0, 5).join('\n')}`;
        }
      }

      await dialog.alert({
        title: $_('page.settings.data.import_completed'),
        message,
        variant: result.failed > 0 ? 'warning' : 'default',
      });
    } catch (err: unknown) {
      console.error('Import failed:', err);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('page.settings.data.import_failed', {
          values: { error: err instanceof Error ? err.message : String(err) },
        }),
        variant: 'danger',
      });
    } finally {
      importing = false;
      input.value = '';
    }
  }

  function handleResetServerUrl() {
    serverUrlForm.url = getDefaultServerUrl();
    serverUrlForm.error = '';
    serverUrlForm.showRestartWarning = false;
  }
</script>

<div
  class="h-full bg-background overflow-y-auto overflow-x-hidden"
  style="scrollbar-width: none !important; scrollbar-gutter: stable;"
>
  <div class="max-w-3xl mx-auto p-4 md:p-8">
    <!-- Header -->
    <div class="flex items-center gap-4 mb-8">
      <button
        onclick={() => goto('/')}
        class="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
        title={$_('page.settings.back')}
      >
        <ArrowLeft size={20} />
      </button>
      <h1 class="text-2xl font-bold text-foreground">{$_('page.settings.title')}</h1>
    </div>

    <!-- Tabs: Option A - Icon-only on mobile, with text on desktop -->
    <div class="flex gap-2 md:gap-1 mb-8 border-b border-border overflow-x-auto scrollbar-none">
      <button
        onclick={() => (activeTab = 'appearance')}
        title={$_('page.settings.tabs.appearance')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'appearance'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <Palette size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.appearance')}</span>
      </button>
      <button
        onclick={() => (activeTab = 'editor')}
        title={$_('page.settings.tabs.editor')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'editor'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <Edit3 size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.editor')}</span>
      </button>
      <button
        onclick={() => (activeTab = 'security')}
        title={$_('page.settings.tabs.security')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'security'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <Shield size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.security')}</span>
      </button>
      <button
        onclick={() => (activeTab = 'account')}
        title={$_('page.settings.tabs.account')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'account'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <User size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.account')}</span>
      </button>
      <button
        onclick={() => (activeTab = 'ai')}
        title={$_('page.settings.tabs.ai')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'ai'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <Sparkles size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.ai')}</span>
      </button>
      <button
        onclick={() => (activeTab = 'data')}
        title={$_('page.settings.tabs.data')}
        class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
					{activeTab === 'data'
          ? 'border-primary text-primary'
          : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        <Database size={18} />
        <span class="hidden sm:inline">{$_('page.settings.tabs.data')}</span>
      </button>
      {#if isTauriApp}
        <button
          onclick={() => (activeTab = 'connection')}
          title="Connection"
          class="flex items-center gap-2 px-3 md:px-4 py-3 text-sm font-medium transition-colors border-b-2 -mb-px flex-shrink-0 whitespace-nowrap
						{activeTab === 'connection'
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          <Globe size={18} />
          <span class="hidden sm:inline">Connection</span>
        </button>
      {/if}
    </div>

    <!-- Tab Content -->
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
          <h3 class="text-sm font-medium text-muted-foreground mb-4">Performance (Experimental)</h3>
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
      <div class="space-y-8">
        <!-- Encryption Locked Warning -->
        {#if !encryption.isEncryptionUnlocked()}
          <div
            class="p-4 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
          >
            <div class="flex items-start gap-3">
              <AlertTriangle size={20} class="text-orange-700 dark:text-orange-400 mt-0.5" />
              <div class="flex-1">
                <div class="font-medium text-orange-950 dark:text-orange-200 mb-1">
                  {$_('page.settings.security.encryption_locked_title')}
                </div>
                <div class="text-sm text-orange-900 dark:text-orange-300">
                  {$_('page.settings.security.encryption_locked_description')}
                </div>
              </div>
            </div>
          </div>
        {/if}

        <!-- Security Level -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-4">
            {$_('page.settings.security.security_level_title')}
          </h3>
          <div class="space-y-3">
            <!-- Paranoid -->
            <button
              onclick={() => handleSecurityLevelChange('paranoid')}
              disabled={!encryption.isEncryptionUnlocked() || isSavingSecurityLevel}
              class="w-full flex items-start gap-4 p-4 rounded-lg border-2 transition-all text-left
								{securityLevel === 'paranoid'
                ? 'border-success bg-success/10'
                : 'border-border hover:border-success/50 bg-card'}
								disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div
                class="mt-1 {securityLevel === 'paranoid'
                  ? 'text-success'
                  : 'text-muted-foreground'}"
              >
                <ShieldCheck size={24} />
              </div>
              <div class="flex-1">
                <div class="font-medium text-foreground text-base mb-1">
                  {$_('page.settings.security.paranoid_title')}
                </div>
                <div class="text-sm text-muted-foreground mb-2">
                  {$_('page.settings.security.paranoid_subtitle')}
                </div>
                <div class="text-xs text-muted-foreground space-y-1">
                  <div>{$_('page.settings.security.paranoid_no_auto_unlock')}</div>
                  <div>{$_('page.settings.security.paranoid_key_in_ram')}</div>
                  <div>{$_('page.settings.security.paranoid_password_on_refresh')}</div>
                </div>
              </div>
            </button>

            <!-- Balanced -->
            <button
              onclick={() => handleSecurityLevelChange('balanced')}
              disabled={!encryption.isEncryptionUnlocked() || isSavingSecurityLevel}
              class="w-full flex items-start gap-4 p-4 rounded-lg border-2 transition-all text-left
								{securityLevel === 'balanced'
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-primary/50 bg-card'}
								disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div
                class="mt-1 {securityLevel === 'balanced'
                  ? 'text-primary'
                  : 'text-muted-foreground'}"
              >
                <Shield size={24} />
              </div>
              <div class="flex-1">
                <div class="flex items-center gap-2 mb-1">
                  <div class="font-medium text-foreground text-base">
                    {$_('page.settings.security.balanced_title')}
                  </div>
                  <span class="text-xs bg-primary/20 text-primary px-2 py-0.5 rounded">
                    {$_('page.settings.security.balanced_tag')}
                  </span>
                </div>
                <div class="text-sm text-muted-foreground mb-2">
                  {$_('page.settings.security.balanced_subtitle')}
                </div>
                <div class="text-xs text-muted-foreground space-y-1">
                  <div>{$_('page.settings.security.balanced_auto_unlock')}</div>
                  <div>{$_('page.settings.security.balanced_auto_lock')}</div>
                  <div>{$_('page.settings.security.balanced_browser_encrypted')}</div>
                </div>
              </div>
            </button>

            <!-- Convenient -->
            <button
              onclick={() => handleSecurityLevelChange('convenient')}
              disabled={!encryption.isEncryptionUnlocked() || isSavingSecurityLevel}
              class="w-full flex items-start gap-4 p-4 rounded-lg border-2 transition-all text-left
								{securityLevel === 'convenient'
                ? 'border-orange-500 bg-orange-500/10'
                : 'border-border hover:border-orange-500/50 bg-card'}
								disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div
                class="mt-1 {securityLevel === 'convenient'
                  ? 'text-orange-500'
                  : 'text-muted-foreground'}"
              >
                <ShieldOff size={24} />
              </div>
              <div class="flex-1">
                <div class="font-medium text-foreground text-base mb-1">
                  {$_('page.settings.security.convenient_title')}
                </div>
                <div class="text-sm text-muted-foreground mb-2">
                  {$_('page.settings.security.convenient_subtitle')}
                </div>
                <div class="text-xs text-muted-foreground space-y-1">
                  <div>{$_('page.settings.security.convenient_webauthn')}</div>
                  <div>{$_('page.settings.security.convenient_persistent_kek')}</div>
                  <div>{$_('page.settings.security.convenient_optional_autolock')}</div>
                </div>
              </div>
            </button>
          </div>

          <!-- Info/Warning boxes -->
          {#if securityLevel === 'balanced'}
            <div class="mt-4 p-3 rounded-lg bg-primary/10 border border-primary/30">
              <div class="flex items-start gap-2">
                <div class="text-primary mt-0.5">ℹ️</div>
                <div class="text-sm text-foreground">
                  {$_('page.settings.security.balanced_info')}
                </div>
              </div>
            </div>
          {:else if securityLevel === 'convenient'}
            <div
              class="mt-4 p-3 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
            >
              <div class="flex items-start gap-2">
                <AlertTriangle size={16} class="text-orange-700 dark:text-orange-400 mt-0.5" />
                <div class="text-sm text-orange-900 dark:text-orange-300">
                  <strong>{$_('common.note')}</strong>
                  {$_('page.settings.security.convenient_warning')}
                </div>
              </div>
            </div>
          {/if}
        </div>

        <!-- Auto-Lock Timeout -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-4">
            {$_('page.settings.security.autolock_timeout_title')}
          </h3>
          <div class="space-y-4">
            <div>
              <label for="auto-lock-timeout" class="block text-sm font-medium text-foreground mb-2">
                {$_('page.settings.security.autolock_timeout_label')}
              </label>
              <select
                id="auto-lock-timeout"
                bind:value={autoLockTimeout}
                onchange={handleAutoLockTimeoutChange}
                disabled={securityLevel === 'paranoid' ||
                  !encryption.isEncryptionUnlocked() ||
                  isSavingAutoLockTimeout}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <option value={0}>{$_('page.settings.security.autolock_never')}</option>
                <option value={5}>{$_('page.settings.security.autolock_5_min')}</option>
                <option value={15}>{$_('page.settings.security.autolock_15_min')}</option>
                <option value={30}>{$_('page.settings.security.autolock_30_min')}</option>
                <option value={60}>{$_('page.settings.security.autolock_60_min')}</option>
              </select>
            </div>

            {#if securityLevel === 'paranoid'}
              <div class="text-sm text-muted-foreground">
                {$_('page.settings.security.autolock_paranoid_info')}
              </div>
            {/if}
          </div>
        </div>

        <!-- Security Keys (FIDO2 2FA) -->
        <div class="p-6 rounded-lg border border-border bg-card">
          <h3 class="text-lg font-medium text-foreground mb-1">Security Keys</h3>
          <p class="text-sm text-muted-foreground mb-4">
            Hardware Security Keys (YubiKey etc.) als zweiten Faktor beim Login verwenden.
          </p>
          <SecurityKeyManager onUpdate={load2FAStatus} />
        </div>

        <!-- Biometric Devices -->
        <div class="p-6 rounded-lg border border-border bg-card">
          <WebAuthnDeviceManager
            credentials={webAuthnCredentials}
            onUpdate={loadSecurityPreferences}
          />
        </div>

        <!-- Note Encryption Migration -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-4">
            {$_('page.settings.security.encryption_migration_title')}
          </h3>

          {#if isLoadingMigrationStats}
            <div class="p-4 rounded-lg border border-border bg-card flex items-center gap-3">
              <Loader2 size={20} class="animate-spin text-muted-foreground" />
              <span class="text-muted-foreground">{$_('common.loading')}</span>
            </div>
          {:else if migrationStats}
            <div class="p-4 rounded-lg border border-border bg-card space-y-4">
              <!-- Statistics Grid -->
              <div class="grid grid-cols-3 gap-3">
                <div class="text-center p-3 rounded-lg bg-muted/50">
                  <div class="text-2xl font-bold text-foreground">{migrationStats.total}</div>
                  <div class="text-xs text-muted-foreground">
                    {$_('page.settings.security.migration_total')}
                  </div>
                </div>
                <div class="text-center p-3 rounded-lg bg-success/10">
                  <div class="flex items-center justify-center gap-1">
                    <Lock size={16} class="text-success" />
                    <span class="text-2xl font-bold text-success">{migrationStats.encrypted}</span>
                  </div>
                  <div class="text-xs text-muted-foreground">
                    {$_('page.settings.security.migration_encrypted')}
                  </div>
                </div>
                <div
                  class="text-center p-3 rounded-lg {migrationStats.plaintext > 0
                    ? 'bg-orange-500/10'
                    : 'bg-muted/50'}"
                >
                  <div class="flex items-center justify-center gap-1">
                    <Unlock
                      size={16}
                      class={migrationStats.plaintext > 0
                        ? 'text-orange-500'
                        : 'text-muted-foreground'}
                    />
                    <span
                      class="text-2xl font-bold {migrationStats.plaintext > 0
                        ? 'text-orange-600 dark:text-orange-400'
                        : 'text-muted-foreground'}">{migrationStats.plaintext}</span
                    >
                  </div>
                  <div class="text-xs text-muted-foreground">
                    {$_('page.settings.security.migration_plaintext')}
                  </div>
                </div>
              </div>

              <!-- Status and Action -->
              {#if migrationStats.plaintext > 0}
                <div
                  class="p-3 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-300 dark:border-orange-800"
                >
                  <div class="flex items-start gap-2 mb-3">
                    <AlertTriangle
                      size={16}
                      class="text-orange-700 dark:text-orange-400 mt-0.5 flex-shrink-0"
                    />
                    <div class="text-sm text-orange-900 dark:text-orange-300">
                      {$_('page.settings.security.migration_warning', {
                        values: { count: migrationStats.plaintext },
                      })}
                    </div>
                  </div>
                  <button
                    onclick={() => goto('/settings/migration')}
                    disabled={!encryption.isEncryptionUnlocked()}
                    class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
											font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <RefreshCw size={16} />
                    {$_('page.settings.security.migration_button')}
                    <ArrowRight size={16} />
                  </button>
                </div>
              {:else if migrationStats.total > 0}
                <div class="p-3 rounded-lg bg-success/15 border border-success/30">
                  <div class="flex items-center gap-2">
                    <Check size={16} class="text-success" />
                    <span class="text-sm text-success">
                      {$_('page.settings.security.migration_complete')}
                    </span>
                  </div>
                </div>
              {:else}
                <div class="text-sm text-muted-foreground">
                  {$_('page.settings.security.migration_no_notes')}
                </div>
              {/if}
            </div>
          {:else}
            <div class="p-4 rounded-lg border border-border bg-card text-sm text-muted-foreground">
              {$_('page.settings.security.migration_error')}
            </div>
          {/if}
        </div>
      </div>
    {:else if activeTab === 'account'}
      <div class="space-y-8">
        <!-- Current User Info -->
        {#if auth.getCurrentUser()}
          <div class="p-4 rounded-lg bg-muted/50 border border-border">
            <div class="text-sm text-muted-foreground">
              {$_('page.settings.account.logged_in_as')}
            </div>
            <div class="font-medium text-foreground">{auth.getCurrentUser()?.username}</div>
            <div class="text-sm text-muted-foreground">{auth.getCurrentUser()?.email}</div>
          </div>
        {/if}

        <!-- Two-Factor Authentication -->
        <div class="p-4 rounded-lg border border-border">
          <div class="flex items-start gap-4">
            <div class="p-2 rounded-lg bg-primary/10">
              <Shield size={24} class="text-primary" />
            </div>
            <div class="flex-1 space-y-4">
              <div>
                <h3 class="text-lg font-medium text-foreground">
                  {$_('page.settings.account.twofa_title')}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {$_('page.settings.account.twofa_description')}
                </p>
              </div>

              {#if isLoadingTfa}
                <div class="flex items-center gap-2 text-muted-foreground">
                  <Loader2 size={16} class="animate-spin" />
                  <span class="text-sm">{$_('common.loading')}</span>
                </div>
              {:else if tfaStatus?.enabled}
                <!-- 2FA Enabled -->
                <div class="space-y-4">
                  <div class="flex items-center gap-2">
                    <ShieldCheck size={18} class="text-success" />
                    <span class="text-sm font-medium text-success">
                      {$_('page.settings.account.twofa_enabled_since', {
                        values: { date: formatDate(tfaStatus.verified_at) },
                      })}
                    </span>
                  </div>

                  <div class="text-sm text-muted-foreground">
                    {$_('page.settings.account.twofa_backup_codes_remaining')}<span
                      class="font-medium text-foreground">{tfaStatus.unused_backup_codes}/10</span
                    >
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <button
                      onclick={handleRegenerateBackupCodes}
                      disabled={isRegeneratingCodes}
                      class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-border
												bg-background text-foreground hover:bg-muted disabled:opacity-50 transition-colors"
                    >
                      {#if isRegeneratingCodes}
                        <Loader2 size={14} class="animate-spin" />
                      {:else}
                        <RefreshCw size={14} />
                      {/if}
                      {$_('page.settings.account.new_backup_codes')}
                    </button>
                    <button
                      onclick={() => (showDisableDialog = true)}
                      class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg
												bg-red-500/10 text-red-600 dark:text-red-400 hover:bg-red-500/20 transition-colors"
                    >
                      <ShieldOff size={14} />
                      {$_('page.settings.account.disable_twofa')}
                    </button>
                  </div>
                </div>
              {:else}
                <!-- 2FA Not Enabled -->
                <div class="space-y-4">
                  <div class="flex items-center gap-2">
                    <ShieldOff size={18} class="text-muted-foreground" />
                    <span class="text-sm text-muted-foreground"
                      >{$_('page.settings.account.twofa_not_enabled')}</span
                    >
                  </div>

                  <button
                    onclick={() => (showSetupDialog = true)}
                    class="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
											font-medium hover:bg-primary/90 transition-colors"
                  >
                    <Shield size={16} />
                    {$_('page.settings.account.setup_twofa')}
                  </button>
                </div>
              {/if}
            </div>
          </div>
        </div>

        <!-- Change Email -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-4">
            {$_('page.settings.account.change_email_title')}
          </h3>
          <form onsubmit={handleEmailSubmit} class="space-y-4">
            <div>
              <label for="new-email" class="block text-sm font-medium text-foreground mb-1">
                {$_('page.settings.account.new_email_label')}
              </label>
              <input
                id="new-email"
                type="email"
                bind:value={emailForm.newEmail}
                disabled={settings.getIsChangingEmail()}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                placeholder={$_('page.settings.account.new_email_placeholder')}
              />
            </div>
            <div>
              <label for="email-password" class="block text-sm font-medium text-foreground mb-1">
                {$_('page.settings.account.current_password_label')}
              </label>
              <input
                id="email-password"
                type="password"
                bind:value={emailForm.password}
                disabled={settings.getIsChangingEmail()}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                placeholder={$_('page.settings.account.current_password_placeholder')}
              />
            </div>

            {#if emailForm.error}
              <div class="text-sm text-red-500">{emailForm.error}</div>
            {/if}

            <button
              type="submit"
              disabled={settings.getIsChangingEmail()}
              class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
								font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {#if settings.getIsChangingEmail()}
                <Loader2 size={16} class="animate-spin" />
                {$_('common.saving')}
              {:else}
                {$_('page.settings.account.change_email_button')}
              {/if}
            </button>
          </form>
        </div>

        <!-- Change Password -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-4">
            {$_('page.settings.account.change_password_title')}
          </h3>
          <form onsubmit={handlePasswordSubmit} class="space-y-4">
            <div>
              <label for="current-password" class="block text-sm font-medium text-foreground mb-1">
                {$_('page.settings.account.current_password_label')}
              </label>
              <input
                id="current-password"
                type="password"
                bind:value={passwordForm.currentPassword}
                disabled={passwordForm.isChanging}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
              />
            </div>
            <div>
              <label for="new-password" class="block text-sm font-medium text-foreground mb-1">
                {$_('page.settings.account.new_password_label')}
              </label>
              <input
                id="new-password"
                type="password"
                bind:value={passwordForm.newPassword}
                disabled={passwordForm.isChanging}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                placeholder={$_('page.settings.account.new_password_placeholder')}
              />
            </div>
            <div>
              <label for="confirm-password" class="block text-sm font-medium text-foreground mb-1">
                {$_('page.settings.account.confirm_password_label')}
              </label>
              <input
                id="confirm-password"
                type="password"
                bind:value={passwordForm.confirmPassword}
                disabled={passwordForm.isChanging}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
              />
            </div>

            {#if passwordForm.error}
              <div class="text-sm text-red-500">{passwordForm.error}</div>
            {/if}

            {#if passwordForm.reWrappingProgress}
              <div class="text-sm text-muted-foreground flex items-center gap-2">
                <Loader2 size={16} class="animate-spin" />
                {passwordForm.reWrappingProgress}
              </div>
            {/if}

            <button
              type="submit"
              disabled={passwordForm.isChanging}
              class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
								font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {#if passwordForm.isChanging}
                <Loader2 size={16} class="animate-spin" />
                {passwordForm.reWrappingProgress || $_('common.saving')}
              {:else}
                {$_('page.settings.account.change_password_button')}
              {/if}
            </button>
          </form>
        </div>

        <!-- Info about session invalidation -->
        <div class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground">
          <strong>{$_('common.note')}</strong>
          {$_('page.settings.account.session_invalidation_info')}
        </div>
      </div>
    {:else if activeTab === 'ai'}
      <div class="space-y-8">
        <!-- Claude API Key (BYOK) -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-2">
            {$_('page.settings.ai.claude_title')}
          </h3>
          <p class="text-sm text-muted-foreground mb-4">
            {$_('page.settings.ai.claude_description')}
          </p>

          {#if isLoadingClaudeKeyStatus}
            <div class="p-4 rounded-lg border border-border bg-card flex items-center gap-3">
              <Loader2 size={20} class="animate-spin text-muted-foreground" />
              <span class="text-muted-foreground">{$_('common.loading')}</span>
            </div>
          {:else if claudeApiKeyStatus?.has_key}
            <!-- API Key is configured -->
            <div class="p-4 rounded-lg border border-success/30 bg-success/10">
              <div class="flex items-start gap-3">
                <Key size={20} class="text-success mt-0.5" />
                <div class="flex-1">
                  <div class="font-medium text-success mb-1">
                    {$_('page.settings.ai.api_key_configured')}
                  </div>
                  <div class="text-sm text-success font-mono">
                    {claudeApiKeyStatus.masked_key}
                  </div>
                  {#if claudeApiKeyStatus.updated_at}
                    <div class="text-xs text-success mt-2">
                      {$_('page.settings.ai.api_key_updated_at', {
                        values: {
                          date: new Date(claudeApiKeyStatus.updated_at).toLocaleDateString(),
                        },
                      })}
                    </div>
                  {/if}
                </div>
                <button
                  onclick={handleDeleteClaudeApiKey}
                  disabled={claudeKeyForm.isDeleting}
                  class="p-2 rounded-lg text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors disabled:opacity-50"
                  title={$_('page.settings.ai.delete_api_key')}
                >
                  {#if claudeKeyForm.isDeleting}
                    <Loader2 size={18} class="animate-spin" />
                  {:else}
                    <Trash2 size={18} />
                  {/if}
                </button>
              </div>
            </div>

            <!-- Option to update the key -->
            <details class="mt-4">
              <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
                {$_('page.settings.ai.update_api_key')}
              </summary>
              <form onsubmit={handleSaveClaudeApiKey} class="mt-4 space-y-4">
                <div>
                  <label
                    for="claude-api-key-update"
                    class="block text-sm font-medium text-foreground mb-1"
                  >
                    {$_('page.settings.ai.new_api_key_label')}
                  </label>
                  <div class="relative">
                    <input
                      id="claude-api-key-update"
                      type={claudeKeyForm.showKey ? 'text' : 'password'}
                      bind:value={claudeKeyForm.apiKey}
                      disabled={claudeKeyForm.isSaving}
                      class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
												focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                      placeholder="sk-ant-api03-..."
                    />
                    <button
                      type="button"
                      onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
                      class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
                    >
                      {#if claudeKeyForm.showKey}
                        <EyeOff size={18} />
                      {:else}
                        <Eye size={18} />
                      {/if}
                    </button>
                  </div>
                </div>

                {#if claudeKeyForm.error}
                  <div class="text-sm text-red-500">{claudeKeyForm.error}</div>
                {/if}

                <button
                  type="submit"
                  disabled={claudeKeyForm.isSaving}
                  class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
										font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {#if claudeKeyForm.isSaving}
                    <Loader2 size={16} class="animate-spin" />
                    {$_('common.saving')}
                  {:else}
                    {$_('page.settings.ai.update_api_key_button')}
                  {/if}
                </button>
              </form>
            </details>
          {:else}
            <!-- No API Key configured -->
            <form onsubmit={handleSaveClaudeApiKey} class="space-y-4">
              <div>
                <label for="claude-api-key" class="block text-sm font-medium text-foreground mb-1">
                  {$_('page.settings.ai.api_key_label')}
                </label>
                <div class="relative">
                  <input
                    id="claude-api-key"
                    type={claudeKeyForm.showKey ? 'text' : 'password'}
                    bind:value={claudeKeyForm.apiKey}
                    disabled={claudeKeyForm.isSaving}
                    class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
											focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                    placeholder="sk-ant-api03-..."
                  />
                  <button
                    type="button"
                    onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
                  >
                    {#if claudeKeyForm.showKey}
                      <EyeOff size={18} />
                    {:else}
                      <Eye size={18} />
                    {/if}
                  </button>
                </div>
                <p class="text-xs text-muted-foreground mt-1">
                  {$_('page.settings.ai.claude_key_hint')}
                </p>
              </div>

              {#if claudeKeyForm.error}
                <div class="text-sm text-red-500">{claudeKeyForm.error}</div>
              {/if}

              <button
                type="submit"
                disabled={claudeKeyForm.isSaving}
                class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
									font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {#if claudeKeyForm.isSaving}
                  <Loader2 size={16} class="animate-spin" />
                  {$_('common.saving')}
                {:else}
                  <Key size={16} />
                  {$_('page.settings.ai.save_api_key_button')}
                {/if}
              </button>
            </form>
          {/if}
        </div>

        <!-- Gemini API Key (BYOK) -->
        <div>
          <h3 class="text-lg font-medium text-foreground mb-2">
            {$_('page.settings.ai.gemini_title')}
          </h3>
          <p class="text-sm text-muted-foreground mb-4">
            {$_('page.settings.ai.gemini_description')}
          </p>

          {#if isLoadingGeminiKeyStatus}
            <div class="p-4 rounded-lg border border-border bg-card flex items-center gap-3">
              <Loader2 size={20} class="animate-spin text-muted-foreground" />
              <span class="text-muted-foreground">{$_('common.loading')}</span>
            </div>
          {:else if geminiApiKeyStatus?.has_key}
            <!-- API Key is configured -->
            <div class="p-4 rounded-lg border border-success/30 bg-success/10">
              <div class="flex items-start gap-3">
                <Key size={20} class="text-success mt-0.5" />
                <div class="flex-1">
                  <div class="font-medium text-success mb-1">
                    {$_('page.settings.ai.api_key_configured')}
                  </div>
                  <div class="text-sm text-success font-mono">
                    {geminiApiKeyStatus.masked_key}
                  </div>
                  {#if geminiApiKeyStatus.updated_at}
                    <div class="text-xs text-success mt-2">
                      {$_('page.settings.ai.api_key_updated_at', {
                        values: {
                          date: new Date(geminiApiKeyStatus.updated_at).toLocaleDateString(),
                        },
                      })}
                    </div>
                  {/if}
                </div>
                <button
                  onclick={handleDeleteGeminiApiKey}
                  disabled={geminiKeyForm.isDeleting}
                  class="p-2 rounded-lg text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors disabled:opacity-50"
                  title={$_('page.settings.ai.delete_api_key')}
                >
                  {#if geminiKeyForm.isDeleting}
                    <Loader2 size={18} class="animate-spin" />
                  {:else}
                    <Trash2 size={18} />
                  {/if}
                </button>
              </div>
            </div>

            <!-- Option to update the key -->
            <details class="mt-4">
              <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
                {$_('page.settings.ai.update_api_key')}
              </summary>
              <form onsubmit={handleSaveGeminiApiKey} class="mt-4 space-y-4">
                <div>
                  <label
                    for="gemini-api-key-update"
                    class="block text-sm font-medium text-foreground mb-1"
                  >
                    {$_('page.settings.ai.new_api_key_label')}
                  </label>
                  <div class="relative">
                    <input
                      id="gemini-api-key-update"
                      type={geminiKeyForm.showKey ? 'text' : 'password'}
                      bind:value={geminiKeyForm.apiKey}
                      disabled={geminiKeyForm.isSaving}
                      class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
												focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                      placeholder="AIzaSy..."
                    />
                    <button
                      type="button"
                      onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
                      class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
                    >
                      {#if geminiKeyForm.showKey}
                        <EyeOff size={18} />
                      {:else}
                        <Eye size={18} />
                      {/if}
                    </button>
                  </div>
                </div>

                {#if geminiKeyForm.error}
                  <div class="text-sm text-red-500">{geminiKeyForm.error}</div>
                {/if}

                <button
                  type="submit"
                  disabled={geminiKeyForm.isSaving}
                  class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
										font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {#if geminiKeyForm.isSaving}
                    <Loader2 size={16} class="animate-spin" />
                    {$_('common.saving')}
                  {:else}
                    {$_('page.settings.ai.update_api_key_button')}
                  {/if}
                </button>
              </form>
            </details>
          {:else}
            <!-- No API Key configured -->
            <form onsubmit={handleSaveGeminiApiKey} class="space-y-4">
              <div>
                <label for="gemini-api-key" class="block text-sm font-medium text-foreground mb-1">
                  {$_('page.settings.ai.api_key_label')}
                </label>
                <div class="relative">
                  <input
                    id="gemini-api-key"
                    type={geminiKeyForm.showKey ? 'text' : 'password'}
                    bind:value={geminiKeyForm.apiKey}
                    disabled={geminiKeyForm.isSaving}
                    class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
											focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                    placeholder="AIzaSy..."
                  />
                  <button
                    type="button"
                    onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
                  >
                    {#if geminiKeyForm.showKey}
                      <EyeOff size={18} />
                    {:else}
                      <Eye size={18} />
                    {/if}
                  </button>
                </div>
                <p class="text-xs text-muted-foreground mt-1">
                  {$_('page.settings.ai.gemini_key_hint')}
                </p>
              </div>

              {#if geminiKeyForm.error}
                <div class="text-sm text-red-500">{geminiKeyForm.error}</div>
              {/if}

              <button
                type="submit"
                disabled={geminiKeyForm.isSaving}
                class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
									font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {#if geminiKeyForm.isSaving}
                  <Loader2 size={16} class="animate-spin" />
                  {$_('common.saving')}
                {:else}
                  <Key size={16} />
                  {$_('page.settings.ai.save_api_key_button')}
                {/if}
              </button>
            </form>
          {/if}
        </div>

        <!-- AI Features Info -->
        <div class="p-4 rounded-lg bg-primary/10 border border-primary/30">
          <div class="flex items-start gap-3">
            <Sparkles size={20} class="text-primary mt-0.5" />
            <div class="flex-1 text-sm text-foreground">
              <div class="font-medium mb-2">{$_('page.settings.ai.features_title')}</div>
              <ul class="list-disc list-inside space-y-1">
                <li>{$_('page.settings.ai.feature_summaries')}</li>
                <li>{$_('page.settings.ai.feature_tagging')}</li>
                <li>{$_('page.settings.ai.feature_linking')}</li>
              </ul>
            </div>
          </div>
        </div>

        <!-- Privacy Notice -->
        <div
          class="p-4 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
        >
          <div class="flex items-start gap-3">
            <AlertTriangle size={20} class="text-orange-700 dark:text-orange-400 mt-0.5" />
            <div class="flex-1">
              <div class="font-medium text-orange-950 dark:text-orange-200 mb-1">
                {$_('page.settings.ai.privacy_title')}
              </div>
              <div class="text-sm text-orange-900 dark:text-orange-300">
                {$_('page.settings.ai.privacy_notice')}
              </div>
            </div>
          </div>
        </div>
      </div>
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
          <button
            onclick={handleExport}
            class="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors"
          >
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
            class="flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-background text-foreground font-medium hover:bg-muted disabled:opacity-50 transition-colors"
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
        <div class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground">
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
                      Die neue Server-URL wurde gespeichert. Bitte melde dich ab und neu an, um die
                      Änderung anzuwenden.
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
        <div class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground">
          <strong>Hinweis:</strong> Du kannst xelanote selbst hosten. Weitere Informationen findest
          du in der
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
  const updatePasswordForm = (next: Partial<PasswordFormState>) => {
    Object.assign(passwordForm, next);
  };
