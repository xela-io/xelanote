<script lang="ts">
  import { Copy, Download, Check } from 'lucide-svelte';

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
    <div
      class="p-4 rounded-lg bg-amber-500/10 border border-amber-500/30 text-sm text-amber-700 dark:text-amber-400"
    >
      <strong>Wichtig:</strong> Diese Codes werden nur einmal angezeigt. Speichere sie jetzt an einem
      sicheren Ort! Jeder Code kann nur einmal verwendet werden.
    </div>
  {/if}

  <!-- Codes Grid -->
  <div class="grid grid-cols-2 gap-2">
    {#each codes as code (code)}
      <div class="px-3 py-2 bg-muted rounded-lg font-mono text-sm text-center text-foreground">
        {code}
      </div>
    {/each}
  </div>

  <!-- Action Buttons -->
  <div class="flex gap-2">
    <button
      onclick={copyAllCodes}
      class="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-border
				bg-background text-foreground hover:bg-muted transition-colors"
    >
      {#if copied}
        <Check size={16} class="text-success" />
        Kopiert!
      {:else}
        <Copy size={16} />
        Alle kopieren
      {/if}
    </button>
    <button
      onclick={downloadCodes}
      class="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg border border-border
				bg-background text-foreground hover:bg-muted transition-colors"
    >
      <Download size={16} />
      Als Datei speichern
    </button>
  </div>

  <!-- Confirmation -->
  {#if onConfirm}
    <div class="pt-4 border-t border-border space-y-4">
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
        class="w-full px-4 py-2 rounded-lg bg-primary text-primary-foreground font-medium
					hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        Fertig
      </button>
    </div>
  {/if}
</div>
