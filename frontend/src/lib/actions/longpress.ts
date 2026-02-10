/**
 * Long-press gesture action for Svelte (touch-only)
 *
 * Triggers after 500ms stationary touch. Suppresses the browser's
 * contextmenu event that fires after a long press so that only the
 * custom handler runs.
 */

export interface LongPressOptions {
  /** Callback when long press is detected */
  onLongPress: (detail: { clientX: number; clientY: number }) => void;
  /** Hold duration in ms (default: 500) */
  duration?: number;
  /** Max movement in px before cancelling (default: 10) */
  moveThreshold?: number;
}

export function longpress(node: HTMLElement, options: LongPressOptions) {
  const { onLongPress, duration = 500, moveThreshold = 10 } = options;

  let timer: ReturnType<typeof setTimeout> | null = null;
  let startX = 0;
  let startY = 0;
  let _longPressTriggered = false;

  function handlePointerDown(e: PointerEvent) {
    if (e.pointerType !== 'touch') return;

    startX = e.clientX;
    startY = e.clientY;
    _longPressTriggered = false;

    timer = setTimeout(() => {
      _longPressTriggered = true;

      // Suppress the browser contextmenu that fires after long-press on touch
      document.addEventListener(
        'contextmenu',
        (evt) => {
          evt.preventDefault();
          evt.stopImmediatePropagation();
        },
        { capture: true, once: true }
      );

      onLongPress({ clientX: startX, clientY: startY });

      // Reset flag after current event loop cycle
      setTimeout(() => {
        _longPressTriggered = false;
      }, 0);
    }, duration);
  }

  function handlePointerMove(e: PointerEvent) {
    if (!timer) return;
    const dx = e.clientX - startX;
    const dy = e.clientY - startY;
    if (Math.sqrt(dx * dx + dy * dy) > moveThreshold) {
      cancel();
    }
  }

  function handlePointerUp() {
    cancel();
  }

  function handlePointerCancel() {
    cancel();
  }

  function cancel() {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
  }

  node.addEventListener('pointerdown', handlePointerDown);
  node.addEventListener('pointermove', handlePointerMove);
  node.addEventListener('pointerup', handlePointerUp);
  node.addEventListener('pointercancel', handlePointerCancel);

  return {
    destroy() {
      cancel();
      node.removeEventListener('pointerdown', handlePointerDown);
      node.removeEventListener('pointermove', handlePointerMove);
      node.removeEventListener('pointerup', handlePointerUp);
      node.removeEventListener('pointercancel', handlePointerCancel);
    },
  };
}
