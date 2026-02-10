<script lang="ts">
  import { Sparkles } from 'lucide-svelte';

  import type { Note } from '$lib/api';

  interface Props {
    note: Note;
    isSelected: boolean;
    onclick: () => void;
  }

  const { note, isSelected, onclick }: Props = $props();
  let isDragging = $state(false);

  function handleDragStart(e: DragEvent) {
    if (!e.dataTransfer) return;

    // Store note data in DataTransfer
    const dragData = {
      id: note.id,
      title: note.title,
      folder_path: note.folder_path,
    };

    e.dataTransfer.setData('application/x-xelanote-note', JSON.stringify(dragData));
    e.dataTransfer.effectAllowed = 'move';
    isDragging = true;
  }

  function handleDragEnd() {
    isDragging = false;
  }
</script>

<button
  draggable="true"
  ondragstart={handleDragStart}
  ondragend={handleDragEnd}
  {onclick}
  class="note-item"
  class:selected={isSelected}
  class:dragging={isDragging}
>
  <span class="note-title">{note.title}</span>
  {#if note.ai_enabled}
    <span class="ai-badge" title="Claude API aktiviert">
      <Sparkles size={12} />
    </span>
  {/if}
</button>

<style>
  .note-item {
    width: 100%;
    text-align: left;
    padding: 8px 16px;
    font-size: 14px;
    color: var(--color-sidebar-foreground);
    background: transparent;
    border: none;
    cursor: grab;
    overflow: hidden;
    white-space: nowrap;
    border-radius: 4px;
    transition: background-color 0.15s;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .note-title {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ai-badge {
    flex-shrink: 0;
    color: var(--color-primary);
    opacity: 0.7;
    display: flex;
    align-items: center;
  }

  .note-item:hover .ai-badge {
    opacity: 1;
  }

  .note-item:hover {
    background-color: var(--color-sidebar-accent);
  }

  .note-item.selected {
    background-color: var(--color-sidebar-accent);
  }

  .note-item.dragging {
    opacity: 0.5;
    cursor: grabbing;
  }

  /* Mobile-optimized sizes for screens < 640px */
  @media (max-width: 639px) {
    .note-item {
      font-size: 15px;
      padding: 10px 16px;
    }
  }
</style>
