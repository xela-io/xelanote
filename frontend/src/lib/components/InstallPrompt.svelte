<script lang="ts">
  import { Download, X } from 'lucide-svelte';

  interface BeforeInstallPromptEvent extends Event {
    prompt(): Promise<void>;
    userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
  }

  const { onClose }: { onClose: () => void } = $props();

  let deferredPrompt = $state<BeforeInstallPromptEvent | null>(null);
  let showPrompt = $state(false);

  // Listen for beforeinstallprompt event
  if (typeof window !== 'undefined') {
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault();
      deferredPrompt = e;
      showPrompt = true;
    });
  }

  async function handleInstall() {
    if (!deferredPrompt) return;

    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;

    console.log(`User ${outcome === 'accepted' ? 'accepted' : 'dismissed'} the install prompt`);

    deferredPrompt = null;
    showPrompt = false;
    onClose();
  }

  function handleDismiss() {
    showPrompt = false;
    onClose();
  }
</script>

{#if showPrompt}
  <div
    class="fixed bottom-4 left-4 right-4 sm:left-auto sm:right-4 z-50 bg-background border border-border rounded-lg shadow-lg p-4 max-w-sm"
  >
    <div class="flex items-start gap-3">
      <div class="bg-primary/10 p-2 rounded-lg">
        <Download size={24} class="text-primary" />
      </div>
      <div class="flex-1">
        <h3 class="font-semibold mb-1">Install xelanote</h3>
        <p class="text-sm text-muted-foreground mb-3">
          Install xelanote as an app for quick access and offline support
        </p>
        <div class="flex gap-2">
          <button
            onclick={handleInstall}
            class="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium"
          >
            Install
          </button>
          <button
            onclick={handleDismiss}
            class="px-4 py-2 bg-accent text-accent-foreground rounded-lg hover:bg-accent/80 transition-colors text-sm"
          >
            Not now
          </button>
        </div>
      </div>
      <button onclick={handleDismiss} class="text-muted-foreground hover:text-foreground">
        <X size={20} />
      </button>
    </div>
  </div>
{/if}
