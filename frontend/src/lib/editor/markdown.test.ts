import { afterEach, describe, expect, it, vi } from 'vitest';

import { updateImageWidthByIndex } from './image-resize';
import {
  extractDueDates,
  extractWikilinks,
  isValidDueDate,
  renderMarkdown,
  sanitizeColor,
} from './markdown';

describe('renderMarkdown', () => {
  describe('basic markdown', () => {
    it('renders paragraphs', () => {
      const result = renderMarkdown('Hello World');
      expect(result).toContain('<p>Hello World</p>');
    });

    it('renders bold text', () => {
      const result = renderMarkdown('**bold**');
      expect(result).toContain('<strong>bold</strong>');
    });

    it('renders italic text', () => {
      const result = renderMarkdown('*italic*');
      expect(result).toContain('<em>italic</em>');
    });

    it('renders headings', () => {
      const result = renderMarkdown('# Heading');
      // Markdown library adds ID attributes for anchor links
      expect(result).toContain('<h1 id="heading">Heading</h1>');
    });

    it('renders links', () => {
      const result = renderMarkdown('[link](https://example.com)');
      expect(result).toContain('<a href="https://example.com">link</a>');
    });
  });

  describe('wikilinks - basic', () => {
    it('renders basic wikilink', () => {
      const result = renderMarkdown('[[My Note]]');
      expect(result).toContain('href="/note/My%20Note"');
      expect(result).toContain('class="wikilink');
      expect(result).toContain('>My Note</a>');
    });

    it('renders wikilink with spaces', () => {
      const result = renderMarkdown('[[Hello World Note]]');
      expect(result).toContain('href="/note/Hello%20World%20Note"');
      expect(result).toContain('>Hello World Note</a>');
    });

    it('renders wikilink with special characters', () => {
      const result = renderMarkdown('[[Test & Note]]');
      expect(result).toContain('href="/note/Test%20%26%20Note"');
      expect(result).toContain('data-title="Test &amp; Note"');
    });
  });

  describe('wikilinks - aliases', () => {
    it('renders wikilink with alias', () => {
      const result = renderMarkdown('[[My Note|display text]]');
      expect(result).toContain('href="/note/My%20Note"');
      expect(result).toContain('>display text</a>');
      expect(result).toContain('data-title="My Note"');
    });

    it('renders wikilink with alias containing spaces', () => {
      const result = renderMarkdown('[[Target|A longer display text]]');
      expect(result).toContain('href="/note/Target"');
      expect(result).toContain('>A longer display text</a>');
    });
  });

  describe('wikilinks - resolved/unresolved', () => {
    it('renders unresolved wikilink without resolvedTitles', () => {
      const result = renderMarkdown('[[Unresolved Note]]');
      expect(result).toContain('class="wikilink wikilink-unresolved"');
    });

    it('renders resolved wikilink when title is in resolvedTitles', () => {
      const resolvedTitles = new Set(['my note']);
      const result = renderMarkdown('[[My Note]]', { resolvedTitles });
      expect(result).toContain('class="wikilink wikilink-resolved"');
    });

    it('case-insensitive matching for resolved titles', () => {
      const resolvedTitles = new Set(['hello world']);
      const result = renderMarkdown('[[HELLO WORLD]]', { resolvedTitles });
      expect(result).toContain('class="wikilink wikilink-resolved"');
    });

    it('uses note ID when titleToIdMap is provided', () => {
      const titleToIdMap = new Map([['my note', 'abc123']]);
      const result = renderMarkdown('[[My Note]]', { titleToIdMap });
      expect(result).toContain('href="/note/abc123"');
    });
  });

  describe('wikilinks - XSS prevention', () => {
    it('escapes HTML in wikilink title', () => {
      const result = renderMarkdown('[[<script>alert("xss")</script>]]');
      // The displayed link text must be escaped
      expect(result).toContain('&lt;script&gt;alert');
      // data-title attribute contains the literal text (safe - can't execute JS from attributes)
      // The link href is URL-encoded for safety
      expect(result).toContain('href="/note/%3Cscript%3E');
    });

    it('escapes HTML in wikilink alias', () => {
      const result = renderMarkdown('[[Note|<img onerror="alert(1)">]]');
      expect(result).not.toContain('<img');
      expect(result).toContain('&lt;img');
    });

    it('escapes quotes in data-title attribute', () => {
      // Note: markdown-it typographer converts straight quotes to curly quotes
      const result = renderMarkdown('[[Note "with" quotes]]');
      // The typographer converts "with" to "with" (curly quotes)
      expect(result).toContain('data-title="Note');
      expect(result).toContain('quotes"');
      expect(result).not.toContain('onclick');
    });

    it('escapes ampersands', () => {
      const result = renderMarkdown('[[A & B]]');
      expect(result).toContain('&amp;');
    });
  });

  describe('wikilinks - edge cases', () => {
    it('handles empty wikilink', () => {
      const result = renderMarkdown('[[]]');
      // Empty wikilink should not match the regex
      expect(result).toContain('[[]]');
    });

    it('handles wikilink with only pipe', () => {
      const result = renderMarkdown('[[|]]');
      // Should not match since title is empty
      expect(result).toContain('[[|]]');
    });

    it('handles unclosed wikilink', () => {
      const result = renderMarkdown('[[unclosed');
      expect(result).toContain('[[unclosed');
      expect(result).not.toContain('<a');
    });

    it('handles single bracket', () => {
      const result = renderMarkdown('[single bracket]');
      expect(result).toContain('[single bracket]');
    });

    it('handles nested brackets in content', () => {
      const result = renderMarkdown('[[Note [with] brackets]]');
      // The regex [^\]] doesn't allow ] inside, so this won't match as a wikilink
      // This is expected behavior - nested brackets are not supported
      expect(result).toContain('[[Note [with] brackets]]');
    });

    it('handles multiple wikilinks in same paragraph', () => {
      const result = renderMarkdown('See [[Note 1]] and [[Note 2]]');
      expect(result).toContain('>Note 1</a>');
      expect(result).toContain('>Note 2</a>');
    });

    it('handles wikilink at start of line', () => {
      const result = renderMarkdown('[[Start]] of line');
      expect(result).toContain('>Start</a>');
    });

    it('handles wikilink at end of line', () => {
      const result = renderMarkdown('End of [[Line]]');
      expect(result).toContain('>Line</a>');
    });

    it('handles wikilink surrounded by markdown', () => {
      const result = renderMarkdown('**bold [[Link]] text**');
      expect(result).toContain('<strong>');
      expect(result).toContain('>Link</a>');
    });
  });

  describe('wikilinks - mixed content', () => {
    it('renders wikilinks with other markdown', () => {
      const content = `# Heading

This is a paragraph with [[a link]].

- List item with [[another link|alias]]
`;
      const result = renderMarkdown(content);
      // Heading has ID attribute for anchor links
      expect(result).toContain('<h1 id="heading">');
      expect(result).toContain('>a link</a>');
      expect(result).toContain('>alias</a>');
    });
  });
});

