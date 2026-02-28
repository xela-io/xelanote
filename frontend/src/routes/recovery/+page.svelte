<script lang="ts">
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { ApiError } from '$lib/api';
  import {
    getRecoverySaltByEmail,
    getRecoveryWrappedDEKs,
    resetPasswordWithRecoveryToken,
    verifyRecoveryKey,
  } from '$lib/api/auth';
  import { e2eEncryption } from '$lib/crypto/e2e';
  import { fromBase64Standard } from '$lib/crypto/sodium';

  let email = $state('');
  let recoveryKey = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let isSubmitting = $state(false);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  async function handleRecoveryReset() {
    errorMessage = null;
    successMessage = null;

    if (!email.trim() || !recoveryKey || !newPassword || !confirmPassword) {
      errorMessage = $_('page.recovery.error_required_fields');
      return;
    }
    if (newPassword.length < 8) {
      errorMessage = $_('page.recovery.error_password_length');
      return;
    }
    if (newPassword !== confirmPassword) {
      errorMessage = $_('page.recovery.error_password_mismatch');
      return;
    }

    isSubmitting = true;
    try {
      const { salt } = await getRecoverySaltByEmail(email.trim().toLowerCase());
      const verify = await verifyRecoveryKey(email.trim().toLowerCase(), recoveryKey);
      const wrappedDEKs = await getRecoveryWrappedDEKs(verify.recovery_reset_token);

      let reWrappedNoteDEKs: Record<string, string> | undefined;
      let reWrappedVersionDEKs: Record<string, string> | undefined;

      if (wrappedDEKs.notes.length > 0 || wrappedDEKs.versions.length > 0) {
        if (!verify.encryption_salt) {
          throw new Error($_('page.recovery.error_missing_encryption_salt'));
        }

        const recoverySalt = fromBase64Standard(salt);
        const encryptionSalt = fromBase64Standard(verify.encryption_salt);
        const reWrapped = await e2eEncryption.reWrapRecoveryDEKs(
          wrappedDEKs.notes,
          wrappedDEKs.versions,
          recoveryKey,
          newPassword,
          recoverySalt,
          encryptionSalt
        );

        reWrappedNoteDEKs = Object.fromEntries(reWrapped.notes);
        reWrappedVersionDEKs = Object.fromEntries(reWrapped.versions);
      }

      await resetPasswordWithRecoveryToken({
        recovery_reset_token: verify.recovery_reset_token,
        new_password: newPassword,
        re_wrapped_note_deks: reWrappedNoteDEKs,
        re_wrapped_version_deks: reWrappedVersionDEKs,
      });

      successMessage = $_('page.recovery.success');
      recoveryKey = '';
      newPassword = '';
      confirmPassword = '';
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          errorMessage = $_('page.recovery.error_invalid_credentials');
        } else if (err.status === 409) {
          errorMessage = $_('page.recovery.error_conflict');
        } else if (err.status === 400) {
          errorMessage = err.message || $_('page.recovery.error_bad_request');
        } else {
          errorMessage = $_('page.recovery.error_generic');
        }
      } else {
        errorMessage = err instanceof Error ? err.message : $_('page.recovery.error_generic');
      }
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="recovery-container">
  <div class="recovery-card">
    <h1>{$_('page.recovery.title')}</h1>
    <p class="subtitle">{$_('page.recovery.subtitle')}</p>

    <form
      onsubmit={(e) => {
        e.preventDefault();
        handleRecoveryReset();
      }}
    >
      <div class="form-group">
        <label for="email">{$_('common.email')}</label>
        <input id="email" type="email" bind:value={email} placeholder="you@example.com" />
      </div>

      <div class="form-group">
        <label for="recovery-key">{$_('page.recovery.recovery_key')}</label>
        <input id="recovery-key" type="password" bind:value={recoveryKey} />
      </div>

      <div class="form-group">
        <label for="new-password">{$_('page.recovery.new_password')}</label>
        <input id="new-password" type="password" bind:value={newPassword} />
      </div>

      <div class="form-group">
        <label for="confirm-password">{$_('page.recovery.confirm_password')}</label>
        <input id="confirm-password" type="password" bind:value={confirmPassword} />
      </div>

      {#if errorMessage}
        <div class="error-message">{errorMessage}</div>
      {/if}
      {#if successMessage}
        <div class="success-message">{successMessage}</div>
      {/if}

      <button type="submit" class="action-button" disabled={isSubmitting}>
        {isSubmitting ? $_('page.recovery.submitting') : $_('page.recovery.submit')}
      </button>
    </form>

    <button type="button" class="back-button" onclick={() => goto('/login')}>
      {$_('common.back')}
    </button>
  </div>
</div>

<style>
  .recovery-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    min-height: 100dvh;
    padding: 1rem;
    background: var(--color-background);
  }

  .recovery-card {
    width: 100%;
    max-width: 460px;
    padding: 1.5rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-card);
    box-shadow: var(--shadow-md, 0 4px 6px rgba(0, 0, 0, 0.1));
  }

  h1 {
    margin: 0;
    text-align: center;
    font-size: 1.5rem;
    color: var(--color-foreground);
  }

  .subtitle {
    margin: 0.75rem 0 1.5rem;
    text-align: center;
    color: var(--color-muted-foreground);
    font-size: 0.95rem;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--color-foreground);
  }

  input {
    width: 100%;
    box-sizing: border-box;
    padding: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-background);
    color: var(--color-foreground);
  }

  input:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary), transparent 80%);
  }

  .action-button {
    width: 100%;
    border: none;
    border-radius: var(--radius-sm);
    padding: 0.75rem;
    font-weight: 600;
    background: var(--color-primary);
    color: var(--color-primary-foreground);
    cursor: pointer;
  }

  .action-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .back-button {
    width: 100%;
    margin-top: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    padding: 0.75rem;
    background: var(--color-background);
    color: var(--color-foreground);
    cursor: pointer;
  }

  .error-message {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-destructive);
    background: color-mix(in oklch, var(--color-destructive), transparent 85%);
    color: var(--color-destructive);
    font-size: 0.875rem;
  }

  .success-message {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border-radius: var(--radius-sm);
    border: 1px solid color-mix(in oklch, var(--color-primary), black 10%);
    background: color-mix(in oklch, var(--color-primary), transparent 85%);
    color: var(--color-primary);
    font-size: 0.875rem;
  }
</style>
