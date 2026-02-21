import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import CanvasToolbar from './CanvasToolbar.svelte';

describe('CanvasToolbar', () => {
  const tools = [
    { action: 'add-text', label: 'Text', shortcut: 'T' },
    { action: 'add-file', label: 'Note', shortcut: 'N' },
    { action: 'add-link', label: 'Link', shortcut: 'L' },
    { action: 'add-group', label: 'Group', shortcut: 'G' },
  ] as const;

  it('renders all 4 tool buttons', () => {
    const onAction = vi.fn();
    const { getByRole } = render(CanvasToolbar, { props: { onAction } });

    const toolbar = getByRole('toolbar', { name: 'Canvas tools' });
    expect(toolbar).toBeInTheDocument();

    for (const tool of tools) {
      const btn = getByRole('button', { name: `${tool.label} (${tool.shortcut})` });
      expect(btn).toBeInTheDocument();
    }
  });

  it.each(tools)('clicking $label button calls onAction with "$action"', async (tool) => {
    const onAction = vi.fn();
    const { getByRole } = render(CanvasToolbar, { props: { onAction } });

    const btn = getByRole('button', { name: `${tool.label} (${tool.shortcut})` });
    await fireEvent.click(btn);

    expect(onAction).toHaveBeenCalledTimes(1);
    expect(onAction).toHaveBeenCalledWith(tool.action);
  });

  it('sets draggable attribute on all buttons', () => {
    const onAction = vi.fn();
    const { getByRole } = render(CanvasToolbar, { props: { onAction } });

    for (const tool of tools) {
      const btn = getByRole('button', { name: `${tool.label} (${tool.shortcut})` });
      expect(btn).toHaveAttribute('draggable', 'true');
    }
  });

  it('sets correct MIME data on drag start', async () => {
    const onAction = vi.fn();
    const { getByRole } = render(CanvasToolbar, { props: { onAction } });

    const btn = getByRole('button', { name: 'Text (T)' });

    const setData = vi.fn();
    const dataTransfer = {
      effectAllowed: '',
      setData,
    };

    await fireEvent.dragStart(btn, { dataTransfer });

    expect(dataTransfer.effectAllowed).toBe('copy');
    expect(setData).toHaveBeenCalledWith('application/x-xelanote-canvas-tool', 'add-text');
    expect(setData).toHaveBeenCalledWith('text/plain', 'add-text');
  });

  it('has correct title attributes with shortcuts', () => {
    const onAction = vi.fn();
    const { getByRole } = render(CanvasToolbar, { props: { onAction } });

    for (const tool of tools) {
      const btn = getByRole('button', { name: `${tool.label} (${tool.shortcut})` });
      expect(btn).toHaveAttribute('title', `${tool.label} (${tool.shortcut})`);
    }
  });
});
