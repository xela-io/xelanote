<script lang="ts">
  import { ChevronDown, ChevronUp, GripVertical } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    title: string;
    subtitle: string;
    collapsed: boolean;
    isDragOver: boolean;
    order: number;
    ariaLabel: string;
    onToggle: () => void;
    onDragStart: (event: DragEvent) => void;
    onDragOver: (event: DragEvent) => void;
    onDrop: (event: DragEvent) => void;
    onDragEnd: () => void;
    children: Snippet;
  }

  const {
    title,
    subtitle,
    collapsed,
    isDragOver,
    order,
    ariaLabel,
    onToggle,
    onDragStart,
    onDragOver,
    onDrop,
    onDragEnd,
    children,
  }: Props = $props();
</script>

<section
  role="group"
  aria-label={ariaLabel}
  ondragover={onDragOver}
  ondrop={onDrop}
  style={`order: ${order}`}
  class={`ui-panel p-4 sm:p-5 ${isDragOver ? 'border-primary/50 ring-1 ring-primary/20' : ''}`}
>
  <div class="mb-3 flex items-start justify-between gap-3">
    <div>
      <div class="text-[11px] uppercase tracking-[0.12em] text-muted-foreground">{title}</div>
      <div class="mt-1 text-sm font-medium text-foreground">{subtitle}</div>
    </div>
    <div class="flex items-center gap-1">
      <button
        type="button"
        draggable="true"
        ondragstart={onDragStart}
        ondragend={onDragEnd}
        class="ui-icon-button ui-icon-button-sm"
        title={$_('component.dashboard_section.reorder_title')}
        aria-label={$_('component.dashboard_section.reorder_title')}
      >
        <GripVertical size={14} />
      </button>
      <button
        type="button"
        onclick={onToggle}
        class="ui-icon-button ui-icon-button-sm"
        aria-expanded={!collapsed}
        aria-label={$_(
          collapsed ? 'component.dashboard_section.expand' : 'component.dashboard_section.collapse'
        )}
      >
        {#if collapsed}
          <ChevronDown size={14} />
        {:else}
          <ChevronUp size={14} />
        {/if}
      </button>
    </div>
  </div>

  {#if collapsed}
    <div class="text-xs text-muted-foreground">{$_('page.home.collapsed')}</div>
  {:else}
    {@render children()}
  {/if}
</section>
