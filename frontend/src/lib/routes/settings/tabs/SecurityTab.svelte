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
  export let autoLockTimeout: number;
  export let isSavingAutoLockTimeout: boolean;
  export let handleSecurityLevelChange: (level: 'paranoid' | 'balanced' | 'convenient') => void;
  export let handleAutoLockTimeoutChange: () => void;
  export let webAuthnCredentials: Array<import('$lib/crypto/webauthn').WebAuthnCredential>;
  export let load2FAStatus: () => void;
  export let loadSecurityPreferences: () => void;
  export let isLoadingMigrationStats: boolean;
  export let migrationStats: MigrationStats | null;
</script>

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
          <div class="text-sm text-foreground">{$_('page.settings.security.balanced_info')}</div>
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
    <WebAuthnDeviceManager credentials={webAuthnCredentials} onUpdate={loadSecurityPreferences} />
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
              <span class="text-sm text-success">{$_('page.settings.security.migration_complete')}</span>
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
