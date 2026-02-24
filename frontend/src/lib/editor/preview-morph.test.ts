import { describe, expect, it } from 'vitest';

import { morphPreview } from './preview-morph';

describe('morphPreview', () => {
  it('updates innerHTML to match new content', () => {
    const container = document.createElement('div');
    container.innerHTML = '<p>old text</p>';

    morphPreview(container, '<p>new text</p>');

    expect(container.innerHTML).toContain('new text');
    expect(container.innerHTML).not.toContain('old text');
  });

  it('preserves <details> open state', () => {
    const container = document.createElement('div');
    container.innerHTML = '<details><summary>Title</summary><p>Content</p></details>';

    // Open the details element
    const details = container.querySelector('details')!;
    details.open = true;

    // Morph with same structure (closed by default in HTML)
    morphPreview(container, '<details><summary>Title</summary><p>Updated content</p></details>');

    const morphedDetails = container.querySelector('details')!;
    expect(morphedDetails.open).toBe(true);
    expect(morphedDetails.textContent).toContain('Updated content');
  });

  it('adds new elements', () => {
    const container = document.createElement('div');
    container.innerHTML = '<p>first</p>';

    morphPreview(container, '<p>first</p><p>second</p>');

    expect(container.querySelectorAll('p')).toHaveLength(2);
  });

  it('removes old elements', () => {
    const container = document.createElement('div');
    container.innerHTML = '<p>first</p><p>second</p>';

    morphPreview(container, '<p>first</p>');

    expect(container.querySelectorAll('p')).toHaveLength(1);
  });

  it('handles empty new content', () => {
    const container = document.createElement('div');
    container.innerHTML = '<p>content</p>';

    morphPreview(container, '');

    expect(container.children).toHaveLength(0);
  });

  it('handles empty old content', () => {
    const container = document.createElement('div');
    container.innerHTML = '';

    morphPreview(container, '<p>new content</p>');

    expect(container.querySelector('p')?.textContent).toBe('new content');
  });
});
