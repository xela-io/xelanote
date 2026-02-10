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
    if (!isNaN(val) && val >= 1 && val <= 999) {
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

<div class="space-y-3">
  <div class="grid grid-cols-2 gap-3">
    <!-- Servings -->
    <div>
      <label for="recipe-servings" class="text-xs text-muted-foreground block mb-1">
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
        class="w-full px-2 py-1.5 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
      />
    </div>

    <!-- Difficulty -->
    <div>
      <label for="recipe-difficulty" class="text-xs text-muted-foreground block mb-1">
        {$_('page.recipes.difficulty')}
      </label>
      <select
        id="recipe-difficulty"
        bind:value={difficulty}
        onchange={scheduleUpdate}
        disabled={readonly}
        class="w-full px-2 py-1.5 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
      >
        {#each difficulties as d (d.value)}
          <option value={d.value}>
            {d.labelKey ? $_(d.labelKey) : '-'}
          </option>
        {/each}
      </select>
    </div>

    <!-- Prep Time -->
    <div>
      <label
        for="recipe-prep-time"
        class="text-xs text-muted-foreground flex items-center gap-1 mb-1"
      >
        <Clock size={12} />
        {$_('page.recipes.prep_time')}
      </label>
      <div class="flex items-center gap-1">
        <input
          id="recipe-prep-time"
          type="number"
          bind:value={prepTime}
          oninput={scheduleUpdate}
          disabled={readonly}
          min="0"
          placeholder="–"
          class="w-full px-2 py-1.5 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
        />
        <span class="text-xs text-muted-foreground">min</span>
      </div>
    </div>

    <!-- Cook Time -->
    <div>
      <label
        for="recipe-cook-time"
        class="text-xs text-muted-foreground flex items-center gap-1 mb-1"
      >
        <ChefHat size={12} />
        {$_('page.recipes.cook_time')}
      </label>
      <div class="flex items-center gap-1">
        <input
          id="recipe-cook-time"
          type="number"
          bind:value={cookTime}
          oninput={scheduleUpdate}
          disabled={readonly}
          min="0"
          placeholder="–"
          class="w-full px-2 py-1.5 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
        />
        <span class="text-xs text-muted-foreground">min</span>
      </div>
    </div>
  </div>

  <!-- Source URL -->
  <div>
    <label class="text-xs text-muted-foreground flex items-center gap-1 mb-1">
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
        class="w-full px-2 py-1.5 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
      />
      {#if sourceUrl && !readonly}
        <a
          href={sourceUrl}
          target="_blank"
          rel="noopener noreferrer"
          class="p-1.5 rounded hover:bg-accent shrink-0"
          title={$_('page.recipes.open_source')}
        >
          <ExternalLink size={14} />
        </a>
      {/if}
    </div>
  </div>
</div>
