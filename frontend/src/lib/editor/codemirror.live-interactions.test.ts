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
    expect(openSpy).toHaveBeenCalledWith('https://example.com/docs', '_blank', 'noopener,noreferrer');

    view.destroy();
  });

  it('expands completed group when clicking live completed toggle', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const view = createEditor(parent, {
      doc: 'Active\n- [ ] Open task\n- [x] Done 1\n- [x] Done 2',
    });

    setLivePreviewMode(view, true);

    const before = parent.querySelectorAll('.cm-live-task-checkbox');
    expect(before.length).toBe(1);

    const toggle = parent.querySelector('.cm-live-completed-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    const after = parent.querySelectorAll('.cm-live-task-checkbox');
    expect(after.length).toBe(3);

    const toggleAgain = parent.querySelector('.cm-live-completed-toggle') as HTMLElement | null;
    expect(toggleAgain).not.toBeNull();
    toggleAgain?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggleAgain?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    const afterCollapse = parent.querySelectorAll('.cm-live-task-checkbox');
    expect(afterCollapse.length).toBe(1);

    view.destroy();
  });

  it('keeps completed-toggle interaction latency within budget', () => {
    const parent = document.createElement('div');
    document.body.appendChild(parent);

    const lines = ['Active', '- [ ] Open task'];
    for (let i = 0; i < 220; i++) {
      lines.push(`- [x] Done ${i}`);
    }

    const samples: number[] = [];
    setLivePreviewProfilerSink((sample) => {
      if (sample.phase === 'build' && sample.reason === 'forceRebuild') {
        samples.push(sample.ms);
      }
    });

    const view = createEditor(parent, {
      doc: lines.join('\n'),
    });
    setLivePreviewMode(view, true);

    const toggle = parent.querySelector('.cm-live-completed-toggle') as HTMLElement | null;
    expect(toggle).not.toBeNull();
    toggle?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
    toggle?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

    expect(samples.length).toBeGreaterThanOrEqual(1);
    const avg = samples.reduce((sum, value) => sum + value, 0) / samples.length;
    const max = Math.max(...samples);

    // Relaxed CI-friendly limits that still catch substantial interaction regressions.
    expect(avg).toBeLessThan(8);
    expect(max).toBeLessThan(20);

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
