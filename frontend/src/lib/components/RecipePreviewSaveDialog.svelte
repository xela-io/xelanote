<script lang="ts">
  import { Loader2, Plus, Save, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { GeneratedIngredient, GeneratedRecipe } from '$lib/api';
  import { saveGeneratedRecipe } from '$lib/api';

  import BaseDialog from './ui/BaseDialog.svelte';

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
        source_url: recipe.source_url ?? null,
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
      <div class="ui-alert ui-alert-danger mb-4 text-sm">
        {error}
      </div>
    {/if}

    <div class="space-y-4">
      <!-- Title -->
      <div>
        <label for="gen-title" class="ui-label mb-1">
          {$_('page.recipes.suggestions.edit_title')}
        </label>
        <input id="gen-title" type="text" bind:value={title} class="ui-input text-sm" />
      </div>

      <!-- Metadata row -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div>
          <label for="gen-servings" class="ui-label text-xs mb-1">
            {$_('page.recipes.suggestions.edit_servings')}
          </label>
          <input
            id="gen-servings"
            type="number"
            min="1"
            max="999"
            bind:value={servings}
            class="ui-input text-sm px-2 py-1.5"
          />
        </div>
        <div>
          <label for="gen-prep" class="ui-label text-xs mb-1">
            {$_('page.recipes.suggestions.edit_prep_time')}
          </label>
          <input
            id="gen-prep"
            type="number"
            min="0"
            bind:value={prepTime}
            class="ui-input text-sm px-2 py-1.5"
          />
        </div>
        <div>
          <label for="gen-cook" class="ui-label text-xs mb-1">
            {$_('page.recipes.suggestions.edit_cook_time')}
          </label>
          <input
            id="gen-cook"
            type="number"
            min="0"
            bind:value={cookTime}
            class="ui-input text-sm px-2 py-1.5"
          />
        </div>
        <div>
          <label for="gen-difficulty" class="ui-label text-xs mb-1">
            {$_('page.recipes.suggestions.edit_difficulty')}
          </label>
          <select id="gen-difficulty" bind:value={difficulty} class="ui-select text-sm px-2 py-1.5">
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
          <button onclick={addIngredient} class="ui-button ui-button-ghost text-xs px-2 py-1">
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
                class="ui-input w-16 px-2 py-1 text-sm text-right"
              />
              <input
                type="text"
                placeholder={$_('page.recipes.unit')}
                bind:value={ing.unit}
                class="ui-input w-16 px-2 py-1 text-sm"
              />
              <input
                type="text"
                placeholder={$_('page.recipes.ingredient_name')}
                bind:value={ing.name}
                class="ui-input flex-1 px-2 py-1 text-sm"
              />
              <button
                onclick={() => removeIngredient(i)}
                class="ui-icon-button ui-icon-button-sm text-muted-foreground hover:text-destructive"
              >
                <Trash2 size={14} />
              </button>
            </div>
          {/each}
        </div>
      </div>

      <!-- Instructions -->
      <div>
        <label for="gen-instructions" class="ui-label mb-1">
          {$_('page.recipes.suggestions.edit_instructions')}
        </label>
        <textarea
          id="gen-instructions"
          bind:value={instructions}
          class="ui-textarea min-h-[200px] p-3 text-sm font-mono resize-y"
        ></textarea>
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <button onclick={onClose} class="ui-button ui-button-secondary text-sm">
      {$_('common.cancel')}
    </button>
    <button
      onclick={handleSave}
      disabled={saving || !title.trim()}
      class="ui-button ui-button-primary text-sm"
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
