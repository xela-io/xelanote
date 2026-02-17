<script lang="ts">
  import { Search, X } from 'lucide-svelte';

  import { quickSearch } from '$lib/api';
  import type { Note } from '$lib/api/types';

  type Props = {
    open: boolean;
    onSelect: (note: Note) => void;
    onClose: () => void;
  };

  const { open, onSelect, onClose }: Props = $props();

  let query = $state('');
  let results = $state<Note[]>([]);

  $effect(() => {
    if (open) {
      searchNotes(query);
    }
  });

  async function searchNotes(q: string) {
    try {
      const response = await quickSearch(q, 20);
      results = response.notes.filter((n: Note) => n.note_type !== 'canvas');
    } catch {
      results = [];
    }
  }

  function handleSelect(note: Note) {
    onSelect(note);
    onClose();
    query = '';
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
      query = '';
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="canvas-note-picker-overlay" onclick={onClose} onkeydown={handleKeydown}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="canvas-note-picker" onclick={(e) => e.stopPropagation()}>
      <div class="canvas-note-picker-header">
        <div class="canvas-note-picker-search">
          <Search size={16} class="text-muted-foreground" />
          <input
            type="text"
            placeholder="Search notes..."
            bind:value={query}
            autofocus
            onkeydown={handleKeydown}
          />
        </div>
        <button class="canvas-note-picker-close" onclick={onClose}>
          <X size={16} />
        </button>
      </div>

      <div class="canvas-note-picker-results">
        {#each results as note (note.id)}
          <button class="canvas-note-picker-item" onclick={() => handleSelect(note)}>
            <span class="canvas-note-picker-title">{note.title}</span>
            <span class="canvas-note-picker-path">{note.folder_path}</span>
          </button>
        {:else}
          <div class="canvas-note-picker-empty">No notes found</div>
        {/each}
      </div>
    </div>
  </div>
{/if}

<style>
  .canvas-note-picker-overlay {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: color-mix(in oklch, var(--color-foreground) 20%, transparent);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .canvas-note-picker {
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 0.75rem;
    box-shadow: 0 16px 48px color-mix(in oklch, var(--color-foreground) 20%, transparent);
    width: min(90vw, 32rem);
    max-height: 70vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .canvas-note-picker-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    border-bottom: 1px solid var(--color-border);
  }

  .canvas-note-picker-search {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
  }

  .canvas-note-picker-search input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    font-size: 0.875rem;
    color: var(--color-foreground);
  }

  .canvas-note-picker-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    border-radius: 6px;
    color: var(--color-muted-foreground);
    cursor: pointer;
  }

  .canvas-note-picker-close:hover {
    background: var(--color-accent);
  }

  .canvas-note-picker-results {
    overflow-y: auto;
    max-height: 60vh;
  }

  .canvas-note-picker-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 100%;
    padding: 12px;
    border: none;
    background: transparent;
    text-align: left;
    cursor: pointer;
    border-radius: 6px;
    margin: 2px 4px;
  }

  .canvas-note-picker-item:hover {
    background: var(--color-accent);
  }

  .canvas-note-picker-title {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-foreground);
  }

  .canvas-note-picker-path {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .canvas-note-picker-empty {
    padding: 24px;
    text-align: center;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
  }
</style>
