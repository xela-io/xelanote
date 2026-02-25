<script lang="ts">
  import { ChefHat, Clock, ExternalLink } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeMetadata } from '$lib/api';

  interface Props {
    metadata: RecipeMetadata | null;
    readonly?: boolean;
    onupdate: (data: {
      servings: number;
      prep_time_minutes?: number | null;
      cook_time_minutes?: number | null;
      source_url?: string | null;
      difficulty?: string | null;
    }) => void;
  }

  const { metadata, readonly = false, onupdate }: Props = $props();

  const MIN_SERVINGS = 1;
  const MAX_SERVINGS = 999;

  let servings = $state(4);
  let prepTime = $state('');
  let cookTime = $state('');
  let sourceUrl = $state('');
  let difficulty = $state('');
  let lastMetadataId = $state('');

  // Sync when metadata changes from outside
  $effect(() => {
    const id = metadata?.note_id ?? '';
    const updatedAt = metadata?.updated_at ?? '';
    const key = `${id}:${updatedAt}`;
    if (key !== lastMetadataId) {
      lastMetadataId = key;
      servings = metadata?.servings ?? 4;
      prepTime = metadata?.prep_time_minutes?.toString() ?? '';
      cookTime = metadata?.cook_time_minutes?.toString() ?? '';
      sourceUrl = metadata?.source_url ?? '';
      difficulty = metadata?.difficulty ?? '';
    }
  });

  let saveTimeout: ReturnType<typeof setTimeout> | null = null;

  function scheduleUpdate() {
    if (readonly) return;
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      onupdate({
        servings,
        prep_time_minutes: prepTime ? parseInt(prepTime) : null,
        cook_time_minutes: cookTime ? parseInt(cookTime) : null,
        source_url: sourceUrl || null,
        difficulty: difficulty || null,
      });
    }, 1000);
  }

  function handleServingsChange(e: Event) {
    const val = parseInt((e.target as HTMLInputElement).value);
    if (!isNaN(val) && val >= MIN_SERVINGS && val <= MAX_SERVINGS) {
      servings = val;
      scheduleUpdate();
    }
  }

  const difficulties = [
    { value: '', label: '' },
    { value: 'easy', labelKey: 'page.recipes.difficulty_easy' },
    { value: 'medium', labelKey: 'page.recipes.difficulty_medium' },
    { value: 'hard', labelKey: 'page.recipes.difficulty_hard' },
  ];
</script>

<div class="ui-fieldset">
  <div class="ui-form-grid-2">
    <!-- Servings -->
    <div class="ui-form-row">
      <label for="recipe-servings" class="ui-label">
        {$_('page.recipes.servings')}
      </label>
      <input
        id="recipe-servings"
        type="number"
        value={servings}
        oninput={handleServingsChange}
        disabled={readonly}
        min="1"
        max="999"
        class="ui-input"
      />
    </div>

    <!-- Difficulty -->
    <div class="ui-form-row">
      <label for="recipe-difficulty" class="ui-label">
        {$_('page.recipes.difficulty')}
      </label>
      <select
        id="recipe-difficulty"
        bind:value={difficulty}
        onchange={scheduleUpdate}
        disabled={readonly}
        class="ui-select"
      >
        {#each difficulties as d (d.value)}
          <option value={d.value}>
            {d.labelKey ? $_(d.labelKey) : '-'}
          </option>
        {/each}
      </select>
    </div>

    <!-- Prep Time -->
    <div class="ui-form-row">
      <label for="recipe-prep-time" class="ui-label mb-0 flex items-center gap-1">
        <Clock size={12} />
        {$_('page.recipes.prep_time')}
      </label>
      <div class="ui-control-row">
        <input
          id="recipe-prep-time"
          type="number"
          bind:value={prepTime}
          oninput={scheduleUpdate}
          disabled={readonly}
          min="0"
          placeholder="–"
          class="ui-input w-full text-sm"
        />
        <span class="ui-form-help min-w-7 text-right">min</span>
      </div>
    </div>

    <!-- Cook Time -->
    <div class="ui-form-row">
      <label for="recipe-cook-time" class="ui-label mb-0 flex items-center gap-1">
        <ChefHat size={12} />
        {$_('page.recipes.cook_time')}
      </label>
      <div class="ui-control-row">
        <input
          id="recipe-cook-time"
          type="number"
          bind:value={cookTime}
          oninput={scheduleUpdate}
          disabled={readonly}
          min="0"
          placeholder="–"
          class="ui-input w-full text-sm"
        />
        <span class="ui-form-help min-w-7 text-right">min</span>
      </div>
    </div>
  </div>

  <!-- Source URL -->
  <div class="ui-form-row">
    <label class="ui-label mb-0 flex items-center gap-1">
      <ExternalLink size={12} />
      {$_('page.recipes.source_url')}
    </label>
    <div class="flex gap-1">
      <input
        type="url"
        bind:value={sourceUrl}
        oninput={scheduleUpdate}
        disabled={readonly}
        placeholder="https://..."
        class="ui-input"
      />
      {#if sourceUrl && !readonly}
        <a
          href={sourceUrl}
          target="_blank"
          rel="noopener noreferrer"
          class="ui-icon-button shrink-0 p-2"
          title={$_('page.recipes.open_source')}
        >
          <ExternalLink size={14} />
        </a>
      {/if}
    </div>
  </div>
</div>

<style>
  @media (max-width: 639px) {
    :global(.ui-fieldset) {
      gap: 0.7rem;
    }

    :global(.ui-form-grid-2) {
      gap: 0.65rem;
    }

    :global(.ui-form-row) {
      gap: 0.22rem;
    }

    :global(.ui-label) {
      font-size: 0.8rem;
      margin-bottom: 0.2rem;
    }

    :global(.ui-control-row) {
      gap: 0.3rem;
    }

    :global(.ui-form-help) {
      font-size: 0.72rem;
    }

    :global(.ui-input),
    :global(.ui-select) {
      min-height: 2.5rem;
      padding-block: 0.45rem;
    }
  }
</style>
