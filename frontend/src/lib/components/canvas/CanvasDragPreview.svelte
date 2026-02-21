<script lang="ts">
  import { ExternalLink, FileText, Group as GroupIcon, Type } from 'lucide-svelte';

  import type { ToolbarAction } from '$lib/components/canvas/canvas-toolbar-tools';

  interface Props {
    action: ToolbarAction;
    containerX: number;
    containerY: number;
    width: number;
    height: number;
  }

  const { action, containerX, containerY, width, height }: Props = $props();

  function getToolLabel(a: ToolbarAction): string {
    switch (a) {
      case 'add-text':
        return 'Text';
      case 'add-file':
        return 'Note';
      case 'add-link':
        return 'Link';
      case 'add-group':
        return 'Group';
    }
  }
</script>

<div
  class={`tool-drag-preview tool-drag-preview--${action}`}
  style={`left:${containerX}px;top:${containerY}px;width:${width}px;height:${height}px;`}
  aria-hidden="true"
>
  {#if action === 'add-text'}
    <div class="tool-drag-preview-header">
      <Type size={14} />
      <span>{getToolLabel(action)}</span>
    </div>
    <div class="tool-drag-preview-text-lines">
      <span></span>
      <span></span>
      <span></span>
    </div>
  {:else if action === 'add-file'}
    <div class="tool-drag-preview-header">
      <FileText size={14} />
      <span>{getToolLabel(action)}</span>
    </div>
    <div class="tool-drag-preview-text-lines">
      <span></span>
      <span></span>
    </div>
  {:else if action === 'add-link'}
    <div class="tool-drag-preview-header">
      <ExternalLink size={14} />
      <span>{getToolLabel(action)}</span>
    </div>
    <div class="tool-drag-preview-link-url">example.com/path</div>
    <div class="tool-drag-preview-text-lines">
      <span></span>
    </div>
  {:else}
    <div class="tool-drag-preview-group-label">
      <GroupIcon size={14} />
      <span>{getToolLabel(action)}</span>
    </div>
  {/if}
</div>

<style>
  .tool-drag-preview {
    position: absolute;
    z-index: 20;
    border-radius: 0.6rem;
    border: 1px solid var(--color-border);
    background: color-mix(in oklch, var(--color-card) 90%, transparent);
    box-shadow:
      0 0 0 1px color-mix(in oklch, var(--color-ring) 30%, transparent),
      0 14px 32px color-mix(in oklch, var(--color-foreground) 16%, transparent);
    pointer-events: none;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
  }

  .tool-drag-preview-header {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-foreground);
  }

  .tool-drag-preview-text-lines {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .tool-drag-preview-text-lines span {
    display: block;
    height: 8px;
    border-radius: 999px;
    background: color-mix(in oklch, var(--color-muted-foreground) 26%, transparent);
  }

  .tool-drag-preview-text-lines span:nth-child(1) {
    width: 88%;
  }

  .tool-drag-preview-text-lines span:nth-child(2) {
    width: 70%;
  }

  .tool-drag-preview-text-lines span:nth-child(3) {
    width: 55%;
  }

  .tool-drag-preview-link-url {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-muted-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tool-drag-preview-group-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: color-mix(in oklch, var(--color-foreground) 76%, var(--color-primary));
  }

  .tool-drag-preview--add-group {
    border-style: dashed;
    border-width: 1.5px;
    background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  }

  .tool-drag-preview--add-text,
  .tool-drag-preview--add-file,
  .tool-drag-preview--add-link {
    border-left-width: 3px;
    border-left-color: var(--color-primary);
  }
</style>
