<script lang="ts">
  import { X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    open: boolean;
    editName?: string;
    editColor?: string | null;
    onsave: (name: string, color: string | null) => void;
    onclose: () => void;
  }

  const { open, editName = '', editColor = null, onsave, onclose }: Props = $props();

  let name = $state('');
  let color = $state('');

  const COLORS = [
    { value: '', label: 'Default' },
    { value: '#ef4444', label: 'Red' },
    { value: '#f97316', label: 'Orange' },
    { value: '#eab308', label: 'Yellow' },
    { value: '#22c55e', label: 'Green' },
    { value: '#3b82f6', label: 'Blue' },
    { value: '#8b5cf6', label: 'Purple' },
    { value: '#ec4899', label: 'Pink' },
  ];

  // Reset state when dialog opens
  $effect(() => {
    if (open) {
      name = editName ?? '';
      color = editColor || '';
    }
  });

  function handleSubmit() {
    if (!name.trim()) return;
    onsave(name.trim(), color || null);
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog-backdrop" onmousedown={onclose}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="dialog-content" onmousedown={(e) => e.stopPropagation()}>
      <div class="dialog-header">
        <h2 class="dialog-title">
          {editName ? $_('page.shopping.list_name') : $_('page.shopping.new_list')}
        </h2>
        <button type="button" class="dialog-close" onclick={onclose}>
          <X size={18} />
        </button>
      </div>

      <form
        onsubmit={(e) => {
          e.preventDefault();
          handleSubmit();
        }}
        class="dialog-body"
      >
        <label class="dialog-label">
          {$_('page.shopping.list_name')}
          <input type="text" bind:value={name} class="dialog-input" autofocus maxlength={200} />
        </label>

        <div class="dialog-label">
          {$_('page.shopping.list_color')}
          <div class="color-picker">
            {#each COLORS as c (c.value)}
              <button
                type="button"
                class="color-swatch"
                class:selected={color === c.value}
                style={c.value ? `background-color: ${c.value}` : ''}
                onclick={() => (color = c.value)}
                title={c.label}
              >
                {#if !c.value}
                  <span class="no-color">–</span>
                {/if}
              </button>
            {/each}
          </div>
        </div>

        <div class="dialog-actions">
          <button type="button" class="btn-secondary" onclick={onclose}>
            {$_('common.cancel', { default: 'Cancel' })}
          </button>
          <button type="submit" class="btn-primary" disabled={!name.trim()}>
            {editName ? $_('common.save', { default: 'Save' }) : $_('page.shopping.new_list')}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .dialog-backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
  }

  .dialog-content {
    background: var(--color-surface-100);
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 24rem;
    margin: 1rem;
    box-shadow: var(--shadow-lg, 0 10px 15px -3px rgba(0, 0, 0, 0.1));
  }

  .dialog-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--color-surface-300);
  }

  .dialog-title {
    font-size: var(--text-base);
    font-weight: 600;
  }

  .dialog-close {
    color: var(--color-text-muted);
    cursor: pointer;
  }

  .dialog-body {
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .dialog-label {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    font-size: var(--text-sm);
    font-weight: 500;
  }

  .dialog-input {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-md);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .dialog-input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .color-picker {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .color-swatch {
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: border-color 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .color-swatch:not([style]) {
    background: var(--color-surface-300);
  }

  .color-swatch.selected {
    border-color: var(--color-text);
  }

  .no-color {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.5rem;
  }

  .btn-secondary {
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface-200);
    cursor: pointer;
  }

  .btn-secondary:hover {
    background: var(--color-surface-300);
  }

  .btn-primary {
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    color: white;
    background: var(--color-primary);
    cursor: pointer;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary:not(:disabled):hover {
    opacity: 0.9;
  }
</style>
