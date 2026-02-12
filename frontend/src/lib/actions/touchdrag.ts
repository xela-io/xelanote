/**
 * Touch Drag & Drop action for Svelte
 *
 * Uses event delegation via data-* attributes on tree items.
 * State machine: IDLE -> PENDING (pointerdown) -> DRAGGING (hold timer) -> IDLE (drop/cancel)
 *
 * Attach to a scroll container; tree items need:
 *   data-drag-type="folder|note"
 *   data-drag-id="123"
 *   data-drag-path="/path" (folders)
 *   data-drag-folder-path="/path" (notes)
 *   data-drag-title="Note Title" (notes)
 *
 * Elements with [data-no-drag] are excluded from drag initiation.
 */

export interface TouchDragData {
  type: 'folder' | 'note' | 'root-dropzone';
  id: string;
  path?: string;
  folder_path?: string;
  title?: string;
}

export type DropPosition = 'before' | 'after' | 'into';

export interface TouchDragOptions {
  /** Hold duration in ms before drag starts (default: 300) */
  holdDuration?: number;
  /** Whether touch drag is enabled */
  enabled?: () => boolean;
  /** Called when a valid drop occurs */
  onDrop: (dragData: TouchDragData, targetData: TouchDragData, position: DropPosition) => void;
  /** Called when drag starts */
  onDragStart?: () => void;
  /** Called when drag ends (drop or cancel) */
  onDragEnd?: () => void;
}

type State = 'IDLE' | 'PENDING' | 'DRAGGING';

const MOVE_THRESHOLD = 10; // px - cancel PENDING if moved further

