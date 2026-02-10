<script lang="ts">
  import { _ } from 'svelte-i18n';

  import {
    deleteWebAuthnCredential,
    isPlatformAuthenticatorAvailable,
    isWebAuthnSupported,
    registerWebAuthnCredential,
    type WebAuthnCredential,
  } from '$lib/crypto/webauthn';
  import { getCurrentUser } from '$lib/stores/auth.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import { error as showError, success } from '$lib/stores/toast.svelte';

  interface Props {
    credentials: WebAuthnCredential[];
    onUpdate?: () => void;
  }

  const { credentials, onUpdate }: Props = $props();

  // Local state
  let isAdding = $state(false);
  let isCheckingSupport = $state(true);
  let isSupported = $state(false);
  let deviceName = $state('');
  let removingCredentialId = $state<string | null>(null);

  // Check WebAuthn support on mount
  $effect(() => {
    checkSupport();
  });

  async function checkSupport() {
    isCheckingSupport = true;
    try {
      const supported = isWebAuthnSupported();
      const platformAvailable = supported ? await isPlatformAuthenticatorAvailable() : false;
      isSupported = supported && platformAvailable;
    } catch (error) {
      console.error('Error checking WebAuthn support:', error);
      isSupported = false;
    } finally {
      isCheckingSupport = false;
    }
  }

  async function handleAddDevice() {
    const user = getCurrentUser();
    if (!user) {
      showError('Not authenticated');
      return;
    }

    isAdding = true;
    try {
      const newCred = await registerWebAuthnCredential(
        user.id,
        user.username,
        deviceName.trim() || undefined
      );

      success(`Device "${newCred.device_name}" registered successfully`);
      deviceName = '';

      // Reload preferences to get updated credentials list
      if (onUpdate) {
        onUpdate();
      } else {
        // Fallback: reload settings
        await settings.loadPreferences();
      }
    } catch (err) {
      console.error('Failed to add WebAuthn device:', err);
      if (err instanceof Error) {
        showError(err.message);
      } else {
        showError('Failed to register device');
      }
    } finally {
      isAdding = false;
    }
  }

  async function handleRemoveDevice(credential: WebAuthnCredential) {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('component.webauthn.remove_device_confirm', {
        values: { name: credential.device_name },
      }),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });

    if (!confirmed) return;

    removingCredentialId = credential.credential_id;
    try {
      await deleteWebAuthnCredential(credential.credential_id);
      success(`Device "${credential.device_name}" removed`);

      // Reload preferences
      if (onUpdate) {
        onUpdate();
      } else {
        await settings.loadPreferences();
      }
    } catch (err) {
      console.error('Failed to remove WebAuthn device:', err);
      showError('Failed to remove device');
    } finally {
      removingCredentialId = null;
    }
  }

  function formatDate(dateString: string): string {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      return 'Today';
    } else if (diffDays === 1) {
      return 'Yesterday';
    } else if (diffDays < 7) {
      return `${diffDays} days ago`;
    } else {
      return date.toLocaleDateString();
    }
  }
</script>

