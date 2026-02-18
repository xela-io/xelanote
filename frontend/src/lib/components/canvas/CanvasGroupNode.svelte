<script lang="ts">
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  const label = $derived((data.label as string) || '');
  const color = $derived((data.color as string) || '4'); // Default green
  const borderColor = $derived(getCanvasColor(color) || 'var(--color-border)');
  const bgColor = $derived(getCanvasBgColor(color));
</script>

<NodeResizer
  minWidth={200}
  minHeight={150}
  isVisible={selected}
  lineStyle="border-color: var(--color-ring);"
  handleStyle="background: var(--color-ring); width: 8px; height: 8px;"
/>

<div
  class="canvas-group-node"
  class:selected
  style:border-color={borderColor}
  style:background={bgColor
    ? `color-mix(in oklch, ${bgColor} 25%, transparent)`
    : 'color-mix(in oklch, var(--color-muted) 10%, transparent)'}
>
  {#if label}
    <span class="canvas-group-label" style:color={borderColor}>{label}</span>
  {/if}
</div>

<Handle type="source" position={Position.Right} />
<Handle type="source" position={Position.Bottom} />
<Handle type="target" position={Position.Left} />
<Handle type="target" position={Position.Top} />

<style>
  .canvas-group-node {
    border: 1.5px dashed var(--color-border);
    border-radius: 0.75rem;
    padding: 32px 16px 16px 16px;
    min-width: 200px;
    min-height: 150px;
    width: 100%;
    height: 100%;
    position: relative;
  }

  .canvas-group-node.selected {
    border-color: var(--color-ring);
    box-shadow: 0 0 0 1px var(--color-ring);
  }

  .canvas-group-label {
    position: absolute;
    top: 8px;
    left: 12px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    opacity: 0.85;
    user-select: none;
  }
</style>
