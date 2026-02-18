<script lang="ts">
  import { Copy, Pencil, Trash2 } from 'lucide-svelte';

  import { CANVAS_COLOR_PRESETS } from './canvas-colors';

  type Props = {
    x: number;
    y: number;
    onClose: () => void;
    onDelete: () => void;
    onDuplicate: () => void;
    onColorChange: (color: string) => void;
    onRename?: () => void;
    currentColor?: string;
  };

  const { x, y, onClose, onDelete, onDuplicate, onColorChange, onRename, currentColor }: Props =
    $props();

  function handleClickOutside(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest('.canvas-context-menu')) {
      onClose();
    }
  }
</script>

<svelte:window onclick={handleClickOutside} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="canvas-context-menu"
  style:left="{x}px"
  style:top="{y}px"
  oncontextmenu={(e) => e.preventDefault()}
>
  <div class="canvas-context-colors">
    {#each CANVAS_COLOR_PRESETS as preset (preset.id)}
      <button
        class="canvas-color-dot"
        class:active={currentColor === preset.id}
        style:background={`var(${preset.cssVar})`}
        title={preset.name}
        onclick={() => {
          onColorChange(preset.id);
          onClose();
        }}
      ></button>
    {/each}
    <button
      class="canvas-color-dot canvas-color-none"
      class:active={!currentColor}
      title="No color"
      onclick={() => {
        onColorChange('');
        onClose();
      }}
    >
      &times;
    </button>
  </div>

  <div class="canvas-context-separator"></div>

  {#if onRename}
    <button
      class="canvas-context-item"
      onclick={() => {
        onRename();
        onClose();
      }}
    >
      <Pencil size={16} />
      <span>Rename</span>
    </button>
  {/if}

  <button
    class="canvas-context-item"
    onclick={() => {
      onDuplicate();
      onClose();
    }}
  >
    <Copy size={16} />
    <span>Duplicate</span>
  </button>

  <button
    class="canvas-context-item canvas-context-item--danger"
    onclick={() => {
      onDelete();
      onClose();
    }}
  >
    <Trash2 size={16} />
    <span>Delete</span>
  </button>
</div>

<style>
  .canvas-context-menu {
    position: fixed;
    z-index: 50;
    background: var(--color-popover);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    box-shadow: 0 8px 28px color-mix(in oklch, var(--color-foreground) 15%, transparent);
    padding: 4px;
    min-width: 180px;
    animation: scaleUp 150ms ease;
  }

  @keyframes scaleUp {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  .canvas-context-colors {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
  }

  .canvas-color-dot {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: transform 150ms ease;
  }

  .canvas-color-dot:hover {
    transform: scale(1.15);
  }

  .canvas-color-dot.active {
    border-color: var(--color-ring);
    transform: scale(1.15);
  }

  .canvas-color-none {
    background: var(--color-muted) !important;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: var(--color-muted-foreground);
  }

  .canvas-context-separator {
    height: 1px;
    background: var(--color-border);
    margin: 4px 0;
  }

  .canvas-context-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    border: none;
    background: transparent;
    border-radius: 4px;
    font-size: 0.875rem;
    color: var(--color-foreground);
    cursor: pointer;
    text-align: left;
  }

  .canvas-context-item:hover {
    background: var(--color-accent);
  }

  .canvas-context-item--danger:hover {
    background: color-mix(in oklch, var(--color-destructive) 15%, transparent);
    color: var(--color-destructive);
  }
</style>
