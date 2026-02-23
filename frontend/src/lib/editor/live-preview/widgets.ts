// Live preview widget classes for inline text, hidden syntax, tasks, tables, headings, task groups

import { RangeSetBuilder } from '@codemirror/state';
import { Decoration, WidgetType } from '@codemirror/view';

import type { TableBlock } from './table-parser';

const lineDecorationCache = new Map<string, Decoration>();
const LIVE_TASK_DRAG_HANDLE_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="12" r="1"/><circle cx="9" cy="5" r="1"/><circle cx="9" cy="19" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="5" r="1"/><circle cx="15" cy="19" r="1"/></svg>';

export class InlineTextWidget extends WidgetType {
  constructor(
    private readonly text: string,
    private readonly className: string,
    private readonly dataAttrs?: Record<string, string>
  ) {
    super();
  }

  eq(other: InlineTextWidget): boolean {
    return (
      this.text === other.text &&
      this.className === other.className &&
      JSON.stringify(this.dataAttrs ?? {}) === JSON.stringify(other.dataAttrs ?? {})
    );
  }

  toDOM(): HTMLElement {
    const span = document.createElement('span');
    span.className = this.className;
    span.textContent = this.text;
    if (this.dataAttrs) {
      for (const [key, value] of Object.entries(this.dataAttrs)) {
        span.dataset[key] = value;
      }
    }
    return span;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

export class HiddenSyntaxWidget extends WidgetType {
  eq(): boolean {
    return true;
  }

  toDOM(): HTMLElement {
    const span = document.createElement('span');
    span.className = 'cm-live-hidden-syntax';
    span.setAttribute('aria-hidden', 'true');
    return span;
  }
}

export const hiddenSyntaxDecoration = Decoration.replace({
  widget: new HiddenSyntaxWidget(),
});

export function addProtectedWidget(
  builder: RangeSetBuilder<Decoration>,
  protectedRanges: Array<{ from: number; to: number }>,
  fromPos: number,
  toPos: number,
  widget: WidgetType
) {
  builder.add(
    fromPos,
    toPos,
    Decoration.replace({
      widget,
    })
  );
  protectedRanges.push({ from: fromPos, to: toPos });
}

export function getLineDecoration(className: string): Decoration {
  const cached = lineDecorationCache.get(className);
  if (cached) return cached;

  const decoration = Decoration.line({ attributes: { class: className } });
  lineDecorationCache.set(className, decoration);
  return decoration;
}

export class TaskCheckboxWidget extends WidgetType {
  constructor(
    private readonly checked: boolean,
    private readonly lineNumber: number
  ) {
    super();
  }

  eq(other: TaskCheckboxWidget): boolean {
    return this.checked === other.checked && this.lineNumber === other.lineNumber;
  }

  toDOM(): HTMLElement {
    const wrapper = document.createElement('span');
    wrapper.className = `cm-live-task-checkbox ${this.checked ? 'is-checked' : ''}`;
    wrapper.dataset.checked = String(this.checked);
    wrapper.dataset.line = String(this.lineNumber);
    wrapper.setAttribute('role', 'checkbox');
    wrapper.setAttribute('aria-checked', String(this.checked));
    wrapper.tabIndex = -1;

    const dragHandle = document.createElement('span');
    dragHandle.className = 'cm-live-task-drag-handle';
    dragHandle.dataset.line = String(this.lineNumber);
    dragHandle.setAttribute('aria-hidden', 'true');
    // F2-08: Use DOM method instead of innerHTML. SVG is a hardcoded constant.
    const svgContainer = document.createElement('span');
    svgContainer.innerHTML = LIVE_TASK_DRAG_HANDLE_SVG; // hardcoded constant, safe
    if (svgContainer.firstChild) dragHandle.appendChild(svgContainer.firstChild);

    const input = document.createElement('input');
    input.type = 'checkbox';
    input.className = 'cm-live-task-checkbox-input';
    input.checked = this.checked;
    input.tabIndex = -1;
    input.setAttribute('aria-hidden', 'true');
    wrapper.appendChild(dragHandle);
    wrapper.appendChild(input);
    return wrapper;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

export class TableWidget extends WidgetType {
  constructor(private readonly block: TableBlock) {
    super();
  }

  eq(other: TableWidget): boolean {
    if (
      this.block.startLine !== other.block.startLine ||
      this.block.endLine !== other.block.endLine ||
      this.block.headerCells.length !== other.block.headerCells.length ||
      this.block.rows.length !== other.block.rows.length
    ) {
      return false;
    }
    for (let i = 0; i < this.block.headerCells.length; i++) {
      if (this.block.headerCells[i] !== other.block.headerCells[i]) return false;
    }
    for (let i = 0; i < this.block.alignments.length; i++) {
      if (this.block.alignments[i] !== other.block.alignments[i]) return false;
    }
    for (let i = 0; i < this.block.rows.length; i++) {
      const thisRow = this.block.rows[i];
      const otherRow = other.block.rows[i];
      if (thisRow.length !== otherRow.length) return false;
      for (let j = 0; j < thisRow.length; j++) {
        if (thisRow[j] !== otherRow[j]) return false;
      }
    }
    return true;
  }

  toDOM(): HTMLElement {
    const wrapper = document.createElement('div');
    wrapper.className = 'cm-table-widget';
    wrapper.dataset.startLine = String(this.block.startLine);

    const table = document.createElement('table');
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');

    for (let i = 0; i < this.block.headerCells.length; i++) {
      const th = document.createElement('th');
      th.textContent = this.block.headerCells[i];
      const align = this.block.alignments[i];
      if (align) th.style.textAlign = align;
      headerRow.appendChild(th);
    }
    thead.appendChild(headerRow);
    table.appendChild(thead);

    if (this.block.rows.length > 0) {
      const tbody = document.createElement('tbody');
      for (const row of this.block.rows) {
        const tr = document.createElement('tr');
        for (let i = 0; i < this.block.headerCells.length; i++) {
          const td = document.createElement('td');
          td.textContent = row[i] ?? '';
          const align = this.block.alignments[i];
          if (align) td.style.textAlign = align;
          tr.appendChild(td);
        }
        tbody.appendChild(tr);
      }
      table.appendChild(tbody);
    }

    wrapper.appendChild(table);
    return wrapper;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

export class HeadingToggleWidget extends WidgetType {
  constructor(
    private readonly sectionKey: string,
    private readonly collapsed: boolean
  ) {
    super();
  }

  eq(other: HeadingToggleWidget): boolean {
    return this.sectionKey === other.sectionKey && this.collapsed === other.collapsed;
  }

  toDOM(): HTMLElement {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'cm-live-heading-toggle';
    button.dataset.section = this.sectionKey;
    button.dataset.collapsed = String(this.collapsed);
    button.setAttribute(
      'aria-label',
      this.collapsed ? 'Expand heading section' : 'Collapse heading section'
    );
    button.textContent = this.collapsed ? '▷' : '-';
    return button;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

export class CompletedTaskGroupToggleWidget extends WidgetType {
  constructor(
    private readonly groupKey: string,
    private readonly collapsed: boolean,
    private readonly count: number,
    private readonly startLine: number
  ) {
    super();
  }

  eq(other: CompletedTaskGroupToggleWidget): boolean {
    return (
      this.groupKey === other.groupKey &&
      this.collapsed === other.collapsed &&
      this.count === other.count &&
      this.startLine === other.startLine
    );
  }

  toDOM(): HTMLElement {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'cm-live-task-group-toggle';
    button.dataset.taskGroup = this.groupKey;
    button.dataset.line = String(this.startLine);
    button.setAttribute(
      'aria-label',
      this.collapsed ? 'Erledigte Aufgaben aufklappen' : 'Erledigte Aufgaben einklappen'
    );
    button.textContent = this.collapsed ? '▷' : '-';
    if (!this.collapsed && this.count > 1) {
      // Vertically center across the group: each line = 1.7rem (line-height of preview lines)
      // Use rem (not em) because this button has font-size: 0.8em
      button.style.top = `${this.count * 0.85}rem`;
    }
    return button;
  }

  ignoreEvent(): boolean {
    return false;
  }
}

export class CompletedTaskGroupSummaryWidget extends WidgetType {
  constructor(
    private readonly groupKey: string,
    private readonly count: number,
    private readonly startLine: number
  ) {
    super();
  }

  eq(other: CompletedTaskGroupSummaryWidget): boolean {
    return (
      this.groupKey === other.groupKey &&
      this.count === other.count &&
      this.startLine === other.startLine
    );
  }

  toDOM(): HTMLElement {
    const span = document.createElement('span');
    span.className = 'cm-live-task-group-summary';
    span.dataset.taskGroup = this.groupKey;
    span.dataset.line = String(this.startLine);
    span.textContent = `${this.count} erledigte Aufgaben`;
    return span;
  }

  ignoreEvent(): boolean {
    return false;
  }
}
