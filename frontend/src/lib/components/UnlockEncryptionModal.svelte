<script lang="ts">
  import { Loader2, Lock } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
  import { fromBase64Standard } from '$lib/crypto/sodium';
  import {
    authenticateWithWebAuthn,
    isPlatformAuthenticatorAvailable,
    isWebAuthnSupported,
  } from '$lib/crypto/webauthn';
  import { getCurrentUser } from '$lib/stores/auth.svelte';
  import * as autoLock from '$lib/stores/auto-lock.svelte';
  import { setupEncryption, tryRestoreKEK } from '$lib/stores/encryption.svelte';

  interface Props {
    isOpen: boolean;
    onSuccess: () => void;
    onCancel?: () => void;
  }

  let { isOpen = $bindable(), onSuccess, onCancel }: Props = $props();

  let password = $state('');
  let error = $state('');
  let isUnlocking = $state(false);
  let isWebAuthnUnlocking = $state(false);
  let showWebAuthn = $state(false);
  let webAuthnAvailable = $state(false);

  // Check WebAuthn availability when modal opens
  $effect(() => {
    if (isOpen) {
      checkWebAuthnAvailability();
    }
  });

  async function checkWebAuthnAvailability() {
    try {
      // Check if WebAuthn is supported and user has credentials
      const supported = isWebAuthnSupported();
      const platformAvailable = supported ? await isPlatformAuthenticatorAvailable() : false;

      // Check if user has any registered credentials
      const prefs = await api.getPreferences();
      const hasCredentials = prefs.webauthn_credentials && prefs.webauthn_credentials.length > 0;

      // Don't show WebAuthn in paranoid mode (KEK is never persisted)
      const isParanoid = prefs.security_level === 'paranoid';

      webAuthnAvailable = supported && platformAvailable && hasCredentials && !isParanoid;
      showWebAuthn = webAuthnAvailable;
    } catch (err) {
      console.error('Failed to check WebAuthn availability:', err);
      webAuthnAvailable = false;
      showWebAuthn = false;
    }
  }

  async function handleUnlock() {
    if (!password) {
      error = $_('component.unlock_encryption.error_empty_password');
      return;
    }

    error = '';
    isUnlocking = true;

    try {
      const user = getCurrentUser();
      if (!user) {
        error = $_('component.unlock_encryption.error_no_user');
        isUnlocking = false;
        return;
      }

      // Get encryption salt from backend
      const currentUser = await api.getCurrentUser();
      if (!currentUser.encryption_salt) {
        error = $_('component.unlock_encryption.error_no_salt');
        isUnlocking = false;
        return;
      }

      const salt = fromBase64Standard(currentUser.encryption_salt);

      // Load user preferences to get security level
      const prefs = await api.getPreferences();
      const securityLevel = (prefs.security_level || 'balanced') as
        | 'paranoid'
        | 'balanced'
        | 'convenient';

      // Setup encryption with password (will persist KEK if security_level != paranoid)
      await setupEncryption(password, user.id, salt, securityLevel);

      // Restart auto-lock timer after successful unlock
      const autoLockTimeout = prefs.auto_lock_timeout ?? 15;
      autoLock.initAutoLock(autoLockTimeout);

      // Clear password from memory
      password = '';

      // Close modal and notify success
      isOpen = false;
      onSuccess();
    } catch (err) {
      console.error('Failed to unlock encryption:', err);
      error = err instanceof Error ? err.message : $_('component.unlock_encryption.error_unlock');
    } finally {
      isUnlocking = false;
    }
  }

  async function handleWebAuthnUnlock() {
    error = '';
    isWebAuthnUnlocking = true;

    try {
      const user = getCurrentUser();
      if (!user) {
        error = $_('component.unlock_encryption.error_no_user');
        isWebAuthnUnlocking = false;
        return;
      }

      // Authenticate with WebAuthn
      const authenticated = await authenticateWithWebAuthn();

      if (!authenticated) {
        error = $_('component.unlock_encryption.error_biometric');
        return;
      }

      // Restore KEK from IndexedDB (no password needed)
      const restored = await tryRestoreKEK(user.id);

      if (!restored) {
        // KEK not in IndexedDB - first time after device registration
        error = $_('component.unlock_encryption.error_first_time');
        showWebAuthn = false; // Hide WebAuthn button until password unlock
        return;
      }

      // Load preferences for auto-lock timeout
      const prefs = await api.getPreferences();
      const autoLockTimeout = prefs.auto_lock_timeout ?? 15;
      autoLock.initAutoLock(autoLockTimeout);

      // Close modal and notify success
      isOpen = false;
      onSuccess();
    } catch (err) {
      console.error('Failed to unlock with WebAuthn:', err);
      error =
        err instanceof Error
          ? err.message
          : $_('component.unlock_encryption.error_biometric_generic');
    } finally {
      isWebAuthnUnlocking = false;
    }
  }

  function handleCancel() {
    password = '';
    error = '';
    isOpen = false;
    if (onCancel) {
      onCancel();
    }
  }

  // Handle Enter key
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !isUnlocking) {
      handleUnlock();
    }
  }
