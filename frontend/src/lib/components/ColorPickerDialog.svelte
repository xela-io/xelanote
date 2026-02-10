<script lang="ts">
  import { untrack } from 'svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    currentColor: string | null | undefined;
    onClose: () => void;
    onSelect: (color: string | null) => void;
  }

  const { currentColor, onClose, onSelect }: Props = $props();

  // Preset colors - 10 colors inspired by VS Code and common project colors
  const presetColors = [
    '#E74C3C', // Red
    '#E67E22', // Orange
    '#F1C40F', // Yellow
    '#2ECC71', // Green
    '#1ABC9C', // Teal
    '#3498DB', // Blue
    '#9B59B6', // Purple
    '#E91E63', // Pink
    '#795548', // Brown
    '#607D8B', // Blue Grey
  ];

  let selectedColor = $state<string | null>(untrack(() => currentColor ?? null));
  let customColorInput = $state(untrack(() => currentColor ?? '#3498DB'));

  function handlePresetClick(color: string) {
    selectedColor = color;
    customColorInput = color;
  }

  function handleCustomColorChange(e: Event) {
    const input = e.target as HTMLInputElement;
    selectedColor = input.value;
    customColorInput = input.value;
  }

  function handleConfirm() {
    onSelect(selectedColor);
    onClose();
  }

  function handleRemove() {
    onSelect(null);
    onClose();
  }
</script>

<BaseDialog
  open={true}
  title={$_('component.color_picker.title')}
  {onClose}
  size="sm"
  footerAlign="between"
>
  {#snippet content()}
    <div class="space-y-4">
      <!-- Preview -->
      <div class="flex items-center gap-3">
        <span class="text-sm font-medium">{$_('component.color_picker.preview')}:</span>
        <div class="flex items-center gap-2 px-3 py-2 bg-secondary rounded-md">
          {#if selectedColor}
            <div class="w-1 h-6 rounded-sm" style="background-color: {selectedColor}"></div>
          {:else}
            <div class="w-1 h-6 rounded-sm bg-muted"></div>
          {/if}
          <span class="text-sm">Beispielordner</span>
        </div>
      </div>

      <!-- Preset Colors -->
      <div class="space-y-2">
        <span class="text-sm font-medium">{$_('component.color_picker.presets')}:</span>
        <div class="grid grid-cols-5 gap-2">
          {#each presetColors as color (color)}
            <button
              type="button"
              class="w-10 h-10 rounded-md border-2 transition-transform hover:scale-110"
              class:ring-2={selectedColor === color}
              class:ring-offset-2={selectedColor === color}
              class:ring-primary={selectedColor === color}
              style="background-color: {color}; border-color: {selectedColor === color
                ? 'var(--color-primary)'
                : 'transparent'}"
              onclick={() => handlePresetClick(color)}
              title={color}
            ></button>
          {/each}
        </div>
      </div>

      <!-- Custom Color -->
      <div class="space-y-2">
        <label for="custom-color" class="text-sm font-medium">
          {$_('component.color_picker.custom')}:
        </label>
        <div class="flex items-center gap-2">
          <input
            id="custom-color"
            type="color"
            value={customColorInput}
            onchange={handleCustomColorChange}
            class="w-12 h-10 rounded cursor-pointer border border-border"
          />
          <input
            type="text"
            value={customColorInput}
            readonly
            class="flex-1 px-3 py-2 bg-secondary border border-border rounded-md text-sm font-mono"
          />
        </div>
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <button
      type="button"
      onclick={handleRemove}
      class="px-4 py-2 text-sm text-destructive hover:bg-destructive/10 rounded-md"
    >
      {$_('component.color_picker.remove')}
    </button>
    <div class="flex gap-2">
      <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
        {$_('common.cancel')}
      </button>
      <button
        type="button"
        onclick={handleConfirm}
        class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md"
      >
        {$_('component.color_picker.set')}
      </button>
    </div>
  {/snippet}
</BaseDialog>

<style>
  /* Color input styling */
  input[type='color'] {
    appearance: none;
    -webkit-appearance: none;
    padding: 0;
  }

  input[type='color']::-webkit-color-swatch-wrapper {
    padding: 0;
  }

  input[type='color']::-webkit-color-swatch {
    border: none;
    border-radius: 4px;
  }
</style>