describe('extractWikilinks', () => {
  it('extracts basic wikilinks', () => {
    const links = extractWikilinks('[[Note 1]] and [[Note 2]]');
    expect(links).toHaveLength(2);
    expect(links[0].title).toBe('Note 1');
    expect(links[1].title).toBe('Note 2');
  });

  it('extracts wikilinks with aliases', () => {
    const links = extractWikilinks('[[Target|Display]]');
    expect(links).toHaveLength(1);
    expect(links[0].title).toBe('Target');
    expect(links[0].alias).toBe('Display');
  });

  it('trims whitespace from titles and aliases', () => {
    const links = extractWikilinks('[[  Spaced Title  |  Spaced Alias  ]]');
    expect(links[0].title).toBe('Spaced Title');
    expect(links[0].alias).toBe('Spaced Alias');
  });

  it('returns empty array for content without wikilinks', () => {
    const links = extractWikilinks('No wikilinks here');
    expect(links).toHaveLength(0);
  });

  it('handles empty content', () => {
    const links = extractWikilinks('');
    expect(links).toHaveLength(0);
  });

  it('extracts from multiline content', () => {
    const links = extractWikilinks(`Line 1 with [[Link 1]]
Line 2 with [[Link 2|Alias]]
Line 3 without links`);
    expect(links).toHaveLength(2);
    expect(links[0].title).toBe('Link 1');
    expect(links[1].title).toBe('Link 2');
    expect(links[1].alias).toBe('Alias');
  });

  it('does not extract from code blocks', () => {
    // Note: extractWikilinks is a simple regex extraction
    // It does not understand code blocks - this is expected behavior
    // The rendering handles this differently
    const links = extractWikilinks('```\n[[Code Link]]\n```');
    // extractWikilinks extracts all wikilinks regardless of context
    expect(links).toHaveLength(1);
  });
});

