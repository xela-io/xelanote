<script lang="ts">
  import { ChevronDown, ChevronUp, Loader2, Plus, X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeImage } from '$lib/api';
  import * as recipeStore from '$lib/stores/recipes.svelte';

  interface Props {
    images: RecipeImage[];
    noteId: string;
    readonly?: boolean;
  }

  const { images, noteId, readonly = false }: Props = $props();

  let uploading = $state(false);
  let uploadProgress = $state('');
  let lightboxImage = $state<RecipeImage | null>(null);
  let fileInput = $state<HTMLInputElement | null>(null);
  let reorderTimeout: ReturnType<typeof setTimeout> | null = null;

  function handleFileSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files?.length) return;

    const files = Array.from(input.files);
    uploadFiles(files);
    input.value = '';
  }

  async function uploadFiles(files: File[]) {
    uploading = true;
    uploadProgress = '';
    try {
      await recipeStore.addImages(noteId, files, (current, total) => {
        uploadProgress = `${current}/${total}`;
      });
    } finally {
      uploading = false;
      uploadProgress = '';
    }
  }

  async function handleDelete(imageId: number) {
    await recipeStore.deleteImage(noteId, imageId);
  }

  function handleCaptionBlur(imageId: number, value: string) {
    const caption = value.trim() || null;
    recipeStore.updateImageCaption(noteId, imageId, caption);
  }

  function moveImage(index: number, direction: -1 | 1) {
    const newIndex = index + direction;
    if (newIndex < 0 || newIndex >= images.length) return;

    const ids = images.map((img) => img.id);
    [ids[index], ids[newIndex]] = [ids[newIndex], ids[index]];

    if (reorderTimeout) clearTimeout(reorderTimeout);
    reorderTimeout = setTimeout(() => {
      recipeStore.reorderImages(noteId, ids);
    }, 500);
  }

  function openLightbox(image: RecipeImage) {
    lightboxImage = image;
  }

  function closeLightbox() {
    lightboxImage = null;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && lightboxImage) {
      closeLightbox();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="recipe-image-gallery" class:no-images={images.length === 0}>
  <div class="image-grid grid grid-cols-2 sm:grid-cols-3 gap-2">
    {#each images as image, index (image.id)}
      <div class="image-card">
        <div class="image-wrapper">
          <button type="button" class="image-button" onclick={() => openLightbox(image)}>
            <img
              src={image.image_url}
              alt={image.caption || $_('page.recipes.images')}
              class="thumbnail"
              loading="lazy"
            />
          </button>
          {#if !readonly}
            <button
              type="button"
              class="delete-button"
              title={$_('page.recipes.delete_image')}
              onclick={() => handleDelete(image.id)}
            >
              <X size={14} />
            </button>
            <div class="reorder-buttons">
              {#if index > 0}
                <button type="button" class="reorder-button" onclick={() => moveImage(index, -1)}>
                  <ChevronUp size={14} />
                </button>
              {/if}
              {#if index < images.length - 1}
                <button type="button" class="reorder-button" onclick={() => moveImage(index, 1)}>
                  <ChevronDown size={14} />
                </button>
              {/if}
            </div>
          {/if}
        </div>
        {#if !readonly}
          <input
            type="text"
            class="caption-input"
            placeholder={$_('page.recipes.image_caption_placeholder')}
            value={image.caption ?? ''}
            onblur={(e) => handleCaptionBlur(image.id, (e.target as HTMLInputElement).value)}
          />
        {:else if image.caption}
          <span class="caption-text">{image.caption}</span>
        {/if}
      </div>
    {/each}

    {#if !readonly}
      <button
        type="button"
        class="upload-card"
        onclick={() => fileInput?.click()}
        disabled={uploading}
      >
        {#if uploading}
          <Loader2 size={24} class="animate-spin" />
          <span class="upload-label">{uploadProgress}</span>
        {:else}
          <Plus size={24} />
          <span class="upload-label">{$_('page.recipes.add_image')}</span>
        {/if}
      </button>
      <input
        bind:this={fileInput}
        type="file"
        accept="image/*"
        multiple
        class="hidden"
        onchange={handleFileSelect}
      />
    {/if}
  </div>
</div>

{#if lightboxImage}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="lightbox-overlay" onclick={closeLightbox}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="lightbox-content" onclick={(e) => e.stopPropagation()}>
      <img
        src={lightboxImage.image_url}
        alt={lightboxImage.caption || $_('page.recipes.images')}
        class="lightbox-image"
      />
      {#if lightboxImage.caption}
        <p class="lightbox-caption">{lightboxImage.caption}</p>
      {/if}
      <button type="button" class="lightbox-close" onclick={closeLightbox}>
        <X size={24} />
      </button>
    </div>
  </div>
{/if}

<style>
  .recipe-image-gallery {
    margin-bottom: 0;
  }

  .image-card {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .image-wrapper {
    position: relative;
    aspect-ratio: 1;
    overflow: hidden;
    border-radius: var(--radius-lg);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
  }

  .image-button {
    width: 100%;
    height: 100%;
    padding: 0;
    border: none;
    cursor: pointer;
    background: none;
  }

  .thumbnail {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .delete-button {
    position: absolute;
    top: 0.25rem;
    right: 0.25rem;
    padding: 0.25rem;
    border-radius: 50%;
    background: color-mix(in oklch, var(--color-danger) 90%, transparent);
    color: white;
    border: none;
    cursor: pointer;
    opacity: 0;
    transition: opacity var(--duration-fast) var(--ease-default);
  }

  .image-wrapper:hover .delete-button {
    opacity: 1;
  }

  .reorder-buttons {
    position: absolute;
    top: 0.25rem;
    left: 0.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    opacity: 0;
    transition: opacity var(--duration-fast) var(--ease-default);
  }

  .image-wrapper:hover .reorder-buttons {
    opacity: 1;
  }

  .reorder-button {
    padding: 0.125rem;
    border-radius: var(--radius-sm);
    background: color-mix(in oklch, var(--color-bg) 80%, transparent);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    cursor: pointer;
  }

  .caption-input {
    width: 100%;
    padding: 0.125rem 0.25rem;
    font-size: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-bg);
    color: var(--color-text);
  }

  .caption-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .caption-text {
    font-size: 0.75rem;
    color: var(--color-muted);
    padding: 0.125rem 0.25rem;
  }

  .upload-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    aspect-ratio: 1;
    border: 2px dashed var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-bg);
    color: var(--color-muted);
    cursor: pointer;
    transition:
      border-color 0.15s,
      color 0.15s;
  }

  @media (min-width: 1024px) {
    .recipe-image-gallery {
      max-width: 19rem;
    }

    .recipe-image-gallery.no-images .image-grid {
      grid-template-columns: minmax(0, 1fr);
      max-width: 8.75rem;
    }

    .recipe-image-gallery.no-images .upload-card {
      aspect-ratio: 1;
    }
  }

  .upload-card:hover:not(:disabled) {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  .upload-card:disabled {
    cursor: wait;
  }

  .upload-label {
    font-size: 0.75rem;
  }

  /* Lightbox */
  .lightbox-overlay {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: color-mix(in oklch, var(--color-bg) 90%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem;
  }

  .lightbox-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
  }

  .lightbox-image {
    max-width: 100%;
    max-height: 85vh;
    object-fit: contain;
    border-radius: var(--radius-lg);
  }

  .lightbox-caption {
    text-align: center;
    margin-top: 0.5rem;
    color: var(--color-text);
    font-size: 0.875rem;
  }

  .lightbox-close {
    position: absolute;
    top: -1rem;
    right: -1rem;
    padding: 0.5rem;
    border-radius: 50%;
    background: var(--color-surface);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    cursor: pointer;
  }
</style>
