<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

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
      <div class="space-y-2">
        <label for="table-cols-input" class="text-sm font-medium"
          >{$_('component.editor.table_insert.columns')}:</label
        >
        <input
          id="table-cols-input"
          type="number"
          bind:value={cols}
          bind:this={colsInput}
          min="1"
          max="20"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        />
      </div>
      <div class="space-y-2">
        <label for="table-rows-input" class="text-sm font-medium"
          >{$_('component.editor.table_insert.rows')}:</label
        >
        <input
          id="table-rows-input"
          type="number"
          bind:value={rows}
          min="1"
          max="50"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        />
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('dialog.cancel')}
    </button>
    <button
      type="button"
      onclick={handleInsert}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md"
    >
      {$_('component.editor.table_insert.insert')}
    </button>
  {/snippet}
</BaseDialog>