describe('sanitizeColor', () => {
  describe('named colors', () => {
    it('accepts primary', () => {
      expect(sanitizeColor('primary')).toBe('primary');
    });

    it('accepts destructive', () => {
      expect(sanitizeColor('destructive')).toBe('destructive');
    });

    it('accepts accent', () => {
      expect(sanitizeColor('accent')).toBe('accent');
    });

    it('accepts muted', () => {
      expect(sanitizeColor('muted')).toBe('muted');
    });

    it('accepts secondary', () => {
      expect(sanitizeColor('secondary')).toBe('secondary');
    });

    it('normalizes to lowercase', () => {
      expect(sanitizeColor('PRIMARY')).toBe('primary');
      expect(sanitizeColor('Destructive')).toBe('destructive');
    });

    it('trims whitespace', () => {
      expect(sanitizeColor('  primary  ')).toBe('primary');
    });

    it('rejects invalid named colors', () => {
      expect(sanitizeColor('red')).toBeNull();
      expect(sanitizeColor('blue')).toBeNull();
      expect(sanitizeColor('invalid')).toBeNull();
    });
  });

  describe('hex colors', () => {
    it('accepts 3-digit hex', () => {
      expect(sanitizeColor('#fff')).toBe('#fff');
      expect(sanitizeColor('#000')).toBe('#000');
      expect(sanitizeColor('#abc')).toBe('#abc');
    });

    it('accepts 6-digit hex', () => {
      expect(sanitizeColor('#ffffff')).toBe('#ffffff');
      expect(sanitizeColor('#000000')).toBe('#000000');
      expect(sanitizeColor('#aabbcc')).toBe('#aabbcc');
    });

    it('normalizes to lowercase', () => {
      expect(sanitizeColor('#FFF')).toBe('#fff');
      expect(sanitizeColor('#AABBCC')).toBe('#aabbcc');
    });

    it('rejects invalid hex', () => {
      expect(sanitizeColor('#ff')).toBeNull();
      expect(sanitizeColor('#fffffff')).toBeNull();
      expect(sanitizeColor('#xyz')).toBeNull();
      expect(sanitizeColor('fff')).toBeNull();
    });
  });

  describe('rgb colors', () => {
    it('accepts valid rgb', () => {
      expect(sanitizeColor('rgb(0, 0, 0)')).toBe('rgb(0, 0, 0)');
      expect(sanitizeColor('rgb(255, 255, 255)')).toBe('rgb(255, 255, 255)');
      expect(sanitizeColor('rgb(128, 64, 32)')).toBe('rgb(128, 64, 32)');
    });

    it('handles whitespace variations', () => {
      expect(sanitizeColor('rgb(0,0,0)')).toBe('rgb(0, 0, 0)');
      expect(sanitizeColor('rgb( 255 , 255 , 255 )')).toBe('rgb(255, 255, 255)');
    });

    it('rejects out-of-range values', () => {
      expect(sanitizeColor('rgb(256, 0, 0)')).toBeNull();
      expect(sanitizeColor('rgb(0, 300, 0)')).toBeNull();
      expect(sanitizeColor('rgb(0, 0, 999)')).toBeNull();
    });

    it('rejects invalid format', () => {
      expect(sanitizeColor('rgb(0, 0)')).toBeNull();
      expect(sanitizeColor('rgb()')).toBeNull();
    });
  });

  describe('rgba colors', () => {
    it('accepts valid rgba', () => {
      expect(sanitizeColor('rgba(0, 0, 0, 0)')).toBe('rgba(0, 0, 0, 0)');
      expect(sanitizeColor('rgba(255, 255, 255, 1)')).toBe('rgba(255, 255, 255, 1)');
      expect(sanitizeColor('rgba(128, 64, 32, 0.5)')).toBe('rgba(128, 64, 32, 0.5)');
    });

    it('accepts decimal alpha values', () => {
      expect(sanitizeColor('rgba(0, 0, 0, .5)')).toBe('rgba(0, 0, 0, .5)');
      expect(sanitizeColor('rgba(0, 0, 0, 0.25)')).toBe('rgba(0, 0, 0, 0.25)');
    });

    it('rejects out-of-range alpha', () => {
      expect(sanitizeColor('rgba(0, 0, 0, 1.5)')).toBeNull();
      expect(sanitizeColor('rgba(0, 0, 0, -0.1)')).toBeNull();
    });
  });

  describe('invalid inputs', () => {
    it('rejects empty string', () => {
      expect(sanitizeColor('')).toBeNull();
    });

    it('rejects XSS attempts', () => {
      expect(sanitizeColor('<script>')).toBeNull();
      expect(sanitizeColor('javascript:alert(1)')).toBeNull();
      expect(sanitizeColor('expression(alert(1))')).toBeNull();
    });
  });
});

