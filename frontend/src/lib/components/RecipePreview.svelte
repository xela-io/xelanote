<script lang="ts">
  import { ChefHat, ChevronDown, Clock, ExternalLink, Users } from 'lucide-svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import type { RecipeImage, RecipeMetadata, ScaledIngredient } from '$lib/api';

  import RecipeScaleControl from './RecipeScaleControl.svelte';

  interface Props {
    title: string;
    metadata: RecipeMetadata | null;
    images?: RecipeImage[];
    scaledIngredients: ScaledIngredient[];
    content: string;
    targetServings: number;
    baseServings?: number;
    onServingsChange?: (servings: number) => void;
  }

  const {
    title,
    metadata,
    images = [],
    scaledIngredients,
    content,
    targetServings,
    baseServings,
    onServingsChange,
  }: Props = $props();

  const effectiveBaseServings = $derived(baseServings ?? metadata?.servings ?? 4);

  const totalTime = $derived(() => {
    const prep = metadata?.prep_time_minutes ?? 0;
    const cook = metadata?.cook_time_minutes ?? 0;
    return prep + cook;
  });

  // Group scaled ingredients by group_name
  const groupedIngredients = $derived(() => {
    const map = new SvelteMap<string, ScaledIngredient[]>();
    for (const ing of scaledIngredients) {
      const key = ing.group_name ?? '';
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(ing);
    }
    return map;
  });

  let ingredientsOpen = $state(true);

  function difficultyLabel(d: string | null | undefined): string {
    if (!d) return '';
    switch (d) {
      case 'easy':
        return $_('page.recipes.difficulty_easy');
      case 'medium':
        return $_('page.recipes.difficulty_medium');
      case 'hard':
        return $_('page.recipes.difficulty_hard');
      default:
        return d;
    }
  }
</script>

<div class="recipe-preview mx-auto max-w-3xl space-y-5 sm:space-y-6">
  <!-- Header -->
  <div class="ui-panel p-4 sm:p-5 space-y-3">
    <h1 class="text-2xl sm:text-3xl font-bold tracking-tight">{title}</h1>

    <!-- Meta info badges -->
    <div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
      {#if onServingsChange}
        <div class="rounded-full border border-border/60 bg-background/40 px-2.5 py-1">
          <RecipeScaleControl
            servings={targetServings}
            baseServings={effectiveBaseServings}
            onchange={onServingsChange}
          />
        </div>
      {:else}
        <span class="recipe-meta-chip">
          <Users size={14} />
          {targetServings}
          {$_('page.recipes.servings')}
        </span>
      {/if}
      {#if metadata?.prep_time_minutes}
        <span class="recipe-meta-chip">
          <Clock size={14} />
          {$_('page.recipes.prep_time')}: {metadata.prep_time_minutes} min
        </span>
      {/if}
      {#if metadata?.cook_time_minutes}
        <span class="recipe-meta-chip">
          <ChefHat size={14} />
          {$_('page.recipes.cook_time')}: {metadata.cook_time_minutes} min
        </span>
      {/if}
      {#if totalTime() > 0}
        <span class="recipe-meta-chip font-medium">
          {$_('page.recipes.total_time')}: {totalTime()} min
        </span>
      {/if}
      {#if metadata?.difficulty}
        <span class="recipe-meta-chip text-xs">
          {difficultyLabel(metadata.difficulty)}
        </span>
      {/if}
    </div>

    {#if metadata?.source_url}
      <a
        href={metadata.source_url}
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-1 rounded-md border border-border/70 px-2.5 py-1.5 text-sm text-primary hover:bg-accent/60"
      >
        <ExternalLink size={12} />
        {$_('page.recipes.source')}
      </a>
    {/if}
  </div>

  <!-- Images -->
  {#if images.length > 0}
    <div class="ui-panel p-3 sm:p-4">
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        {#each images as image (image.id)}
          <div>
            <img
              src={image.image_url}
              alt={image.caption || $_('page.recipes.images')}
              class="w-full aspect-square object-cover rounded-lg"
              loading="lazy"
            />
            {#if image.caption}
              <p class="text-xs text-muted-foreground mt-1">{image.caption}</p>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Cooking section: ingredients (sticky) + instructions -->
  {#if scaledIngredients.length > 0 || content}
    <div class="space-y-4">
      {#if scaledIngredients.length > 0}
        <section class="ui-panel overflow-hidden">
          <!-- Sticky ingredients header -->
          <button
            onclick={() => (ingredientsOpen = !ingredientsOpen)}
            class="sticky top-0 z-10 flex w-full cursor-pointer items-center justify-between border-b border-border/60 bg-background/75 px-4 py-3 text-left text-lg font-semibold backdrop-blur-sm"
            aria-expanded={ingredientsOpen}
          >
            <span>
              {$_('page.recipes.ingredients')}
              <span class="ml-1 text-sm font-normal text-muted-foreground">
                ({scaledIngredients.length})
              </span>
            </span>
            <ChevronDown
              size={20}
              class="transition-transform duration-200 {ingredientsOpen
                ? 'rotate-0'
                : '-rotate-90'}"
            />
          </button>

          <!-- Collapsible ingredients list -->
          {#if ingredientsOpen}
            <div class="p-4">
              {#each [...groupedIngredients().entries()] as [groupName, ingredients] (groupName)}
                {#if groupName}
                  <h3 class="ui-kicker mt-4 mb-2 first:mt-0">
                    {groupName}
                  </h3>
                {/if}
                <ul class="recipe-ingredient-list">
                  {#each ingredients as ing, ii (ii)}
                    <li class="recipe-ingredient-line" class:opacity-60={ing.optional}>
                      <span class="font-medium text-right shrink-0">
                        {ing.display_amount}
                      </span>
                      <span class="text-muted-foreground shrink-0">
                        {ing.unit ?? ''}
                      </span>
                      <span class="min-w-0">
                        {ing.name}
                        {#if ing.optional}
                          <span class="ml-1 text-xs text-muted-foreground"
                            >({$_('page.recipes.optional')})</span
                          >
                        {/if}
                      </span>
                    </li>
                  {/each}
                </ul>
              {/each}
            </div>
          {/if}
        </section>
      {/if}

      <!-- Instructions (rendered as markdown-like text) -->
      {#if content}
        <section class="ui-panel p-4 sm:p-5">
          <h2 class="mb-3 text-lg font-semibold tracking-tight">
            {$_('page.recipes.instructions')}
          </h2>
          <div class="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap leading-6">
            {content}
          </div>
        </section>
      {/if}
    </div>
  {/if}
</div>

<style>
  .recipe-meta-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    border-radius: 999px;
    border: 1px solid hsl(var(--border) / 0.6);
    background: hsl(var(--background) / 0.4);
    padding: 0.35rem 0.6rem;
  }

  .recipe-ingredient-list {
    display: grid;
    gap: 0.2rem;
  }

  .recipe-ingredient-line {
    display: grid;
    grid-template-columns: 4.25rem 3rem minmax(0, 1fr);
    align-items: start;
    gap: 0.5rem;
    padding: 0.45rem 0.2rem;
    border-bottom: 1px dashed hsl(var(--border) / 0.45);
    font-size: 0.92rem;
  }

  .recipe-ingredient-line:last-child {
    border-bottom: none;
  }

  @media (max-width: 639px) {
    .recipe-ingredient-line {
      grid-template-columns: 4rem 2.75rem minmax(0, 1fr);
      gap: 0.4rem;
      padding-inline: 0;
      font-size: 0.9rem;
    }
  }
</style>
