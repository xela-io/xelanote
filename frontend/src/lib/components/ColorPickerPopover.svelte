<script lang="ts">
  import { Palette, Hash, X } from 'lucide-svelte';
  import { sanitizeColor } from '$lib/editor/markdown';

  interface Props {
    onSelect: (color: string) => void;
    onClose: () => void;
  }

  const { onSelect, onClose }: Props = $props();

  let activeTab = $state<'palette' | 'custom'>('palette');
  let customHex = $state('#');
  let hexError = $state('');

  // Named colors (design tokens)
  const namedColors = [
    { name: 'primary', label: 'Primär', cssVar: 'var(--color-primary)' },
    { name: 'destructive', label: 'Warnung', cssVar: 'var(--color-destructive)' },
    { name: 'accent', label: 'Akzent', cssVar: 'var(--color-accent-foreground)' },
    { name: 'muted', label: 'Gedämpft', cssVar: 'var(--color-muted-foreground)' },
    { name: 'secondary', label: 'Sekundär', cssVar: 'var(--color-secondary-foreground)' },
  ];

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  function handleBackdropClick() {
    onClose();
  }

  function handlePopoverClick(e: MouseEvent) {
    e.stopPropagation();
  }

  function selectNamedColor(name: string) {
    onSelect(name);
    onClose();
  }

  function handleHexInput(e: Event) {
    const value = (e.currentTarget as HTMLInputElement).value;
    customHex = value;

    // Validate hex color
    if (value && value !== '#') {
      const sanitized = sanitizeColor(value);
      if (sanitized) {
        hexError = '';
      } else {
        hexError = 'Ungültiges Hex-Format (z.B. #fff oder #ffffff)';
      }
    } else {
      hexError = '';
    }
  }

  function handleHexSubmit() {
    const sanitized = sanitizeColor(customHex);
    if (sanitized) {
      onSelect(sanitized);
      onClose();
    }
  }

  function handleColorPickerChange(e: Event) {
    const value = (e.currentTarget as HTMLInputElement).value;
    customHex = value;
    hexError = '';
  }

  function handleHexKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !hexError && customHex !== '#') {
      handleHexSubmit();
    }
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 z-40 md:bg-transparent bg-black/50"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<!-- Popover: Bottom sheet on mobile, centered on desktop -->
<div
  class="fixed z-50 bg-background border border-border shadow-lg flex flex-col
		md:top-1/2 md:left-1/2 md:-translate-x-1/2 md:-translate-y-1/2 md:w-80 md:rounded-lg md:max-h-[calc(100vh-6rem)]
		bottom-0 left-0 right-0 max-h-[60vh] rounded-t-2xl animate-bottom-sheet"
  onkeydown={handleKeydown}
  onclick={handlePopoverClick}
  role="dialog"
  aria-label="Textfarbe wählen"
  tabindex="-1"
>
  <!-- Mobile handle bar -->
  <div class="md:hidden flex justify-center pt-2 pb-1">
    <div class="w-12 h-1 bg-muted-foreground/30 rounded-full"></div>
  </div>

  <!-- Header -->
  <div class="flex items-center justify-between px-4 py-3 border-b border-border">
    <h3 class="text-sm font-semibold">Textfarbe wählen</h3>
    <button
      type="button"
      onclick={onClose}
      class="p-1 hover:bg-accent rounded-md"
      aria-label="Schließen"
    >
      <X size={16} />
    </button>
  </div>

  <!-- Tabs -->
  <div class="flex border-b border-border">
    <button
      type="button"
      onclick={() => (activeTab = 'palette')}
      class="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors"
      class:border-primary={activeTab === 'palette'}
      class:text-primary={activeTab === 'palette'}
      class:border-transparent={activeTab !== 'palette'}
      class:text-muted-foreground={activeTab !== 'palette'}
    >
      <Palette size={16} />
      Palette
    </button>
    <button
      type="button"
      onclick={() => (activeTab = 'custom')}
      class="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors"
      class:border-primary={activeTab === 'custom'}
      class:text-primary={activeTab === 'custom'}
      class:border-transparent={activeTab !== 'custom'}
      class:text-muted-foreground={activeTab !== 'custom'}
    >
      <Hash size={16} />
      Eigene
    </button>
  </div>

  <!-- Content -->
  <div class="p-4 overflow-y-auto">
    {#if activeTab === 'palette'}
      <div class="grid grid-cols-1 gap-2">
        {#each namedColors as color (color.name)}
          <button
            onclick={() => selectNamedColor(color.name)}
            class="flex items-center gap-3 p-3 hover:bg-accent rounded-md transition-colors text-left"
          >
            <div
              class="w-6 h-6 rounded-full border border-border flex-shrink-0"
              style="background-color: {color.cssVar};"
            ></div>
            <div class="flex-1">
              <span class="text-sm font-medium">{color.label}</span>
              <span class="text-xs text-muted-foreground ml-2">{color.name}</span>
            </div>
          </button>
        {/each}
      </div>
    {:else}
      <div class="space-y-4">
        <!-- Hex Input -->
        <div>
          <label for="hex-input" class="block text-sm font-medium mb-2"> Hex-Farbcode </label>
          <div class="flex gap-2">
            <input
              id="hex-input"
              type="text"
              value={customHex}
              oninput={handleHexInput}
              onkeydown={handleHexKeydown}
              placeholder="#ff0000"
              class="flex-1 px-3 py-2 bg-background border border-border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-ring font-mono"
            />
            <button
              onclick={handleHexSubmit}
              disabled={hexError !== '' || customHex === '#' || customHex === ''}
              class="px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-primary/90 transition-colors"
            >
              OK
            </button>
          </div>
          {#if hexError}
            <p class="text-xs text-destructive mt-1">{hexError}</p>
          {/if}
        </div>

        <!-- Native Color Picker -->
        <div>
          <label for="color-picker" class="block text-sm font-medium mb-2"> Farbwähler </label>
          <div class="flex gap-2 items-center">
            <input
              id="color-picker"
              type="color"
              value={customHex.startsWith('#') && customHex.length >= 4 ? customHex : '#000000'}
              oninput={handleColorPickerChange}
              class="w-12 h-10 p-0 border border-border rounded cursor-pointer"
            />
            <span class="text-sm text-muted-foreground">
              Wähle eine Farbe aus der Systempalette
            </span>
          </div>
        </div>

        <!-- Preview -->
        {#if customHex !== '#' && !hexError}
          <div class="pt-2 border-t border-border">
            <span class="text-sm font-medium">Vorschau:</span>
            <p class="mt-2 text-lg" style="color: {sanitizeColor(customHex) || 'inherit'};">
              Dies ist ein Beispieltext in der gewählten Farbe.
            </p>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
