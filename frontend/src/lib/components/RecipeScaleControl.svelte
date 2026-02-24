<script lang="ts">
  import { Minus, Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    servings: number;
    baseServings: number;
    onchange: (servings: number) => void;
    disabled?: boolean;
  }

  const { servings, baseServings, onchange, disabled = false }: Props = $props();

  const MIN_SERVINGS = 1;
  const MAX_SERVINGS = 999;

  const isScaled = $derived(servings !== baseServings);

  function decrease() {
    if (servings > MIN_SERVINGS) {
      onchange(servings - 1);
    }
  }

  function increase() {
    if (servings < MAX_SERVINGS) {
      onchange(servings + 1);
    }
  }

  function reset() {
    onchange(baseServings);
  }

  function handleInput(e: Event) {
    const value = parseInt((e.target as HTMLInputElement).value);
    if (!isNaN(value) && value >= MIN_SERVINGS && value <= MAX_SERVINGS) {
      onchange(value);
    }
  }
</script>

<div class="flex items-center gap-2">
  <span class="text-sm text-muted-foreground">{$_('page.recipes.servings')}:</span>
  <div class="flex items-center gap-1">
    <button
      onclick={decrease}
      {disabled}
      class="p-1 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
      title={$_('page.recipes.decrease_servings')}
    >
      <Minus size={14} />
    </button>
    <input
      type="number"
      value={servings}
      oninput={handleInput}
      {disabled}
      min="1"
      max="999"
      class="w-14 text-center text-sm px-1 py-0.5 bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring"
    />
    <button
      onclick={increase}
      {disabled}
      class="p-1 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
      title={$_('page.recipes.increase_servings')}
    >
      <Plus size={14} />
    </button>
  </div>
  {#if isScaled}
    <button
      onclick={reset}
      {disabled}
      class="text-xs text-muted-foreground hover:text-foreground underline"
    >
      {$_('page.recipes.reset_servings')}
    </button>
  {/if}
</div>
