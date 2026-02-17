/**
 * Bottom sheet swipe-to-dismiss action for Svelte
 *
 * Adds touch-based swipe-down-to-dismiss behavior to a mobile bottom sheet.
 * The sheet translates downward as the user drags, and dismisses when the
 * swipe distance or velocity exceeds a threshold.
 *
 * Only active on viewports < 640px (sm breakpoint).
 * Respects scrollable inner content: the dismiss gesture only activates
 * when no ancestor between the touch target and the sheet is scrolled down.
 */

export interface BottomSheetOptions {
  /** Called when the user swipes the sheet away */
  onClose: () => void;
  /** Distance in px to trigger dismiss (default: 80) */
  threshold?: number;
  /** Minimum velocity in px/ms to trigger dismiss on shorter distances (default: 0.4) */
  velocityThreshold?: number;
}

export function bottomsheet(node: HTMLElement, options: BottomSheetOptions) {
  const { onClose, threshold = 80, velocityThreshold = 0.4 } = options;

  let startY = 0;
  let currentY = 0;
  let startTime = 0;
  let dragging = false;
  let dismissed = false;

  /**
   * Walk from target up to the sheet node and return the first vertically
   * scrollable ancestor (overflow-y auto/scroll with actual overflow).
   */
  function findScrollableAncestor(target: HTMLElement): HTMLElement | null {
    let el: HTMLElement | null = target;
    while (el && el !== node) {
      if (el.scrollHeight > el.clientHeight) {
        const style = getComputedStyle(el);
        if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
          return el;
        }
      }
      el = el.parentElement;
    }
    return null;
  }

  function handleTouchStart(e: TouchEvent) {
    // Only on mobile-width viewports
    if (window.innerWidth >= 640) return;
    if (dismissed) return;

    // If touch started inside a scrollable area that is scrolled down, let it scroll
    const scrollable = findScrollableAncestor(e.target as HTMLElement);
    if (scrollable && scrollable.scrollTop > 0) return;

    startY = e.touches[0].clientY;
    currentY = startY;
    startTime = Date.now();
    dragging = false;
  }

  function handleTouchMove(e: TouchEvent) {
    if (window.innerWidth >= 640 || dismissed) return;
    if (startTime === 0) return;

    const touchY = e.touches[0].clientY;
    const deltaY = touchY - startY;

    // Only handle downward movement
    if (deltaY <= 0) {
      if (dragging) {
        node.style.transform = '';
        node.style.transition = 'none';
        dragging = false;
      }
      return;
    }

    dragging = true;
    currentY = touchY;

    // Dampen the translation slightly for a natural feel
    node.style.transition = 'none';
    node.style.transform = `translateY(${deltaY}px)`;
  }

  function handleTouchEnd() {
    if (!dragging || dismissed) {
      startTime = 0;
      return;
    }

    const deltaY = currentY - startY;
    const elapsed = Math.max(Date.now() - startTime, 1);
    const velocity = deltaY / elapsed;

    dragging = false;
    startTime = 0;

    if (deltaY >= threshold || velocity >= velocityThreshold) {
      // Dismiss: slide fully out, then call onClose
      dismissed = true;
      node.style.transition = 'transform 0.2s ease-out';
      node.style.transform = 'translateY(100%)';
      setTimeout(() => {
        onClose();
      }, 200);
    } else {
      // Snap back
      node.style.transition = 'transform 0.2s ease-out';
      node.style.transform = '';
    }
  }

  function handleTouchCancel() {
    if (dragging && !dismissed) {
      node.style.transition = 'transform 0.2s ease-out';
      node.style.transform = '';
    }
    dragging = false;
    startTime = 0;
  }

  node.addEventListener('touchstart', handleTouchStart, { passive: true });
  node.addEventListener('touchmove', handleTouchMove, { passive: true });
  node.addEventListener('touchend', handleTouchEnd, { passive: true });
  node.addEventListener('touchcancel', handleTouchCancel, { passive: true });

  return {
    destroy() {
      node.removeEventListener('touchstart', handleTouchStart);
      node.removeEventListener('touchmove', handleTouchMove);
      node.removeEventListener('touchend', handleTouchEnd);
      node.removeEventListener('touchcancel', handleTouchCancel);
    },
  };
}
