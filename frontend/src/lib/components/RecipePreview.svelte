<script lang="ts">
  import { ChefHat, Clock, ExternalLink, Users } from 'lucide-svelte';
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

<div class="recipe-preview space-y-6 max-w-2xl mx-auto">
  <!-- Header -->
  <div class="space-y-2">
    <h1 class="text-2xl font-bold">{title}</h1>

    <!-- Meta info badges -->
    <div class="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
      {#if onServingsChange}
        <RecipeScaleControl
          servings={targetServings}
          baseServings={effectiveBaseServings}
          onchange={onServingsChange}
        />
      {:else}
        <span class="flex items-center gap-1">
          <Users size={14} />
          {targetServings}
          {$_('page.recipes.servings')}
        </span>
      {/if}
      {#if metadata?.prep_time_minutes}
        <span class="flex items-center gap-1">
          <Clock size={14} />
          {$_('page.recipes.prep_time')}: {metadata.prep_time_minutes} min
        </span>
      {/if}
      {#if metadata?.cook_time_minutes}
        <span class="flex items-center gap-1">
          <ChefHat size={14} />
          {$_('page.recipes.cook_time')}: {metadata.cook_time_minutes} min
        </span>
      {/if}
      {#if totalTime() > 0}
        <span class="font-medium">
          {$_('page.recipes.total_time')}: {totalTime()} min
        </span>
      {/if}
      {#if metadata?.difficulty}
        <span class="px-2 py-0.5 rounded-full text-xs bg-accent">
          {difficultyLabel(metadata.difficulty)}
        </span>
      {/if}
    </div>

    {#if metadata?.source_url}
      <a
        href={metadata.source_url}
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
      >
        <ExternalLink size={12} />
        {$_('page.recipes.source')}
      </a>
    {/if}
  </div>

  <!-- Images -->
  {#if images.length > 0}
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
  {/if}

  <!-- Ingredients -->
  {#if scaledIngredients.length > 0}
    <div>
      <h2 class="text-lg font-semibold mb-3">{$_('page.recipes.ingredients')}</h2>
      {#each [...groupedIngredients().entries()] as [groupName, ingredients] (groupName)}
        {#if groupName}
          <h3 class="text-sm font-medium text-muted-foreground mt-3 mb-1">{groupName}</h3>
        {/if}
        <ul class="space-y-1">
          {#each ingredients as ing, ii (ii)}
            <li class="flex gap-2 text-sm" class:opacity-60={ing.optional}>
              <span class="font-medium w-20 text-right shrink-0">
                {ing.display_amount}
              </span>
              <span class="text-muted-foreground w-12 shrink-0">
                {ing.unit ?? ''}
              </span>
              <span>
                {ing.name}
                {#if ing.optional}
                  <span class="text-xs text-muted-foreground">({$_('page.recipes.optional')})</span>
                {/if}
              </span>
            </li>
          {/each}
        </ul>
      {/each}
    </div>
  {/if}

  <!-- Instructions (rendered as markdown-like text) -->
  {#if content}
    <div>
      <h2 class="text-lg font-semibold mb-3">{$_('page.recipes.instructions')}</h2>
      <div class="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap">
        {content}
      </div>
    </div>
  {/if}
</div>
