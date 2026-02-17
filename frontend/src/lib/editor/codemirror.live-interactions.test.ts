import { afterEach, describe, expect, it, vi } from 'vitest';

import { createEditor, setLivePreviewMode } from './codemirror';
import { setLivePreviewProfilerSink } from './live-preview';

describe('codemirror live interactions', () => {
  if (!Range.prototype.getClientRects) {
    Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
  }
  if (!Range.prototype.getBoundingClientRect) {
    Range.prototype.getBoundingClientRect = () => new DOMRect();
  }

  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
    setLivePreviewProfilerSink(null);
  });

  it('toggles task by source line when clicking live task checkbox', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onToggleTaskByLine = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Task',
      onToggleTaskByLine,
    });

    setLivePreviewMode(view, true);

    const checkbox = parent.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();

    checkbox?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onToggleTaskByLine).toHaveBeenCalledTimes(1);
    expect(onToggleTaskByLine).toHaveBeenCalledWith(2, true);

    view.destroy();
  });

  it('prevents editor line selection on mousedown before checkbox click', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onToggleTaskByLine = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Task',
      onToggleTaskByLine,
    });

    setLivePreviewMode(view, true);

    const checkbox = parent.querySelector('.cm-live-task-checkbox') as HTMLElement | null;
    expect(checkbox).not.toBeNull();

    checkbox?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    checkbox?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onToggleTaskByLine).toHaveBeenCalledTimes(1);
    expect(onToggleTaskByLine).toHaveBeenCalledWith(2, true);

    view.destroy();
  });

  it('triggers wikilink callback when clicking live wikilink widget', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const onWikilinkClick = vi.fn();

    const view = createEditor(parent, {
      doc: 'Active\n[[Target Note|Alias]]',
      onWikilinkClick,
    });

    setLivePreviewMode(view, true);

    const wikilink = parent.querySelector('.cm-live-preview-wikilink') as HTMLElement | null;
    expect(wikilink).not.toBeNull();

    wikilink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(onWikilinkClick).toHaveBeenCalledTimes(1);
    expect(onWikilinkClick).toHaveBeenCalledWith('Target Note');

    view.destroy();
  });

  it('opens markdown links from live link widget in new tab', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    const view = createEditor(parent, {
      doc: 'Active\n[Docs](https://example.com/docs)',
    });

    setLivePreviewMode(view, true);

    const link = parent.querySelector('.cm-live-preview-link') as HTMLElement | null;
    expect(link).not.toBeNull();

    link?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    expect(openSpy).toHaveBeenCalledTimes(1);
    expect(openSpy).toHaveBeenCalledWith(
      'https://example.com/docs',
      '_blank',
      'noopener,noreferrer'
    );

    view.destroy();
  });

  it('does not render a line-number gutter', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Open task\n- [x] Done 1\n- [x] Done 2',
    });

    setLivePreviewMode(view, true);

    expect(parent.querySelector('.cm-lineNumbers')).toBeNull();

    view.destroy();
  });

  it('collapses heading section when clicking live heading toggle', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: '# Heading\nline a\nline b\n## Sub\nsub line',
    });
    setLivePreviewMode(view, true);

    const toggle = parent.querySelector('.cm-live-heading-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    const hiddenLines = parent.querySelectorAll('.cm-live-collapsed-line');
    expect(hiddenLines.length).toBeGreaterThan(0);

    view.destroy();
  });
});
