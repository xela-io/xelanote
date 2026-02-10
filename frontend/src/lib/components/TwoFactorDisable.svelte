<script lang="ts">
  import { AlertTriangle, Loader2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    onClose: () => void;
    onSuccess: () => void;
  }

  const { onClose, onSuccess }: Props = $props();

  let password = $state('');
  let code = $state('');
  let useBackupCode = $state(false);
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  let passwordInput: HTMLInputElement | null = null;

  async function handleDisable() {
    if (!password) {
      error = $_('dialog.twofa_disable.error_password');
      return;
    }

    if (!code) {
      error = useBackupCode
        ? $_('dialog.twofa_disable.error_backup_code')
        : $_('dialog.twofa_disable.error_totp_code');
      return;
    }

    isLoading = true;
    error = null;

    try {
      if (useBackupCode) {
        await api.disable2FA(password, undefined, code);
      } else {
        await api.disable2FA(password, code, undefined);
      }
      onSuccess();
    } catch (err) {
      error = err instanceof Error ? err.message : $_('dialog.twofa_disable.error_generic');
    } finally {
      isLoading = false;
    }
  }

  function handleCodeInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (useBackupCode) {
      // Backup code format: XXXX-XXXX (allow alphanumeric and dash)
      input.value = input.value
        .toUpperCase()
        .replace(/[^A-Z0-9-]/g, '')
        .slice(0, 9);
    } else {
      // TOTP: Only digits, max 6
      input.value = input.value.replace(/\D/g, '').slice(0, 6);
    }
    code = input.value;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleDisable();
    }
  }

  function toggleCodeType() {
    useBackupCode = !useBackupCode;
    code = '';
  }

  $effect(() => {
    passwordInput?.focus();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<BaseDialog
  open={true}
  title={$_('dialog.twofa_disable.title')}
  {onClose}
  size="md"
  variant="danger"
>
  {#snippet content()}
    <div class="space-y-6">
      <!-- Warning -->
      <div class="flex items-start gap-3 p-4 rounded-lg bg-amber-500/10 border border-amber-500/30">
        <AlertTriangle size={20} class="text-amber-500 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-amber-700 dark:text-amber-400">
          {@html $_('dialog.twofa_disable.warning')}
        </div>
      </div>

      <!-- Password -->
      <div class="space-y-2">
        <label for="disable-password" class="block text-sm font-medium text-foreground">
          {$_('dialog.twofa_disable.label_password')}
        </label>
        <input
          bind:this={passwordInput}
          id="disable-password"
          type="password"
          bind:value={password}
          disabled={isLoading}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
						focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
        />
      </div>

      <!-- Code Type Toggle -->
      <div class="flex items-center gap-2">
        <button type="button" onclick={toggleCodeType} class="text-sm text-primary hover:underline">
          {useBackupCode ? $_('page.login.use_totp_code') : $_('page.login.use_backup_code')}
        </button>
      </div>

      <!-- Code Input -->
      <div class="space-y-2">
        <label for="disable-code" class="block text-sm font-medium text-foreground">
          {useBackupCode ? $_('page.login.backup_code') : $_('page.login.totp_code')}
        </label>
        <input
          id="disable-code"
          type="text"
          inputmode={useBackupCode ? 'text' : 'numeric'}
          maxlength={useBackupCode ? 9 : 6}
          value={code}
          oninput={handleCodeInput}
          placeholder={useBackupCode ? 'XXXX-XXXX' : '000000'}
          disabled={isLoading}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground font-mono
						focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50
						{useBackupCode ? '' : 'text-center text-lg tracking-widest'}"
        />
      </div>

      {#if error}
        <div class="text-sm text-red-500">{error}</div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button
      type="button"
      onclick={onClose}
      class="flex-1 px-4 py-2 rounded-lg border border-border text-foreground
				hover:bg-muted transition-colors"
    >
      {$_('dialog.cancel')}
    </button>
    <button
      type="button"
      onclick={handleDisable}
      disabled={isLoading}
      class="flex-1 px-4 py-2 rounded-lg bg-red-600 text-white font-medium
				hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors
				flex items-center justify-center gap-2"
    >
      {#if isLoading}
        <Loader2 size={16} class="animate-spin" />
        {$_('dialog.twofa_disable.disabling')}
      {:else}
        {$_('dialog.twofa_disable.title')}
      {/if}
    </button>
  {/snippet}
</BaseDialog>
