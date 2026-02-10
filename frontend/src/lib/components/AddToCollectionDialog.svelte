<script lang="ts">
  import { Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeCollection } from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    open: boolean;
    collections: RecipeCollection[];
    currentCollectionIds: number[];
    onClose: () => void;
    onAdd: (collectionId: number) => void;
    onCreateNew: () => void;
  }

  const { open, collections, currentCollectionIds, onClose, onAdd, onCreateNew }: Props = $props();

  const availableCollections = $derived(
    collections.filter((c) => !currentCollectionIds.includes(c.id))
  );
</script>

<BaseDialog {open} title={$_('page.recipes.add_to_collection')} {onClose} size="sm">
  {#snippet content()}
    <div class="space-y-2">
      {#if availableCollections.length === 0}
        <p class="text-sm text-muted-foreground italic py-4 text-center">
          {$_('page.recipes.no_available_collections')}
        </p>
      {:else}
        {#each availableCollections as coll (coll.id)}
          <button
            onclick={() => {
              onAdd(coll.id);
              onClose();
            }}
            class="w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded hover:bg-accent"
          >
            {#if coll.color}
              <span class="w-3 h-3 rounded-full shrink-0" style="background-color: {coll.color}"
              ></span>
            {/if}
            <span class="flex-1">{coll.name}</span>
            {#if coll.recipe_count}
              <span class="text-xs text-muted-foreground">{coll.recipe_count}</span>
            {/if}
          </button>
        {/each}
      {/if}

      <button
        onclick={() => {
          onClose();
          onCreateNew();
        }}
        class="w-full flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground hover:text-foreground rounded hover:bg-accent border border-dashed border-border mt-2"
      >
        <Plus size={14} />
        {$_('page.recipes.create_collection')}
      </button>
    </div>
  {/snippet}
</BaseDialog>
