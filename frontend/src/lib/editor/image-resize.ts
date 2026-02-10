/**
 * Svelte action for interactive image resizing in markdown preview.
 * Allows users to resize images by dragging the resize handle.
 */

export interface ImageResizeOptions {
  onResize: (imageIndex: number, newWidth: number) => void;
}

/**
 * Svelte action that enables drag-to-resize functionality for images.
 * Attaches to the preview container and handles resize events.
 */
export function imageResize(container: HTMLElement, options: ImageResizeOptions) {
  let activeImg: HTMLImageElement | null = null;
  let activeWrapper: HTMLElement | null = null;
  let startX = 0;
  let startWidth = 0;
  let isResizing = false;

  function handleMouseDown(e: MouseEvent) {
    const handle = (e.target as HTMLElement).closest('.resize-handle');
    if (!handle) return;

    e.preventDefault();
    e.stopPropagation();

    const wrapper = handle.closest('.resizable-image-wrapper') as HTMLElement;
    activeImg = wrapper?.querySelector('img') || null;
    activeWrapper = wrapper;

    if (!activeImg) return;

    isResizing = true;
    wrapper.classList.add('resizing');
    startX = e.clientX;
    startWidth = activeImg.offsetWidth;

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }

  function handleMouseMove(e: MouseEvent) {
    if (!activeImg || !isResizing) return;

    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);
    activeImg.style.width = `${newWidth}px`;
  }

  function handleMouseUp() {
    if (!activeImg || !isResizing) return;

    activeWrapper?.classList.remove('resizing');

    const imageIndex = parseInt(activeImg.dataset.imageIndex || '0', 10);
    const finalWidth = Math.round(activeImg.offsetWidth);

    // Callback to update markdown source
    options.onResize(imageIndex, finalWidth);

    // Cleanup
    isResizing = false;
    activeImg = null;
    activeWrapper = null;
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  }

  // Touch support
  function handleTouchStart(e: TouchEvent) {
    const handle = (e.target as HTMLElement).closest('.resize-handle');
    if (!handle) return;

    e.preventDefault();

    const wrapper = handle.closest('.resizable-image-wrapper') as HTMLElement;
    activeImg = wrapper?.querySelector('img') || null;
    activeWrapper = wrapper;

    if (!activeImg) return;

    isResizing = true;
    wrapper.classList.add('resizing');
    startX = e.touches[0].clientX;
    startWidth = activeImg.offsetWidth;

    document.addEventListener('touchmove', handleTouchMove, { passive: false });
    document.addEventListener('touchend', handleTouchEnd);
  }

  function handleTouchMove(e: TouchEvent) {
    if (!activeImg || !isResizing) return;

    e.preventDefault();
    const delta = e.touches[0].clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);
    activeImg.style.width = `${newWidth}px`;
  }

  function handleTouchEnd() {
    if (!activeImg || !isResizing) return;

    activeWrapper?.classList.remove('resizing');

    const imageIndex = parseInt(activeImg.dataset.imageIndex || '0', 10);
    const finalWidth = Math.round(activeImg.offsetWidth);

    options.onResize(imageIndex, finalWidth);

    isResizing = false;
    activeImg = null;
    activeWrapper = null;
    document.removeEventListener('touchmove', handleTouchMove);
    document.removeEventListener('touchend', handleTouchEnd);
  }

  // Attach event listeners
  container.addEventListener('mousedown', handleMouseDown);
  container.addEventListener('touchstart', handleTouchStart, { passive: false });

  return {
    update(newOptions: ImageResizeOptions) {
      options = newOptions;
    },
    destroy() {
      container.removeEventListener('mousedown', handleMouseDown);
      container.removeEventListener('touchstart', handleTouchStart);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.removeEventListener('touchmove', handleTouchMove);
      document.removeEventListener('touchend', handleTouchEnd);
    },
  };
}

/**
 * Updates the width attribute of an image in markdown content by its index.
 *
 * @param content - The markdown content
 * @param targetIndex - 1-based index of the image to update
 * @param newWidth - New width value in pixels
 * @returns Updated markdown content with {width=...} syntax
 */
export function updateImageWidthByIndex(
  content: string,
  targetIndex: number,
  newWidth: number
): string {
  // Pattern matches: ![alt](url) optionally followed by {width=...}
  const imagePattern = /!\[([^\]]*)\]\(([^)]+)\)(\{width=\d+%?\})?/g;
  let currentIndex = 0;

  return content.replace(imagePattern, (match, alt, url, _existingWidth) => {
    currentIndex++;
    if (currentIndex !== targetIndex) {
      return match;
    }

    // Build the new image markdown with width attribute
    return `![${alt}](${url}){width=${newWidth}}`;
  });
}
