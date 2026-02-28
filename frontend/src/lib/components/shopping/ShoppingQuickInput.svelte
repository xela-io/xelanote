<script lang="ts">
  import { Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { parseShoppingInput } from '$lib/utils/shopping-parser';

  interface Props {
    onsubmit: (
      items: Array<{ name: string; quantity: number | null; unit: string | null }>
    ) => void;
    disabled?: boolean;
  }

  const { onsubmit, disabled = false }: Props = $props();

  let input = $state('');

  function handleSubmit() {
    const parsed = parseShoppingInput(input);
    if (parsed.length === 0) return;
    onsubmit(parsed);
    input = '';
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }
</script>

<div class="shopping-quick-input">
  <input
    type="text"
    bind:value={input}
    onkeydown={handleKeydown}
    placeholder={$_('page.shopping.quick_input_placeholder')}
    {disabled}
    class="shopping-quick-input-field"
  />
  <button
    type="button"
    onclick={handleSubmit}
    disabled={disabled || !input.trim()}
    class="shopping-quick-input-btn"
    aria-label={$_('page.shopping.add_item')}
  >
    <Plus size={18} />
  </button>
</div>

<style>
  .shopping-quick-input {
    display: flex;
    gap: 0.5rem;
    padding: 0.75rem;
    border-bottom: 1px solid var(--color-surface-300);
  }

  .shopping-quick-input-field {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-md);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .shopping-quick-input-field:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 2px
      var(--color-primary-alpha-20, rgba(var(--color-primary-rgb, 59, 130, 246), 0.2));
  }

  .shopping-quick-input-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem;
    border-radius: var(--radius-md);
    background: var(--color-primary);
    color: white;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .shopping-quick-input-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .shopping-quick-input-btn:not(:disabled):hover {
    opacity: 0.9;
  }
</style>
