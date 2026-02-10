<script lang="ts">
  import { _ } from 'svelte-i18n';
  import BaseDialog from './BaseDialog.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';

  const state = $derived(dialog.getConfirmState());
  const isOpen = $derived(state !== null);

  function handleConfirm() {
    console.log('[ConfirmDialog] handleConfirm called');
    dialog.resolveConfirm(true);
  }

  function handleCancel() {
    console.log('[ConfirmDialog] handleCancel called');
    dialog.resolveConfirm(false);
  }

  const buttonBaseClasses =
    'px-4 py-2 rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';
  const cancelButtonClasses = `${buttonBaseClasses} bg-secondary text-secondary-foreground hover:bg-secondary/80 focus:ring-ring`;
  const confirmButtonClasses = $derived(
    state?.variant === 'danger'
      ? `${buttonBaseClasses} bg-destructive text-destructive-foreground hover:bg-destructive/90 focus:ring-destructive`
      : `${buttonBaseClasses} bg-primary text-primary-foreground hover:bg-primary/90 focus:ring-primary`
  );
</script>

<BaseDialog
  open={isOpen}
  title={state?.title ?? $_('dialog.confirm_title')}
  onClose={handleCancel}
  variant={state?.variant ?? 'default'}
  size="sm"
  closeOnBackdrop={false}
>
  {#snippet content()}
    <p class="text-foreground">{state?.message ?? ''}</p>
  {/snippet}

  {#snippet footer()}
    <button type="button" class={cancelButtonClasses} onclick={handleCancel}>
      {state?.cancelText ?? $_('dialog.cancel')}
    </button>
    <button type="button" class={confirmButtonClasses} onclick={handleConfirm}>
      {state?.confirmText ?? $_('dialog.confirm')}
    </button>
  {/snippet}
</BaseDialog>
