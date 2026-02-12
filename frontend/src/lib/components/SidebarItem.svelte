<script lang="ts">
  import type { ComponentType } from 'svelte';

  interface Props {
    icon?: ComponentType;
    label: string;
    isActive?: boolean;
    count?: number;
    isDraggable?: boolean;
    onContextMenu?: (event: MouseEvent) => void;
    onClick?: () => void;
    onDragStart?: (event: DragEvent) => void;
    onDragEnd?: (event: DragEvent) => void;
  }

  const {
    icon: IconComponent = undefined,
    label,
    isActive = false,
    count = 0,
    isDraggable = false,
    onContextMenu,
    onClick,
    onDragStart,
    onDragEnd,
  }: Props = $props();

  let isHovered = $state(false);

  const iconSize = 16;

  function handleContextMenu(e: MouseEvent) {
    e.preventDefault();
    onContextMenu?.(e);
  }

  function handleDragStart(e: DragEvent) {
    if (!isDraggable) return;
    onDragStart?.(e);
  }

  function handleDragEnd(e: DragEvent) {
    onDragEnd?.(e);
  }
</script>

<div
  class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm cursor-pointer transition-all duration-fast ease-default
		{isActive
    ? 'bg-sidebar-accent/40 text-sidebar-primary border-l-3 border-l-sidebar-primary'
    : 'text-sidebar-foreground hover:bg-sidebar-accent/30'}"
  role="button"
  tabindex="0"
  draggable={isDraggable}
  onmouseenter={() => (isHovered = true)}
  onmouseleave={() => (isHovered = false)}
  oncontextmenu={handleContextMenu}
  onclick={onClick}
  ondragstart={handleDragStart}
  ondragend={handleDragEnd}
  onkeydown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onClick?.();
    }
  }}
>
  <!-- Icon -->
  {#if IconComponent}
    <IconComponent size={iconSize} class="flex-shrink-0 opacity-80" />
  {/if}

  <!-- Label -->
  <span class="flex-1 truncate">{label}</span>

  <!-- Count Badge -->
  {#if count > 0}
    <span
      class="text-xs bg-sidebar-accent/50 text-sidebar-accent-foreground rounded-full px-1.5 py-0.5 flex-shrink-0"
    >
      {count}
    </span>
  {/if}

  <!-- Drag Handle (shown on hover) -->
  {#if isDraggable && isHovered}
    <div class="flex-shrink-0 opacity-40 hover:opacity-70 transition-opacity">
      <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
        <circle cx="9" cy="5" r="1.5"></circle>
        <circle cx="9" cy="12" r="1.5"></circle>
        <circle cx="9" cy="19" r="1.5"></circle>
        <circle cx="15" cy="5" r="1.5"></circle>
        <circle cx="15" cy="12" r="1.5"></circle>
        <circle cx="15" cy="19" r="1.5"></circle>
      </svg>
    </div>
  {/if}
</div>

<style>
  div {
    transition-property: background-color, color, border-color;
    transition-timing-function: var(--ease-default);
    transition-duration: var(--duration-fast);
  }

  @media (hover: hover) {
    div:hover:not([aria-disabled='true']) {
      background-color: var(--color-sidebar-accent);
      opacity: 0.7;
    }
  }

  /* Active state styling */
  div:is(.active) {
    background-color: var(--color-sidebar-accent);
    color: var(--color-sidebar-primary);
    border-left: 3px solid var(--color-sidebar-primary);
    font-weight: 500;
  }

  /* Focus visible styling */
  div:focus-visible {
    outline: 2px solid var(--color-sidebar-ring);
    outline-offset: -2px;
  }

  /* Drag state */
  div[draggable='true']:active {
    opacity: 0.7;
  }
</style>
