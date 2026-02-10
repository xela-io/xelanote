<script lang="ts">
  import { AlertTriangle,KeyRound, Plus, Trash2 } from 'lucide-svelte';

  import { deleteFIDO2Credential, type FIDO2CredentialInfo,listFIDO2Credentials } from '$lib/api';
  import { isFIDO2Supported, registerSecurityKey } from '$lib/crypto/fido2';
  import * as dialog from '$lib/stores/dialog.svelte';
  import { error as showError,success } from '$lib/stores/toast.svelte';

  interface Props {
    onUpdate?: () => void;
  }

  const { onUpdate }: Props = $props();

  let credentials = $state<FIDO2CredentialInfo[]>([]);
  let isLoading = $state(true);
  let isRegistering = $state(false);
  let deviceName = $state('');
  let supported = $state(false);
  let backupCodes = $state<string[] | null>(null);
  let showBackupCodes = $state(false);
  let backupCodesConfirmed = $state(false);

  $effect(() => {
    supported = isFIDO2Supported();
    loadCredentials();
  });

  async function loadCredentials() {
    isLoading = true;
    try {
      credentials = await listFIDO2Credentials();
    } catch (err) {
      console.error('Failed to load FIDO2 credentials:', err);
    } finally {
      isLoading = false;
    }
  }

  async function handleRegister() {
    if (!deviceName.trim()) {
      const date = new Date().toLocaleDateString('de-DE');
      deviceName = `Security Key (${date})`;
    }

    isRegistering = true;
    try {
      const result = await registerSecurityKey(deviceName.trim());
      success('Security Key erfolgreich registriert');
      deviceName = '';

      if (result.backup_codes && result.backup_codes.length > 0) {
        backupCodes = result.backup_codes;
        showBackupCodes = true;
        backupCodesConfirmed = false;
      }

      await loadCredentials();
      onUpdate?.();
    } catch (err) {
      console.error('FIDO2 registration failed:', err);
      showError(err instanceof Error ? err.message : 'Registrierung fehlgeschlagen');
    } finally {
      isRegistering = false;
    }
  }

  async function handleDelete(cred: FIDO2CredentialInfo) {
    const isLast = credentials.length === 1;
    const message = isLast
      ? `"${cred.device_name}" entfernen? Das ist dein letzter Security Key. Ohne diesen kannst du dich nicht mehr per FIDO2 anmelden.`
      : `"${cred.device_name}" wirklich entfernen?`;

    const confirmed = await dialog.confirm({ message, variant: isLast ? 'danger' : 'default' });
    if (!confirmed) return;

    try {
      await deleteFIDO2Credential(cred.id);
      success('Security Key entfernt');
      await loadCredentials();
      onUpdate?.();
    } catch (_err) {
      showError('Löschen fehlgeschlagen');
    }
  }

  function formatDate(dateStr: string): string {
    try {
      return new Date(dateStr).toLocaleDateString('de-DE', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
      });
    } catch {
      return dateStr;
    }
  }

  function dismissBackupCodes() {
    showBackupCodes = false;
    backupCodes = null;
    backupCodesConfirmed = false;
  }
</script>

