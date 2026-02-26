<script lang="ts">
  import {
    ArrowLeft,
    Database,
    Edit3,
    Globe,
    Loader2,
    Palette,
    Shield,
    Sparkles,
    User,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as api from '$lib/api';
  import BackupCodesDisplay from '$lib/components/BackupCodesDisplay.svelte';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import TwoFactorDisable from '$lib/components/TwoFactorDisable.svelte';
  import TwoFactorSetup from '$lib/components/TwoFactorSetup.svelte';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import { isTauri } from '$lib/config';
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
  import AppearanceTab from '$lib/routes/settings/tabs/AppearanceTab.svelte';
  import ConnectionTab from '$lib/routes/settings/tabs/ConnectionTab.svelte';
  import DataTab from '$lib/routes/settings/tabs/DataTab.svelte';
  import EditorTab from '$lib/routes/settings/tabs/EditorTab.svelte';
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
  import * as settings from '$lib/stores/settings.svelte';

  // Tab state (connection tab only visible in Tauri)
  let activeTab = $state<
    'appearance' | 'editor' | 'account' | 'security' | 'ai' | 'data' | 'connection'
  >('appearance');

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

  onMount(() => {
    // Detect Tauri at mount time (preload script should be done)
    isTauriApp = isTauri();

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
</script>

<div class="ui-page-shell min-h-0 overflow-hidden">
  <PageHeader
    title={$_('page.settings.title')}
    subtitle={$_('page.settings.subtitle')}
    mobileHeaderMode="topbar"
    mobileSingleRow={true}
    mobileHideSubtitle={true}
    mobileSticky={false}
    class="mb-4 px-3 py-2 sm:px-6 sm:py-4"
    containerClass="mx-auto max-w-3xl"
    subtitleClass="hidden sm:block"
  >
    {#snippet leading()}
      <div class="ui-mobile-topbar-leading">
        <MobileSidebarInlineToggle />
        <button
          onclick={() => goto('/')}
          class="ui-icon-button ui-icon-button-md ui-mobile-topbar-icon"
          title={$_('page.settings.back')}
        >
          <ArrowLeft size={20} />
        </button>
      </div>
    {/snippet}
  </PageHeader>

  <div class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden scrollbar-none overscroll-contain">
    <div class="max-w-3xl mx-auto w-full px-4 pb-24 sm:px-6 sm:pb-8">
      <!-- Tabs: Option A - Icon-only on mobile, with text on desktop -->
      <div class="mb-6 overflow-x-auto scrollbar-none">
        <div class="ui-tablist min-w-max" role="tablist" aria-label={$_('page.settings.title')}>
          <button
            onclick={() => (activeTab = 'appearance')}
            title={$_('page.settings.tabs.appearance')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'appearance'}
          >
            <Palette size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.appearance')}</span>
          </button>
          <button
            onclick={() => (activeTab = 'editor')}
            title={$_('page.settings.tabs.editor')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'editor'}
          >
            <Edit3 size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.editor')}</span>
          </button>
          <button
            onclick={() => (activeTab = 'security')}
            title={$_('page.settings.tabs.security')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'security'}
          >
            <Shield size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.security')}</span>
          </button>
          <button
            onclick={() => (activeTab = 'account')}
            title={$_('page.settings.tabs.account')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'account'}
          >
            <User size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.account')}</span>
          </button>
          <button
            onclick={() => (activeTab = 'ai')}
            title={$_('page.settings.tabs.ai')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'ai'}
          >
            <Sparkles size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.ai')}</span>
          </button>
          <button
            onclick={() => (activeTab = 'data')}
            title={$_('page.settings.tabs.data')}
            class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
            role="tab"
            aria-selected={activeTab === 'data'}
          >
            <Database size={18} />
            <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.data')}</span>
          </button>
          {#if isTauriApp}
            <button
              onclick={() => (activeTab = 'connection')}
              title={$_('page.settings.tabs.connection')}
              class="ui-tab flex-shrink-0 whitespace-nowrap px-3 md:px-4 py-2.5"
              role="tab"
              aria-selected={activeTab === 'connection'}
            >
              <Globe size={18} />
              <span class="text-[9px] sm:text-sm">{$_('page.settings.tabs.connection')}</span>
            </button>
          {/if}
        </div>
      </div>

      <!-- Tab Content -->
      <div class="ui-panel ui-panel-mobile-flat p-4 sm:p-5">
        {#if activeTab === 'appearance'}
          <AppearanceTab />
        {:else if activeTab === 'editor'}
          <EditorTab />
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
          <DataTab />
        {:else if activeTab === 'connection' && isTauriApp}
          <ConnectionTab />
        {/if}
      </div>
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
