<script lang="ts">
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';
  import { ExternalLink } from 'lucide-svelte';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  const url = $derived((data.url as string) || '');
  const color = $derived(data.color as string | undefined);
  const borderColor = $derived(getCanvasColor(color));
  const bgColor = $derived(getCanvasBgColor(color));

  const displayUrl = $derived(() => {
    try {
      const u = new URL(url);
      return u.hostname + (u.pathname !== '/' ? u.pathname : '');
    } catch {
      return url;
    }
  });
</script>

<NodeResizer
  minWidth={150}
  minHeight={100}
  isVisible={selected}
  lineStyle="border-color: var(--color-ring);"
  handleStyle="background: var(--color-ring); width: 8px; height: 8px;"
/>

<div
  class="canvas-link-node"
  class:selected
  style:border-left-color={borderColor}
  style:background={bgColor ? `color-mix(in oklch, ${bgColor} 40%, var(--color-card))` : undefined}
>
  <div class="canvas-link-header">
    <ExternalLink size={16} class="text-muted-foreground shrink-0" />
    <span class="canvas-link-url">{displayUrl()}</span>
  </div>
  <div class="canvas-link-full">{url}</div>
</div>

<Handle type="source" position={Position.Right} />
<Handle type="source" position={Position.Bottom} />
<Handle type="target" position={Position.Left} />
<Handle type="target" position={Position.Top} />

<style>
  .canvas-link-node {
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    padding: 12px;
    font-family: var(--font-sans);
    min-width: 150px;
    min-height: 80px;
    width: 100%;
    height: 100%;
    box-shadow: 0 1px 3px color-mix(in oklch, var(--color-foreground) 8%, transparent);
    transition: box-shadow 200ms ease;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .canvas-link-node:hover {
    box-shadow: 0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-link-node.selected {
    border-color: var(--color-ring);
    box-shadow:
      0 0 0 2px var(--color-ring),
      0 4px 12px color-mix(in oklch, var(--color-foreground) 12%, transparent);
  }

  .canvas-link-node[style*='border-left-color'] {
    border-left-width: 3px;
  }

  .canvas-link-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .canvas-link-url {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .canvas-link-full {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