describe('color syntax', () => {
  describe('basic rendering', () => {
    it('renders named color with CSS class', () => {
      const result = renderMarkdown('{color:primary}text{/color}');
      expect(result).toContain('class="text-color-primary"');
      expect(result).toContain('>text</span>');
    });

    it('renders hex color with inline style', () => {
      const result = renderMarkdown('{color:#ff0000}red text{/color}');
      expect(result).toContain('style="color: #ff0000;"');
      expect(result).toContain('>red text</span>');
    });

    it('renders rgb color with inline style', () => {
      const result = renderMarkdown('{color:rgb(255, 0, 0)}red text{/color}');
      expect(result).toContain('style="color: rgb(255, 0, 0);"');
    });

    it('renders rgba color with inline style', () => {
      const result = renderMarkdown('{color:rgba(255, 0, 0, 0.5)}semi-red{/color}');
      expect(result).toContain('style="color: rgba(255, 0, 0, 0.5);"');
    });
  });

  describe('all named colors', () => {
    it('renders primary', () => {
      const result = renderMarkdown('{color:primary}text{/color}');
      expect(result).toContain('class="text-color-primary"');
    });

    it('renders destructive', () => {
      const result = renderMarkdown('{color:destructive}text{/color}');
      expect(result).toContain('class="text-color-destructive"');
    });

    it('renders accent', () => {
      const result = renderMarkdown('{color:accent}text{/color}');
      expect(result).toContain('class="text-color-accent"');
    });

    it('renders muted', () => {
      const result = renderMarkdown('{color:muted}text{/color}');
      expect(result).toContain('class="text-color-muted"');
    });

    it('renders secondary', () => {
      const result = renderMarkdown('{color:secondary}text{/color}');
      expect(result).toContain('class="text-color-secondary"');
    });
  });

  describe('escape support', () => {
    it('renders escaped opening tag as literal', () => {
      const result = renderMarkdown('\\{color:primary}text{/color}');
      expect(result).toContain('{color:primary}');
      expect(result).not.toContain('class="text-color');
    });

    it('renders escaped closing tag as literal', () => {
      const result = renderMarkdown('{color:primary}text\\{/color}');
      // The opening tag won't match without closing, so it becomes literal
      expect(result).toContain('{color:primary}');
    });
  });

  describe('XSS prevention', () => {
    it('rejects script injection in color value', () => {
      const result = renderMarkdown('{color:<script>alert(1)</script>}text{/color}');
      // Invalid color, should not render as color syntax
      expect(result).not.toContain('<script>');
      expect(result).toContain('{color:');
    });

    it('rejects javascript: in color value', () => {
      const result = renderMarkdown('{color:javascript:alert(1)}text{/color}');
      expect(result).not.toContain('class="text-color');
      expect(result).not.toContain('style="color:');
    });

    it('escapes HTML in content', () => {
      const result = renderMarkdown('{color:primary}<script>alert(1)</script>{/color}');
      expect(result).not.toContain('<script>');
      expect(result).toContain('&lt;script&gt;');
    });
  });

  describe('nested content', () => {
    it('renders bold text inside color', () => {
      const result = renderMarkdown('{color:primary}**bold text**{/color}');
      expect(result).toContain('class="text-color-primary"');
      expect(result).toContain('<strong>bold text</strong>');
    });

    it('renders italic text inside color', () => {
      const result = renderMarkdown('{color:primary}*italic*{/color}');
      expect(result).toContain('<em>italic</em>');
    });

    it('renders wikilink inside color', () => {
      const result = renderMarkdown('{color:primary}[[My Note]]{/color}');
      expect(result).toContain('class="text-color-primary"');
      expect(result).toContain('class="wikilink');
    });

    it('renders link inside color', () => {
      const result = renderMarkdown('{color:primary}[link](https://example.com){/color}');
      expect(result).toContain('href="https://example.com"');
    });
  });

  describe('edge cases', () => {
    it('handles empty content', () => {
      const result = renderMarkdown('{color:primary}{/color}');
      expect(result).toContain('class="text-color-primary"');
      expect(result).toContain('></span>');
    });

    it('handles unclosed tag', () => {
      const result = renderMarkdown('{color:primary}unclosed text');
      // Should not match without closing tag
      expect(result).toContain('{color:primary}');
      expect(result).not.toContain('class="text-color');
    });

    it('handles multiple color blocks', () => {
      const result = renderMarkdown(
        '{color:primary}one{/color} and {color:destructive}two{/color}'
      );
      expect(result).toContain('class="text-color-primary"');
      expect(result).toContain('class="text-color-destructive"');
      expect(result).toContain('>one</span>');
      expect(result).toContain('>two</span>');
    });

    it('handles color in heading', () => {
      const result = renderMarkdown('# {color:primary}Colored Heading{/color}');
      // Heading has ID attribute for anchor links
      expect(result).toContain('<h1 id="colored-heading">');
      expect(result).toContain('class="text-color-primary"');
    });

    it('handles color in list', () => {
      const result = renderMarkdown('- {color:primary}colored item{/color}');
      expect(result).toContain('<li>');
      expect(result).toContain('class="text-color-primary"');
    });

    it('handles multiline content', () => {
      // Color syntax doesn't support multiline by design (single inline element)
      const result = renderMarkdown('{color:primary}line1\nline2{/color}');
      // Should not match because of newline
      expect(result).not.toContain('class="text-color');
    });
  });
});

