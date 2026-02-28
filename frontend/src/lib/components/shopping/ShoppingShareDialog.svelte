<script lang="ts">
  import { Trash2, X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { ShoppingListShare } from '$lib/api/types';

  interface Props {
    open: boolean;
    shares: ShoppingListShare[];
    onshare: (userId: number, role: 'viewer' | 'editor') => void;
    onupdaterole: (userId: number, role: 'viewer' | 'editor') => void;
    onremove: (userId: number) => void;
    onclose: () => void;
  }

  const { open, shares, onshare, onupdaterole, onremove, onclose }: Props = $props();

  let userId = $state('');
  let role = $state<'editor' | 'viewer'>('editor');

  function handleShare() {
    const id = parseInt(userId, 10);
    if (isNaN(id)) return;
    onshare(id, role);
    userId = '';
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog-backdrop" onmousedown={onclose}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="dialog-content" onmousedown={(e) => e.stopPropagation()}>
      <div class="dialog-header">
        <h2 class="dialog-title">{$_('page.shopping.share_list')}</h2>
        <button type="button" class="dialog-close" onclick={onclose}>
          <X size={18} />
        </button>
      </div>

      <div class="dialog-body">
        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleShare();
          }}
          class="share-form"
        >
          <input
            type="number"
            bind:value={userId}
            placeholder={$_('page.shopping.share_with')}
            class="share-input"
          />
          <select bind:value={role} class="share-role-select">
            <option value="editor">{$_('page.shopping.role_editor')}</option>
            <option value="viewer">{$_('page.shopping.role_viewer')}</option>
          </select>
          <button type="submit" class="btn-primary" disabled={!userId}>
            {$_('page.shopping.share_list')}
          </button>
        </form>

        {#if shares.length > 0}
          <div class="share-list">
            {#each shares as share (share.shared_with_user_id)}
              <div class="share-row">
                <span class="share-name"
                  >{share.shared_with_name || `User #${share.shared_with_user_id}`}</span
                >
                <select
                  value={share.role}
                  onchange={(e) =>
                    onupdaterole(
                      share.shared_with_user_id,
                      (e.target as HTMLSelectElement).value as 'viewer' | 'editor'
                    )}
                  class="share-role-select-sm"
                >
                  <option value="editor">{$_('page.shopping.role_editor')}</option>
                  <option value="viewer">{$_('page.shopping.role_viewer')}</option>
                </select>
                <button
                  type="button"
                  class="share-remove-btn"
                  onclick={() => onremove(share.shared_with_user_id)}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
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
    max-width: 28rem;
    margin: 1rem;
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

  .share-form {
    display: flex;
    gap: 0.5rem;
  }

  .share-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-md);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .share-role-select,
  .share-role-select-sm {
    padding: 0.5rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-md);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .share-role-select-sm {
    padding: 0.25rem 0.5rem;
    font-size: var(--text-xs);
  }

  .btn-primary {
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    color: white;
    background: var(--color-primary);
    cursor: pointer;
    white-space: nowrap;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .share-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .share-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    border-radius: var(--radius-md);
    background: var(--color-surface-200);
  }

  .share-name {
    flex: 1;
    font-size: var(--text-sm);
  }

  .share-remove-btn {
    display: flex;
    padding: 0.25rem;
    color: var(--color-text-muted);
    cursor: pointer;
    border-radius: var(--radius-sm);
  }

  .share-remove-btn:hover {
    color: var(--color-error, #ef4444);
    background: var(--color-surface-300);
  }
</style>
