<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

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
      <div class="space-y-2">
        <label for="folder-path-input" class="text-sm font-medium"
          >{$_('dialog.create_folder.label')}:</label
        >
        <input
          id="folder-path-input"
          type="text"
          bind:value={path}
          bind:this={pathInput}
          placeholder="/Projects/Work"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring font-mono"
        />
        <p class="text-xs text-muted-foreground">
          {$_('dialog.create_folder.hint')}
        </p>
      </div>

      {#if errorMessage}
        <div class="text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-2 rounded">
          {errorMessage}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('dialog.cancel')}
    </button>
    <button
      type="button"
      onclick={handleCreate}
      disabled={!path.trim()}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {$_('common.create')}
    </button>
  {/snippet}
</BaseDialog>
