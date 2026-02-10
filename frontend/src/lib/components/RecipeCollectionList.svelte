<script lang="ts">
  import { BookOpen, Pencil, Plus, Trash2, Users } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeCollection } from '$lib/api';

  interface Props {
    collections: RecipeCollection[];
    onEdit: (collection: RecipeCollection) => void;
    onDelete: (collectionId: number) => void;
    onCreate: () => void;
    onSelect: (collectionId: number) => void;
    onShare?: (collectionId: number) => void;
  }

  const { collections, onEdit, onDelete, onCreate, onSelect, onShare }: Props = $props();
</script>

<div class="space-y-2">
  {#if collections.length === 0}
    <p class="text-sm text-muted-foreground italic py-4 text-center">
      {$_('page.recipes.no_collections')}
    </p>
  {:else}
    {#each collections as coll (coll.id)}
      <div class="group flex items-center gap-2 px-3 py-2 rounded hover:bg-accent">
        <button onclick={() => onSelect(coll.id)} class="flex-1 flex items-center gap-2 text-left">
          {#if coll.color}
            <span class="w-3 h-3 rounded-full shrink-0" style="background-color: {coll.color}"
            ></span>
          {:else}
            <BookOpen size={14} class="text-muted-foreground shrink-0" />
          {/if}
          <span class="text-sm flex-1">{coll.name}</span>
          {#if coll.recipe_count}
            <span class="text-xs text-muted-foreground">{coll.recipe_count}</span>
          {/if}
        </button>

        <div class="flex gap-1 opacity-0 group-hover:opacity-100">
          {#if onShare}
            <button
              onclick={() => onShare(coll.id)}
              class="p-1 rounded hover:bg-accent"
              title={$_('sharing.collection_title')}
            >
              <Users size={12} />
            </button>
          {/if}
          <button
            onclick={() => onEdit(coll)}
            class="p-1 rounded hover:bg-accent"
            title={$_('common.edit')}
          >
            <Pencil size={12} />
          </button>
          <button
            onclick={() => onDelete(coll.id)}
            class="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive"
            title={$_('common.delete')}
          >
            <Trash2 size={12} />
          </button>
        </div>
      </div>
    {/each}
  {/if}

  <button
    onclick={onCreate}
    class="w-full flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground hover:text-foreground rounded hover:bg-accent border border-dashed border-border"
  >
    <Plus size={14} />
    {$_('page.recipes.create_collection')}
  </button>
</div>
