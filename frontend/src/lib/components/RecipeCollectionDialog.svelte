<script lang="ts">
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeCollection } from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

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
      <div class="space-y-2">
        <label for="coll-name" class="text-sm font-medium"
          >{$_('page.recipes.collection_name')}</label
        >
        <input
          id="coll-name"
          type="text"
          bind:value={name}
          bind:this={nameInput}
          placeholder={$_('page.recipes.collection_name_placeholder')}
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        />
      </div>

      <div class="space-y-2">
        <label for="coll-desc" class="text-sm font-medium"
          >{$_('page.recipes.collection_description')}</label
        >
        <textarea
          id="coll-desc"
          bind:value={description}
          placeholder={$_('page.recipes.collection_description_placeholder')}
          rows="2"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring resize-none"
        ></textarea>
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('dialog.cancel')}
    </button>
    <button
      type="button"
      onclick={handleSave}
      disabled={!name.trim()}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {collection ? $_('common.save') : $_('common.create')}
    </button>
  {/snippet}
</BaseDialog>
