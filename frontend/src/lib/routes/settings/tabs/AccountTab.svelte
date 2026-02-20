<script lang="ts">
  import {
    ArrowRight,
    Download,
    Key,
    Loader2,
    Lock,
    LogOut,
    Shield,
    ShieldOff,
    Trash2,
    User,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { EmailFormState } from '$lib/routes/settings/account-forms';
  import type { PasswordFormState } from '$lib/routes/settings/password-change';

  export let auth: typeof import('$lib/stores/auth.svelte');
  export let emailForm: EmailFormState;
  export let passwordForm: PasswordFormState;
  export let handleEmailSubmit: (e: Event) => void;
  export let handlePasswordSubmit: (e: Event) => void;
  export let tfaStatus:
    | (import('$lib/api').TwoFactorStatus & { backup_code_regenerated_at?: string })
    | null;
  export let isLoadingTfa: boolean;
  export let showSetupDialog: boolean;
  export let showDisableDialog: boolean;
  export let handleRegenerateBackupCodes: () => void;
  export let formatDate: (date: string) => string;
  export let handleSettingsLogout: () => void;
</script>

<div class="space-y-8">
  <!-- Current User Info -->
  {#if auth.getCurrentUser()}
    <div class="p-4 rounded-lg bg-muted/50 border border-border">
      <div class="text-sm text-muted-foreground">{$_('page.settings.account.logged_in_as')}</div>
      <div class="flex items-center gap-2 mt-1">
        <User size={16} class="text-muted-foreground" />
        <span class="font-medium text-foreground">{auth.getCurrentUser()?.username}</span>
        {#if auth.getCurrentUser()?.is_admin}
          <span class="text-xs bg-primary/20 text-primary px-2 py-0.5 rounded">
            {$_('page.settings.account.admin')}
          </span>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Change Email -->
  <div class="p-6 rounded-lg border border-border bg-card">
    <h3 class="text-lg font-medium text-foreground mb-1">
      {$_('page.settings.account.change_email_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">
      {$_('page.settings.account.change_email_description')}
    </p>

    <form class="space-y-4" onsubmit={handleEmailSubmit}>
      {#if emailForm.error}
        <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/30">
          <div class="flex items-center gap-2">
            <ShieldOff size={16} class="text-destructive" />
            <span class="text-sm text-destructive">{emailForm.error}</span>
          </div>
        </div>
      {/if}

      <div>
        <label for="settings-new-email" class="block text-sm font-medium text-foreground mb-2">
          {$_('page.settings.account.new_email')}
        </label>
        <input
          id="settings-new-email"
          type="email"
          bind:value={emailForm.newEmail}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50"
          placeholder="you@example.com"
        />
      </div>

      <div>
        <label for="settings-email-password" class="block text-sm font-medium text-foreground mb-2">
          {$_('page.settings.account.password')}
        </label>
        <input
          id="settings-email-password"
          type="password"
          bind:value={emailForm.password}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50"
        />
      </div>

      <button
        type="submit"
        class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
					font-medium hover:bg-primary/90 transition-colors"
      >
        <ArrowRight size={16} />
        {$_('page.settings.account.change_email_button')}
      </button>
    </form>
  </div>

  <!-- Change Password -->
  <div class="p-6 rounded-lg border border-border bg-card">
    <h3 class="text-lg font-medium text-foreground mb-1">
      {$_('page.settings.account.change_password_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">
      {$_('page.settings.account.change_password_description')}
    </p>

    <form class="space-y-4" onsubmit={handlePasswordSubmit}>
      {#if passwordForm.error}
        <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/30">
          <div class="flex items-center gap-2">
            <ShieldOff size={16} class="text-destructive" />
            <span class="text-sm text-destructive">{passwordForm.error}</span>
          </div>
        </div>
      {/if}

      <div>
        <label
          for="settings-current-password"
          class="block text-sm font-medium text-foreground mb-2"
        >
          {$_('page.settings.account.current_password')}
        </label>
        <input
          id="settings-current-password"
          type="password"
          bind:value={passwordForm.currentPassword}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50"
        />
      </div>

      <div>
        <label for="settings-new-password" class="block text-sm font-medium text-foreground mb-2">
          {$_('page.settings.account.new_password')}
        </label>
        <input
          id="settings-new-password"
          type="password"
          bind:value={passwordForm.newPassword}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50"
        />
      </div>

      <div>
        <label
          for="settings-confirm-password"
          class="block text-sm font-medium text-foreground mb-2"
        >
          {$_('page.settings.account.confirm_password')}
        </label>
        <input
          id="settings-confirm-password"
          type="password"
          bind:value={passwordForm.confirmPassword}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50"
        />
      </div>

      <button
        type="submit"
        disabled={passwordForm.isChanging}
        class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
					font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
      >
        {#if passwordForm.isChanging}
          <Loader2 size={16} class="animate-spin" />
          {$_('page.settings.account.changing_password')}
        {:else}
          <Lock size={16} />
          {$_('page.settings.account.change_password_button')}
        {/if}
      </button>

      {#if passwordForm.reWrappingProgress}
        <div class="text-sm text-muted-foreground">{passwordForm.reWrappingProgress}</div>
      {/if}
    </form>
  </div>

  <!-- Two-Factor Authentication -->
  <div class="p-6 rounded-lg border border-border bg-card">
    <div class="flex items-start justify-between gap-4 mb-4">
      <div>
        <h3 class="text-lg font-medium text-foreground mb-1">
          {$_('page.settings.account.two_factor_title')}
        </h3>
        <p class="text-sm text-muted-foreground">
          {$_('page.settings.account.two_factor_description')}
        </p>
      </div>
      <div class="flex items-center gap-2">
        {#if isLoadingTfa}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {:else if tfaStatus?.enabled}
          <div class="flex items-center gap-1 text-success text-sm">
            <Shield size={16} />
            {$_('page.settings.account.two_factor_enabled')}
          </div>
        {:else}
          <div class="flex items-center gap-1 text-muted-foreground text-sm">
            <ShieldOff size={16} />
            {$_('page.settings.account.two_factor_disabled')}
          </div>
        {/if}
      </div>
    </div>

    <div class="space-y-3">
      {#if !tfaStatus?.enabled}
        <button
          onclick={() => (showSetupDialog = true)}
          class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
						font-medium hover:bg-primary/90 transition-colors"
        >
          <Key size={16} />
          {$_('page.settings.account.enable_two_factor')}
        </button>
      {:else}
        <button
          onclick={() => (showDisableDialog = true)}
          class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive
						font-medium hover:bg-destructive/10 transition-colors"
        >
          <Trash2 size={16} />
          {$_('page.settings.account.disable_two_factor')}
        </button>
      {/if}

      {#if tfaStatus?.enabled}
        <button
          onclick={handleRegenerateBackupCodes}
          class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-border bg-card
						font-medium hover:border-primary/50 transition-colors"
        >
          <Download size={16} />
          {$_('page.settings.account.regenerate_backup_codes')}
        </button>
      {/if}
    </div>
  </div>

  {#if tfaStatus?.backup_code_regenerated_at}
    <div class="text-xs text-muted-foreground">
      {$_('page.settings.account.backup_codes_regenerated', {
        values: { date: formatDate(tfaStatus.backup_code_regenerated_at) },
      })}
    </div>
  {/if}

  <div class="p-6 rounded-lg border border-destructive/40 bg-destructive/5">
    <h3 class="text-lg font-medium text-foreground mb-1">{$_('common.logout')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.sidebar.confirm_logout')}</p>
    <button
      onclick={handleSettingsLogout}
      class="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-destructive text-destructive font-medium hover:bg-destructive/10 transition-colors"
    >
      <LogOut size={16} />
      {$_('common.logout')}
    </button>
  </div>
</div>
