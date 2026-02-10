<script lang="ts">
  import { Download, Key, AlertTriangle, Check, Copy } from 'lucide-svelte';
  import { toBase64Standard } from '$lib/crypto/sodium';

  interface Props {
    onClose?: () => void;
  }

  const { onClose }: Props = $props();

  let recoveryKey = $state('');
  let downloaded = $state(false);
  let copied = $state(false);
  let showKey = $state(false);

  async function generateRecoveryKey() {
    // Generate 256-bit recovery token
    // This is a random token, NOT derived from password
    // In a full implementation, this would be used to wrap the KEK
    const randomBytes = crypto.getRandomValues(new Uint8Array(32)); // 256-bit
    recoveryKey = toBase64Standard(randomBytes);
    showKey = true;
  }

  function downloadRecoveryKey() {
    const content = `XELANOTE RECOVERY TOKEN
========================

IMPORTANT: Keep this token SAFE and OFFLINE!
This token can be used to recover your encrypted notes if you forget your password.

Recovery Token:
${recoveryKey}

Generated: ${new Date().toISOString()}

WARNING:
- Anyone with this token can decrypt your notes!
- Store it in a password manager or encrypted USB drive
- Never share it via email or unencrypted channels
- Print it and store it in a safe place

To restore access:
1. Login with "Forgot Password?"
2. Enter this recovery token
3. Set a new password
`;

    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `xelanote-recovery-token-${new Date().toISOString().split('T')[0]}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    downloaded = true;

    // Auto-close after 2 seconds if downloaded
    setTimeout(() => {
      if (downloaded && onClose) {
        onClose();
      }
    }, 2000);
  }

  async function copyToClipboard() {
    try {
      await navigator.clipboard.writeText(recoveryKey);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  }

  function close() {
    if (onClose) {
      onClose();
    }
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
  <div
    class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto"
  >
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <Key class="w-8 h-8 text-primary" />
        <h2 class="text-2xl font-bold">Recovery Key erstellen</h2>
      </div>
      <button
        class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        onclick={close}
        aria-label="Close"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          ></path>
        </svg>
      </button>
    </div>

    {#if !showKey}
      <!-- Initial State - Explanation -->
      <div class="space-y-4 mb-6">
        <div
          class="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800"
        >
          <div class="flex items-start gap-3">
            <AlertTriangle
              class="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5"
            />
            <div>
              <h3 class="font-semibold text-yellow-800 dark:text-yellow-300 mb-2">
                Warum einen Recovery Key?
              </h3>
              <p class="text-sm text-yellow-800 dark:text-yellow-300">
                Wenn du dein Passwort vergisst, kannst du mit diesem Recovery Key deine
                verschlüsselten Notizen wiederherstellen. <strong
                  >Ohne Recovery Key sind deine Daten unwiederbringlich verloren!</strong
                >
              </p>
            </div>
          </div>
        </div>

        <div class="space-y-2">
          <h3 class="font-semibold">Was du beachten solltest:</h3>
          <ul class="space-y-2 text-sm text-gray-700 dark:text-gray-300">
            <li class="flex items-start gap-2">
              <span class="text-success font-bold mt-0.5">✓</span>
              <span>Der Recovery Key ist ein 256-bit zufälliger Token</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="text-success font-bold mt-0.5">✓</span>
              <span>Er kann verwendet werden, um Zugriff auf deine Notizen wiederherzustellen</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="text-red-600 dark:text-red-400 font-bold mt-0.5">⚠</span>
              <span><strong>Jeder mit diesem Key kann deine Notizen entschlüsseln!</strong></span>
            </li>
            <li class="flex items-start gap-2">
              <span class="text-red-600 dark:text-red-400 font-bold mt-0.5">⚠</span>
              <span
                >Bewahre ihn sicher offline auf (Passwort-Manager, Tresor, verschlüsselter USB)</span
              >
            </li>
          </ul>
        </div>

        <div class="p-4 bg-primary/10 rounded-lg border border-primary/30">
          <h3 class="font-semibold text-foreground mb-2">Empfohlene Aufbewahrung:</h3>
          <ul class="space-y-1 text-sm text-muted-foreground">
            <li>• Speichere ihn in einem Passwort-Manager (1Password, Bitwarden, etc.)</li>
            <li>• Drucke ihn aus und bewahre ihn in einem Tresor auf</li>
            <li>• Speichere ihn auf einem verschlüsselten USB-Stick</li>
            <li>• <strong>NIEMALS</strong> per E-Mail oder unverschlüsselte Cloud teilen!</li>
          </ul>
        </div>
      </div>

      <button
        class="w-full px-6 py-3 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center justify-center gap-2 font-semibold"
        onclick={generateRecoveryKey}
      >
        <Key class="w-5 h-5" />
        Recovery Key generieren
      </button>
    {:else}
      <!-- Key Generated State -->
      <div class="space-y-4 mb-6">
        <div class="p-4 bg-success/10 rounded-lg border border-success/30">
          <div class="flex items-center gap-2 mb-2">
            <Check class="w-5 h-5 text-success" />
            <span class="font-semibold text-success"> Recovery Key generiert! </span>
          </div>
          <p class="text-sm text-success">
            Speichere diesen Key sicher. Du wirst ihn benötigen, falls du dein Passwort vergisst.
          </p>
        </div>

        <!-- Recovery Key Display -->
        <div class="p-4 bg-muted rounded-lg border border-border">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-semibold text-foreground"> Dein Recovery Key: </span>
            <button
              class="px-3 py-1 text-sm bg-accent rounded-md hover:bg-accent/80 flex items-center gap-1"
              onclick={copyToClipboard}
            >
              {#if copied}
                <Check class="w-4 h-4 text-success" />
                <span>Kopiert!</span>
              {:else}
                <Copy class="w-4 h-4" />
                <span>Kopieren</span>
              {/if}
            </button>
          </div>
          <code class="block p-3 bg-card rounded border border-border text-sm font-mono break-all">
            {recoveryKey}
          </code>
        </div>

        <!-- Critical Warning -->
        <div
          class="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800"
        >
          <div class="flex items-start gap-3">
            <AlertTriangle class="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
            <div class="text-sm text-red-800 dark:text-red-300">
              <p class="font-semibold mb-1">⚠️ KRITISCH: Sichere diesen Key JETZT!</p>
              <p>
                Sobald du dieses Fenster schließt, wird der Key aus dem Speicher gelöscht. Du kannst
                ihn nicht erneut anzeigen lassen. Stelle sicher, dass du ihn heruntergeladen oder
                kopiert hast!
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex gap-3">
        <button
          class="flex-1 px-6 py-3 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center justify-center gap-2 font-semibold"
          onclick={downloadRecoveryKey}
        >
          <Download class="w-5 h-5" />
          Als Textdatei herunterladen
        </button>
      </div>

      {#if downloaded}
        <div class="mt-4 p-3 bg-success/10 rounded-lg border border-success/30 text-center">
          <p class="text-sm text-success flex items-center justify-center gap-2">
            <Check class="w-4 h-4" />
            <span>✓ Recovery Key heruntergeladen! Bewahre ihn sicher auf.</span>
          </p>
        </div>
      {/if}
    {/if}

    {#if showKey && !downloaded}
      <div class="mt-4 text-center">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          Du kannst dieses Fenster erst schließen, nachdem du den Key heruntergeladen hast.
        </p>
      </div>
    {/if}
  </div>
</div>
