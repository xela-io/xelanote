<script module>
  let counter = 0;
  let scrollLockCount = 0;

  interface StackEntry {
    id: string;
    onClose: () => void;
    trap: import('focus-trap').FocusTrap;
  }

  const dialogStack: StackEntry[] = [];

  function lockScroll() {
    if (++scrollLockCount === 1) document.body.style.overflow = 'hidden';
  }

  function unlockScroll() {
    if (--scrollLockCount === 0) document.body.style.overflow = '';
  }
</script>

<script lang="ts">
  import { createFocusTrap, type FocusTrap } from 'focus-trap';
  import { X } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    open: boolean;
    title: string;
    onClose: () => void;
    size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl';
    variant?: 'default' | 'danger';
    showCloseButton?: boolean;
    closeOnBackdrop?: boolean;
    closeOnEscape?: boolean;
    scrollable?: boolean;
    footerAlign?: 'end' | 'between';
    content?: Snippet;
    footer?: Snippet;
  }

  const {
    open,
    title,
    onClose,
    size = 'md',
    variant = 'default',
    showCloseButton = true,
    closeOnBackdrop = true,
    closeOnEscape = true,
    scrollable = false,
    footerAlign = 'end',
    content,
    footer,
  }: Props = $props();

  const dialogId = `dialog-${counter++}`;

  let dialogRef: HTMLDivElement | null = $state(null);
  let focusTrap: FocusTrap | null = null;
  let previousActiveElement: Element | null = null;

  const sizeClasses: Record<string, string> = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-5xl',
  };

  const variantClasses: Record<string, string> = {
    default: '',
    danger: 'border-destructive/50',
  };

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && closeOnEscape && dialogStack.at(-1)?.id === dialogId) {
      e.preventDefault();
      onClose();
    }
  }

  function handleBackdropClick() {
    if (closeOnBackdrop) {
      onClose();
    }
  }

  $effect(() => {
    if (open && dialogRef) {
      // Store the currently focused element to restore later
      previousActiveElement = document.activeElement;

      // Pause the previous top-of-stack trap
      const previousTrap = dialogStack.at(-1)?.trap;
      previousTrap?.pause();

      // Create and activate focus trap
      focusTrap = createFocusTrap(dialogRef, {
        escapeDeactivates: false, // We handle escape ourselves
        allowOutsideClick: true,
        fallbackFocus: dialogRef,
        returnFocusOnDeactivate: false, // We handle this ourselves
      });

      // Small delay to ensure DOM is ready
      requestAnimationFrame(() => {
        focusTrap?.activate();
      });

      // Push to dialog stack
      dialogStack.push({ id: dialogId, onClose, trap: focusTrap! });

      // Prevent body scroll (reference-counted)
      lockScroll();

      return () => {
        // Remove from dialog stack
        const idx = dialogStack.findIndex((d) => d.id === dialogId);
        if (idx !== -1) dialogStack.splice(idx, 1);

        // Cleanup focus trap
        focusTrap?.deactivate();
        focusTrap = null;

        // Unpause the new top-of-stack trap
        const newTop = dialogStack.at(-1)?.trap;
        newTop?.unpause();

        // Unlock scroll (reference-counted)
        unlockScroll();

        // Restore focus to previous element
        if (previousActiveElement instanceof HTMLElement) {
          previousActiveElement.focus();
        }
      };
    }
  });
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

{#if open}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/50 z-50 animate-fade-in"
    aria-hidden="true"
    onclick={handleBackdropClick}
    role="presentation"
  ></div>

  <!-- Dialog Container -->
  <div
    bind:this={dialogRef}
    role="dialog"
    aria-modal="true"
    aria-labelledby={dialogId}
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
    tabindex="-1"
  >
    <!-- Dialog Panel -->
    <div
      class="bg-background border border-border rounded-lg shadow-lg w-full {sizeClasses[
        size
      ]} {variantClasses[variant]} animate-scale-up"
      onclick={(e) => e.stopPropagation()}
      role="presentation"
    >
      <!-- Header -->
      <header class="flex items-center justify-between p-4 border-b border-border">
        <h2 id={dialogId} class="text-lg font-semibold text-foreground">
          {title}
        </h2>
        {#if showCloseButton}
          <button
            type="button"
            onclick={onClose}
            class="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            aria-label={$_('accessibility.close_dialog')}
          >
            <X size={18} />
          </button>
        {/if}
      </header>

      <!-- Content -->
      {#if content}
        <div class="p-4 {scrollable ? 'overflow-y-auto max-h-[70vh]' : ''}">
          {@render content()}
        </div>
      {/if}

      <!-- Footer -->
      {#if footer}
        <footer
          class="flex items-center gap-2 p-4 border-t border-border bg-muted/30"
          class:justify-end={footerAlign === 'end'}
          class:justify-between={footerAlign === 'between'}
        >
          {@render footer()}
        </footer>
      {/if}
    </div>
  </div>
{/if}
