<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';
  import { untrack } from 'svelte';

  import { createCanvasEditor, updateEditorContent } from '$lib/editor/codemirror';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  let editorContainer: HTMLDivElement | undefined = $state();
  let editorView: EditorView | undefined = $state();
  let suppressExternalSync = false;

  const color = $derived(data.color as string | undefined);
  const borderColor = $derived(getCanvasColor(color));
  const bgColor = $derived(getCanvasBgColor(color));

  // Mount CodeMirror editor
  $effect(() => {
    if (!editorContainer) return;
    const initialText = untrack(() => (data.text as string) || '');
    editorView = createCanvasEditor(editorContainer, {
      doc: initialText,
      onChange: (content) => {
        suppressExternalSync = true;
        data.text = content;
        suppressExternalSync = false;
        // Bubble up for auto-save
        editorContainer?.dispatchEvent(new CustomEvent('canvastextchange', { bubbles: true }));
      },
      onSave: () => {
        editorContainer?.dispatchEvent(new CustomEvent('canvassave', { bubbles: true }));
      },
      onWikilinkClick: (title) => {
        editorContainer?.dispatchEvent(
          new CustomEvent('wikilinkclick', { bubbles: true, detail: { title } })
        );
      },
      onToggleTaskByLine: (lineNumber, checked) => {
        if (!editorView) return;
        const line = editorView.state.doc.line(lineNumber);
        const newText = checked
          ? line.text.replace(/\[ \]/, '[x]')
          : line.text.replace(/\[x\]/i, '[ ]');
        editorView.dispatch({
          changes: { from: line.from, to: line.to, insert: newText },
        });
      },
    });
    return () => {
      editorView?.destroy();
      editorView = undefined;
    };
  });

  // Sync external updates (e.g. remote/sync) into the editor
  $effect(() => {
    const text = (data.text as string) || '';
    if (editorView && !suppressExternalSync) {
      updateEditorContent(editorView, text);
    }
  });
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
>
  <!-- nodrag nowheel nopan: SvelteFlow escape hatches to prevent mouse events from being captured -->
  <div
    class="canvas-text-editor nodrag nowheel nopan"
    bind:this={editorContainer}
    onkeydown={(e) => e.stopPropagation()}
  ></div>
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
    overflow: hidden;
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

  .canvas-text-editor {
    width: 100%;
    height: 100%;
  }
</style>
