<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { Loader2, Save, Trash2, Plus } from 'lucide-svelte';
  import BaseDialog from './ui/BaseDialog.svelte';
  import type { GeneratedRecipe, GeneratedIngredient } from '$lib/api';
  import { saveGeneratedRecipe } from '$lib/api';

  interface Props {
    open: boolean;
    recipe: GeneratedRecipe;
    onClose: () => void;
    onSaved: (noteId: string) => void;
  }

  const { open, recipe, onClose, onSaved }: Props = $props();

  // Editable state (initialized from recipe prop)
  let title = $state('');
  let servings = $state(4);
  let prepTime = $state<number | null>(null);
  let cookTime = $state<number | null>(null);
  let difficulty = $state<string | null>(null);
  let ingredients = $state<GeneratedIngredient[]>([]);
  let instructions = $state('');

  let saving = $state(false);
  let error = $state<string | null>(null);

  // Initialize state when recipe changes
  $effect(() => {
    if (recipe && open) {
      title = recipe.title;
      servings = recipe.servings;
      prepTime = recipe.prep_time_minutes ?? null;
      cookTime = recipe.cook_time_minutes ?? null;
      difficulty = recipe.difficulty ?? null;
      ingredients = recipe.ingredients.map((i) => ({ ...i }));
      instructions = recipe.instructions;
      error = null;
    }
  });

  function addIngredient() {
    ingredients = [
      ...ingredients,
      { name: '', amount: null, unit: null, scalable: true, optional: false },
    ];
  }

  function removeIngredient(index: number) {
    ingredients = ingredients.filter((_, i) => i !== index);
  }

  async function handleSave() {
    if (!title.trim()) return;
    saving = true;
    error = null;

    try {
      const result = await saveGeneratedRecipe({
        title: title.trim(),
        instructions,
        servings,
        prep_time_minutes: prepTime,
        cook_time_minutes: cookTime,
        difficulty,
        ingredients: ingredients.filter((i) => i.name.trim()),
        folder_path: '/Rezepte',
      });
      onSaved(result.note_id);
    } catch (err: unknown) {
      error = err instanceof Error ? err.message : $_('page.recipes.suggestions.ai_error');
    } finally {
      saving = false;
    }
  }
</script>

<BaseDialog
  {open}
  title={$_('page.recipes.suggestions.preview_save')}
  {onClose}
  size="xl"
  scrollable
>
  {#snippet content()}
    {#if error}
      <div class="p-3 mb-4 bg-destructive/10 text-destructive rounded-md text-sm">
        {error}
      </div>
    {/if}

    <div class="space-y-4">
      <!-- Title -->
      <div>
        <label for="gen-title" class="block text-sm font-medium mb-1">
          {$_('page.recipes.suggestions.edit_title')}
        </label>
        <input
          id="gen-title"
          type="text"
          bind:value={title}
          class="w-full px-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>

      <!-- Metadata row -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div>
          <label for="gen-servings" class="block text-xs text-muted-foreground mb-1">
            {$_('page.recipes.suggestions.edit_servings')}
          </label>
          <input
            id="gen-servings"
            type="number"
            min="1"
            max="999"
            bind:value={servings}
            class="w-full px-2 py-1.5 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <label for="gen-prep" class="block text-xs text-muted-foreground mb-1">
            {$_('page.recipes.suggestions.edit_prep_time')}
          </label>
          <input
            id="gen-prep"
            type="number"
            min="0"
            bind:value={prepTime}
            class="w-full px-2 py-1.5 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <label for="gen-cook" class="block text-xs text-muted-foreground mb-1">
            {$_('page.recipes.suggestions.edit_cook_time')}
          </label>
          <input
            id="gen-cook"
            type="number"
            min="0"
            bind:value={cookTime}
            class="w-full px-2 py-1.5 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <label for="gen-difficulty" class="block text-xs text-muted-foreground mb-1">
            {$_('page.recipes.suggestions.edit_difficulty')}
          </label>
          <select
            id="gen-difficulty"
            bind:value={difficulty}
            class="w-full px-2 py-1.5 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value={null}>–</option>
            <option value="easy">{$_('page.recipes.difficulty_easy')}</option>
            <option value="medium">{$_('page.recipes.difficulty_medium')}</option>
            <option value="hard">{$_('page.recipes.difficulty_hard')}</option>
          </select>
        </div>
      </div>

      <!-- Ingredients -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <div class="block text-sm font-medium">
            {$_('page.recipes.suggestions.edit_ingredients')}
          </div>
          <button
            onclick={addIngredient}
            class="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
          >
            <Plus size={12} />
            {$_('page.recipes.add_ingredient')}
          </button>
        </div>
        <div class="space-y-1.5">
          {#each ingredients as ing, i (i)}
            <div class="flex items-center gap-2">
              <input
                type="number"
                step="any"
                placeholder="–"
                bind:value={ing.amount}
                class="w-16 px-2 py-1 bg-background border border-border rounded text-sm text-right focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="text"
                placeholder={$_('page.recipes.unit')}
                bind:value={ing.unit}
                class="w-16 px-2 py-1 bg-background border border-border rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="text"
                placeholder={$_('page.recipes.ingredient_name')}
                bind:value={ing.name}
                class="flex-1 px-2 py-1 bg-background border border-border rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <button
                onclick={() => removeIngredient(i)}
                class="p-1 text-muted-foreground hover:text-destructive"
              >
                <Trash2 size={14} />
              </button>
            </div>
          {/each}
        </div>
      </div>

      <!-- Instructions -->
      <div>
        <label for="gen-instructions" class="block text-sm font-medium mb-1">
          {$_('page.recipes.suggestions.edit_instructions')}
        </label>
        <textarea
          id="gen-instructions"
          bind:value={instructions}
          class="w-full min-h-[200px] p-3 bg-background border border-border rounded-md text-sm font-mono resize-y focus:outline-none focus:ring-1 focus:ring-primary"
        ></textarea>
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <button
      onclick={onClose}
      class="px-4 py-2 text-sm rounded-md hover:bg-accent transition-colors"
    >
      {$_('common.cancel')}
    </button>
    <button
      onclick={handleSave}
      disabled={saving || !title.trim()}
      class="flex items-center gap-2 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
    >
      {#if saving}
        <Loader2 size={14} class="animate-spin" />
        {$_('page.recipes.suggestions.saving')}
      {:else}
        <Save size={14} />
        {$_('page.recipes.suggestions.save_recipe')}
      {/if}
    </button>
  {/snippet}
</BaseDialog>
