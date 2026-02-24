<script lang="ts">
  import { AlertTriangle, Check, Loader2, RefreshCw } from 'lucide-svelte';

  import { getDefaultServerUrl, getServerUrl, setServerUrl } from '$lib/config';

  const serverUrlForm = $state({
    url: getServerUrl(),
    error: '',
    isSaving: false,
    showRestartWarning: false,
  });

  function validateServerUrl(url: string): boolean {
    try {
      const parsed = new URL(url);
      return parsed.protocol === 'https:' || parsed.protocol === 'http:';
    } catch {
      return false;
    }
  }

  async function handleServerUrlSubmit(e: Event) {
    e.preventDefault();
    serverUrlForm.error = '';

    const url = serverUrlForm.url.trim();

    if (!url) {
      serverUrlForm.error = 'Server URL ist erforderlich';
      return;
    }

    if (!validateServerUrl(url)) {
      serverUrlForm.error = 'Ungültige URL. Bitte geben Sie eine gültige HTTP(S) URL ein.';
      return;
    }

    serverUrlForm.isSaving = true;

    try {
      // Test connection to new server
      const testResponse = await fetch(`${url}/health`);
      if (!testResponse.ok) {
        serverUrlForm.error = 'Server nicht erreichbar oder nicht kompatibel';
        return;
      }

      // Save the new URL
      setServerUrl(url);

      // Show restart warning - user needs to re-login
      serverUrlForm.showRestartWarning = true;
    } catch (err) {
      console.error('Failed to connect to server:', err);
      serverUrlForm.error = 'Verbindung zum Server fehlgeschlagen';
    } finally {
      serverUrlForm.isSaving = false;
    }
  }

  function handleResetServerUrl() {
    serverUrlForm.url = getDefaultServerUrl();
    serverUrlForm.error = '';
    serverUrlForm.showRestartWarning = false;
  }
</script>

<div class="space-y-8">
  <!-- Server URL -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-4">Server-Verbindung</h3>
    <p class="text-sm text-muted-foreground mb-4">
      Verbinde dich mit xelanote.com oder deinem eigenen Server.
    </p>

    <form onsubmit={handleServerUrlSubmit} class="space-y-4">
      <div>
        <label for="server-url" class="block text-sm font-medium text-foreground mb-1">
          Server URL
        </label>
        <input
          id="server-url"
          type="url"
          bind:value={serverUrlForm.url}
          disabled={serverUrlForm.isSaving}
          class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground
								focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
          placeholder="https://xelanote.com"
        />
      </div>

      {#if serverUrlForm.error}
        <div class="text-sm text-red-500">{serverUrlForm.error}</div>
      {/if}

      {#if serverUrlForm.showRestartWarning}
        <div
          class="p-4 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
        >
          <div class="flex items-start gap-3">
            <AlertTriangle size={20} class="text-orange-700 dark:text-orange-400 mt-0.5" />
            <div class="flex-1">
              <div class="font-medium text-orange-950 dark:text-orange-200 mb-1">
                Server URL geändert
              </div>
              <div class="text-sm text-orange-900 dark:text-orange-300">
                Die neue Server-URL wurde gespeichert. Bitte melde dich ab und neu an, um die
                Änderung anzuwenden.
              </div>
            </div>
          </div>
        </div>
      {/if}

      <div class="flex gap-2">
        <button
          type="submit"
          disabled={serverUrlForm.isSaving}
          class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
								font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {#if serverUrlForm.isSaving}
            <Loader2 size={16} class="animate-spin" />
            Verbinde...
          {:else}
            <Check size={16} />
            Verbindung testen & speichern
          {/if}
        </button>
        <button
          type="button"
          onclick={handleResetServerUrl}
          disabled={serverUrlForm.isSaving}
          class="flex items-center gap-2 px-4 py-2 rounded-lg border border-border
								bg-background text-foreground hover:bg-muted disabled:opacity-50 transition-colors"
        >
          <RefreshCw size={16} />
          Standard
        </button>
      </div>
    </form>
  </div>

  <!-- Self-hosted info -->
  <div class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground">
    <strong>Hinweis:</strong> Du kannst xelanote selbst hosten. Weitere Informationen findest du in
    der
    <a
      href="https://github.com/xela-io/xelanote"
      target="_blank"
      rel="noopener noreferrer"
      class="underline hover:no-underline"
    >
      Dokumentation
    </a>.
  </div>
</div>
