<script lang="ts">
  import { Check, ChevronDown, Columns, Edit, Eye, ScanEye } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  type EditorMode = 'edit' | 'split' | 'preview' | 'live';

  interface Props {
    editorMode: EditorMode;
    isMobile?: boolean;
    onSetEditorMode: (mode: EditorMode) => void;
  }

  const { editorMode, isMobile = false, onSetEditorMode }: Props = $props();

  let mobileMenuOpen = $state(false);
  let mobileTriggerRect = $state<DOMRect | null>(null);

  const modeOptions = $derived.by(() => {
    const base = [
      { value: 'live' as const, label: $_('component.editor.toolbar.mode_live') },
      { value: 'edit' as const, label: $_('component.editor.toolbar.mode_edit') },
      { value: 'preview' as const, label: $_('component.editor.toolbar.mode_preview') },
    ];
    if (isMobile) return base;
    return [...base, { value: 'split' as const, label: $_('component.editor.toolbar.mode_split') }];
  });

  function getModeLabel(mode: EditorMode) {
    if (mode === 'live') return $_('component.editor.toolbar.mode_live');
    if (mode === 'edit') return $_('component.editor.toolbar.mode_edit');
    if (mode === 'preview') return $_('component.editor.toolbar.mode_preview');
    return $_('component.editor.toolbar.mode_split');
  }

  const mobileMenuStyle = $derived.by(() => {
    if (!mobileTriggerRect) return '';
    const top = mobileTriggerRect.bottom + 6;
    const minWidth = Math.max(168, mobileTriggerRect.width);
    const maxLeft = Math.max(8, window.innerWidth - minWidth - 8);
    const left = Math.min(maxLeft, Math.max(8, mobileTriggerRect.left));
    return `top: ${top}px; left: ${left}px; min-width: ${minWidth}px; max-width: calc(100vw - 16px);`;
  });

  function toggleMobileMenu(e: MouseEvent) {
    mobileTriggerRect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    mobileMenuOpen = !mobileMenuOpen;
  }

  function closeMobileMenu() {
    mobileMenuOpen = false;
  }

  function selectMobileMode(mode: EditorMode) {
    onSetEditorMode(mode);
    closeMobileMenu();
  }
</script>

{#if isMobile}
  <button
    type="button"
    class="h-8 rounded-md border border-border bg-background px-2 text-xs text-foreground hover:bg-accent toolbar-btn flex-shrink-0 inline-flex items-center gap-1"
    onclick={toggleMobileMenu}
    aria-label={$_('component.editor.toolbar.mode_group')}
    aria-expanded={mobileMenuOpen}
    aria-haspopup="menu"
    title={getModeLabel(editorMode)}
  >
    <span class="truncate max-w-[6.5rem]">{getModeLabel(editorMode)}</span>
    <ChevronDown size={14} class={`transition-transform ${mobileMenuOpen ? 'rotate-180' : ''}`} />
  </button>

  {#if mobileMenuOpen}
    <button
      type="button"
      class="fixed inset-0 z-40 bg-transparent"
      onclick={closeMobileMenu}
      aria-label={$_('component.editor.toolbar.more_options')}
    ></button>

    <div
      class="fixed z-50 rounded-lg border border-border bg-background shadow-lg p-1.5"
      style={mobileMenuStyle}
      role="menu"
      aria-label={$_('component.editor.toolbar.mode_group')}
    >
      <div class="px-2 pb-1 text-[10px] uppercase tracking-wide text-muted-foreground">
        {$_('component.editor.toolbar.mode_group')}
      </div>
      {#each modeOptions as mode (mode.value)}
        <button
          type="button"
          class="w-full flex items-center gap-2 px-2 py-1.5 text-left text-xs rounded-md hover:bg-accent"
          onclick={() => selectMobileMode(mode.value)}
          role="menuitemradio"
          aria-checked={editorMode === mode.value}
        >
          {#if editorMode === mode.value}
            <Check size={14} />
          {:else}
            <span class="w-[14px]"></span>
          {/if}
          <span>{mode.label}</span>
        </button>
      {/each}
    </div>
  {/if}
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
