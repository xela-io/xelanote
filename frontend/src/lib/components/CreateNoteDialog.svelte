<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
  import * as features from '$lib/stores/features.svelte';

  interface Props {
    open: boolean;
    folderPath: string;
    onClose: () => void;
    onCreate: (title: string, noteType?: string) => void;
  }

  const { open, folderPath, onClose, onCreate }: Props = $props();

  let title = $state('');
  let noteType = $state<'note' | 'canvas'>('note');
  let titleInput: HTMLInputElement | null = null;

  const canvasEnabled = $derived(features.getCanvasFeatureEnabled());

  $effect(() => {
    if (open) {
      title = '';
      noteType = 'note';
      tick().then(() => {
        titleInput?.focus();
      });
    }
  });

  function handleCreate() {
    if (title.trim()) {
      onCreate(title.trim(), noteType === 'canvas' ? 'canvas' : undefined);
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleCreate();
    }
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

<BaseDialog {open} title={$_('dialog.create_note.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <div class="text-sm text-muted-foreground">
        {$_('dialog.create_note.folder')}: <span class="font-mono">{folderPath}</span>
      </div>

      {#if canvasEnabled}
        <div class="ui-tablist">
          <button
            type="button"
            class="ui-tab"
            class:is-active={noteType === 'note'}
            onclick={() => (noteType = 'note')}
          >
            {$_('dialog.create_note.type_note')}
          </button>
          <button
            type="button"
            class="ui-tab"
            class:is-active={noteType === 'canvas'}
            onclick={() => (noteType = 'canvas')}
          >
            {$_('dialog.create_note.type_canvas')}
          </button>
        </div>
      {/if}

      <DialogField forId="note-title-input" label={$_('dialog.create_note.label') + ':'}>
        <input
          id="note-title-input"
          type="text"
          bind:value={title}
          bind:this={titleInput}
          placeholder={$_('dialog.create_note.placeholder')}
          class="ui-input"
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
        disabled={!title.trim()}
        class="ui-button ui-button-primary text-sm"
      >
        {$_('common.create')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