export function touchdrag(container: HTMLElement, options: TouchDragOptions) {
  const { holdDuration = 300, enabled = () => true, onDrop, onDragStart, onDragEnd } = options;

  let state: State = 'IDLE';
  let holdTimer: ReturnType<typeof setTimeout> | null = null;
  let startX = 0;
  let startY = 0;
  let activePointerId: number | null = null;

  // Drag state
  let dragData: TouchDragData | null = null;
  let sourceElement: HTMLElement | null = null;
  let ghostElement: HTMLElement | null = null;
  let currentTarget: HTMLElement | null = null;
  let currentPosition: DropPosition | null = null;
  let scrollRAF: number | null = null;

  // --- Helpers ---

  function findDragItem(target: EventTarget | null): HTMLElement | null {
    if (!(target instanceof Element)) return null;
    // Skip elements inside no-drag zones (kebab button, expand button)
    if (target.closest('[data-no-drag]')) return null;
    // Find nearest drag-enabled ancestor within the container
    const item = target.closest('[data-drag-type]');
    if (!item || !container.contains(item)) return null;
    return item as HTMLElement;
  }

  function extractDragData(el: HTMLElement): TouchDragData | null {
    const type = el.dataset.dragType as 'folder' | 'note' | 'root-dropzone' | undefined;
    const id = el.dataset.dragId;
    if (!type || !id) return null;
    return {
      type,
      id,
      path: el.dataset.dragPath,
      folder_path: el.dataset.dragFolderPath,
      title: el.dataset.dragTitle,
    };
  }

  function createGhost(el: HTMLElement): HTMLElement {
    const ghost = document.createElement('div');
    const maxWidth = Math.min(200, container.offsetWidth * 0.65);

    // Get text content from the tree item
    const nameEl = el.querySelector('.node-name');
    const text = nameEl?.textContent || el.textContent?.trim() || '';

    ghost.textContent = text;
    Object.assign(ghost.style, {
      position: 'fixed',
      top: '0',
      left: '0',
      width: `${maxWidth}px`,
      padding: '8px 12px',
      background: 'var(--color-sidebar-accent, #e5e7eb)',
      color: 'var(--color-sidebar-foreground, #1f2937)',
      borderRadius: '6px',
      fontSize: '13px',
      fontFamily: 'inherit',
      boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
      pointerEvents: 'none',
      zIndex: '9999',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
      opacity: '0.92',
      transform: 'translate3d(-9999px, -9999px, 0)',
      willChange: 'transform',
    });

    document.body.appendChild(ghost);
    return ghost;
  }

  function moveGhost(x: number, y: number) {
    if (!ghostElement) return;
    ghostElement.style.transform = `translate3d(${x - 20}px, ${y - 20}px, 0)`;
  }

  function removeGhost() {
    if (ghostElement) {
      ghostElement.remove();
      ghostElement = null;
    }
  }

  function clearIndicators() {
    // Remove all touch-drop indicator classes from any elements
    const indicators = container.querySelectorAll(
      '.touch-drop-before, .touch-drop-after, .touch-drop-into'
    );
    for (const el of indicators) {
      el.classList.remove('touch-drop-before', 'touch-drop-after', 'touch-drop-into');
    }
    currentTarget = null;
    currentPosition = null;
  }

  function setIndicator(target: HTMLElement, position: DropPosition) {
    // Clear previous indicators first
    clearIndicators();

    // Find the .tree-item ancestor for the indicator class
    let treeItem: HTMLElement | null = target;
    while (treeItem && !treeItem.classList.contains('tree-item')) {
      treeItem = treeItem.parentElement;
    }
    if (!treeItem) treeItem = target;

    treeItem.classList.add(`touch-drop-${position}`);
    currentTarget = target;
    currentPosition = position;
  }

  function computeDropPosition(target: HTMLElement, clientY: number): DropPosition {
    const type = target.dataset.dragType;
    // Find .tree-item for bounding rect
    let treeItem: HTMLElement | null = target;
    while (treeItem && !treeItem.classList.contains('tree-item')) {
      treeItem = treeItem.parentElement;
    }
    const rect = (treeItem || target).getBoundingClientRect();
    const relY = clientY - rect.top;
    const height = rect.height;

    if (type === 'folder') {
      if (relY < height * 0.25) return 'before';
      if (relY > height * 0.75) return 'after';
      return 'into';
    } else if (type === 'root-dropzone') {
      return 'into';
    } else {
      // note: top 50% = before, bottom 50% = after
      return relY < height * 0.5 ? 'before' : 'after';
    }
  }

  // --- Auto-scroll ---

  function startAutoScroll(clientY: number) {
    stopAutoScroll();
    const rect = container.getBoundingClientRect();
    const edgeSize = 50;
    const topDist = clientY - rect.top;
    const bottomDist = rect.bottom - clientY;

    let speed = 0;
    if (topDist < edgeSize && container.scrollTop > 0) {
      speed = -Math.max(2, (edgeSize - topDist) * 0.15);
    } else if (
      bottomDist < edgeSize &&
      container.scrollTop < container.scrollHeight - container.clientHeight
    ) {
      speed = Math.max(2, (edgeSize - bottomDist) * 0.15);
    }

    if (speed === 0) return;

    function scroll() {
      container.scrollTop += speed;
      scrollRAF = requestAnimationFrame(scroll);
    }
    scrollRAF = requestAnimationFrame(scroll);
  }

  function stopAutoScroll() {
    if (scrollRAF !== null) {
      cancelAnimationFrame(scrollRAF);
      scrollRAF = null;
    }
  }

  // --- State transitions ---

  function handlePointerDown(e: PointerEvent) {
    if (state !== 'IDLE') return;
    if (e.pointerType !== 'touch') return;
    if (!enabled()) return;

    const item = findDragItem(e.target);
    if (!item) return;

    const data = extractDragData(item);
    if (!data) return;

    state = 'PENDING';
    startX = e.clientX;
    startY = e.clientY;
    activePointerId = e.pointerId;
    dragData = data;
    sourceElement = item;

    // Start hold timer
    holdTimer = setTimeout(() => {
      if (state !== 'PENDING') return;
      enterDragging(e.clientX, e.clientY);
    }, holdDuration);

    // Don't block scroll during PENDING. If the user scrolls, the browser
    // fires pointercancel which cancels the drag via handleCancel.
    // This preserves smooth native scrolling on touch devices.
    // Drag only starts if the user holds perfectly still for holdDuration.

    // Listen on window for move/up/cancel during PENDING
    window.addEventListener('pointermove', handlePointerMovePending, { passive: true });
    window.addEventListener('pointerup', handlePointerUpPending, { passive: true });
    window.addEventListener('pointercancel', handleCancel, { passive: true });
  }

  function handlePointerMovePending(e: PointerEvent) {
    if (e.pointerId !== activePointerId) return;
    const dx = e.clientX - startX;
    const dy = e.clientY - startY;
    if (Math.sqrt(dx * dx + dy * dy) > MOVE_THRESHOLD) {
      cancelDrag();
    }
  }

  function handlePointerUpPending() {
    // Finger lifted before hold timer fired - cancel
    cancelDrag();
  }

  function enterDragging(x: number, y: number) {
    if (!sourceElement || !dragData) return;

    state = 'DRAGGING';

    // Clear any text selection caused by long-press
    window.getSelection()?.removeAllRanges();

    // Haptic feedback (no-op on iOS)
    navigator.vibrate?.(15);

    // Suppress browser contextmenu from the long-press
    document.addEventListener('contextmenu', suppressContextMenu, { capture: true, once: true });

    // Create ghost
    ghostElement = createGhost(sourceElement);
    moveGhost(x, y);

    // Mark source
    let treeItem: HTMLElement | null = sourceElement;
    while (treeItem && !treeItem.classList.contains('tree-item')) {
      treeItem = treeItem.parentElement;
    }
    if (treeItem) treeItem.classList.add('touch-dragging-source');

    // Block scrolling via non-passive touchmove (touch-action:none doesn't work mid-gesture)
    document.addEventListener('touchmove', preventTouchScroll, { passive: false });

    // Body class for global styling
    document.body.classList.add('touch-drag-active');

    // Remove PENDING listeners, add DRAGGING listeners on window
    removePendingListeners();
    window.addEventListener('pointermove', handlePointerMoveDragging);
    window.addEventListener('pointerup', handleDrop);
    window.addEventListener('pointercancel', handleCancel);

    onDragStart?.();
  }

  function handlePointerMoveDragging(e: PointerEvent) {
    if (e.pointerId !== activePointerId) return;
    e.preventDefault(); // Prevent scroll only during DRAGGING

    moveGhost(e.clientX, e.clientY);

    // Find target under finger (ghost has pointer-events: none)
    const hitEl = document.elementFromPoint(e.clientX, e.clientY);
    if (!hitEl || !(hitEl instanceof Element)) {
      clearIndicators();
      stopAutoScroll();
      return;
    }

    // Walk up to nearest [data-drag-type] using closest() (works for SVG + HTML)
    const target = hitEl.closest('[data-drag-type]') as HTMLElement | null;

    if (target && container.contains(target)) {
      const targetData = extractDragData(target);
      // Don't show indicator on self
      if (targetData && dragData && targetData.id !== dragData.id) {
        const position = computeDropPosition(target, e.clientY);
        setIndicator(target, position);
      } else {
        clearIndicators();
      }
    } else {
      clearIndicators();
    }

    // Auto-scroll near edges
    startAutoScroll(e.clientY);
  }

  function handleDrop(e: PointerEvent) {
    if (e.pointerId !== activePointerId) return;

    try {
      if (currentTarget && currentPosition && dragData) {
        const targetData = extractDragData(currentTarget);
        if (targetData && targetData.id !== dragData.id) {
          onDrop(dragData, targetData, currentPosition);
        }
      }
    } finally {
      cleanup();
    }
  }

  function handleCancel() {
    cancelDrag();
  }

  function preventTouchScroll(e: TouchEvent) {
    if (state === 'DRAGGING') {
      e.preventDefault();
    }
  }

  function suppressContextMenu(e: Event) {
    e.preventDefault();
    e.stopImmediatePropagation();
  }

  // --- Cleanup ---

  function removePendingListeners() {
    window.removeEventListener('pointermove', handlePointerMovePending);
    window.removeEventListener('pointerup', handlePointerUpPending);
    window.removeEventListener('pointercancel', handleCancel);
  }

  function removeDraggingListeners() {
    window.removeEventListener('pointermove', handlePointerMoveDragging);
    window.removeEventListener('pointerup', handleDrop);
    window.removeEventListener('pointercancel', handleCancel);
  }

  function cancelDrag() {
    if (state === 'PENDING') {
      removePendingListeners();
    } else if (state === 'DRAGGING') {
      cleanup();
      return;
    }

    if (holdTimer) {
      clearTimeout(holdTimer);
      holdTimer = null;
    }

    state = 'IDLE';
    dragData = null;
    sourceElement = null;
    activePointerId = null;
  }

  function cleanup() {
    try {
      // Clear hold timer
      if (holdTimer) {
        clearTimeout(holdTimer);
        holdTimer = null;
      }

      // Remove ghost
      removeGhost();

      // Clear indicators
      clearIndicators();

      // Remove source class
      const sources = container.querySelectorAll('.touch-dragging-source');
      for (const el of sources) {
        el.classList.remove('touch-dragging-source');
      }

      // Remove scroll blocker
      document.removeEventListener('touchmove', preventTouchScroll);

      // Remove body class
      document.body.classList.remove('touch-drag-active');

      // Stop auto-scroll
      stopAutoScroll();

      // Remove listeners
      removePendingListeners();
      removeDraggingListeners();

      // Remove contextmenu suppressor if still attached
      document.removeEventListener('contextmenu', suppressContextMenu, {
        capture: true,
      } as EventListenerOptions);
    } finally {
      state = 'IDLE';
      dragData = null;
      sourceElement = null;
      activePointerId = null;

      onDragEnd?.();
    }
  }

  // --- Attach ---

  container.addEventListener('pointerdown', handlePointerDown, { passive: true });

  return {
    destroy() {
      cleanup();
      container.removeEventListener('pointerdown', handlePointerDown);
    },
  };
}
