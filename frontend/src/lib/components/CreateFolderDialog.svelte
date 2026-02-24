<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    onCreate: (path: string) => void;
  }

  const { open, onClose, onCreate }: Props = $props();

  let path = $state('/');
  let errorMessage = $state<string | null>(null);
  let pathInput: HTMLInputElement | null = null;

  $effect(() => {
    if (open) {
      path = '/';
      errorMessage = null;
      tick().then(() => {
        pathInput?.focus();
      });
    }
  });

  function handleCreate() {
    errorMessage = null;

    if (!path.trim()) {
      errorMessage = $_('dialog.create_folder.error_empty');
      return;
    }

    if (!path.startsWith('/')) {
      errorMessage = $_('dialog.create_folder.error_slash');
      return;
    }

    if (path.includes('..')) {
      errorMessage = $_('dialog.create_folder.error_dotdot');
      return;
    }

    onCreate(path.trim());
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleCreate();
    }
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

<BaseDialog {open} title={$_('dialog.create_folder.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <DialogField
        forId="folder-path-input"
        label={$_('dialog.create_folder.label') + ':'}
        help={$_('dialog.create_folder.hint')}
        error={errorMessage}
      >
        <input
          id="folder-path-input"
          type="text"
          bind:value={path}
          bind:this={pathInput}
          placeholder="/Projects/Work"
          class="ui-input font-mono"
        />
      </DialogField>
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button type="button" onclick={onClose} class="ui-button ui-button-secondary text-sm">
        {$_('dialog.cancel')}
      </button>
      <button
        type="button"
        onclick={handleCreate}
        disabled={!path.trim()}
        class="ui-button ui-button-primary text-sm"
      >
        {$_('common.create')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
