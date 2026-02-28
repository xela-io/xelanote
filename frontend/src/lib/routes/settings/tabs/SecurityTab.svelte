<script lang="ts">
  import {
    AlertTriangle,
    ArrowRight,
    Check,
    Loader2,
    Lock,
    RefreshCw,
    Shield,
    ShieldCheck,
    ShieldOff,
    Unlock,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import SecurityKeyManager from '$lib/components/SecurityKeyManager.svelte';
  import WebAuthnDeviceManager from '$lib/components/WebAuthnDeviceManager.svelte';
  import type { MigrationStats } from '$lib/routes/settings/migration-stats';

  export let encryption: typeof import('$lib/stores/encryption.svelte');
  export let securityLevel: 'paranoid' | 'balanced' | 'convenient';
  export let isSavingSecurityLevel: boolean;
  export let isSettingRecoveryKey: boolean;
  export let autoLockTimeout: number;
  export let isSavingAutoLockTimeout: boolean;
  export let handleSecurityLevelChange: (level: 'paranoid' | 'balanced' | 'convenient') => void;
  export let handleAutoLockTimeoutChange: () => void;
  export let handleSetupRecoveryKey: () => void;
  export let webAuthnCredentials: Array<import('$lib/crypto/webauthn').WebAuthnCredential>;
  export let load2FAStatus: () => void;
  export let loadSecurityPreferences: () => void;
  export let isLoadingMigrationStats: boolean;
  export let migrationStats: MigrationStats | null;
</script>

<div class="space-y-8">
  <!-- Encryption Locked Warning -->
  {#if !encryption.isEncryptionUnlocked()}
    <div class="ui-alert ui-alert-warning">
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
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.security.security_level_title')}
    </h3>
    <div class="space-y-3">
      <!-- Paranoid -->
      <button
        onclick={() => handleSecurityLevelChange('paranoid')}
        disabled={!encryption.isEncryptionUnlocked() || isSavingSecurityLevel}
        class="ui-select-card ui-select-card-success {securityLevel === 'paranoid'
          ? 'is-active'
          : ''}"
      >
        <div class="mt-1 {securityLevel === 'paranoid' ? 'text-success' : 'text-muted-foreground'}">
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
        class="ui-select-card ui-select-card-primary {securityLevel === 'balanced'
          ? 'is-active'
          : ''}"
      >
        <div class="mt-1 {securityLevel === 'balanced' ? 'text-primary' : 'text-muted-foreground'}">
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
        class="ui-select-card ui-select-card-warning {securityLevel === 'convenient'
          ? 'is-active'
          : ''}"
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
      <div class="ui-alert ui-alert-info mt-4">
        <div class="flex items-start gap-2">
          <div class="text-primary mt-0.5">ℹ️</div>
          <div class="text-sm text-foreground">{$_('page.settings.security.balanced_info')}</div>
        </div>
      </div>
    {:else if securityLevel === 'convenient'}
      <div class="ui-alert ui-alert-warning mt-4">
        <div class="flex items-start gap-2">
          <AlertTriangle size={16} class="text-orange-700 dark:text-orange-400 mt-0.5" />
          <div class="text-sm text-orange-900 dark:text-orange-300">
            <strong>{$_('common.note')}</strong>
            {$_('page.settings.security.convenient_warning')}
          </div>
        </div>
      </div>
    {/if}
  </section>

  <!-- Auto-Lock Timeout -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.security.autolock_timeout_title')}
    </h3>
    <div class="ui-fieldset">
      <div class="ui-form-row">
        <label for="auto-lock-timeout" class="ui-label mb-0">
          {$_('page.settings.security.autolock_timeout_label')}
        </label>
        <select
          id="auto-lock-timeout"
          bind:value={autoLockTimeout}
          onchange={handleAutoLockTimeoutChange}
          disabled={securityLevel === 'paranoid' ||
            !encryption.isEncryptionUnlocked() ||
            isSavingAutoLockTimeout}
          class="ui-select"
        >
          <option value={0}>{$_('page.settings.security.autolock_never')}</option>
          <option value={5}>{$_('page.settings.security.autolock_5_min')}</option>
          <option value={15}>{$_('page.settings.security.autolock_15_min')}</option>
          <option value={30}>{$_('page.settings.security.autolock_30_min')}</option>
          <option value={60}>{$_('page.settings.security.autolock_60_min')}</option>
        </select>
      </div>

      {#if securityLevel === 'paranoid'}
        <div class="ui-form-help">
          {$_('page.settings.security.autolock_paranoid_info')}
        </div>
      {/if}
    </div>
  </section>

  <!-- Security Keys (FIDO2 2FA) -->
  <div class="ui-panel p-5 sm:p-6">
    <h3 class="ui-form-section-title mb-2">{$_('page.settings.security.security_keys_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">
      {$_('page.settings.security.security_keys_description')}
    </p>
    <SecurityKeyManager onUpdate={load2FAStatus} />
  </div>

  <!-- Recovery Key Setup -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.security.recovery_setup_title')}
    </h3>
    <div class="ui-panel p-4 sm:p-5">
      <p class="text-sm text-muted-foreground mb-4">
        {$_('page.settings.security.recovery_setup_description')}
      </p>
      <button
        onclick={handleSetupRecoveryKey}
        disabled={!encryption.isEncryptionUnlocked() || isSettingRecoveryKey}
        class="ui-button ui-button-primary"
      >
        {#if isSettingRecoveryKey}
          <Loader2 size={16} class="animate-spin" />
          {$_('page.settings.security.recovery_setup_in_progress')}
        {:else}
          <RefreshCw size={16} />
          {$_('page.settings.security.recovery_setup_button')}
        {/if}
      </button>
    </div>
  </section>

  <!-- Biometric Devices -->
  <div class="ui-panel p-5 sm:p-6">
    <WebAuthnDeviceManager credentials={webAuthnCredentials} onUpdate={loadSecurityPreferences} />
  </div>

  <!-- Note Encryption Migration -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.security.encryption_migration_title')}
    </h3>

    {#if isLoadingMigrationStats}
      <div class="ui-panel-soft p-4 flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if migrationStats}
      <div class="ui-panel p-4 space-y-4">
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
                class={migrationStats.plaintext > 0 ? 'text-orange-500' : 'text-muted-foreground'}
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
          <div class="ui-alert ui-alert-warning">
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
              class="ui-button ui-button-primary w-full"
            >
              <RefreshCw size={16} />
              {$_('page.settings.security.migration_button')}
              <ArrowRight size={16} />
            </button>
          </div>
        {:else if migrationStats.total > 0}
          <div class="ui-alert ui-alert-success">
            <div class="flex items-center gap-2">
              <Check size={16} class="text-success" />
              <span class="text-sm text-success"
                >{$_('page.settings.security.migration_complete')}</span
              >
            </div>
          </div>
        {:else}
          <div class="text-sm text-muted-foreground">
            {$_('page.settings.security.migration_no_notes')}
          </div>
        {/if}
      </div>
    {:else}
      <div class="ui-panel-soft p-4 text-sm text-muted-foreground">
        {$_('page.settings.security.migration_error')}
      </div>
    {/if}
  </section>
</div>
