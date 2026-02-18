<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { Handle, NodeResizer, Position } from '@xyflow/svelte';
  import { FileText } from 'lucide-svelte';
  import { untrack } from 'svelte';

  import { getNote } from '$lib/api/notes';
  import { getApiBaseUrl } from '$lib/config';
  import { createCanvasEditor } from '$lib/editor/codemirror';
  import * as encryption from '$lib/stores/encryption.svelte';
  import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

  import { getCanvasBgColor, getCanvasColor } from './canvas-colors';

  const { data, selected } = $props<{ data: Record<string, unknown>; selected?: boolean }>();

  const file = $derived((data.file as string) || '');
  const noteId = $derived((data['x-xelanote-note-id'] as string) || '');
  const color = $derived(data.color as string | undefined);
  const borderColor = $derived(getCanvasColor(color));
  const bgColor = $derived(getCanvasBgColor(color));

  // Strip query parameters before checking extension
  const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.webp'];
  const filePathname = $derived.by(() => {
    try {
      return new URL(file, 'http://x').pathname;
    } catch {
      return file;
    }
  });
  const isImage = $derived(imageExtensions.some((ext) => filePathname.toLowerCase().endsWith(ext)));

  // Build the full image src URL for uploaded images
  // Upload URLs are like /api/uploads/{userId}/{filename} — works as relative path in web,
  // but needs the server base URL for desktop (Tauri) where getApiBaseUrl() returns http://...
  const imageSrc = $derived.by(() => {
    if (!file.startsWith('/api/uploads/')) return file;
    const base = getApiBaseUrl();
    // Web: base is "/api" → strip to get "" + file = "/api/uploads/..."
    // Desktop: base is "http://host:port/api" → strip to get "http://host:port" + file
    return base.replace(/\/api$/, '') + file;
  });

  // Track image load error to fall back to icon
  let imageError = $state(false);
  // Reset error state when file changes
  $effect(() => {
    void file;
    imageError = false;
  });

  // Note content fetching — no synchronous $state writes inside the effect
  // to avoid re-render loops with SvelteFlow.
  let noteContent = $state<string | null>(null);
  let loadState = $state<'idle' | 'loading' | 'locked' | 'error'>('idle');
  let fetchedForId = ''; // plain var, not reactive — prevents duplicate fetches

  $effect(() => {
    const id = noteId;
    const encryptionUnlocked = encryption.isEncryptionUnlocked();
    if (
      !id ||
      (id === fetchedForId && !(loadState === 'locked' && encryptionUnlocked))
    )
      return;
    fetchedForId = id;
    loadState = 'loading';
    noteContent = null;
    getNote(id)
      .then((note) => {
        if (fetchedForId !== id) return;
        if (note.content_encrypted && note.encrypted_content) {
          if (!encryption.isEncryptionUnlocked()) {
            loadState = 'locked';
            noteContent = null;
            return;
          }
          try {
            const decrypted = encryption.decryptNote(note.encrypted_title || null, {
              ciphertext: note.encrypted_content,
              metadata: parseEncryptionMetadata(note.encryption_metadata),
            });
            noteContent = decrypted.content;
            loadState = 'idle';
            return;
          } catch {
            loadState = 'error';
            noteContent = null;
            return;
          }
        }
        noteContent = note.content || '';
        loadState = 'idle';
      })
      .catch(() => {
        if (fetchedForId === id) {
          loadState = 'error';
          noteContent = null;
          fetchedForId = ''; // allow retry
        }
      });
  });

  // Read-only CodeMirror editor for note preview.
  // editorView is a plain variable (not $state) to avoid reactive writes inside $effect.
  let editorContainer: HTMLDivElement | undefined = $state();
  let editorView: EditorView | undefined;

  // Mount/destroy editor when container appears/disappears
  $effect(() => {
    if (!editorContainer) return;
    const text = untrack(() => noteContent || '');
    editorView = createCanvasEditor(editorContainer, {
      doc: text,
      readOnly: true,
    });
    return () => {
      editorView?.destroy();
      editorView = undefined;
    };
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
  class="canvas-file-node"
  class:selected
  class:has-image={isImage && !imageError}
  style:border-left-color={borderColor}
  style:background={bgColor ? `color-mix(in oklch, ${bgColor} 40%, var(--color-card))` : undefined}
>
  {#if isImage && !imageError}
    <div class="canvas-file-image">
      <img src={imageSrc} alt={file} draggable="false" onerror={() => (imageError = true)} />
    </div>
  {:else}
    <div class="canvas-file-header">
      <FileText size={16} class="text-muted-foreground shrink-0" />
      <span class="canvas-file-title">{file}</span>
    </div>
    {#if data.subpath}
      <div class="canvas-file-subpath">{data.subpath}</div>
    {/if}
    {#if noteId && noteContent !== null}
      <div class="canvas-file-preview nodrag nowheel nopan" bind:this={editorContainer}></div>
    {:else if noteId && loadState === 'locked'}
      <div class="canvas-file-placeholder">Encrypted note is locked</div>
    {:else if noteId && loadState === 'error'}
      <div class="canvas-file-placeholder">Failed to load note</div>
    {:else if noteId}
      <div class="canvas-file-placeholder">Loading...</div>
    {:else}
      <div class="canvas-file-placeholder">Click to open note</div>
    {/if}
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

  .canvas-file-node.has-image {
    padding: 4px;
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
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }

  .canvas-file-preview :global(.cm-editor) {
    height: 100%;
  }

  .canvas-file-preview :global(.cm-scroller) {
    overflow: auto;
  }

  .canvas-file-placeholder {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
    flex: 1;
    overflow: hidden;
  }

  .canvas-file-image {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    overflow: hidden;
  }

  .canvas-file-image img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
</style>
