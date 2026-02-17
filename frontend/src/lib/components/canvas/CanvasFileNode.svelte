<script lang="ts">
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';
  import { FileText, Image } from 'lucide-svelte';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  const file = $derived((data.file as string) || '');
  const color = $derived(data.color as string | undefined);
  const borderColor = $derived(getCanvasColor(color));
  const bgColor = $derived(getCanvasBgColor(color));

  // Detect if this is an image file
  const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.bmp'];
  const isImage = $derived(imageExtensions.some((ext) => file.toLowerCase().endsWith(ext)));
</script>

<NodeResizer
  minWidth={150}
  minHeight={100}
  isVisible={selected}
  lineStyle="border-color: var(--color-ring);"
  handleStyle="background: var(--color-ring); width: 8px; height: 8px;"
/>

<div
  class="canvas-file-node"
  class:selected
  style:border-left-color={borderColor}
  style:background={bgColor ? `color-mix(in oklch, ${bgColor} 40%, var(--color-card))` : undefined}
>
  {#if isImage}
    <div class="canvas-file-image">
      <Image size={24} class="text-muted-foreground" />
      <span class="canvas-file-title">{file}</span>
    </div>
  {:else}
    <div class="canvas-file-header">
      <FileText size={16} class="text-muted-foreground shrink-0" />
      <span class="canvas-file-title">{file}</span>
    </div>
    {#if data.subpath}
      <div class="canvas-file-subpath">{data.subpath}</div>
    {/if}
    <div class="canvas-file-preview">Click to open note</div>
  {/if}
</div>

<Handle type="source" position={Position.Right} />
<Handle type="source" position={Position.Bottom} />
<Handle type="target" position={Position.Left} />
<Handle type="target" position={Position.Top} />

<style>
  .canvas-file-node {
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    padding: 12px;
    font-family: var(--font-sans, Inter, sans-serif);
    min-width: 150px;
    min-height: 100px;
    width: 100%;
    height: 100%;
    box-shadow: 0 1px 3px color-mix(in oklch, var(--color-foreground) 8%, transparent);
    transition: box-shadow 200ms ease;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .canvas-file-node:hover {
    box-shadow: 0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-file-node.selected {
    border-color: var(--color-ring);
    box-shadow:
      0 0 0 2px var(--color-ring),
      0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-file-node[style*='border-left-color'] {
    border-left-width: 3px;
  }

  .canvas-file-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .canvas-file-title {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--color-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .canvas-file-subpath {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .canvas-file-preview {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
    flex: 1;
    overflow: hidden;
  }

  .canvas-file-image {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
  }
</style>
