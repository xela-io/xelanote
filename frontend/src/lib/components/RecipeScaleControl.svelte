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

<div class="recipe-scale-control">
  <span class="recipe-scale-label text-sm text-muted-foreground"
    >{$_('page.recipes.servings')}:</span
  >
  <div class="recipe-scale-stepper">
    <button
      onclick={decrease}
      {disabled}
      class="recipe-scale-btn hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
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
      class="recipe-scale-input bg-background border border-border focus:outline-none focus:ring-1 focus:ring-ring"
    />
    <button
      onclick={increase}
      {disabled}
      class="recipe-scale-btn hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
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

<style>
  .recipe-scale-control {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .recipe-scale-label {
    flex-shrink: 0;
  }

  .recipe-scale-stepper {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  .recipe-scale-btn {
    width: 2rem;
    height: 2rem;
    min-width: 2rem;
    min-height: 2rem;
    border-radius: 0.5rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
  }

  .recipe-scale-input {
    width: 3.5rem;
    text-align: center;
    font-size: 0.875rem;
    padding: 0.35rem 0.25rem;
    border-radius: 0.5rem;
  }

  @media (max-width: 639px) {
    .recipe-scale-control {
      gap: 0.4rem;
    }

    .recipe-scale-stepper {
      gap: 0.2rem;
    }

    .recipe-scale-btn {
      width: 2.5rem;
      height: 2.5rem;
      min-width: 2.5rem;
      min-height: 2.5rem;
    }

    .recipe-scale-input {
      width: 3.75rem;
      min-height: 2.5rem;
      padding-block: 0.45rem;
    }
  }
</style>
