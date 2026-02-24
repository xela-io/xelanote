<script lang="ts">
  import { Check, Copy, Download } from 'lucide-svelte';

  interface Props {
    codes: string[];
    showWarning?: boolean;
    onConfirm?: () => void;
  }

  const { codes, showWarning = true, onConfirm }: Props = $props();

  let copied = $state(false);
  let confirmed = $state(false);

  async function copyAllCodes() {
    const text = codes.join('\n');
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch (err) {
      console.error('Failed to copy codes:', err);
    }
  }

  function downloadCodes() {
    const text = `xelanote Backup-Codes\n${'='.repeat(30)}\n\nDiese Codes können jeweils nur einmal verwendet werden.\nBewahre sie sicher auf!\n\n${codes.join('\n')}\n\nGeneriert am: ${new Date().toLocaleDateString('de-DE')}`;
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'xelanote-backup-codes.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  function handleConfirm() {
    if (confirmed && onConfirm) {
      onConfirm();
    }
  }
</script>

<div class="space-y-4">
  {#if showWarning}
    <div class="ui-alert ui-alert-warning">
      <strong>Wichtig:</strong> Diese Codes werden nur einmal angezeigt. Speichere sie jetzt an einem
      sicheren Ort! Jeder Code kann nur einmal verwendet werden.
    </div>
  {/if}

  <!-- Codes Grid -->
  <div class="ui-panel-soft grid grid-cols-2 gap-2 p-3">
    {#each codes as code (code)}
      <div class="ui-list-item px-3 py-2 font-mono text-sm text-center text-foreground">
        {code}
      </div>
    {/each}
  </div>

  <!-- Action Buttons -->
  <div class="flex gap-2">
    <button onclick={copyAllCodes} class="ui-button ui-button-secondary flex-1">
      {#if copied}
        <Check size={16} class="text-success" />
        Kopiert!
      {:else}
        <Copy size={16} />
        Alle kopieren
      {/if}
    </button>
    <button onclick={downloadCodes} class="ui-button ui-button-secondary flex-1">
      <Download size={16} />
      Als Datei speichern
    </button>
  </div>

  <!-- Confirmation -->
  {#if onConfirm}
    <div class="ui-panel-soft space-y-4 border-t-0 p-4">
      <label class="flex items-start gap-3 cursor-pointer">
        <input
          type="checkbox"
          bind:checked={confirmed}
          class="mt-1 w-4 h-4 rounded border-border text-primary focus:ring-primary"
        />
        <span class="text-sm text-foreground">
          Ich habe die Backup-Codes sicher gespeichert und verstehe, dass sie nur einmal angezeigt
          werden.
        </span>
      </label>
      <button
        onclick={handleConfirm}
        disabled={!confirmed}
        class="ui-button ui-button-primary w-full"
      >
        Fertig
      </button>
    </div>
  {/if}
</div>