<div class="webauthn-device-manager">
  <div class="header">
    <div class="title-section">
      <h3>Biometric Devices</h3>
      <p class="description">
        Use Touch ID, Face ID, or Windows Hello to quickly unlock encrypted notes without typing
        your password.
      </p>
    </div>
  </div>

  {#if isCheckingSupport}
    <div class="status-message">
      <div class="spinner"></div>
      <span>Checking device compatibility...</span>
    </div>
  {:else if !isSupported}
    <div class="status-message warning">
      <svg
        width="20"
        height="20"
        viewBox="0 0 20 20"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <circle cx="10" cy="10" r="8"></circle>
        <line x1="10" y1="6" x2="10" y2="10"></line>
        <circle cx="10" cy="14" r="0.5" fill="currentColor"></circle>
      </svg>
      <div>
        <strong>Biometric authentication not available</strong>
        <p>
          Your device or browser does not support Touch ID, Face ID, or Windows Hello. You can still
          use password-based encryption.
        </p>
      </div>
    </div>
  {:else}
    <!-- Add Device Section -->
    <div class="add-device-section">
      <div class="input-row">
        <input
          type="text"
          bind:value={deviceName}
          placeholder="Device name (optional)"
          disabled={isAdding}
          onkeydown={(e) => {
            if (e.key === 'Enter' && !isAdding) {
              handleAddDevice();
            }
          }}
        />
        <button class="btn-primary" onclick={handleAddDevice} disabled={isAdding}>
          {#if isAdding}
            <div class="spinner-small"></div>
            <span>Registering...</span>
          {:else}
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="8" y1="4" x2="8" y2="12"></line>
              <line x1="4" y1="8" x2="12" y2="8"></line>
            </svg>
            <span>Add Device</span>
          {/if}
        </button>
      </div>
      <p class="hint">Leave blank to auto-generate a name like "Safari on Mac (Jan 20, 2026)"</p>
    </div>

    <!-- Devices List -->
    {#if credentials.length === 0}
      <div class="empty-state">
        <svg
          width="48"
          height="48"
          viewBox="0 0 48 48"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="8" y="12" width="32" height="24" rx="2"></rect>
          <circle cx="24" cy="24" r="4"></circle>
          <path d="M 16 36 L 16 40 A 2 2 0 0 0 18 42 L 30 42 A 2 2 0 0 0 32 40 L 32 36"></path>
        </svg>
        <p>No biometric devices registered yet</p>
        <span class="empty-hint">Add your first device above to enable biometric unlock</span>
      </div>
    {:else}
      <div class="devices-list">
        {#each credentials as credential (credential.id)}
          <div class="device-card">
            <div class="device-icon">
              <svg
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <rect x="4" y="6" width="16" height="12" rx="1"></rect>
                <circle cx="12" cy="12" r="2"></circle>
              </svg>
            </div>
            <div class="device-info">
              <div class="device-name">{credential.device_name}</div>
              <div class="device-meta">
                <span>Added {formatDate(credential.created_at)}</span>
                {#if credential.last_used_at}
                  <span class="separator">•</span>
                  <span>Last used {formatDate(credential.last_used_at)}</span>
                {/if}
              </div>
            </div>
            <button
              class="btn-remove"
              onclick={() => handleRemoveDevice(credential)}
              disabled={removingCredentialId === credential.credential_id}
              title="Remove device"
            >
              {#if removingCredentialId === credential.credential_id}
                <div class="spinner-small"></div>
              {:else}
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 16 16"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <line x1="4" y1="4" x2="12" y2="12"></line>
                  <line x1="12" y1="4" x2="4" y2="12"></line>
                </svg>
              {/if}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .webauthn-device-manager {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .title-section h3 {
    margin: 0 0 0.5rem 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--color-foreground);
  }

  .description {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    line-height: 1.5;
  }

  .status-message {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
  }

  .status-message.warning {
    background: color-mix(in oklch, var(--color-warning), transparent 90%);
    border-color: color-mix(in oklch, var(--color-warning), transparent 70%);
    color: var(--color-warning);
    flex-direction: column;
    align-items: flex-start;
  }

  .status-message.warning svg {
    flex-shrink: 0;
  }

  .status-message.warning strong {
    display: block;
    margin-bottom: 0.25rem;
    color: var(--color-warning);
  }

  .status-message.warning p {
    margin: 0;
    color: var(--color-muted-foreground);
    font-size: 0.875rem;
  }

  .add-device-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .input-row {
    display: flex;
    gap: 0.75rem;
  }

  .input-row input {
    flex: 1;
    padding: 0.625rem 0.875rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    font-size: 0.875rem;
    color: var(--color-foreground);
    transition: all 0.15s ease;
  }

  .input-row input:focus {
    outline: none;
    border-color: var(--color-primary);
    background: var(--color-background);
  }

  .input-row input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1rem;
    background: var(--color-primary);
    color: var(--color-primary-foreground);
    border: none;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
    white-space: nowrap;
  }

  .btn-primary:hover:not(:disabled) {
    background: color-mix(in oklch, var(--color-primary), black 10%);
    transform: translateY(-1px);
  }

  .btn-primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
  }

  .hint {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--color-muted-foreground);
  }

  .devices-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .device-card {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    transition: all 0.15s ease;
  }

  .device-card:hover {
    border-color: var(--color-border);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }

  .device-icon {
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    color: var(--color-muted-foreground);
  }

  .device-info {
    flex: 1;
    min-width: 0;
  }

  .device-name {
    font-weight: 500;
    color: var(--color-foreground);
    font-size: 0.9375rem;
    margin-bottom: 0.25rem;
  }

  .device-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8125rem;
    color: var(--color-muted-foreground);
  }

  .separator {
    color: var(--color-border);
  }

  .btn-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--color-border);
    border-radius: 6px;
    color: var(--color-muted-foreground);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-remove:hover:not(:disabled) {
    background: color-mix(in oklch, var(--color-destructive), transparent 90%);
    border-color: var(--color-destructive);
    color: var(--color-destructive);
  }

  .btn-remove:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 1rem;
    text-align: center;
    color: var(--color-muted-foreground);
  }

  .empty-state svg {
    margin-bottom: 1rem;
    opacity: 0.5;
  }

  .empty-state p {
    margin: 0 0 0.5rem 0;
    font-size: 0.9375rem;
    color: var(--color-muted-foreground);
  }

  .empty-hint {
    font-size: 0.8125rem;
  }

  .spinner,
  .spinner-small {
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .spinner {
    width: 16px;
    height: 16px;
  }

  .spinner-small {
    width: 12px;
    height: 12px;
    border-width: 1.5px;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 640px) {
    .input-row {
      flex-direction: column;
    }

    .device-card {
      padding: 0.875rem;
    }

    .device-icon {
      width: 36px;
      height: 36px;
    }
  }
</style>
