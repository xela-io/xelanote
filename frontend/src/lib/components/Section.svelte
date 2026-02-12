<script lang="ts">
  import { ChevronDown } from 'lucide-svelte';
  import { type Snippet, untrack } from 'svelte';

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
				transition-colors duration-fast ease-default
				focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sidebar-ring"
      aria-expanded={isOpen}
    >
      <span>{title}</span>
      <ChevronDown
        size={16}
        class="transition-transform duration-base ease-default
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
      class="overflow-hidden max-h-screen transition-all duration-slow ease-default"
      style={`max-height: ${isOpen ? '500px' : '0'}; opacity: ${isOpen ? '1' : '0'}`}
    >
      {@render children?.()}
    </div>
  {/if}
</section>

<style>
  section {
    transition-property: all;
    transition-timing-function: var(--ease-default);
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
      max-height var(--duration-slow) var(--ease-default),
      opacity var(--duration-base) var(--ease-default);
  }
</style>
