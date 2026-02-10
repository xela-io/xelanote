<script lang="ts">
  import { AlertCircle, AlertTriangle, Info } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import * as dialog from '$lib/stores/dialog.svelte';

  import BaseDialog from './BaseDialog.svelte';

  const state = $derived(dialog.getAlertState());
  const isOpen = $derived(state !== null);

  function handleClose() {
    dialog.resolveAlert();
  }

  const buttonBaseClasses =
    'px-4 py-2 rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';

  const confirmButtonClasses = $derived(() => {
    switch (state?.variant) {
      case 'danger':
        return `${buttonBaseClasses} bg-destructive text-destructive-foreground hover:bg-destructive/90 focus:ring-destructive`;
      case 'warning':
        return `${buttonBaseClasses} bg-yellow-500 text-white hover:bg-yellow-600 focus:ring-yellow-500`;
      default:
        return `${buttonBaseClasses} bg-primary text-primary-foreground hover:bg-primary/90 focus:ring-primary`;
    }
  });

  const IconComponent = $derived(() => {
    switch (state?.variant) {
      case 'danger':
        return AlertCircle;
      case 'warning':
        return AlertTriangle;
      default:
        return Info;
    }
  });

  const iconColorClass = $derived(() => {
    switch (state?.variant) {
      case 'danger':
        return 'text-destructive';
      case 'warning':
        return 'text-yellow-500';
      default:
        return 'text-primary';
    }
  });
</script>

<BaseDialog
  open={isOpen}
  title={state?.title ?? $_('common.note')}
  onClose={handleClose}
  variant={state?.variant === 'danger' ? 'danger' : 'default'}
  size="sm"
>
  {#snippet content()}
    {@const Icon = IconComponent()}
    <div class="flex gap-3">
      <div class={iconColorClass()}>
        <Icon size={24} />
      </div>
      <p class="text-foreground flex-1">{state?.message ?? ''}</p>
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" class={confirmButtonClasses()} onclick={handleClose}>
      {state?.confirmText ?? $_('common.close')}
    </button>
  {/snippet}
</BaseDialog>