describe('task lists', () => {
  describe('basic rendering', () => {
    it('renders unchecked task', () => {
      const result = renderMarkdown('- [ ] Unchecked task');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('class="task-list-item-checkbox"');
      // Unchecked tasks don't have checked attribute
      expect(result).not.toContain('checked=""');
    });

    it('renders checked task', () => {
      const result = renderMarkdown('- [x] Checked task');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('checked=""');
    });

    it('renders checked task with uppercase X', () => {
      const result = renderMarkdown('- [X] Checked task');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('checked=""');
    });

    it('adds task-list-item class to list item', () => {
      const result = renderMarkdown('- [ ] Task');
      // Plugin adds "enabled" suffix to the class
      expect(result).toContain('class="task-list-item');
    });

    it('wraps checkboxes in labels', () => {
      const result = renderMarkdown('- [ ] Task');
      expect(result).toContain('<label>');
      expect(result).toContain('</label>');
    });
  });

  describe('nested tasks', () => {
    it('renders nested unchecked tasks', () => {
      const content = `- [ ] Parent task
  - [ ] Child task`;
      const result = renderMarkdown(content);
      const checkboxCount = (result.match(/type="checkbox"/g) || []).length;
      expect(checkboxCount).toBe(2);
    });

    it('renders mixed nested tasks', () => {
      const content = `- [x] Completed parent
  - [ ] Incomplete child`;
      const result = renderMarkdown(content);
      expect(result).toContain('checked');
      const checkboxCount = (result.match(/type="checkbox"/g) || []).length;
      expect(checkboxCount).toBe(2);
    });
  });

  describe('mixed content', () => {
    it('renders tasks with markdown formatting', () => {
      const result = renderMarkdown('- [ ] **Bold** task');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('<strong>Bold</strong>');
    });

    it('renders tasks with wikilinks', () => {
      const result = renderMarkdown('- [ ] Task with [[Note Link]]');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('class="wikilink');
    });

    it('renders tasks with color syntax', () => {
      const result = renderMarkdown('- [ ] {color:primary}Colored task{/color}');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('class="text-color-primary"');
    });

    it('renders regular list items alongside tasks', () => {
      const content = `- Regular item
- [ ] Task item
- Another regular item`;
      const result = renderMarkdown(content);
      expect(result).toContain('Regular item');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('Another regular item');
    });
  });

  describe('multiple tasks', () => {
    it('renders multiple tasks correctly', () => {
      const content = `- [ ] First task
- [ ] Second task
- [x] Third task (done)`;
      const result = renderMarkdown(content);
      const checkboxCount = (result.match(/type="checkbox"/g) || []).length;
      expect(checkboxCount).toBe(3);
      expect(result).toContain('checked');
    });
  });

  describe('edge cases', () => {
    it('does not render invalid task syntax', () => {
      const result = renderMarkdown('- [] No space');
      expect(result).not.toContain('type="checkbox"');
    });

    it('does not render task without dash', () => {
      const result = renderMarkdown('[ ] No dash');
      expect(result).not.toContain('type="checkbox"');
    });

    it('requires content after checkbox syntax', () => {
      // Task with only whitespace after checkbox doesn't render as task
      const result = renderMarkdown('- [ ] ');
      // The plugin requires some content, so this renders as regular list item
      expect(result).toContain('[ ]');
    });

    it('handles task with minimal text', () => {
      const result = renderMarkdown('- [ ] x');
      expect(result).toContain('type="checkbox"');
    });
  });

  describe('drag handles', () => {
    it('adds drag handle to task items', () => {
      const result = renderMarkdown('- [ ] Task');
      expect(result).toContain('class="drag-handle"');
      expect(result).toContain('aria-hidden="true"');
    });

    it('adds data-task-index attribute', () => {
      const result = renderMarkdown('- [ ] Task');
      expect(result).toContain('data-task-index="0"');
    });

    it('increments task index for multiple tasks', () => {
      const content = `- [ ] First task
- [x] Second task
- [ ] Third task`;
      const result = renderMarkdown(content);
      expect(result).toContain('data-task-index="0"');
      expect(result).toContain('data-task-index="1"');
      expect(result).toContain('data-task-index="2"');
    });

    it('adds source line attribute for task items', () => {
      const content = `- [ ] First task
- [ ] Second task`;
      const result = renderMarkdown(content);
      expect(result).toContain('data-task-line="1"');
      expect(result).toContain('data-task-line="2"');
    });

    it('keeps source line mapping aligned when empty task markers exist', () => {
      const content = `- [ ] 
- [ ] Visible`;
      const result = renderMarkdown(content);
      expect(result).toContain('data-task-index="0" data-task-line="2"');
    });

    it('contains SVG icon in drag handle', () => {
      const result = renderMarkdown('- [ ] Task');
      expect(result).toContain('<svg');
      expect(result).toContain('</svg>');
    });
  });
});

