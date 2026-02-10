<script lang="ts">
  import type { ComponentType, Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  import { animationDurations, easing } from '$lib/design/tokens';

  type IconComponent = ComponentType;

  interface Props extends HTMLButtonAttributes {
    variant?: 'primary' | 'secondary' | 'ghost' | 'outline' | 'destructive';
    size?: 'sm' | 'md' | 'lg';
    icon?: IconComponent;
    iconPosition?: 'left' | 'right';
    loading?: boolean;
    fullWidth?: boolean;
    iconOnly?: boolean;
    children?: Snippet;
  }

  const {
    variant = 'primary',
    size = 'md',
    icon: IconComponent = undefined,
    iconPosition = 'left',
    loading = false,
    fullWidth = false,
    iconOnly = false,
    disabled = false,
    children,
    ...rest
  }: Props = $props();

  const iconSize = $derived.by(() => (size === 'sm' ? 16 : size === 'md' ? 18 : 20));

  // Compute classes based on variant and size
  const getVariantClasses = (variant: string) => {
    switch (variant) {
      case 'primary':
        return 'bg-primary text-primary-foreground hover:shadow-md focus-visible:shadow-md';
      case 'secondary':
        return 'bg-secondary text-secondary-foreground hover:shadow-md focus-visible:shadow-md';
      case 'ghost':
        return 'bg-transparent text-foreground hover:bg-accent/50 focus-visible:bg-accent/30';
      case 'outline':
        return 'bg-transparent text-foreground border border-border hover:bg-accent/50 focus-visible:bg-accent/30';
      case 'destructive':
        return 'bg-destructive text-destructive-foreground hover:shadow-md focus-visible:shadow-md';
      default:
        return '';
    }
  };

  const getSizeClasses = (size: string) => {
    switch (size) {
      case 'sm':
        return 'px-3 py-1.5 text-sm h-8';
      case 'md':
        return 'px-4 py-2 text-base h-10';
      case 'lg':
        return 'px-6 py-3 text-base h-12';
      default:
        return '';
    }
  };
</script>

<button
  class="{getVariantClasses(variant)} {getSizeClasses(size)}
		flex items-center justify-center gap-2 rounded-lg font-medium
		transition-all duration-[{animationDurations.fast}ms] ease-[{easing.default}]
		focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring
		disabled:opacity-50 disabled:cursor-not-allowed
		{fullWidth ? 'w-full' : ''}
		{loading || disabled ? 'pointer-events-none' : 'hover:shadow-md active:scale-98'}"
  {disabled}
  {...rest}
>
  {#if loading}
    <svg
      class="animate-spin h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
      ></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
  {:else if IconComponent && iconPosition === 'left'}
    <IconComponent size={iconSize} class="flex-shrink-0" />
  {/if}

  {#if !iconOnly}
    <span>{@render children?.()}</span>
  {/if}

  {#if IconComponent && iconPosition === 'right' && !loading}
    <IconComponent size={iconSize} class="flex-shrink-0" />
  {/if}
</button>

<style>
  button {
    /* Ensure smooth transitions for color and shadow changes */
    transition-property: all;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    transition-duration: 150ms;
  }

  button:active:not(:disabled) {
    /* Press effect - scale down briefly */
    transform: scale(0.98);
  }

  button:hover:not(:disabled) {
    /* Lift effect */
    transform: translateY(-1px);
  }

  /* Focus visible styling */
  button:focus-visible {
    outline: 2px solid var(--color-ring);
    outline-offset: 2px;
  }

  /* Disabled state */
  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
