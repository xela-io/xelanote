<script lang="ts">
  import { PenLine } from 'lucide-svelte';

  interface Props {
    size?: 'sm' | 'md' | 'lg' | 'xl';
    variant?: 'default' | 'badge';
    uppercase?: boolean;
    showIcon?: boolean;
  }

  const { size = 'md', variant = 'default', uppercase = false, showIcon = true }: Props = $props();

  // Icon-Größen passend zu Text-Größen
  const iconSizes = { sm: 14, md: 16, lg: 22, xl: 28 };
</script>

<span
  class="logo-wrapper {size}"
  class:uppercase
  class:has-icon={showIcon && variant !== 'badge'}
  role="img"
  aria-label="xelanote"
>
  {#if showIcon && variant !== 'badge'}
    <PenLine size={iconSizes[size]} class="logo-icon" />
  {/if}
  <span class="logo-text {variant}">xelanote</span>
</span>

<style>
  .logo-wrapper {
    display: inline-flex;
    align-items: center;
    gap: 0.35em;
    transition:
      transform var(--duration-base) var(--ease-default),
      filter var(--duration-slow) var(--ease-default);
    user-select: none;
    cursor: default;
  }

  .logo-wrapper.has-icon:hover {
    transform: scale(1.03);
    filter: drop-shadow(0 0 8px var(--color-sidebar-primary));
  }

  .logo-wrapper :global(.logo-icon) {
    color: var(--color-sidebar-primary);
    transition: transform var(--duration-slow) var(--ease-default);
    flex-shrink: 0;
  }

  .logo-wrapper:hover :global(.logo-icon) {
    transform: rotate(-15deg);
  }

  .logo-text {
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--color-sidebar-primary); /* Fallback */
    background: linear-gradient(
      135deg,
      var(--color-sidebar-primary) 0%,
      color-mix(in oklch, var(--color-sidebar-primary) 60%, oklch(70% 0.18 300)) 50%,
      color-mix(in oklch, var(--color-sidebar-primary) 40%, oklch(65% 0.2 330)) 100%
    );
    background-size: 200% 200%;
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
    animation: gradient-shift 8s ease-in-out infinite;
  }

  /* Desktop: only animate on hover */
  @media (hover: hover) {
    .logo-text {
      animation-play-state: paused;
    }
    .logo-wrapper:hover .logo-text {
      animation-play-state: running;
      animation-duration: 3s;
    }
  }

  /* Touch: slow subtle animation */
  @media (hover: none) {
    .logo-text {
      animation-duration: 16s;
    }
  }

  @keyframes gradient-shift {
    0%,
    100% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
  }

  .sm {
    font-size: 0.875rem;
  }
  .md {
    font-size: 1rem;
  }
  .lg {
    font-size: 1.5rem;
  }
  .xl {
    font-size: 2rem;
  }

  .uppercase {
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .badge {
    background: var(--color-sidebar-primary);
    -webkit-text-fill-color: var(--color-sidebar-primary-foreground);
    color: var(--color-sidebar-primary-foreground);
    padding: 0.4rem 0.8rem;
    border-radius: var(--radius-md);
    font-size: 0.8rem;
    animation: none;
  }

  /* Fallback für Browser ohne color-mix */
  @supports not (color: color-mix(in oklch, red, blue)) {
    .logo-text:not(.badge) {
      background: linear-gradient(135deg, var(--color-sidebar-primary), var(--color-ring));
      background-size: 200% 200%;
      -webkit-background-clip: text;
      background-clip: text;
    }
  }

  /* Fallback für sehr alte Browser */
  @supports not (background-clip: text) {
    .logo-text:not(.badge) {
      color: var(--color-sidebar-primary);
      background: none;
      -webkit-text-fill-color: initial;
      animation: none;
    }
  }

  /* Reduzierte Bewegung respektieren */
  @media (prefers-reduced-motion: reduce) {
    .logo-wrapper {
      transition: none;
    }
    .logo-wrapper:hover {
      transform: none;
    }
    .logo-wrapper:hover :global(.logo-icon) {
      transform: none;
    }
    .logo-text {
      animation: none;
      background-position: 0% 50%;
    }
    .logo-wrapper:hover .logo-text {
      animation: none;
    }
  }
</style>