{#if !supported}
  <div class="unsupported-notice">
    <AlertTriangle size={16} />
    <span>Dein Browser unterstützt keine Security Keys (WebAuthn).</span>
  </div>
{:else}
  <!-- Backup codes display -->
  {#if showBackupCodes && backupCodes}
    <div class="backup-codes-overlay">
      <div class="backup-codes-card">
        <h4>Backup Codes</h4>
        <p class="backup-codes-info">
          Speichere diese Codes sicher ab. Du benötigst sie, falls du deinen Security Key verlierst.
        </p>
        <div class="backup-codes-grid">
          {#each backupCodes as code (code)}
            <code class="backup-code">{code}</code>
          {/each}
        </div>
        <label class="backup-confirm-label">
          <input type="checkbox" bind:checked={backupCodesConfirmed} />
          Ich habe die Codes sicher gespeichert
        </label>
        <button
          type="button"
          class="btn-primary"
          disabled={!backupCodesConfirmed}
          onclick={dismissBackupCodes}
        >
          Verstanden
        </button>
      </div>
    </div>
  {/if}

  <!-- Add new key -->
  <div class="add-key-row">
    <input
      type="text"
      bind:value={deviceName}
      placeholder="Name (z.B. YubiKey)"
      disabled={isRegistering}
      class="device-name-input"
    />
    <button type="button" onclick={handleRegister} disabled={isRegistering} class="btn-add">
      {#if isRegistering}
        Warte auf Key...
      {:else}
        <Plus size={16} />
        Hinzufügen
      {/if}
    </button>
  </div>

  <!-- Credential list -->
  {#if isLoading}
    <p class="loading-text">Lade...</p>
  {:else if credentials.length === 0}
    <p class="empty-text">Noch keine Security Keys registriert.</p>
  {:else}
    <div class="credential-list">
      {#each credentials as cred (cred.id)}
        <div class="credential-item">
          <div class="credential-info">
            <div class="credential-icon">
              <KeyRound size={18} />
            </div>
            <div>
              <span class="credential-name">{cred.device_name || 'Security Key'}</span>
              <span class="credential-meta">
                Erstellt: {formatDate(cred.created_at)}
                {#if cred.last_used_at}
                  · Zuletzt: {formatDate(cred.last_used_at)}
                {/if}
              </span>
            </div>
          </div>
          <button
            type="button"
            class="btn-delete"
            title="Entfernen"
            onclick={() => handleDelete(cred)}
          >
            <Trash2 size={16} />
          </button>
        </div>
      {/each}
    </div>
  {/if}
{/if}

<style>
  .unsupported-notice {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem;
    font-size: 0.875rem;
    color: var(--color-warning);
    background: color-mix(in oklch, var(--color-warning), transparent 90%);
    border-radius: 6px;
  }

  .add-key-row {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .device-name-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    color: var(--color-foreground);
  }

  .device-name-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .btn-add {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-primary-foreground);
    background: var(--color-primary);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    white-space: nowrap;
  }

  .btn-add:hover:not(:disabled) {
    background: color-mix(in oklch, var(--color-primary), black 15%);
  }

  .btn-add:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .credential-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .credential-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 6px;
  }

  .credential-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .credential-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    background: color-mix(in oklch, var(--color-primary), transparent 85%);
    border-radius: 6px;
    color: var(--color-primary);
  }

  .credential-name {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-foreground);
  }

  .credential-meta {
    display: block;
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .btn-delete {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    color: var(--color-muted-foreground);
    background: transparent;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .btn-delete:hover {
    color: var(--color-destructive);
    background: color-mix(in oklch, var(--color-destructive), transparent 90%);
  }

  .loading-text,
  .empty-text {
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    text-align: center;
    padding: 1rem 0;
  }

  /* Backup codes overlay */
  .backup-codes-overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
    padding: 1rem;
  }

  .backup-codes-card {
    max-width: 400px;
    width: 100%;
    padding: 1.5rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.15);
  }

  .backup-codes-card h4 {
    margin: 0 0 0.5rem;
    font-size: 1.125rem;
    color: var(--color-foreground);
  }

  .backup-codes-info {
    margin: 0 0 1rem;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
  }

  .backup-codes-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .backup-code {
    padding: 0.5rem;
    font-size: 0.875rem;
    font-family: monospace;
    text-align: center;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    color: var(--color-foreground);
  }

  .backup-confirm-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1rem;
    font-size: 0.875rem;
    color: var(--color-foreground);
    cursor: pointer;
  }

  .btn-primary {
    width: 100%;
    padding: 0.625rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--color-primary-foreground);
    background: var(--color-primary);
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
