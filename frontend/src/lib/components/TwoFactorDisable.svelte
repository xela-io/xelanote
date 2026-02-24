<script lang="ts">
  import DOMPurify from 'isomorphic-dompurify';
  import { AlertTriangle, Loader2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';

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
      <div class="ui-alert ui-alert-warning flex items-start gap-3">
        <AlertTriangle size={20} class="text-amber-500 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-amber-700 dark:text-amber-400">
          {@html DOMPurify.sanitize($_('dialog.twofa_disable.warning'))}
        </div>
      </div>

      <!-- Password -->
      <DialogField forId="disable-password" label={$_('dialog.twofa_disable.label_password')}>
        <input
          bind:this={passwordInput}
          id="disable-password"
          type="password"
          bind:value={password}
          disabled={isLoading}
          class="ui-input w-full"
        />
      </DialogField>

      <!-- Code Type Toggle -->
      <div class="flex items-center gap-2">
        <button type="button" onclick={toggleCodeType} class="text-sm text-primary hover:underline">
          {useBackupCode ? $_('page.login.use_totp_code') : $_('page.login.use_backup_code')}
        </button>
      </div>

      <!-- Code Input -->
      <DialogField
        forId="disable-code"
        label={useBackupCode ? $_('page.login.backup_code') : $_('page.login.totp_code')}
      >
        <input
          id="disable-code"
          type="text"
          inputmode={useBackupCode ? 'text' : 'numeric'}
          maxlength={useBackupCode ? 9 : 6}
          value={code}
          oninput={handleCodeInput}
          placeholder={useBackupCode ? 'XXXX-XXXX' : '000000'}
          disabled={isLoading}
          class="ui-input w-full font-mono {useBackupCode
            ? ''
            : 'text-center text-lg tracking-widest'}"
        />
      </DialogField>

      {#if error}
        <div class="ui-alert ui-alert-danger text-sm">{error}</div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button type="button" onclick={onClose} class="ui-button ui-button-secondary flex-1">
        {$_('dialog.cancel')}
      </button>
      <button
        type="button"
        onclick={handleDisable}
        disabled={isLoading}
        class="ui-button ui-button-danger flex-1"
      >
        {#if isLoading}
          <Loader2 size={16} class="animate-spin" />
          {$_('dialog.twofa_disable.disabling')}
        {:else}
          {$_('dialog.twofa_disable.title')}
        {/if}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
