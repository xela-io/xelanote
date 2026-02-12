<script lang="ts">
  import type { ComponentType, Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  import { animationDurations, easing } from '$lib/design/tokens';

  type IconComponent = ComponentType;

  interface Props extends HTMLButtonAttributes {
    icon?: IconComponent;
    iconOnly?: boolean;
    variant?: 'primary' | 'secondary' | 'ghost' | 'destructive';
    children?: Snippet;
  }

  const {
    icon: IconComponent = undefined,
    iconOnly = false,
    variant = 'primary',
    disabled = false,
    children,
    ...rest
  }: Props = $props();

  const iconSize = 18;

  const getVariantClasses = (variant: string) => {
    switch (variant) {
      case 'primary':
        return 'text-sidebar-primary hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent/50';
      case 'secondary':
        return 'text-sidebar-foreground hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent/50';
      case 'ghost':
        return 'text-sidebar-foreground hover:bg-sidebar-accent/30 focus-visible:bg-sidebar-accent/30';
      case 'destructive':
        return 'text-destructive hover:bg-red-500/10 focus-visible:bg-red-500/10 hover:text-red-600';
      default:
        return '';
    }
  };
</script>

<button
  class="{getVariantClasses(variant)}
		flex items-center justify-start gap-3 px-4 py-2.5 rounded-lg text-sm
		transition-all duration-[{animationDurations.fast}ms] ease-[{easing.default}]
		focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sidebar-ring
		disabled:opacity-50 disabled:cursor-not-allowed hover:no-underline
		w-full text-left"
  {disabled}
  {...rest}
>
  {#if IconComponent}
    <IconComponent size={iconSize} class="flex-shrink-0" />
  {/if}

  {#if !iconOnly}
    <span class="truncate">{@render children?.()}</span>
  {/if}
</button>

<style>
  button {
    transition-property: background-color, color, border-color;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    transition-duration: 150ms;
  }

  button:active:not(:disabled) {
    transform: scale(0.98);
    transition-duration: 75ms;
  }

  @media (hover: hover) {
    button:hover:not(:disabled) {
      transform: translateY(-1px);
    }
  }

  button:focus-visible {
    outline: 2px solid var(--color-sidebar-ring);
    outline-offset: -2px;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
