interface PreviewInteractionOptions {
  featureTaskLists: boolean;
  getLastTaskClickTime: () => number;
  setLastTaskClickTime: (value: number) => void;
  onWikilink: (title: string) => void;
  onToggleTask: (index: number, checked: boolean) => void;
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
    // Timestamp-based debounce: ignore clicks within 300ms of last click
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

    options.log?.('[TaskSort] Preview click detected, target:', target.tagName, target.className);

    const checkbox = target.matches('input.task-list-item-checkbox')
      ? (target as HTMLInputElement)
      : (target
          .closest('label')
          ?.querySelector('input.task-list-item-checkbox') as HTMLInputElement | null);

    options.log?.('[TaskSort] Checkbox found:', checkbox ? 'yes' : 'no');

    if (checkbox) {
      const previewContainer = checkbox.closest('.markdown-preview');
      options.log?.('[TaskSort] Preview container found:', previewContainer ? 'yes' : 'no');

      if (previewContainer) {
        const taskItem = checkbox.closest('li.task-list-item');
        const checkboxIndex = taskItem
          ? parseInt(taskItem.getAttribute('data-task-index') || '-1', 10)
          : -1;
        options.log?.('[TaskSort] Checkbox index:', checkboxIndex, 'checked:', checkbox.checked);

        if (checkboxIndex !== -1) {
          // Update timestamp before processing
          options.setLastTaskClickTime(now);
          options.onToggleTask(checkboxIndex, checkbox.checked);
        }
      }
    }
  }
}

export function handleTocClick(slug: string) {
  const previewContainer = document.querySelector('.markdown-preview');
  if (previewContainer) {
    const heading = previewContainer.querySelector(`#${CSS.escape(slug)}`);
    if (heading) {
      heading.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }
}
