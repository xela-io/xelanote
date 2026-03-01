import { EditorView } from '@codemirror/view';

interface PreviewInteractionOptions {
  featureTaskLists: boolean;
  getLastTaskClickTime: () => number;
  setLastTaskClickTime: (value: number) => void;
  onWikilink: (title: string) => void;
  onToggleTask: (index: number, checked: boolean, lineNumber?: number) => void;
  log?: (...args: unknown[]) => void;
}

export function handlePreviewClick(e: MouseEvent, options: PreviewInteractionOptions) {
  const target = e.target as HTMLElement;

  // Wikilinks
  if (target.classList.contains('wikilink')) {
    e.preventDefault();
    const title = target.dataset.title;
    if (title) {
      options.onWikilink(title);
    }
    return;
  }

  // Task list checkboxes - handle clicks on checkbox or its label
  if (options.featureTaskLists) {
    options.log?.('[TaskSort] Preview click detected, target:', target.tagName, target.className);

    const checkbox = target.matches('input.task-list-item-checkbox')
      ? (target as HTMLInputElement)
      : (target
          .closest('label')
          ?.querySelector('input.task-list-item-checkbox') as HTMLInputElement | null);

    options.log?.('[TaskSort] Checkbox found:', checkbox ? 'yes' : 'no');

    if (checkbox) {
      // Timestamp-based debounce: ignore clicks within 300ms of last checkbox click.
      // Moved after checkbox detection so non-checkbox clicks (e.g. wikilinks)
      // are never blocked by the debounce.
      const now = Date.now();
      if (now - options.getLastTaskClickTime() < 300) {
        options.log?.(
          '[TaskSort] Ignoring click - debounce active (',
          now - options.getLastTaskClickTime(),
          'ms since last)'
        );
        e.preventDefault();
        return;
      }

      // Prevent browser from toggling the checkbox or firing a synthetic click
      // from the <label>. We manage checkbox state entirely through the markdown
      // source → re-render cycle. Without this, clicking the label text (not the
      // checkbox square directly) reads the OLD checked state because the browser
      // hasn't toggled it yet at this point, while the debounce blocks the
      // subsequent synthetic click that would have the correct state.
      e.preventDefault();

      // Read the rendered (markdown-authoritative) checked state from the HTML
      // attribute, not the DOM property. The HTML attribute reflects what
      // markdown-it rendered and is unaffected by browser click toggling.
      // We invert it to get the user's intended new state.
      const isCurrentlyChecked = checkbox.hasAttribute('checked');
      const newChecked = !isCurrentlyChecked;

      const previewContainer = checkbox.closest('.markdown-preview');
      options.log?.('[TaskSort] Preview container found:', previewContainer ? 'yes' : 'no');

      if (previewContainer) {
        const taskItem = checkbox.closest('li.task-list-item');
        const checkboxIndex = taskItem
          ? parseInt(taskItem.getAttribute('data-task-index') || '-1', 10)
          : -1;
        const lineNumber = taskItem
          ? parseInt(taskItem.getAttribute('data-task-line') || '-1', 10)
          : -1;
        options.log?.(
          '[TaskSort] Checkbox index:',
          checkboxIndex,
          'line:',
          lineNumber,
          'newChecked:',
          newChecked
        );

        if (checkboxIndex !== -1) {
          // Update timestamp before processing
          options.setLastTaskClickTime(now);
          options.onToggleTask(
            checkboxIndex,
            newChecked,
            Number.isInteger(lineNumber) && lineNumber > 0 ? lineNumber : undefined
          );
        }
      }
    }
  }
}

function slugifyHeading(text: string, slugCounts: Map<string, number>): string {
  let slug = text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-');
  const count = slugCounts.get(slug) || 0;
  slugCounts.set(slug, count + 1);
  if (count > 0) {
    slug = `${slug}-${count}`;
  }
  return slug;
}

function findHeadingLineBySlug(content: string, slug: string): number | null {
  const lines = content.split('\n');
  const slugCounts = new Map<string, number>();
  let inCodeBlock = false;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].trimStart().startsWith('```')) {
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) continue;
    const match = lines[i].match(/^(#{1,6})\s+(.+)$/);
    if (!match) continue;
    const currentSlug = slugifyHeading(match[2].trim(), slugCounts);
    if (currentSlug === slug) {
      return i + 1;
    }
  }
  return null;
}

export function handleLiveTocClick(
  slug: string,
  content: string,
  liveEditorView: EditorView
): boolean {
  const lineNumber = findHeadingLineBySlug(content, slug);
  if (!lineNumber || lineNumber > liveEditorView.state.doc.lines) {
    return false;
  }

  const line = liveEditorView.state.doc.line(lineNumber);
  liveEditorView.dispatch({
    selection: { anchor: line.from },
    effects: EditorView.scrollIntoView(line.from, { y: 'start', yMargin: 16 }),
  });
  liveEditorView.focus();
  return true;
}

interface TocClickOptions {
  content?: string;
  liveEditorView?: EditorView;
}

/**
 * Find the nearest scrollable ancestor of an element.
 */
function findScrollContainer(el: HTMLElement): HTMLElement | null {
  let current = el.parentElement;
  while (current) {
    const style = getComputedStyle(current);
    const overflow = style.overflowY;
    if (overflow === 'auto' || overflow === 'scroll') {
      return current;
    }
    current = current.parentElement;
  }
  return null;
}

/** Small top margin so the heading isn't flush against the container edge. */
const SCROLL_MARGIN_TOP = 8;

export function handleTocClick(slug: string, options: TocClickOptions = {}) {
  const previewContainer = document.querySelector('.markdown-preview');
  if (previewContainer) {
    const heading = previewContainer.querySelector<HTMLElement>(`#${CSS.escape(slug)}`);
    if (heading) {
      // Use targeted scrollTo on the specific scroll container instead of
      // scrollIntoView, which scrolls ALL scrollable ancestors and can cause
      // the page to jump on mobile or in split-mode layouts.
      const scrollContainer = findScrollContainer(heading);
      if (scrollContainer) {
        const headingRect = heading.getBoundingClientRect();
        const containerRect = scrollContainer.getBoundingClientRect();
        const targetScroll =
          scrollContainer.scrollTop + (headingRect.top - containerRect.top) - SCROLL_MARGIN_TOP;
        scrollContainer.scrollTo({
          top: Math.max(0, targetScroll),
          behavior: 'smooth',
        });
      } else {
        heading.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
      return;
    }
  }

  if (options.content && options.liveEditorView) {
    handleLiveTocClick(slug, options.content, options.liveEditorView);
  }
}