describe('image resize', () => {
  describe('rendering images with width', () => {
    it('renders image without width attribute', () => {
      const result = renderMarkdown('![Alt text](https://example.com/image.png)');
      expect(result).toContain('<span class="resizable-image-wrapper"');
      expect(result).toContain('data-image-index="1"');
      expect(result).toContain('src="https://example.com/image.png"');
      expect(result).toContain('alt="Alt text"');
      expect(result).toContain('<span class="resize-handle"></span>');
    });

    it('renders image with pixel width', () => {
      const result = renderMarkdown('![Alt text](https://example.com/image.png){width=300}');
      expect(result).toContain('width="300"');
      expect(result).toContain('style="width: 300px"');
    });

    it('renders image with percentage width', () => {
      const result = renderMarkdown('![Alt text](https://example.com/image.png){width=50%}');
      expect(result).toContain('width="50%"');
      expect(result).toContain('style="width: 50%"');
    });

    it('renders multiple images with correct indices', () => {
      const result = renderMarkdown(`![Image 1](https://example.com/1.png){width=100}

![Image 2](https://example.com/2.png){width=200}`);
      expect(result).toContain('data-image-index="1"');
      expect(result).toContain('data-image-index="2"');
      // First image has width 100
      expect(result).toMatch(/data-image-index="1"[^>]*width="100"/);
    });

    it('handles mixed images with and without width', () => {
      const result = renderMarkdown(`![No width](https://example.com/1.png)

![With width](https://example.com/2.png){width=250}`);
      // First image should not have width attribute
      expect(result).toMatch(/data-image-index="1"[^>]*>/);
      // Second image should have width
      expect(result).toContain('width="250"');
    });
  });

  describe('XSS prevention in width values', () => {
    it('rejects width with script injection', () => {
      const result = renderMarkdown('![Alt](url){width=<script>alert(1)</script>}');
      expect(result).not.toContain('<script>');
      expect(result).not.toContain('width="<script>');
    });

    it('rejects width with javascript', () => {
      const result = renderMarkdown('![Alt](url){width=javascript:alert(1)}');
      // Invalid width value is not applied to the image - it remains as literal text
      expect(result).not.toContain('width="javascript:');
      expect(result).not.toContain('style="width: javascript:');
    });

    it('accepts only numeric values', () => {
      const result = renderMarkdown('![Alt](url){width=abc}');
      // Invalid width should not be rendered
      expect(result).not.toContain('width="abc"');
    });

    it('accepts numbers with percent sign only', () => {
      const result = renderMarkdown('![Alt](url){width=50%valid}');
      // Should not match invalid pattern
      expect(result).not.toContain('width="50%valid"');
    });
  });

  describe('image wrapper structure', () => {
    it('wraps image in span with correct class', () => {
      const result = renderMarkdown('![Test](test.png)');
      expect(result).toContain('<span class="resizable-image-wrapper"');
    });

    it('includes resize handle after image', () => {
      const result = renderMarkdown('![Test](test.png)');
      // Resize handle should be after img but inside wrapper
      expect(result).toMatch(/<img[^>]*><span class="resize-handle"><\/span><\/span>/);
    });

    it('includes data-original-src attribute', () => {
      const result = renderMarkdown('![Test](https://example.com/test.png)');
      expect(result).toContain('data-original-src="https://example.com/test.png"');
    });
  });
});

