<script lang="ts">
  import { ArrowLeft, BookOpen, ChefHat, Edit, Eye, Loader2 } from 'lucide-svelte';
  import { onDestroy, onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  const collectionId = $derived(Number(page.params.id));

  // Find the collection info from loaded shared collections
  const collectionInfo = $derived(
    recipes.getSharedCollections().find((c) => c.id === collectionId)
  );
  const items = $derived(recipes.getSharedCollectionItems());
  const loading = $derived(recipes.getSharedCollectionsLoading());

  onMount(async () => {
    // Load shared collections if not yet loaded (e.g. direct navigation)
    if (recipes.getSharedCollections().length === 0) {
      await recipes.loadSharedCollections();
    }
    await recipes.loadSharedCollectionItems(collectionId);
  });

  onDestroy(() => {
    // Clear items when leaving
  });

  function handleRecipeClick(noteId: string) {
    goto(`/note/${noteId}`);
    ui.closeSidebarOnMobile();
  }

  function handleBack() {
    goto('/recipes');
  }
</script>

<svelte:head>
  <title>{collectionInfo?.name ?? $_('page.recipes.collections')} - xelanote</title>
</svelte:head>

<div class="h-full flex flex-col">
  <!-- Header -->
  <div class="px-4 py-3 sm:px-6 sm:py-4 border-b border-border shrink-0">
    <div class="flex items-center gap-2 mb-2">
      <MobileSidebarInlineToggle />
      <button
        onclick={handleBack}
        class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft size={16} />
        {$_('sharing.back_to_shared')}
      </button>
    </div>

    {#if collectionInfo}
      <div class="flex items-center gap-3">
        {#if collectionInfo.color}
          <span
            class="w-4 h-4 rounded-full shrink-0"
            style="background-color: {collectionInfo.color}"
          ></span>
        {:else}
          <BookOpen size={20} class="text-muted-foreground shrink-0" />
        {/if}
        <div>
          <h1 class="text-xl font-bold">{collectionInfo.name}</h1>
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <span>{$_('sharing.shared_by', { values: { name: collectionInfo.shared_by } })}</span>
            <span
              class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {collectionInfo.share_role ===
              'editor'
                ? 'bg-primary/10 text-primary'
                : 'bg-muted text-muted-foreground'}"
            >
              {#if collectionInfo.share_role === 'editor'}
                <Edit size={10} />
                {$_('sharing.editable')}
              {:else}
                <Eye size={10} />
                {$_('sharing.read_only')}
              {/if}
            </span>
          </div>
        </div>
      </div>
      {#if collectionInfo.description}
        <p class="text-sm text-muted-foreground mt-2">{collectionInfo.description}</p>
      {/if}
    {:else}
      <h1 class="text-xl font-bold">{$_('page.recipes.collections')}</h1>
    {/if}
  </div>

  <!-- Recipe list -->
  {#if loading}
    <div class="flex items-center justify-center flex-1">
      <Loader2 class="w-8 h-8 animate-spin" />
    </div>
  {:else if items.length === 0}
    <div class="flex-1 flex items-center justify-center">
      <div class="text-center text-muted-foreground">
        <ChefHat class="w-12 h-12 mx-auto mb-3 opacity-50" />
        <p>{$_('page.recipes.no_recipes')}</p>
      </div>
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto p-6">
      <div class="space-y-2 max-w-2xl">
        {#each items as recipe (recipe.id)}
          <button
            onclick={() => handleRecipeClick(recipe.id)}
            class="w-full text-left p-3 rounded-lg border border-border hover:bg-accent transition-colors"
          >
            <span class="font-medium text-sm truncate">{recipe.title}</span>
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>
