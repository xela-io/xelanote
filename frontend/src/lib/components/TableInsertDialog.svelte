<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    onInsert: (rows: number, cols: number) => void;
  }

  const { open, onClose, onInsert }: Props = $props();

  let cols = $state(3);
  let rows = $state(2);
  let colsInput: HTMLInputElement | null = null;

  $effect(() => {
    if (open) {
      cols = 3;
      rows = 2;
      tick().then(() => {
        colsInput?.focus();
        colsInput?.select();
      });
    }
  });

  function handleInsert() {
    const c = Math.max(1, Math.min(20, cols));
    const r = Math.max(1, Math.min(50, rows));
    onInsert(r, c);
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleInsert();
    }
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

<BaseDialog {open} title={$_('component.editor.table_insert.title')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-4">
      <DialogField
        forId="table-cols-input"
        label={$_('component.editor.table_insert.columns') + ':'}
      >
        <input
          id="table-cols-input"
          type="number"
          bind:value={cols}
          bind:this={colsInput}
          min="1"
          max="20"
          class="ui-input"
        />
      </DialogField>
      <DialogField forId="table-rows-input" label={$_('component.editor.table_insert.rows') + ':'}>
        <input
          id="table-rows-input"
          type="number"
          bind:value={rows}
          min="1"
          max="50"
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
      <button type="button" onclick={handleInsert} class="ui-button ui-button-primary text-sm">
        {$_('component.editor.table_insert.insert')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
