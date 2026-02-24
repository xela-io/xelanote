<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeCollection } from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';

  interface Props {
    open: boolean;
    collection?: RecipeCollection | null;
    onClose: () => void;
    onSave: (name: string, description: string | null, color: string | null) => void;
  }

  const { open, collection = null, onClose, onSave }: Props = $props();

  let name = $state('');
  let description = $state('');
  let color = $state('');
  let nameInput: HTMLInputElement | null = null;

  $effect(() => {
    if (open) {
      name = collection?.name ?? '';
      description = collection?.description ?? '';
      color = collection?.color ?? '';
      tick().then(() => nameInput?.focus());
    }
  });

  function handleSave() {
    if (!name.trim()) return;
    onSave(name.trim(), description.trim() || null, color.trim() || null);
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleSave();
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

<BaseDialog
  {open}
  title={collection ? $_('page.recipes.edit_collection') : $_('page.recipes.create_collection')}
  {onClose}
  size="sm"
>
  {#snippet content()}
    <div class="space-y-4">
      <DialogField forId="coll-name" label={$_('page.recipes.collection_name')}>
        <input
          id="coll-name"
          type="text"
          bind:value={name}
          bind:this={nameInput}
          placeholder={$_('page.recipes.collection_name_placeholder')}
          class="ui-input"
        />
      </DialogField>

      <DialogField forId="coll-desc" label={$_('page.recipes.collection_description')}>
        <textarea
          id="coll-desc"
          bind:value={description}
          placeholder={$_('page.recipes.collection_description_placeholder')}
          rows="2"
          class="ui-textarea resize-none"
        ></textarea>
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
        onclick={handleSave}
        disabled={!name.trim()}
        class="ui-button ui-button-primary text-sm"
      >
        {collection ? $_('common.save') : $_('common.create')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