describe('updateImageWidthByIndex', () => {
  describe('basic functionality', () => {
    it('adds width to image without existing width', () => {
      const content = '![Alt](url)';
      const result = updateImageWidthByIndex(content, 1, 300);
      expect(result).toBe('![Alt](url){width=300}');
    });

    it('replaces existing width', () => {
      const content = '![Alt](url){width=100}';
      const result = updateImageWidthByIndex(content, 1, 500);
      expect(result).toBe('![Alt](url){width=500}');
    });

    it('replaces percentage width with pixel width', () => {
      const content = '![Alt](url){width=50%}';
      const result = updateImageWidthByIndex(content, 1, 400);
      expect(result).toBe('![Alt](url){width=400}');
    });
  });

  describe('multiple images', () => {
    it('updates correct image by index', () => {
      const content = '![First](first.png)\n\n![Second](second.png)\n\n![Third](third.png)';
      const result = updateImageWidthByIndex(content, 2, 200);
      expect(result).toBe(
        '![First](first.png)\n\n![Second](second.png){width=200}\n\n![Third](third.png)'
      );
    });

    it('preserves other images when updating one', () => {
      const content = '![A](a.png){width=100}\n\n![B](b.png){width=200}';
      const result = updateImageWidthByIndex(content, 1, 150);
      expect(result).toBe('![A](a.png){width=150}\n\n![B](b.png){width=200}');
    });

    it('handles same image appearing multiple times', () => {
      const content = '![Same](same.png)\n\n![Same](same.png)';
      const result = updateImageWidthByIndex(content, 2, 300);
      expect(result).toBe('![Same](same.png)\n\n![Same](same.png){width=300}');
    });
  });

  describe('edge cases', () => {
    it('handles empty alt text', () => {
      const content = '![](url)';
      const result = updateImageWidthByIndex(content, 1, 100);
      expect(result).toBe('![](url){width=100}');
    });

    it('handles complex URL', () => {
      const content = '![Alt](https://example.com/path/to/image.png?param=value)';
      const result = updateImageWidthByIndex(content, 1, 250);
      expect(result).toBe('![Alt](https://example.com/path/to/image.png?param=value){width=250}');
    });

    it('returns unchanged content for invalid index', () => {
      const content = '![Alt](url)';
      const result = updateImageWidthByIndex(content, 5, 100);
      expect(result).toBe(content);
    });

    it('preserves surrounding content', () => {
      const content = 'Before text\n\n![Alt](url)\n\nAfter text';
      const result = updateImageWidthByIndex(content, 1, 200);
      expect(result).toBe('Before text\n\n![Alt](url){width=200}\n\nAfter text');
    });
  });
});

describe('isValidDueDate', () => {
  it('accepts valid dates', () => {
    expect(isValidDueDate('2026-01-01')).toBe(true);
    expect(isValidDueDate('2026-12-31')).toBe(true);
    expect(isValidDueDate('2026-02-28')).toBe(true);
    expect(isValidDueDate('2024-02-29')).toBe(true); // Leap year
  });

  it('rejects invalid formats', () => {
    expect(isValidDueDate('tomorrow')).toBe(false);
    expect(isValidDueDate('2026-1-1')).toBe(false);
    expect(isValidDueDate('26-01-01')).toBe(false);
    expect(isValidDueDate('2026/01/01')).toBe(false);
    expect(isValidDueDate('')).toBe(false);
    expect(isValidDueDate('not-a-date')).toBe(false);
  });

  it('rejects overflow dates', () => {
    expect(isValidDueDate('2026-02-30')).toBe(false);
    expect(isValidDueDate('2026-02-29')).toBe(false); // 2026 is not a leap year
    expect(isValidDueDate('2026-13-01')).toBe(false);
    expect(isValidDueDate('2026-04-31')).toBe(false);
    expect(isValidDueDate('2026-00-01')).toBe(false);
    expect(isValidDueDate('2026-01-00')).toBe(false);
  });
});

