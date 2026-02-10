<script lang="ts">
  import { Lock,RefreshCw, WifiOff } from 'lucide-svelte';

  import { getIsSyncing, getPendingCount,getSyncProgress } from '$lib/offline/sync-manager.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as network from '$lib/stores/network.svelte';

  const isOnline = $derived(network.getIsOnline());
  const isSyncing = $derived(getIsSyncing());
  const progress = $derived(getSyncProgress());
  const pending = $derived(getPendingCount());
  const encryptionLocked = $derived(!encryption.isEncryptionUnlocked());

  // Determine banner state
  const state = $derived.by(() => {
    if (isSyncing) {
      return 'syncing' as const;
    }
    if (!isOnline && encryptionLocked) {
      return 'locked' as const;
    }
    if (!isOnline && pending > 0) {
      return 'offline-pending' as const;
    }
    if (!isOnline) {
      return 'offline' as const;
    }
    return 'hidden' as const;
  });

  const bgClass = $derived(
    state === 'syncing' ? 'bg-blue-500' : state === 'locked' ? 'bg-amber-600' : 'bg-amber-500'
  );
</script>

{#if state !== 'hidden'}
  <div
    role="alert"
    aria-live="assertive"
    class="fixed bottom-0 left-0 right-0 z-50 {bgClass} text-white px-3 py-1.5 flex items-center justify-center gap-1.5 text-xs shadow-sm"
  >
    {#if state === 'syncing'}
      <RefreshCw size={14} class="animate-spin shrink-0" />
      <span
        >Synchronisiere{#if progress.total > 0}... {progress.current}/{progress.total}{/if}</span
      >
    {:else if state === 'locked'}
      <Lock size={14} class="shrink-0" />
      <span>Offline — Sync wartet auf Entsperrung</span>
    {:else if state === 'offline-pending'}
      <WifiOff size={14} class="shrink-0" />
      <span>Offline — {pending} {pending === 1 ? 'Aenderung wartet' : 'Aenderungen warten'}</span>
    {:else}
      <WifiOff size={14} class="shrink-0" />
      <span>Offline — Aenderungen werden lokal gespeichert</span>
    {/if}
  </div>
{/if}
