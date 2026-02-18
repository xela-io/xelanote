<script lang="ts">
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';
  import { tick } from 'svelte';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  const label = $derived((data.label as string) || '');
  const color = $derived((data.color as string) || '4'); // Default green
  const borderColor = $derived(getCanvasColor(color) || 'var(--color-border)');
  const bgColor = $derived(getCanvasBgColor(color));

  let editing = $state(false);
  let editValue = $state('');
  let inputEl: HTMLInputElement | undefined = $state();
  let groupEl: HTMLDivElement | undefined = $state();

  async function startEditing() {
    editValue = label;
    editing = true;
    await tick();
    inputEl?.focus();
    inputEl?.select();
  }

  function commitEdit() {
    if (!editing) return;
    editing = false;
    const trimmed = editValue.trim();
    if (trimmed && trimmed !== label) {
      data.label = trimmed;
      groupEl?.dispatchEvent(new CustomEvent('canvasgrouplabelchange', { bubbles: true }));
    }
  }

  function cancelEdit() {
    editing = false;
  }

  function handleLabelKeydown(e: KeyboardEvent) {
    e.stopPropagation();
    if (e.key === 'Enter') {
      e.preventDefault();
      commitEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelEdit();
    }
  }
</script>

<NodeResizer
  minWidth={200}
  minHeight={150}
  isVisible={selected}
  lineStyle="border-color: var(--color-ring);"
  handleStyle="background: var(--color-ring); width: 8px; height: 8px;"
/>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={groupEl}
  class="canvas-group-node"
  class:selected
  style:border-color={borderColor}
  style:background={bgColor
    ? `color-mix(in oklch, ${bgColor} 25%, transparent)`
    : 'color-mix(in oklch, var(--color-muted) 10%, transparent)'}
>
  {#if editing}
    <input
      bind:this={inputEl}
      class="canvas-group-label-input nodrag nopan"
      style:color={borderColor}
      bind:value={editValue}
      onblur={commitEdit}
      onkeydown={handleLabelKeydown}
    />
  {:else}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <span
      class="canvas-group-label"
      class:canvas-group-label--empty={!label}
      style:color={borderColor}
      ondblclick={startEditing}>{label || 'Group'}</span
    >
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
    cursor: text;
  }

  .canvas-group-label--empty {
    opacity: 0.4;
  }

  .canvas-group-label-input {
    position: absolute;
    top: 4px;
    left: 8px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    background: transparent;
    border: 1px solid var(--color-ring);
    border-radius: 4px;
    padding: 2px 4px;
    outline: none;
    min-width: 80px;
    max-width: calc(100% - 24px);
  }
</style>