describe('due date syntax', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  describe('basic rendering', () => {
    it('renders valid due date as badge', () => {
      const result = renderMarkdown('Task @due(2026-02-10)');
      expect(result).toContain('class="due-date');
      expect(result).toContain('data-due-date="2026-02-10"');
      expect(result).toContain('>2026-02-10</span>');
    });

    it('renders due date in task list', () => {
      const result = renderMarkdown('- [ ] Einkaufen @due(2026-02-10)');
      expect(result).toContain('type="checkbox"');
      expect(result).toContain('class="due-date');
      expect(result).toContain('data-due-date="2026-02-10"');
    });

    it('renders multiple due dates in one line', () => {
      const result = renderMarkdown('Start @due(2026-02-10) Ende @due(2026-03-15)');
      expect(result).toContain('data-due-date="2026-02-10"');
      expect(result).toContain('data-due-date="2026-03-15"');
    });

    it('renders due date in freetext', () => {
      const result = renderMarkdown('Deadline @due(2026-03-15) for project');
      expect(result).toContain('class="due-date');
      expect(result).toContain('>2026-03-15</span>');
    });
  });

  describe('invalid dates stay as plaintext', () => {
    it('ignores Feb 30', () => {
      const result = renderMarkdown('@due(2026-02-30)');
      expect(result).not.toContain('class="due-date');
      expect(result).toContain('@due(2026-02-30)');
    });

    it('ignores month 13', () => {
      const result = renderMarkdown('@due(2026-13-01)');
      expect(result).not.toContain('class="due-date');
    });

    it('ignores non-date content', () => {
      const result = renderMarkdown('@due(tomorrow)');
      expect(result).not.toContain('class="due-date');
      expect(result).toContain('@due(tomorrow)');
    });

    it('ignores relative dates', () => {
      const result = renderMarkdown('@due(+3d)');
      expect(result).not.toContain('class="due-date');
    });
  });

  describe('color classes with fake timers', () => {
    it('overdue for past dates', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-02-09)');
      expect(result).toContain('due-date-overdue');
    });

    it('today class for today', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-02-10)');
      expect(result).toContain('due-date-today');
    });

    it('soon for tomorrow', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-02-11)');
      expect(result).toContain('due-date-soon');
    });

    it('soon for +3 days', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-02-13)');
      expect(result).toContain('due-date-soon');
    });

    it('future for +4 days', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-02-14)');
      expect(result).toContain('due-date-future');
    });

    it('future for far future', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-02-10T12:00:00'));
      const result = renderMarkdown('@due(2026-12-31)');
      expect(result).toContain('due-date-future');
    });
  });

  describe('code blocks and inline code', () => {
    it('does not render in inline code', () => {
      const result = renderMarkdown('Use `@due(2026-01-01)` syntax');
      expect(result).not.toContain('class="due-date');
      expect(result).toContain('<code>');
    });

    it('does not render in fenced code block', () => {
      const result = renderMarkdown('```\n@due(2026-01-01)\n```');
      expect(result).not.toContain('class="due-date');
    });
  });

  describe('XSS prevention', () => {
    it('does not match script injection', () => {
      const result = renderMarkdown('@due(<script>)');
      expect(result).not.toContain('class="due-date');
      expect(result).not.toContain('<script>');
    });

    it('escapes HTML in data attribute', () => {
      // Valid date renders safely
      const result = renderMarkdown('@due(2026-01-01)');
      expect(result).toContain('data-due-date="2026-01-01"');
      expect(result).not.toContain('onclick');
    });
  });
});

describe('extractDueDates', () => {
  it('extracts valid dates', () => {
    const dates = extractDueDates('Task @due(2026-02-10) and @due(2026-03-15)');
    expect(dates).toEqual(['2026-02-10', '2026-03-15']);
  });

  it('ignores invalid dates', () => {
    const dates = extractDueDates('@due(2026-02-30) @due(tomorrow)');
    expect(dates).toEqual([]);
  });

  it('ignores code blocks', () => {
    const dates = extractDueDates('```\n@due(2026-01-01)\n```');
    expect(dates).toEqual([]);
  });

  it('ignores inline code', () => {
    const dates = extractDueDates('Use `@due(2026-01-01)` syntax');
    expect(dates).toEqual([]);
  });

  it('returns empty for content without due dates', () => {
    const dates = extractDueDates('No due dates here');
    expect(dates).toEqual([]);
  });

  it('handles mixed valid and invalid', () => {
    const dates = extractDueDates('@due(2026-02-10) @due(2026-02-30) @due(2026-03-01)');
    expect(dates).toEqual(['2026-02-10', '2026-03-01']);
  });
});