</script>

<svelte:window onkeydown={isOpen ? handleKeydown : undefined} />

<BaseDialog
  open={isOpen}
  title={$_('component.unlock_encryption.title')}
  onClose={handleCancel}
  size="sm"
  closeOnBackdrop={false}
  closeOnEscape={false}
>
  {#snippet content()}
    <div class="space-y-6">
      <div class="flex items-start gap-3">
        <div
          class="ui-panel-soft flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full"
        >
          <Lock size={20} class="text-amber-500" />
        </div>
        <p class="text-sm text-muted-foreground pt-2">
          {$_('component.unlock_encryption.description')}
        </p>
      </div>

      {#if showWebAuthn}
        <button
          type="button"
          class="ui-button ui-button-primary w-full py-3"
          onclick={handleWebAuthnUnlock}
          disabled={isWebAuthnUnlocking || isUnlocking}
        >
          {#if isWebAuthnUnlocking}
            <Loader2 size={16} class="animate-spin" />
            <span>{$_('component.unlock_encryption.authenticating')}</span>
          {:else}
            <svg
              width="20"
              height="20"
              viewBox="0 0 20 20"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect x="3" y="5" width="14" height="10" rx="1"></rect>
              <circle cx="10" cy="10" r="2"></circle>
            </svg>
            <span>{$_('component.unlock_encryption.unlock_biometric')}</span>
          {/if}
        </button>

        <div class="flex items-center text-center text-sm text-muted-foreground">
          <div class="flex-1 border-b border-border"></div>
          <span class="px-4">{$_('component.unlock_encryption.or_password')}</span>
          <div class="flex-1 border-b border-border"></div>
        </div>
      {/if}

      <DialogField forId="unlock-password" label={$_('common.password')}>
        <input
          id="unlock-password"
          type="password"
          bind:value={password}
          placeholder={$_('component.unlock_encryption.password_placeholder')}
          disabled={isUnlocking || isWebAuthnUnlocking}
          class="ui-input w-full"
        />
      </DialogField>

      {#if error}
        <div class="ui-alert ui-alert-danger">
          {error}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button
        type="button"
        class="ui-button ui-button-secondary"
        onclick={handleCancel}
        disabled={isUnlocking || isWebAuthnUnlocking}
      >
        {$_('dialog.cancel')}
      </button>
      <button
        type="button"
        class="ui-button ui-button-primary"
        onclick={handleUnlock}
        disabled={isUnlocking || isWebAuthnUnlocking}
      >
        {#if isUnlocking}
          <Loader2 size={16} class="animate-spin" />
          {$_('component.unlock_encryption.unlocking')}
        {:else}
          {$_('component.unlock_encryption.unlock')}
        {/if}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
