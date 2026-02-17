<script lang="ts">
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  let editing = $state(false);
  let editText = $state('');
  const displayText = $derived((data.text as string) || '');

  const color = $derived(data.color as string | undefined);
  const borderColor = $derived(getCanvasColor(color));
  const bgColor = $derived(getCanvasBgColor(color));

  function handleDoubleClick() {
    editText = displayText;
    editing = true;
  }

  function handleBlur() {
    editing = false;
    data.text = editText;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      editing = false;
      data.text = editText;
    }
  }
</script>

<NodeResizer
  minWidth={100}
  minHeight={60}
  isVisible={selected}
  lineStyle="border-color: var(--color-ring);"
  handleStyle="background: var(--color-ring); width: 8px; height: 8px;"
/>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="canvas-text-node"
  class:selected
  style:border-left-color={borderColor}
  style:background={bgColor ? `color-mix(in oklch, ${bgColor} 40%, var(--color-card))` : undefined}
  ondblclick={handleDoubleClick}
>
  {#if editing}
    <textarea
      class="canvas-text-edit"
      bind:value={editText}
      onblur={handleBlur}
      onkeydown={handleKeydown}
      autofocus
    ></textarea>
  {:else}
    <div class="canvas-text-content">
      {displayText || 'Double-click to edit...'}
    </div>
  {/if}
</div>

<Handle type="source" position={Position.Right} />
<Handle type="source" position={Position.Bottom} />
<Handle type="target" position={Position.Left} />
<Handle type="target" position={Position.Top} />

<style>
  .canvas-text-node {
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    padding: 12px;
    font-family: var(--font-sans, Inter, sans-serif);
    min-width: 100px;
    min-height: 60px;
    width: 100%;
    height: 100%;
    box-shadow: 0 1px 3px color-mix(in oklch, var(--color-foreground) 8%, transparent);
    transition: box-shadow 200ms ease;
    overflow: auto;
  }

  .canvas-text-node:hover {
    box-shadow: 0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-text-node.selected {
    border-color: var(--color-ring);
    box-shadow:
      0 0 0 2px var(--color-ring),
      0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-text-node[style*='border-left-color'] {
    border-left-width: 3px;
  }

  .canvas-text-content {
    font-size: 0.875rem;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--color-foreground);
  }

  .canvas-text-content:empty::before {
    content: 'Double-click to edit...';
    color: var(--color-muted-foreground);
    font-style: italic;
  }

  .canvas-text-edit {
    width: 100%;
    height: 100%;
    min-height: 40px;
    background: transparent;
    border: none;
    outline: none;
    resize: none;
    font-family: inherit;
    font-size: 0.875rem;
    line-height: 1.6;
    color: var(--color-foreground);
  }
</style>
