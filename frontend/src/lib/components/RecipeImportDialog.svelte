<script lang="ts">
  import { Camera, Globe, Loader2, WandSparkles } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';
  import { locale } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { GeneratedRecipe } from '$lib/api';
  import { importRecipeFromImage, importRecipeFromURL } from '$lib/api';

  import RecipePreviewSaveDialog from './RecipePreviewSaveDialog.svelte';
  import BaseDialog from './ui/BaseDialog.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  const { open, onClose }: Props = $props();

  let tab = $state<'image' | 'url'>('image');
  let loading = $state(false);
  let error = $state<string | null>(null);
  let imageFile = $state<File | null>(null);
  let imagePreview = $state<string | null>(null);
  let sourceURL = $state('');
  let previewRecipe = $state<GeneratedRecipe | null>(null);

  const currentLocale = $derived($locale?.substring(0, 2) ?? 'en');

  $effect(() => {
    if (!open) {
      loading = false;
      error = null;
      imageFile = null;
      imagePreview = null;
      sourceURL = '';
      previewRecipe = null;
      tab = 'image';
    }
  });

  function mapError(err: unknown): string {
    const apiErr = err as { status?: number; message?: string };
    const status = apiErr?.status;

    if (status === 424) {
      const msg = apiErr?.message || '';
      if (msg.includes('vision')) return $_('page.recipes.suggestions.no_vision_provider');
      return $_('page.recipes.suggestions.no_provider');
    }
    if (status === 422) return $_('page.recipes.import.no_recipe_found');
    if (status === 400) return $_('page.recipes.import.invalid_input');
    if (status === 502 || status === 504) return $_('page.recipes.import.fetch_failed');
    return $_('page.recipes.import.extraction_failed');
  }

  function handleImageChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    imageFile = file;
    imagePreview = URL.createObjectURL(file);
    error = null;
  }

  async function extractFromImage() {
    if (!imageFile || loading) return;
    loading = true;
    error = null;
    try {
      previewRecipe = await importRecipeFromImage(imageFile, currentLocale);
    } catch (err: unknown) {
      error = mapError(err);
    } finally {
      loading = false;
    }
  }

  async function extractFromURL() {
    const url = sourceURL.trim();
    if (!url || loading) return;
    loading = true;
    error = null;
    try {
      previewRecipe = await importRecipeFromURL(url, currentLocale);
      if (previewRecipe) {
        previewRecipe.source_url = url;
      }
    } catch (err: unknown) {
      error = mapError(err);
    } finally {
      loading = false;
    }
  }
</script>

<BaseDialog {open} title={$_('page.recipes.import.title')} {onClose} size="lg" scrollable>
  {#snippet content()}
    <div class="space-y-4">
      <div class="inline-flex rounded-md border border-border p-1 bg-accent/40">
        <button
          onclick={() => (tab = 'image')}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded"
          class:bg-background={tab === 'image'}
          class:shadow-sm={tab === 'image'}
        >
          <Camera size={14} />
          {$_('page.recipes.import.from_image')}
        </button>
        <button
          onclick={() => (tab = 'url')}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded"
          class:bg-background={tab === 'url'}
          class:shadow-sm={tab === 'url'}
        >
          <Globe size={14} />
          {$_('page.recipes.import.from_url')}
        </button>
      </div>

      {#if error}
        <div class="p-3 bg-destructive/10 text-destructive rounded-md text-sm">
          {error}
        </div>
      {/if}

      {#if tab === 'image'}
        <div class="space-y-3">
          <label
            class="block rounded-md border border-dashed border-border p-6 text-center cursor-pointer hover:bg-accent/40 transition-colors"
          >
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onchange={handleImageChange}
              class="hidden"
            />
            <div class="text-sm text-muted-foreground">
              {$_('page.recipes.import.drop_or_click')}
            </div>
            <div class="text-xs text-muted-foreground mt-1">
              {$_('page.recipes.suggestions.photo_formats')}
            </div>
          </label>

          {#if imagePreview}
            <img src={imagePreview} alt="Preview" class="w-40 h-40 rounded-md object-cover" />
          {/if}

          <button
            onclick={extractFromImage}
            disabled={!imageFile || loading}
            class="inline-flex items-center gap-2 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            {#if loading}
              <Loader2 size={14} class="animate-spin" />
              {$_('page.recipes.import.extracting')}
            {:else}
              <WandSparkles size={14} />
              {$_('page.recipes.import.extract')}
            {/if}
          </button>
        </div>
      {:else}
        <div class="space-y-3">
          <input
            type="url"
            bind:value={sourceURL}
            placeholder={$_('page.recipes.import.url_placeholder')}
            class="w-full px-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />

          <button
            onclick={extractFromURL}
            disabled={!sourceURL.trim() || loading}
            class="inline-flex items-center gap-2 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            {#if loading}
              <Loader2 size={14} class="animate-spin" />
              {$_('page.recipes.import.extracting')}
            {:else}
              <WandSparkles size={14} />
              {$_('page.recipes.import.extract')}
            {/if}
          </button>
        </div>
      {/if}
    </div>
  {/snippet}
</BaseDialog>

{#if previewRecipe}
  <RecipePreviewSaveDialog
    open={!!previewRecipe}
    recipe={previewRecipe}
    onClose={() => (previewRecipe = null)}
    onSaved={(noteId) => {
      previewRecipe = null;
      onClose();
      goto(`/note/${noteId}`);
    }}
  />
{/if}
