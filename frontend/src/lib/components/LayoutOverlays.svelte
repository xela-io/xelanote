<script lang="ts">
  import type { ComponentType } from 'svelte';

  import { goto } from '$app/navigation';
  import InstallPrompt from '$lib/components/InstallPrompt.svelte';
  import OfflineBanner from '$lib/components/OfflineBanner.svelte';
  import SessionRestoreBanner from '$lib/components/SessionRestoreBanner.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import AlertDialog from '$lib/components/ui/AlertDialog.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import UnlockEncryptionModal from '$lib/components/UnlockEncryptionModal.svelte';
  import * as notes from '$lib/stores/notes.svelte';

  interface Props {
    showOfflineBanner: boolean;
    isSyncing: boolean;
    hasConflicts: boolean;
    conflictDialog: ComponentType | null | undefined;
    showInstallPrompt: boolean;
    isPublic: boolean;
    showSessionRestoreBanner: boolean;
    showUnlockModal: boolean;
    onCloseInstallPrompt: () => void;
  }

  let {
    showOfflineBanner,
    isSyncing,
    hasConflicts,
    conflictDialog: ConflictDialogComponent,
    showInstallPrompt,
    isPublic,
    showSessionRestoreBanner,
    showUnlockModal = $bindable(),
    onCloseInstallPrompt,
  }: Props = $props();
</script>

<!-- Global Toast Notifications -->
<Toast />

<!-- Offline Banner (handles its own visibility based on offline state + sync state) -->
{#if showOfflineBanner || isSyncing}
  <OfflineBanner />
{/if}

{#if showSessionRestoreBanner}
  <SessionRestoreBanner />
{/if}

<!-- Conflict Dialog (lazy-loaded, shown when sync conflicts need resolution) -->
{#if hasConflicts && ConflictDialogComponent}
  <ConflictDialogComponent />
{/if}

<!-- Install Prompt -->
{#if showInstallPrompt && !isPublic}
  <InstallPrompt onClose={onCloseInstallPrompt} />
{/if}

<!-- Global encryption unlock modal -->
<UnlockEncryptionModal
  bind:isOpen={showUnlockModal}
  onSuccess={() => {
    notes.clearError();
  }}
  onCancel={() => {
    notes.clearError();
    goto('/');
  }}
/>

<!-- Global accessible dialogs -->
<ConfirmDialog />
<AlertDialog />
