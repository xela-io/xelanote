// Task list post-processing: drag handles and line number extraction

// SVG icon for drag handle (GripVertical from lucide)
const DRAG_HANDLE_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="12" r="1"/><circle cx="9" cy="5" r="1"/><circle cx="9" cy="19" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="5" r="1"/><circle cx="15" cy="19" r="1"/></svg>`;

/**
 * Extract 1-based line numbers of task list items from raw markdown content.
 */
export function getRenderedTaskLineNumbers(content: string): number[] {
  const lines = content.split('\n');
  const taskLines: number[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const match = /^(\s*(?:[-*+]|\d+[.)]) )\[([xX ])\]/.exec(line);
    if (!match) continue;

    // Match markdown-it-task-lists behavior: no checkbox for empty tasks.
    const taskBody = line.substring(match[0].length).trim();
    if (!taskBody) continue;
    taskLines.push(i + 1); // 1-based line number
  }

  return taskLines;
}

/**
 * Add drag handles and data-task-index attributes to task list items.
 * This post-processes the HTML after markdown-it rendering.
 */
export function addDragHandlesToTasks(html: string, taskLines: number[]): string {
  let taskIndex = 0;

  return html.replace(/<li class="task-list-item([^"]*)">/g, (match, existingClasses) => {
    const index = taskIndex++;
    const handle = `<span class="drag-handle" aria-hidden="true">${DRAG_HANDLE_SVG}</span>`;
    const line = taskLines[index];
    const lineAttr = Number.isInteger(line) ? ` data-task-line="${line}"` : '';
    return `<li class="task-list-item${existingClasses}" data-task-index="${index}"${lineAttr}>${handle}`;
  });
}
