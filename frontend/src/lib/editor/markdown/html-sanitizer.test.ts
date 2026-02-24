import { describe, expect, it } from 'vitest';

import { sanitizeRenderedHtml, sanitizeSvg } from './html-sanitizer';

describe('sanitizeRenderedHtml', () => {
  it('preserves safe HTML elements', () => {
    const html = '<p>Hello <strong>world</strong></p>';
    expect(sanitizeRenderedHtml(html)).toContain('<strong>world</strong>');
  });

  it('strips script tags', () => {
    const html = '<p>Hello</p><script>alert("xss")</script>';
    const result = sanitizeRenderedHtml(html);
    expect(result).not.toContain('script');
    expect(result).not.toContain('alert');
  });

  it('preserves data attributes', () => {
    const html = '<p data-source-line="5">text</p>';
    expect(sanitizeRenderedHtml(html)).toContain('data-source-line="5"');
  });

  it('preserves loading and decoding attributes', () => {
    const html = '<img src="test.png" loading="lazy" decoding="async">';
    const result = sanitizeRenderedHtml(html);
    expect(result).toContain('loading="lazy"');
    expect(result).toContain('decoding="async"');
  });

  it('preserves MathML tags', () => {
    const html = '<math><mrow><mi>x</mi><mo>+</mo><mn>1</mn></mrow></math>';
    const result = sanitizeRenderedHtml(html);
    expect(result).toContain('<math>');
    expect(result).toContain('<mi>');
    expect(result).toContain('<mo>');
    expect(result).toContain('<mn>');
  });

  it('strips event handlers', () => {
    const html = '<p onclick="alert(1)">click me</p>';
    const result = sanitizeRenderedHtml(html);
    expect(result).not.toContain('onclick');
  });
});

describe('sanitizeSvg', () => {
  it('preserves safe SVG elements', () => {
    const svg = '<svg viewBox="0 0 100 100"><rect x="0" y="0" width="100" height="100"/></svg>';
    const result = sanitizeSvg(svg);
    expect(result).toContain('<svg');
    expect(result).toContain('<rect');
  });

  it('preserves text elements', () => {
    const svg = '<svg><text x="10" y="20">Hello</text></svg>';
    const result = sanitizeSvg(svg);
    expect(result).toContain('<text');
    expect(result).toContain('Hello');
  });

  it('preserves transform attributes', () => {
    const svg = '<svg><g transform="translate(10,20)"><rect/></g></svg>';
    const result = sanitizeSvg(svg);
    expect(result).toContain('transform');
  });

  it('strips script tags from SVG', () => {
    const svg = '<svg><script>alert("xss")</script><rect/></svg>';
    const result = sanitizeSvg(svg);
    expect(result).not.toContain('script');
    expect(result).not.toContain('alert');
  });

  it('strips event handlers from SVG', () => {
    const svg = '<svg onload="alert(1)"><rect onclick="alert(2)"/></svg>';
    const result = sanitizeSvg(svg);
    expect(result).not.toContain('onload');
    expect(result).not.toContain('onclick');
  });

  it('preserves marker attributes', () => {
    const svg = '<svg><line marker-end="url(#arrow)"/></svg>';
    const result = sanitizeSvg(svg);
    expect(result).toContain('marker-end');
  });
});
