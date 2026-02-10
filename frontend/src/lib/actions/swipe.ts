/**
 * Swipe gesture action for Svelte
 *
 * Handles touch/pointer-based swipe gestures with edge detection,
 * iOS Safari back-gesture avoidance, and scrollable element handling.
 */

export interface SwipeOptions {
  /** Start of edge zone in pixels (default: 15px to avoid iOS back gesture) */
  edgeZoneStart?: number;
  /** End of edge zone in pixels (default: 50px) */
  edgeZoneEnd?: number;
  /** Minimum horizontal distance to trigger swipe (default: 50px) */
  threshold?: number;
  /** Maximum vertical drift allowed (default: 75px) */
  maxVerticalDrift?: number;
  /** Minimum velocity to trigger swipe on shorter distances (default: 0.3px/ms) */
  velocityThreshold?: number;
  /** Direction(s) to detect */
  direction: 'left' | 'right' | 'both';
  /** Which edge to detect swipe from ('left', 'right', or 'none' for anywhere) */
  edge?: 'left' | 'right' | 'none';
  /** Callback when swipe is detected */
  onSwipe: (direction: 'left' | 'right') => void;
  /** Optional callback for swipe progress (0-1) during gesture */
  onProgress?: (progress: number) => void;
  /** Optional function to check if swipe should be enabled */
  enabled?: () => boolean;
}

interface SwipeState {
  active: boolean;
  captured: boolean;
  startX: number;
  startY: number;
  startTime: number;
  pointerId: number | null;
}

/**
 * Check if an element or its ancestors have horizontal scroll
 */
function isInScrollableElement(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) return false;

  let element: Element | null = target;
  while (element) {
    // Check for overflow-x-auto class or actual horizontal scroll capability
    if (
      element.classList.contains('overflow-x-auto') ||
      element.classList.contains('overflow-x-scroll')
    ) {
      return true;
    }

    const style = getComputedStyle(element);
    if (style.overflowX === 'auto' || style.overflowX === 'scroll') {
      // Only consider it scrollable if content actually overflows
      if (element.scrollWidth > element.clientWidth) {
        return true;
      }
    }

    element = element.parentElement;
  }
  return false;
}

export function swipe(node: HTMLElement, options: SwipeOptions) {
  const {
    edgeZoneStart = 15,
    edgeZoneEnd = 50,
    threshold = 50,
    maxVerticalDrift = 75,
    velocityThreshold = 0.3,
    direction,
    edge = 'none',
    onSwipe,
    onProgress,
    enabled = () => true,
  } = options;

  // Minimum horizontal movement before capturing the pointer (preserves taps/clicks)
  const CAPTURE_THRESHOLD = 10;

  const state: SwipeState = {
    active: false,
    captured: false,
    startX: 0,
    startY: 0,
    startTime: 0,
    pointerId: null,
  };

  function isInEdgeZone(clientX: number): boolean {
    if (edge === 'none') return true;

    const _rect = node.getBoundingClientRect();

    if (edge === 'left') {
      // Check if touch is within edge zone from left side of viewport
      return clientX >= edgeZoneStart && clientX <= edgeZoneEnd;
    } else if (edge === 'right') {
      // Check if touch is within edge zone from right side of viewport
      const rightEdge = window.innerWidth - clientX;
      return rightEdge >= edgeZoneStart && rightEdge <= edgeZoneEnd;
    }

    return false;
  }

  function handlePointerDown(e: PointerEvent) {
    // Only handle touch events (not mouse) for swipe gestures
    if (e.pointerType !== 'touch') return;

    // Check if swipe is enabled
    if (!enabled()) return;

    // Ignore if target is in a horizontally scrollable element
    if (isInScrollableElement(e.target)) return;

    // Check edge zone
    if (!isInEdgeZone(e.clientX)) return;

    state.active = true;
    state.captured = false;
    state.startX = e.clientX;
    state.startY = e.clientY;
    state.startTime = Date.now();
    state.pointerId = e.pointerId;

    // Don't capture pointer yet — wait for enough horizontal movement.
    // Capturing immediately prevents click events from firing on child
    // elements on Android Chrome (pointer capture redirects pointerup
    // to the capturing element, breaking the click event chain).
  }

  function handlePointerMove(e: PointerEvent) {
    if (!state.active || e.pointerId !== state.pointerId) return;

    const deltaX = e.clientX - state.startX;
    const deltaY = e.clientY - state.startY;

    // Cancel if vertical drift is too large (user is scrolling)
    if (Math.abs(deltaY) > maxVerticalDrift) {
      cancelSwipe(e);
      return;
    }

    // Lazily capture pointer once horizontal movement exceeds threshold.
    // This preserves normal tap/click behavior for small movements.
    if (!state.captured && Math.abs(deltaX) >= CAPTURE_THRESHOLD) {
      state.captured = true;
      try {
        node.setPointerCapture(e.pointerId);
      } catch {
        // Ignore if capture fails
      }
    }

    // Report progress if callback provided
    if (onProgress) {
      const progress = Math.min(Math.abs(deltaX) / threshold, 1);
      onProgress(progress);
    }
  }

  function handlePointerUp(e: PointerEvent) {
    if (!state.active || e.pointerId !== state.pointerId) return;

    const deltaX = e.clientX - state.startX;
    const deltaY = e.clientY - state.startY;
    const elapsed = Date.now() - state.startTime;
    const velocity = Math.abs(deltaX) / elapsed;

    // Reset progress
    if (onProgress) {
      onProgress(0);
    }

    // Release pointer capture (only if we actually captured)
    if (state.captured && state.pointerId !== null) {
      try {
        node.releasePointerCapture(state.pointerId);
      } catch {
        // Ignore if already released
      }
    }

    // Check if swipe should be triggered
    const verticalOk = Math.abs(deltaY) <= maxVerticalDrift;
    const distanceOk = Math.abs(deltaX) >= threshold;
    const velocityOk = velocity >= velocityThreshold && Math.abs(deltaX) >= threshold * 0.5;

    if (verticalOk && (distanceOk || velocityOk)) {
      const swipeDirection = deltaX > 0 ? 'right' : 'left';

      // Check if direction matches what we're looking for
      if (direction === 'both' || direction === swipeDirection) {
        onSwipe(swipeDirection);
      }
    }

    resetState();
  }

  function handlePointerCancel(e: PointerEvent) {
    if (state.pointerId === e.pointerId) {
      cancelSwipe(e);
    }
  }

  function cancelSwipe(_e: PointerEvent) {
    if (onProgress) {
      onProgress(0);
    }

    if (state.captured && state.pointerId !== null) {
      try {
        node.releasePointerCapture(state.pointerId);
      } catch {
        // Ignore if already released
      }
    }

    resetState();
  }

  function resetState() {
    state.active = false;
    state.captured = false;
    state.startX = 0;
    state.startY = 0;
    state.startTime = 0;
    state.pointerId = null;
  }

  // Add event listeners
  node.addEventListener('pointerdown', handlePointerDown);
  node.addEventListener('pointermove', handlePointerMove);
  node.addEventListener('pointerup', handlePointerUp);
  node.addEventListener('pointercancel', handlePointerCancel);

  return {
    destroy() {
      node.removeEventListener('pointerdown', handlePointerDown);
      node.removeEventListener('pointermove', handlePointerMove);
      node.removeEventListener('pointerup', handlePointerUp);
      node.removeEventListener('pointercancel', handlePointerCancel);
    },
  };
}
