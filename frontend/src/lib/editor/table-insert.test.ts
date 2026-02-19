import { describe, expect, it } from 'vitest';

import { buildMarkdownTable } from './table-insert';

describe('buildMarkdownTable', () => {
  it('creates a minimal 1x1 table', () => {
    const result = buildMarkdownTable(1, 1);
    const lines = result.split('\n');
    expect(lines).toHaveLength(3); // header + separator + 1 data row
    expect(lines[0]).toMatch(/^\|.*Spalte 1.*\|$/);
    expect(lines[1]).toMatch(/^\|.*---.*\|$/);
    expect(lines[2]).toMatch(/^\|.*\|$/);
  });

  it('creates a 2x3 table with correct dimensions', () => {
    const result = buildMarkdownTable(2, 3);
    const lines = result.split('\n');
    expect(lines).toHaveLength(4); // header + separator + 2 data rows

    // Header should have 3 columns
    const headerPipes = (lines[0].match(/\|/g) ?? []).length;
    expect(headerPipes).toBe(4); // 3 columns = 4 pipes

    // Separator should have 3 columns
    const sepPipes = (lines[1].match(/\|/g) ?? []).length;
    expect(sepPipes).toBe(4);

    // Data rows should have 3 columns each
    for (let i = 2; i < lines.length; i++) {
      const dataPipes = (lines[i].match(/\|/g) ?? []).length;
      expect(dataPipes).toBe(4);
    }
  });

  it('creates a 3x4 table', () => {
    const result = buildMarkdownTable(3, 4);
    const lines = result.split('\n');
    expect(lines).toHaveLength(5); // header + separator + 3 data rows

    // Check header has column names
    expect(lines[0]).toContain('Spalte 1');
    expect(lines[0]).toContain('Spalte 4');

    // Separator line
    expect(lines[1]).toMatch(/^\|( --- \|){4}$/);
  });

  it('has separator line with dashes', () => {
    const result = buildMarkdownTable(1, 2);
    const lines = result.split('\n');
    expect(lines[1]).toContain('---');
  });
});
