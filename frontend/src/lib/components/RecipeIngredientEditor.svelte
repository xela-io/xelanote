<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { SvelteMap } from 'svelte/reactivity';
  import { Plus } from 'lucide-svelte';
  import type { RecipeIngredient, ScaledIngredient } from '$lib/api';
  import RecipeIngredientRow from './RecipeIngredientRow.svelte';

  interface Props {
    ingredients: RecipeIngredient[];
    scaledIngredients?: ScaledIngredient[];
    readonly?: boolean;
    onupdate: (ingredients: RecipeIngredient[]) => void;
  }

  const { ingredients, scaledIngredients, readonly = false, onupdate }: Props = $props();

  // Group ingredients by group_name
  const groups = $derived(() => {
    const map = new SvelteMap<string, RecipeIngredient[]>();
    for (const ing of ingredients) {
      const key = ing.group_name ?? '';
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(ing);
    }
    return map;
  });

  function addIngredient(groupName?: string) {
    const newIng: RecipeIngredient = {
      name: '',
      amount: null,
      unit: null,
      group_name: groupName || null,
      display_order: ingredients.length,
      optional: false,
      scalable: true,
    };
    onupdate([...ingredients, newIng]);
  }

  function updateIngredient(index: number, updated: RecipeIngredient) {
    const newList = [...ingredients];
    newList[index] = updated;
    onupdate(newList);
  }

  function removeIngredient(index: number) {
    const newList = ingredients.filter((_, i) => i !== index);
    // Re-index display_order
    newList.forEach((ing, i) => (ing.display_order = i));
    onupdate(newList);
  }

  function addGroup() {
    const name = prompt($_('page.recipes.group_name_prompt'));
    if (name?.trim()) {
      addIngredient(name.trim());
    }
  }

  function getScaledForIndex(index: number): ScaledIngredient | null {
    if (!scaledIngredients || index >= scaledIngredients.length) return null;
    return scaledIngredients[index];
  }
</script>

<div class="space-y-2">
  {#if ingredients.length === 0}
    <p class="text-sm text-muted-foreground italic py-2">{$_('page.recipes.no_ingredients')}</p>
  {:else}
    {@const grouped = groups()}
    {#each [...grouped.entries()] as [groupName, groupIngredients] (groupName)}
      {#if groupName}
        <div class="mt-3 first:mt-0">
          <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">
            {groupName}
          </h4>
        </div>
      {/if}

      {#each groupIngredients as ingredient, gi (gi)}
        {@const globalIndex = ingredients.indexOf(ingredient)}
        <RecipeIngredientRow
          {ingredient}
          {readonly}
          scaled={getScaledForIndex(globalIndex)}
          onupdate={(updated) => updateIngredient(globalIndex, updated)}
          onremove={() => removeIngredient(globalIndex)}
        />
      {/each}
    {/each}
  {/if}

  {#if !readonly}
    <div class="flex gap-2 pt-2">
      <button
        onclick={() => addIngredient()}
        class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground px-2 py-1 rounded hover:bg-accent"
      >
        <Plus size={14} />
        {$_('page.recipes.add_ingredient')}
      </button>
      <button
        onclick={addGroup}
        class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground px-2 py-1 rounded hover:bg-accent"
      >
        <Plus size={14} />
        {$_('page.recipes.add_group')}
      </button>
    </div>
  {/if}
</div>
