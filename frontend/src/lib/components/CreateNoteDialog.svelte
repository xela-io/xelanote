<script lang="ts">
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    open: boolean;
    folderPath: string;
    onClose: () => void;
    onCreate: (title: string) => void;
  }

  const { open, folderPath, onClose, onCreate }: Props = $props();

  let title = $state('');
  let titleInput: HTMLInputElement | null = null;

  $effect(() => {
    if (open) {
      title = '';
      tick().then(() => {
        titleInput?.focus();
      });
    }
  });

  function handleCreate() {
    if (title.trim()) {
      onCreate(title.trim());
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

      <div class="space-y-2">
        <label for="note-title-input" class="text-sm font-medium"
          >{$_('dialog.create_note.label')}:</label
        >
        <input
          id="note-title-input"
          type="text"
          bind:value={title}
          bind:this={titleInput}
          placeholder={$_('dialog.create_note.placeholder')}
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
      onclick={handleCreate}
      disabled={!title.trim()}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {$_('common.create')}
    </button>
  {/snippet}
</BaseDialog>
