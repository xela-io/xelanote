<script lang="ts">
  import { Columns, Edit, Eye, ScanEye } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  type EditorMode = 'edit' | 'split' | 'preview' | 'live';

  interface Props {
    editorMode: EditorMode;
    isMobile?: boolean;
    onSetEditorMode: (mode: EditorMode) => void;
  }

  const { editorMode, isMobile = false, onSetEditorMode }: Props = $props();

  const modeOptions = $derived.by(() => {
    const base = [
      { value: 'live' as const, label: $_('component.editor.toolbar.mode_live') },
      { value: 'edit' as const, label: $_('component.editor.toolbar.mode_edit') },
      { value: 'preview' as const, label: $_('component.editor.toolbar.mode_preview') },
    ];
    if (isMobile) return base;
    return [...base, { value: 'split' as const, label: $_('component.editor.toolbar.mode_split') }];
  });

  function handleSelectChange(e: Event) {
    const next = (e.currentTarget as HTMLSelectElement).value as EditorMode;
    onSetEditorMode(next);
  }
</script>

{#if isMobile}
  <label class="sr-only" for="editor-mode-select">
    {$_('component.editor.toolbar.more_options')}
  </label>
  <select
    id="editor-mode-select"
    class="h-8 rounded-md border border-border bg-background px-2 text-xs text-foreground hover:bg-accent toolbar-btn flex-shrink-0"
    value={editorMode}
    onchange={handleSelectChange}
    aria-label={$_('component.editor.toolbar.mode_live')}
  >
    {#each modeOptions as mode (mode.value)}
      <option value={mode.value}>{mode.label}</option>
    {/each}
  </select>
{:else}
  <div
    class="inline-flex items-center rounded-md border border-border bg-background p-0.5 gap-0.5 flex-shrink-0"
    role="radiogroup"
    aria-label={$_('component.editor.toolbar.editor_toolbar')}
  >
    {#each modeOptions as mode (mode.value)}
      <button
        type="button"
        class="h-8 w-8 rounded-md inline-flex items-center justify-center hover:bg-accent toolbar-btn"
        class:bg-accent={editorMode === mode.value}
        role="radio"
        aria-checked={editorMode === mode.value}
        aria-label={mode.label}
        title={mode.label}
        onclick={() => onSetEditorMode(mode.value)}
      >
        {#if mode.value === 'live'}
          <ScanEye size={16} />
        {:else if mode.value === 'edit'}
          <Edit size={16} />
        {:else if mode.value === 'preview'}
          <Eye size={16} />
        {:else}
          <Columns size={16} />
        {/if}
      </button>
    {/each}
  </div>
{/if}
