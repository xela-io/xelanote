<script lang="ts">
  import { untrack, type Snippet } from 'svelte';
  import { ChevronDown } from 'lucide-svelte';
  import { animationDurations, easing } from '$lib/design/tokens';

  interface Props {
    title: string;
    collapsible?: boolean;
    defaultOpen?: boolean;
    onToggle?: (isOpen: boolean) => void;
    children?: Snippet;
  }

  const { title, collapsible = false, defaultOpen = true, onToggle, children }: Props = $props();

  let isOpen = $state(untrack(() => defaultOpen));

  function toggleOpen() {
    isOpen = !isOpen;
    onToggle?.(isOpen);
  }
</script>

<section class="flex flex-col">
  <!-- Section Header -->
  {#if collapsible}
    <button
      onclick={toggleOpen}
      class="flex items-center justify-between px-4 py-2 text-xs text-sidebar-foreground uppercase tracking-wider
				border-t border-sidebar-border hover:bg-sidebar-accent/20
				transition-colors duration-[{animationDurations.fast}ms] ease-[{easing.default}]
				focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sidebar-ring"
      aria-expanded={isOpen}
    >
      <span>{title}</span>
      <ChevronDown
        size={16}
        class="transition-transform duration-[{animationDurations.base}ms] ease-[{easing.default}]
					{isOpen ? 'rotate-0' : '-rotate-90'}"
      />
    </button>
  {:else}
    <div
      class="px-4 py-2 text-xs text-sidebar-foreground uppercase tracking-wider border-t border-sidebar-border"
    >
      {title}
    </div>
  {/if}

  <!-- Section Content -->
  {#if !collapsible || isOpen}
    <div
      class="overflow-hidden max-h-screen transition-all duration-[{animationDurations.slow}ms] ease-[{easing.default}]"
      style={`max-height: ${isOpen ? '500px' : '0'}; opacity: ${isOpen ? '1' : '0'}`}
    >
      {@render children?.()}
    </div>
  {/if}
</section>

<style>
  section {
    transition-property: all;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  }

  button:hover {
    background-color: var(--color-sidebar-accent);
    opacity: 0.3;
  }

  button:focus-visible {
    outline: 2px solid var(--color-sidebar-ring);
    outline-offset: -2px;
  }

  div {
    transition:
      max-height 300ms cubic-bezier(0.4, 0, 0.2, 1),
      opacity 200ms cubic-bezier(0.4, 0, 0.2, 1);
  }
</style>
