<script lang="ts">
  import { CheckCircle, Key, Loader2, Shield, Smartphone } from 'lucide-svelte';

  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  import BackupCodesDisplay from './BackupCodesDisplay.svelte';

  interface Props {
    onClose: () => void;
    onSuccess: () => void;
  }

  const { onClose, onSuccess }: Props = $props();

  // Steps: 'intro' | 'qr' | 'verify' | 'backup'
  let step = $state<'intro' | 'qr' | 'verify' | 'backup'>('intro');
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  // Setup data
  let secret = $state('');
  let qrCodeUrl = $state('');
  let qrCodeDataUrl = $state('');
  let backupCodes = $state<string[]>([]);

  // Verification
  let verifyCode = $state('');
  let verifyInput = $state<HTMLInputElement | null>(null);

  const stepTitles: Record<string, string> = {
    intro: 'Zwei-Faktor-Authentifizierung',
    qr: 'QR-Code scannen',
    verify: 'Code bestätigen',
    backup: '2FA aktiviert!',
  };

  const dialogTitle = $derived(stepTitles[step] ?? 'Zwei-Faktor-Authentifizierung');

  async function startSetup() {
    isLoading = true;
    error = null;
    try {
      const setupData = await api.setup2FA();
      secret = setupData.secret;
      qrCodeUrl = setupData.qr_code_url;
      backupCodes = setupData.backup_codes;

      // Generate QR code data URL (lazy import to avoid blocking settings page)
      const QRCode = await import('qrcode');
      qrCodeDataUrl = await QRCode.toDataURL(qrCodeUrl, {
        width: 200,
        margin: 2,
        color: {
          dark: '#000000',
          light: '#ffffff',
        },
      });

      step = 'qr';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Fehler beim Einrichten von 2FA';
    } finally {
      isLoading = false;
    }
  }

  function goToVerify() {
    step = 'verify';
    // Focus input after render
    setTimeout(() => verifyInput?.focus(), 100);
  }

  async function verifySetup() {
    if (verifyCode.length !== 6) {
      error = 'Bitte gib einen 6-stelligen Code ein';
      return;
    }

    isLoading = true;
    error = null;
    try {
      await api.verify2FA(verifyCode);
      step = 'backup';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Ungültiger Code';
      verifyCode = '';
      verifyInput?.focus();
    } finally {
      isLoading = false;
    }
  }

  function handleBackupConfirm() {
    onSuccess();
  }

  function handleCodeInput(e: Event) {
    const input = e.target as HTMLInputElement;
    // Only allow digits
    input.value = input.value.replace(/\D/g, '').slice(0, 6);
    verifyCode = input.value;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && step === 'verify') {
      verifySetup();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<BaseDialog
  open={true}
  title={dialogTitle}
  {onClose}
  size="md"
  closeOnEscape={step === 'intro'}
  closeOnBackdrop={step === 'intro'}
  showCloseButton={step === 'intro'}
  scrollable={true}
>
  {#snippet content()}
    {#if step === 'intro'}
      <!-- Step 1: Introduction -->
      <div class="text-center space-y-6">
        <div class="ui-panel-soft mx-auto flex h-16 w-16 items-center justify-center rounded-full">
          <Shield size={32} class="text-primary" />
        </div>
        <div class="space-y-2">
          <h3 class="text-xl font-semibold text-foreground">Mehr Sicherheit für deinen Account</h3>
          <p class="text-muted-foreground">
            Mit der Zwei-Faktor-Authentifizierung (2FA) wird bei jedem Login zusätzlich ein Code aus
            deiner Authenticator-App benötigt.
          </p>
        </div>
        <div class="ui-panel-soft space-y-3 p-4 text-left">
          <div class="flex items-start gap-3">
            <Smartphone size={20} class="text-primary mt-0.5 flex-shrink-0" />
            <div>
              <div class="font-medium text-foreground">Authenticator-App benötigt</div>
              <div class="text-sm text-muted-foreground">
                Z.B. Google Authenticator, Authy, 1Password oder Bitwarden
              </div>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Key size={20} class="text-primary mt-0.5 flex-shrink-0" />
            <div>
              <div class="font-medium text-foreground">Backup-Codes</div>
              <div class="text-sm text-muted-foreground">
                Du erhältst 10 Einmal-Codes für den Notfall
              </div>
            </div>
          </div>
        </div>
        <button
          type="button"
          onclick={startSetup}
          disabled={isLoading}
          class="ui-button ui-button-primary w-full"
        >
          {#if isLoading}
            <Loader2 size={16} class="animate-spin" />
            Wird eingerichtet...
          {:else}
            2FA einrichten
          {/if}
        </button>
      </div>
    {:else if step === 'qr'}
      <!-- Step 2: QR Code -->
      <div class="space-y-6">
        <div class="text-center space-y-2">
          <p class="text-sm text-muted-foreground">
            Scanne diesen QR-Code mit deiner Authenticator-App
          </p>
        </div>

        <!-- QR Code -->
        <div class="flex justify-center">
          <div class="ui-panel-soft rounded-xl bg-white p-4">
            <img src={qrCodeDataUrl} alt="2FA QR Code" class="w-48 h-48" />
          </div>
        </div>

        <!-- Manual Secret -->
        <div class="ui-panel-soft space-y-2 p-4">
          <div class="text-sm text-muted-foreground text-center">
            Oder gib diesen Code manuell ein:
          </div>
          <div
            class="ui-list-item select-all break-all px-4 py-3 text-center font-mono text-sm text-foreground"
          >
            {secret}
          </div>
        </div>

        <button type="button" onclick={goToVerify} class="ui-button ui-button-primary w-full">
          Weiter
        </button>
      </div>
    {:else if step === 'verify'}
      <!-- Step 3: Verify Code -->
      <div class="space-y-6">
        <div class="text-center space-y-2">
          <p class="text-sm text-muted-foreground">
            Gib den 6-stelligen Code aus deiner Authenticator-App ein
          </p>
        </div>

        <div class="space-y-4">
          <input
            bind:this={verifyInput}
            type="text"
            inputmode="numeric"
            maxlength="6"
            value={verifyCode}
            oninput={handleCodeInput}
            placeholder="000000"
            disabled={isLoading}
            class="ui-input w-full px-4 py-3 text-center text-2xl font-mono tracking-widest"
          />

          {#if error}
            <div class="ui-alert ui-alert-danger text-center">{error}</div>
          {/if}

          <button
            type="button"
            onclick={verifySetup}
            disabled={isLoading || verifyCode.length !== 6}
            class="ui-button ui-button-primary w-full"
          >
            {#if isLoading}
              <Loader2 size={16} class="animate-spin" />
              Wird überprüft...
            {:else}
              Bestätigen
            {/if}
          </button>

          <button
            type="button"
            onclick={() => (step = 'qr')}
            class="ui-button ui-button-secondary w-full"
          >
            Zurück
          </button>
        </div>
      </div>
    {:else if step === 'backup'}
      <!-- Step 4: Backup Codes -->
      <div class="space-y-6">
        <div class="text-center space-y-2">
          <div
            class="ui-panel-soft mx-auto flex h-12 w-12 items-center justify-center rounded-full"
          >
            <CheckCircle size={24} class="text-success" />
          </div>
          <p class="text-sm text-muted-foreground">Speichere jetzt deine Backup-Codes</p>
        </div>

        <BackupCodesDisplay codes={backupCodes} onConfirm={handleBackupConfirm} />
      </div>
    {/if}

    {#if error && step !== 'verify'}
      <div class="ui-alert ui-alert-danger mt-4">
        {error}
      </div>
    {/if}
  {/snippet}
</BaseDialog>
